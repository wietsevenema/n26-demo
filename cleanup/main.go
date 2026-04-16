package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

var (
	projectID = "venema-2026-1"
)

func main() {
	if envProjectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); envProjectID != "" {
		projectID = envProjectID
	}

	// Register multiple patterns to be safe
	http.HandleFunc("/", handleSweep)
	http.HandleFunc("/sweep", handleSweep)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Cleanup service starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleSweep(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received sweep request: %s %s", r.Method, r.URL.Path)
	
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Printf("Failed to create firestore client: %v", err)
		http.Error(w, "Firestore Error", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	now := time.Now()
	query := client.Collection("active_containers").Where("ttl", "<", now)
	iter := query.Documents(ctx)
	defer iter.Stop()

	batch := client.Batch()
	count := 0
	batchSize := 0

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Iteration error: %v", err)
			break
		}

		batch.Delete(doc.Ref)
		count++
		batchSize++

		if batchSize >= 400 {
			if _, err := batch.Commit(ctx); err != nil {
				log.Printf("Batch commit failed: %v", err)
			}
			batch = client.Batch()
			batchSize = 0
		}
	}

	if batchSize > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			log.Printf("Final batch commit failed: %v", err)
		}
	}

	log.Printf("Sweep completed. Cleaned up %d zombie containers.", count)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Cleaned up %d containers", count)
}
