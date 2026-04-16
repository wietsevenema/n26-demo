package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gorilla/websocket"
)

var (
	projectID  = "venema-2026-1"
	instanceID = ""
	upgrader   = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	animals = []string{"🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🐻", "🐼", "🐻‍❄️", "🐨", "🐯", "🦁", "🐮", "🐷", "🐸", "🐵", "🦄", "🐝", "🐙"}
	colors  = []string{"#1967D2", "#C5221F", "#F29900", "#188038", "#5F6368"}

	currentState    *ContainerState
	stateMutex      sync.Mutex
	docRef          *firestore.DocumentRef
	firestoreClient *firestore.Client

	lastTotalCPU uint64
	lastIdleCPU  uint64
)
type ContainerState struct {
	InstanceID    string    `firestore:"instance_id" json:"instance_id"`
	Emoji         string    `firestore:"emoji" json:"emoji"`
	Color         string    `firestore:"color" json:"color"`
	MemoryMB      int64     `firestore:"memory_mb" json:"memory_mb"`
	TotalMemoryMB int64     `firestore:"total_memory_mb" json:"total_memory_mb"`
	CPUUtil       float64   `firestore:"cpu_util" json:"cpu_util"`
	Region        string    `firestore:"region" json:"region"`
	ServiceName   string    `firestore:"service_name" json:"service_name"`
	RevisionName  string    `firestore:"revision_name" json:"revision_name"`
	Status        string    `firestore:"status" json:"status"`
	TTL           time.Time `firestore:"ttl" json:"-"`
}


func main() {
	ctx := context.Background()
	var err error

	instanceID = os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = fmt.Sprintf("inst-%04d", rand.Intn(10000))
	}

	envProjectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if envProjectID != "" {
		projectID = envProjectID
	}

	firestoreClient, err = firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create firestore client: %v", err)
	}
	defer firestoreClient.Close()

	docRef = firestoreClient.Collection("active_containers").Doc(instanceID)

	// Fetch cloud metadata
	region := getRegion()
	serviceName := os.Getenv("K_SERVICE")
	revisionName := os.Getenv("K_REVISION")

	// Initialize global state on boot (Neutral "Empty" state)
	currentState = &ContainerState{
		InstanceID:    instanceID,
		Emoji:         "📦",
		Color:         "#5F6368",
		Status:        "idle",
		MemoryMB:      getMemoryMB(),
		TotalMemoryMB: getMemoryLimitMB(),
		CPUUtil:       getCPUUtil(),
		Region:        region,
		ServiceName:   serviceName,
		RevisionName:  revisionName,
		TTL:           time.Now().Add(2 * time.Minute),
	}
	updateFirestore(ctx)


	// Global ticker for container metrics (belts and suspenders)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			stateMutex.Lock()
			currentState.MemoryMB = getMemoryMB()
			currentState.CPUUtil = getCPUUtil()
			currentState.TTL = time.Now().Add(2 * time.Minute)
			stateMutex.Unlock()
			updateFirestore(context.Background())
		}
	}()

	// Global Signal Handling for Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, cleaning up instance %s...", sig, instanceID)
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = docRef.Delete(cleanupCtx)
		os.Exit(0)
	}()

	http.HandleFunc("/ws", handleAttendee)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s (Instance: %s)", port, instanceID)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func updateFirestore(ctx context.Context) {
	stateMutex.Lock()
	stateCopy := *currentState
	stateMutex.Unlock()

	_, err := docRef.Set(ctx, &stateCopy)
	if err != nil {
		log.Printf("Failed to update firestore: %v", err)
	}
}

