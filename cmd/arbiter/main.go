package main

import (
"context"
"flag"
"fmt"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/jordanhubbard/arbiter/internal/api"
"github.com/jordanhubbard/arbiter/internal/arbiter"
"github.com/jordanhubbard/arbiter/pkg/config"
)

const version = "0.1.0"

func main() {
log.SetFlags(log.LstdFlags | log.Lshortfile)

// Parse command line flags
configPath := flag.String("config", "config.yaml", "Path to configuration file")
showVersion := flag.Bool("version", false, "Show version information")
showHelp := flag.Bool("help", false, "Show help message")
flag.Parse()

if *showVersion {
fmt.Printf("Arbiter v%s\n", version)
return
}

if *showHelp {
fmt.Printf("Arbiter v%s - Agentic Coding Orchestrator\n\n", version)
fmt.Println("Usage: arbiter [options]")
fmt.Println("\nOptions:")
flag.PrintDefaults()
return
}

fmt.Printf("Arbiter v%s - Agentic Coding Orchestrator\n", version)
fmt.Println("An agentic based coding orchestrator for both on-prem and off-prem development")
fmt.Println()

// Load configuration
cfg, err := config.LoadConfig(*configPath)
if err != nil {
log.Printf("Failed to load configuration from %s: %v", *configPath, err)
log.Println("Using default configuration...")
cfg = config.DefaultConfig()
}

// Create arbiter instance
arb := arbiter.New(cfg)

// Initialize arbiter
ctx := context.Background()
if err := arb.Initialize(ctx); err != nil {
log.Fatalf("Failed to initialize arbiter: %v", err)
}

log.Println("Arbiter initialized successfully")

// Start maintenance loop in background
go arb.StartMaintenanceLoop(ctx)

// Create API server
apiServer := api.NewServer(arb, cfg)
handler := apiServer.SetupRoutes()

// Start HTTP server
httpAddr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
server := &http.Server{
Addr:         httpAddr,
Handler:      handler,
ReadTimeout:  cfg.Server.ReadTimeout,
WriteTimeout: cfg.Server.WriteTimeout,
IdleTimeout:  cfg.Server.IdleTimeout,
}

// Start server in goroutine
go func() {
log.Printf("Starting HTTP server on %s", httpAddr)
if cfg.WebUI.Enabled {
log.Printf("Web UI available at http://localhost%s", httpAddr)
}
if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Fatalf("HTTP server failed: %v", err)
}
}()

// Setup graceful shutdown
stop := make(chan os.Signal, 1)
signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

<-stop
log.Println("Shutting down server...")

shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := server.Shutdown(shutdownCtx); err != nil {
log.Printf("Server shutdown error: %v", err)
}

log.Println("Server stopped")
}
