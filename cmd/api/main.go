package main

import (
	"log"
	"net/http"

	"github.com/joaquimrafael/go-task-api/internal/handler"
)

func main() {
	handler := handler.NewTaskHandler()

	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("could not listen on port 8080 %v", err)
	}
}
