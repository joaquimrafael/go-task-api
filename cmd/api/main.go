package main

import (
	"log"
	"net/http"
	"time"

	"github.com/joaquimrafael/go-task-api/internal/handler"
	"github.com/joaquimrafael/go-task-api/internal/repository"
	"github.com/joaquimrafael/go-task-api/internal/service"
)

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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /tasks", taskHandler.List)
	mux.HandleFunc("GET /tasks/{id}", taskHandler.GetByID)
	mux.HandleFunc("POST /tasks", taskHandler.Create)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("serve API: %v", err)
	}
}
