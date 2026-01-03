package main

import (
	"context"
	"os"
	"time"

	"github.com/tungetti/igor/internal/app"
)

func main() {
	// Use app package for lifecycle management when running as a long-lived service
	// For now, we use the CLI directly for command-line usage
	if os.Getenv("IGOR_APP_MODE") == "service" {
		runAsService()
		return
	}

	// Standard CLI mode
	cli := NewCLI()
	exitCode := cli.Run(os.Args[1:])
	os.Exit(exitCode)
}

// runAsService runs Igor as a long-lived service with proper lifecycle management.
// This is intended for future use cases where Igor needs to run as a daemon.
func runAsService() {
	opts := app.Options{
		Version:         Version,
		BuildTime:       BuildTime,
		GitCommit:       GitCommit,
		ShutdownTimeout: 30 * time.Second,
	}

	application := app.New(opts)
	ctx := context.Background()

	// Initialize the application
	configPath := os.Getenv("IGOR_CONFIG")
	if err := application.Initialize(ctx, configPath); err != nil {
		os.Stderr.WriteString("Failed to initialize: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Run with lifecycle management (blocks until shutdown signal)
	if err := application.RunWithLifecycle(ctx); err != nil {
		os.Stderr.WriteString("Application error: " + err.Error() + "\n")
		os.Exit(1)
	}
}
