package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jordanhubbard/agenticorp/pkg/config"
	"github.com/jordanhubbard/agenticorp/pkg/server"
)

func main() {
	fmt.Println("Welcome to AgentiCorp - AI Coding Agent Orchestrator")
	fmt.Println("==================================================")

	// Load or create default configuration
	cfg := config.DefaultConfig()

	// Override with environment variables if set
	if temporalHost := os.Getenv("TEMPORAL_HOST"); temporalHost != "" {
		cfg.Temporal.Host = temporalHost
		log.Printf("Using Temporal host from environment: %s", temporalHost)
	}
	if temporalNamespace := os.Getenv("TEMPORAL_NAMESPACE"); temporalNamespace != "" {
		cfg.Temporal.Namespace = temporalNamespace
		log.Printf("Using Temporal namespace from environment: %s", temporalNamespace)
	}

	fmt.Println("\nAgentiCorp Worker System initialized")
	fmt.Println("See docs/WORKER_SYSTEM.md for usage information")

	// Start the server
	fmt.Println("\nStarting AgentiCorp server...")
	srv := server.NewServer(cfg)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
