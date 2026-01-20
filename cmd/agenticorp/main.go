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

	"github.com/jordanhubbard/agenticorp/internal/api"
	"github.com/jordanhubbard/agenticorp/internal/agenticorp"
	"github.com/jordanhubbard/agenticorp/pkg/config"
)

const version = "0.1.0"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	if *showVersion {
		fmt.Printf("AgentiCorp v%s\n", version)
		return
	}

	cfg, err := config.LoadConfigFromFile(*configPath)
	if err != nil {
		log.Fatalf("failed to load config from %s: %v", *configPath, err)
	}

	arb, err := agenticorp.New(cfg)
	if err != nil {
		log.Fatalf("failed to create agenticorp: %v", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := arb.Initialize(runCtx); err != nil {
		log.Fatalf("failed to initialize agenticorp: %v", err)
	}

	go arb.StartMaintenanceLoop(runCtx)
	if arb.GetTemporalManager() == nil {
		go arb.StartDispatchLoop(runCtx, 10*time.Second)
	}

	apiServer := api.NewServer(arb, cfg)
	handler := apiServer.SetupRoutes()

	httpSrv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		log.Printf("AgentiCorp API listening on %s", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = httpSrv.Shutdown(shutdownCtx)
	arb.Shutdown()

}

func printHelp() {
	fmt.Println("Usage: agenticorp [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -config   Path to configuration file (default: config.yaml)")
	fmt.Println("  -version  Show version information")
	fmt.Println("  -help     Show help message")
}