func handleAttendee(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Assign a vibrant identity only on connection
	stateMutex.Lock()
	currentState.Status = "connected"
	currentState.Emoji = animals[rand.Intn(len(animals))]
	currentState.Color = colors[rand.Intn(len(colors))]
	currentState.TTL = time.Now().Add(2 * time.Minute)
	stateMutex.Unlock()

	ctx := context.Background()
	updateFirestore(ctx)

	// Start a ticker to send back metrics to the attendee UI
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		_ = sendMetrics(conn) // Initial send
		for {
			select {
			case <-ticker.C:
				if err := sendMetrics(conn); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Revert to idle and reset identity completely on disconnect
	defer func() {
		close(done)
		stateMutex.Lock()
		currentState.Status = "idle"
		currentState.Emoji = "📦"
		currentState.Color = "#5F6368"
		currentState.TTL = time.Now().Add(2 * time.Minute)
		stateMutex.Unlock()
		updateFirestore(context.Background())
		log.Printf("WebSocket closed, reverted to neutral idle %s", instanceID)
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return // Exits and triggers defers
		}

		log.Printf("Received WS message: %s", string(message))

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("JSON unmarshal error: %v", err)
			continue
		}

		updated := false
		stateMutex.Lock()
		if emoji, ok := msg["emoji"].(string); ok {
			// Validate emoji is in allowed list
			valid := false
			for _, a := range animals {
				if a == emoji {
					valid = true
					break
				}
			}
			if valid {
				currentState.Emoji = emoji
				updated = true
			}
		}
		if color, ok := msg["color"].(string); ok {
			// Validate color is in allowed list
			valid := false
			for _, c := range colors {
				if c == color {
					valid = true
					break
				}
			}
			if valid {
				currentState.Color = color
				updated = true
			}
		}

		if updated {
			currentState.TTL = time.Now().Add(2 * time.Minute)
		}
		stateMutex.Unlock()

		if updated {
			updateFirestore(ctx)
			sendMetrics(conn)
		}
	}
}

func sendMetrics(conn *websocket.Conn) error {
	stateMutex.Lock()
	stateCopy := *currentState
	stateMutex.Unlock()

	memoryDisplay := fmt.Sprintf("%d MB", stateCopy.MemoryMB)
	if stateCopy.TotalMemoryMB > 0 {
		memoryDisplay = fmt.Sprintf("%d / %d MB", stateCopy.MemoryMB, stateCopy.TotalMemoryMB)
	}

	metricsHTML := fmt.Sprintf(`
		<div id="metrics" hx-swap-oob="innerHTML">
			<p>Instance: %s</p>
			<p>Memory: %s | CPU: %.1f%%</p>
			<p>Region: %s</p>
			<p>Service: %s</p>
			<p>Revision: %s</p>
			<p>Status: %s</p>
		</div>
		<style id="container-preview-style" hx-swap-oob="innerHTML">
			#container-preview { background-color: %s; }
		</style>
	`, stateCopy.InstanceID, memoryDisplay, stateCopy.CPUUtil, stateCopy.Region, stateCopy.ServiceName, stateCopy.RevisionName, stateCopy.Status, stateCopy.Color)
	metricsHTML = strings.ReplaceAll(metricsHTML, "\n", "")
	return conn.WriteMessage(websocket.TextMessage, []byte(metricsHTML))
}

func getMemoryMB() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc / 1024 / 1024)
}

func getMemoryLimitMB() int64 {
	// Try cgroup v2
	data, err := os.ReadFile("/sys/fs/cgroup/memory.max")
	if err != nil {
		// Try cgroup v1
		data, err = os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes")
	}
	if err == nil {
		limitStr := strings.TrimSpace(string(data))
		if limitStr != "max" {
			limit, err := strconv.ParseInt(limitStr, 10, 64)
			if err == nil && limit > 0 && limit < 1024*1024*1024*1024 {
				return limit / 1024 / 1024
			}
		}
	}
	return 0
}

func getCPUUtil() float64 {
	contents, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "cpu" {
			var user, nice, system, idle, iowait, irq, softirq, steal, guest, guest_nice uint64
			fmt.Sscanf(line, "cpu %d %d %d %d %d %d %d %d %d %d", &user, &nice, &system, &idle, &iowait, &irq, &softirq, &steal, &guest, &guest_nice)
			idleTime := idle + iowait
			totalTime := user + nice + system + idle + iowait + irq + softirq + steal

			if lastTotalCPU > 0 {
				idleDiff := float64(idleTime - lastIdleCPU)
				totalDiff := float64(totalTime - lastTotalCPU)
				util := 100.0 * (1.0 - idleDiff/totalDiff)
				lastIdleCPU = idleTime
				lastTotalCPU = totalTime
				return util
			}
			lastIdleCPU = idleTime
			lastTotalCPU = totalTime
			return 0
		}
	}
	return 0
}

func getRegion() string {
	region := os.Getenv("REGION")
	if region != "" {
		return region
	}
	client := &http.Client{Timeout: 1 * time.Second}
	req, _ := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/region", nil)
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		parts := strings.Split(string(b), "/")
		return parts[len(parts)-1]
	}
	return "unknown"
}
