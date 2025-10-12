package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/welasco/adguardfilter/adguardapi"
	logger "github.com/welasco/adguardfilter/common/logger"
	"github.com/welasco/adguardfilter/common/timer"
	"github.com/welasco/adguardfilter/transport"
)

func init() {
	file := os.Getenv("logPath") + os.Getenv("HOSTNAME") + ".log"
	logger.Init(file, os.Getenv("logLevel"))
}

func main() {
	logger.Info("[main][main] Starting AdguardFilter")
	app := transport.Setup()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	logger.Info("[main][main] Starting server on port " + port)

	// Create a channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Start server in a goroutine
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("[main][main] Server error: " + err.Error())
		}
	}()

	logger.Info("[main][main] Server started successfully. Press Ctrl+C to shutdown")

	// Block until we receive a signal
	sig := <-quit
	logger.Info("[main][main] Received shutdown signal: " + sig.String())
	logger.Info("[main][main] Initiating graceful shutdown...")

	// Stop all active timers
	activeTimers := timer.GetAllActiveTimers()
	if len(activeTimers) > 0 {
		logger.Info("[main][main] Stopping ", len(activeTimers), " active timer(s)")
		timer.StopAllTimers()
		logger.Info("[main][main] All timers stopped")

		// Reset blocked services to default
		err := adguardapi.ResetBlockedServices()
		if err != nil {
			logger.Error("[main][main] Failed to reset blocked services")
			logger.Error(err)
		} else {
			logger.Info("[main][main] Successfully reset blocked services to default")
		}

	} else {
		logger.Info("[main][main] No active timers to stop")
	}

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the server gracefully
	logger.Info("[main][main] Shutting down HTTP server...")
	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("[main][main] Server shutdown error: " + err.Error())
	} else {
		logger.Info("[main][main] Server shutdown completed successfully")
	}

	logger.Info("[main][main] AdguardFilter stopped")
}
