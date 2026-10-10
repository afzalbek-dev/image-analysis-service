package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/afzalbek-dev/image-analysis-service/internal/api"
	"github.com/afzalbek-dev/image-analysis-service/internal/config"
	"github.com/afzalbek-dev/image-analysis-service/internal/duplicates"
)

func main() {
	cfg := config.Load()
	dupService := duplicates.NewService()
	server := api.NewServer(cfg, dupService)
	router := server.SetupRouter()

	addr := fmt.Sprintf(":%s", cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeoutSeconds) * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[IMAGE-ANALYSIS-SERVICE] Server listening on http://0.0.0.0%s (env: %s)", addr, cfg.Environment)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	<-shutdown
	log.Println("[IMAGE-ANALYSIS-SERVICE] Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown error: %v", err)
	}
	log.Println("[IMAGE-ANALYSIS-SERVICE] Server stopped.")
}
