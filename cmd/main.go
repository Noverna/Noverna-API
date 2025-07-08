package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"noverna.de/m/v2/internal/api"
	"noverna.de/m/v2/internal/config"
	"noverna.de/m/v2/internal/logger"
)

func main() {
	logger.Info("Starting Noverna-API...")

	if err := config.Init(); err != nil {
		logger.Fatal("Error while loading config file", map[string]any{"error": err})
	}

	server := api.NewServer(config.GetConfig(), nil)

	server.Mount("/ws", websocketHandler())

	gracefulShutdown(server)
}

func websocketHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("WebSocket endpoint placeholder"))
	})
}

func gracefulShutdown(server *api.Server) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start: %v", map[string]any{"error": err})
		}
	}()
	
	<-sigChan
	logger.Info("Shutdown signal received")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := server.Stop(ctx); err != nil {
		logger.Error("Server shutdown failed: %v", map[string]any{"error": err})
	} else {
		logger.Info("Server stopped gracefully")
	}
}