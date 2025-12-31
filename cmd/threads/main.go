package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/salmonumbrella/threads-go/internal/cmd"
)

func main() {
	// Create context that listens for interrupt signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Execute root command
	if err := cmd.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
