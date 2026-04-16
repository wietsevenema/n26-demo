package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
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
	colors  = []string{"#FFADAD", "#FFD6A5", "#FDFFB6", "#CAFFBF", "#9BF6FF", "#A0C4FF", "#BDB2FF", "#FFC6FF"}

	currentState    *ContainerState
	stateMutex      sync.Mutex
	docRef          *firestore.DocumentRef
	firestoreClient *firestore.Client
)

type ContainerState struct {
	InstanceID string    `firestore:"instance_id" json:"instance_id"`
	Emoji      string    `firestore:"emoji" json:"emoji"`
	Color      string    `firestore:"color" json:"color"`
	MemoryMB   int64     `firestore:"memory_mb" json:"memory_mb"`
	Status     string    `firestore:"status" json:"status"`
	LastUpdate time.Time `firestore:"last_update" json:"last_update"`
	TTL        time.Time `firestore:"ttl" json:"-"`
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

	// Initialize global state on boot
	currentState = &ContainerState{
		InstanceID: instanceID,
		Emoji:      "📦",
		Color:      "#e0e0e0",
		Status:     "idle",
		MemoryMB:   getMemoryMB(),
		LastUpdate: time.Now(),
		TTL:        time.Now().Add(2 * time.Minute),
	}
	updateFirestore(ctx)

	// Global ticker for container metrics (belts and suspenders)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			stateMutex.Lock()
			currentState.MemoryMB = getMemoryMB()
			currentState.LastUpdate = time.Now()
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

	// Update global state on connection
	stateMutex.Lock()
	currentState.Status = "connected"
	currentState.Emoji = animals[rand.Intn(len(animals))]
	currentState.Color = colors[rand.Intn(len(colors))]
	currentState.LastUpdate = time.Now()
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

	// Revert to idle on disconnect
	defer func() {
		close(done)
		stateMutex.Lock()
		currentState.Status = "idle"
		currentState.Emoji = "📦"
		currentState.Color = "#e0e0e0"
		currentState.LastUpdate = time.Now()
		currentState.TTL = time.Now().Add(2 * time.Minute)
		stateMutex.Unlock()
		updateFirestore(context.Background())
		log.Printf("WebSocket closed, reverted to idle %s", instanceID)
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
			currentState.Emoji = emoji
			updated = true
		}
		if color, ok := msg["color"].(string); ok {
			currentState.Color = color
			updated = true
		}

		if updated {
			currentState.LastUpdate = time.Now()
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

	metricsHTML := fmt.Sprintf(`
		<div id="metrics" hx-swap-oob="innerHTML">
			<p>Instance: %s</p>
			<p>Memory: %d MB</p>
			<p>Status: %s</p>
		</div>
		<style id="container-preview-style" hx-swap-oob="innerHTML">
			#container-preview { background-color: %s; }
		</style>
	`, stateCopy.InstanceID, stateCopy.MemoryMB, stateCopy.Status, stateCopy.Color)
	metricsHTML = strings.ReplaceAll(metricsHTML, "\n", "")
	return conn.WriteMessage(websocket.TextMessage, []byte(metricsHTML))
}

func getMemoryMB() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc / 1024 / 1024)
}
