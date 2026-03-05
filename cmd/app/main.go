package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prokoleso/etalon-nomenclature/config"
	"github.com/prokoleso/etalon-nomenclature/internal/db"
	"github.com/prokoleso/etalon-nomenclature/internal/service"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger (Development mode for detailed debugging)
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	logger.Info("Starting Etalon Nomenclature Service",
		zap.Int("mailboxes", len(cfg.Mailboxes)),
		zap.Duration("poll_interval", cfg.PollInterval))

	// Initialize database
	database, err := db.New(cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close()

	logger.Info("Database connection established")

	// Create processor service
	processor := service.NewProcessor(cfg, database, logger)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start processor in goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		processor.Run(ctx)
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	case <-done:
		logger.Info("Processor stopped")
	}

	// Wait for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	select {
	case <-done:
		logger.Info("Service stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Warn("Shutdown timeout exceeded")
	}
}
