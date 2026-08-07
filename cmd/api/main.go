package main

import (
	"log"
	"net/http"

	"github.com/joaquimrafael/go-task-api/internal/handler"
	"github.com/joaquimrafael/go-task-api/internal/repository"
)

func main() {
	db, err := repository.OpenSQLite("tasks.db")
	if err != nil {
		log.Fatalf("could not open the database %v", err)
	}
	defer db.Close()

	handler := handler.NewTaskHandler()
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("could not listen on port 8080 %v", err)
	}
}
