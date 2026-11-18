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

	"github.com/mirskov/booking-microservices/notification-service/internal/config"
	"github.com/mirskov/booking-microservices/notification-service/internal/health"
	"github.com/mirskov/booking-microservices/notification-service/internal/mail"
	"github.com/mirskov/booking-microservices/notification-service/internal/notification"
	"github.com/mirskov/booking-microservices/notification-service/internal/storage"
)

func main() {
	if err := run(); err != nil {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Контекст приложения отменяется при SIGINT или SIGTERM.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// 1. Configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// 2. Database
	db, err := storage.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	// 3. Repository
	notificationRepository := notification.NewRepository(db)

	// 4. SMTP sender
	emailSender := mail.NewSMTPSender(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUsername,
		cfg.SMTPPassword,
		cfg.SMTPFrom,
	)

	// 5. Service
	notificationService := notification.NewService(
		notificationRepository,
		emailSender,
	)

	// 6. HTTP handlers
	notificationHandler := notification.NewHandler(notificationService)
	healthHandler := health.NewHandler(db)

	// 7. Router
	mux := http.NewServeMux()

	mux.HandleFunc("POST /events", notificationHandler.ProcessEvent)
	mux.HandleFunc("GET /notifications/{id}", notificationHandler.GetByID)
	mux.HandleFunc("GET /notifications", notificationHandler.List)

	mux.HandleFunc("GET /health/live", healthHandler.Live)
	mux.HandleFunc("GET /health/ready", healthHandler.Ready)

	// 8. HTTP server
	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Запуск сервера в отдельной goroutine.
	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("notification-service listening on :%s", cfg.HTTPPort)

		serverErrors <- server.ListenAndServe()
	}()

	// 9. Ожидание сигнала завершения либо ошибки сервера.
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}

	case <-ctx.Done():
		log.Println("shutdown signal received")
	}

	// 10. Graceful shutdown
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

		return err
	}

	log.Println("notification-service stopped gracefully")

	return nil
}
