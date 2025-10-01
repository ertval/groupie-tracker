package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"groupie-tracker/internal/data"
	"groupie-tracker/internal/web"
)

func main() {
	log.Println("🎸 Starting Groupie Tracker server...")

	// Initialize data store
	store := data.NewStore()

	// Load data with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := store.LoadData(ctx); err != nil {
		log.Fatalf("❌ Failed to load data: %v", err)
	}

	log.Printf("✅ Data loaded successfully - %d artists, %d locations",
		len(store.Artists()), len(store.Locations()))

	// Create web server
	server, err := web.NewServer(store)
	if err != nil {
		log.Fatalf("❌ Failed to create server: %v", err)
	}

	// Setup graceful shutdown
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		log.Println("🛑 Shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("⚠️  Server shutdown error: %v", err)
		}
		os.Exit(0)
	}()

	// Start server
	log.Println("🚀 Server ready!")
	log.Println("🌐 Open: http://localhost:8082")
	log.Println("📊 Health check: http://localhost:8082/health")
	log.Println("Press Ctrl+C to stop")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
