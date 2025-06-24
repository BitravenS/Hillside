package main

import (
	"context"
	"hillside/internal/hub"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Configure logging
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetPrefix("[HUB-MAIN] ")

	log.Printf("Starting Hillside Hub Server...")
	log.Printf("Timestamp: %s", time.Now().Format(time.RFC3339))

	ctx := context.Background()
	h, err := hub.NewHubServer(ctx, "/ip4/0.0.0.0/tcp/4001")
	if err != nil {
		log.Fatalf("Failed to create hub server: %v", err)
	}

	log.Printf("Hub server created successfully")
	h.ListenAddrs()

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Printf("Hub server is running... Press Ctrl+C to stop")

	// Log periodic stats
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				connectedPeers := h.Host.Network().Peers()
				log.Printf("Hub status - Connected peers: %d", len(connectedPeers))
				for _, peerID := range connectedPeers {
					log.Printf("  Connected peer: %s", peerID.String())
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	<-c
	log.Printf("Received shutdown signal, stopping hub server...")

	if err := h.Host.Close(); err != nil {
		log.Printf("Error closing hub server: %v", err)
	} else {
		log.Printf("Hub server stopped gracefully")
	}
}