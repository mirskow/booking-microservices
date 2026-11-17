package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mirskov/booking-microservices/profile-service/internal/config"
	"github.com/mirskov/booking-microservices/profile-service/internal/health"
	"github.com/mirskov/booking-microservices/profile-service/internal/schedule"
	"github.com/mirskov/booking-microservices/profile-service/internal/storage"
	"github.com/mirskov/booking-microservices/profile-service/internal/user"
)

func main() {
	if err := run(); err != nil {
		log.Printf("application stopped with error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize database
	db, err := storage.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer db.Close()

	log.Println("connected to PostgreSQL")

	// Initialize user-service
	userRepository := user.NewRepository(db)
	userService := user.NewService(userRepository)
	userHandler := user.NewHandler(userService)

	// Initialize schedule-service
	scheduleRepository := schedule.NewRepository(db)
	scheduleService := schedule.NewService(scheduleRepository, userService, cfg.SlotDuration)
	scheduleHandler := schedule.NewHandler(scheduleService)

	// Initialize HTTP handlers
	mux := http.NewServeMux()

	healthHandler := health.NewHandler(db)

	mux.HandleFunc("GET /health/live", healthHandler.Live)
	mux.HandleFunc("GET /health/ready", healthHandler.Ready)

	mux.HandleFunc("POST /profiles", userHandler.Create)
	mux.HandleFunc("GET /profiles/{id}", userHandler.GetByID)
	mux.HandleFunc("PUT /profiles/{id}", userHandler.Update)

	mux.HandleFunc("PUT /users/{id}/schedule", scheduleHandler.SetSchedule)
	mux.HandleFunc("GET /users/{id}/schedule", scheduleHandler.GetSchedule)
	mux.HandleFunc("GET /users/{id}/slots", scheduleHandler.GetSlots)

	// Initialize HTTP server
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Start HTTP server
	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("profile service listening on port %d", cfg.Port)

		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			serverErrors <- nil
			return
		}

		serverErrors <- err
	}()

	// Wait for shutdown signal or server error
	stopCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("HTTP server: %w", err)
		}
		return nil

	case <-stopCtx.Done():
		log.Println("shutdown signal received")
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)

		if closeErr := server.Close(); closeErr != nil {
			log.Printf("force close failed: %v", closeErr)
		}

		<-serverErrors
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	<-serverErrors

	log.Println("HTTP server stopped")
	return nil
}
