package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mirskov/booking-microservices/booking-service/internal/booking"
	"github.com/mirskov/booking-microservices/booking-service/internal/clients"
	"github.com/mirskov/booking-microservices/booking-service/internal/config"
	"github.com/mirskov/booking-microservices/booking-service/internal/health"
	"github.com/mirskov/booking-microservices/booking-service/internal/metrics"
	"github.com/mirskov/booking-microservices/booking-service/internal/storage"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := storage.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	// HTTP clients для взаимодействия с другими микросервисами.
	userClient := clients.NewUserClient(cfg.UserServiceURL)

	notificationClient := clients.NewNotificationClient(
		cfg.NotificationServiceURL,
	)

	// Repository.
	repository := booking.NewRepository(db)

	// Business logic.
	service := booking.NewService(
		repository,
		userClient,
		notificationClient,
	)

	// HTTP handlers.
	bookingHandler := booking.NewHandler(service)
	healthHandler := health.NewHandler(db)

	// HTTP router.
	mux := http.NewServeMux()

	bookingHandler.RegisterRoutes(mux)
	healthHandler.RegisterRoutes(mux)
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           metrics.Middleware(mux),
		ReadTimeout:       cfg.HTTPReadTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Контекст для graceful shutdown.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("booking-service starting on port %s", cfg.HTTPPort)

		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}

	case <-ctx.Done():
		log.Println("shutdown signal received")
	}

	// Даем текущим запросам время завершиться.
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)

		if closeErr := server.Close(); closeErr != nil {
			log.Printf("force close server: %v", closeErr)
		}
	}

	log.Println("booking-service stopped")

	return nil
}
