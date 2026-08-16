package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joaquimrafael/go-task-api/internal/handler"
	"github.com/joaquimrafael/go-task-api/internal/repository"
	"github.com/joaquimrafael/go-task-api/internal/service"
)

func newRouter(taskHandler *handler.TaskHandler, healthHandler http.HandlerFunc) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /tasks", taskHandler.List)
	mux.HandleFunc("GET /tasks/{id}", taskHandler.GetByID)
	mux.HandleFunc("POST /tasks", taskHandler.Create)
	mux.HandleFunc("PUT /tasks/{id}", taskHandler.Update)
	mux.HandleFunc("DELETE /tasks/{id}", taskHandler.Delete)
	return mux
}

type apiServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func serveUntilShutdown(
	ctx context.Context,
	server apiServer,
	shutdownTimeout time.Duration,
	logger *slog.Logger,
) error {
	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve after shutdown: %w", err)
	}

	return nil
}

func main() {
	db, err := repository.OpenSQLite("tasks.db")
	if err != nil {
		log.Fatalf("could not open the database %v", err)
	}
	defer db.Close()

	repo, err := repository.NewSQLiteTaskRepository(db)
	if err != nil {
		log.Fatalf("create repository: %v", err)
	}

	taskService, err := service.NewTaskService(repo)
	if err != nil {
		log.Fatalf("create task service: %v", err)
	}

	taskHandler, err := handler.NewTaskHandler(taskService)
	if err != nil {
		log.Fatalf("create task handler: %v", err)
	}

	healthHandler, err := handler.NewHealthHandler(db)
	if err != nil {
		log.Fatalf("create health handler: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           requestLogger(logger, newRouter(taskHandler, healthHandler)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	if err := serveUntilShutdown(signalCtx, server, 5*time.Second, logger); err != nil {
		log.Printf("serve API: %v", err)
	}
}
