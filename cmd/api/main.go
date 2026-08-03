package main

import (
	"log"
	"net/http"

	"github.com/joaquimrafael/go-task-api/internal/handler"
)

func main() {
	http.HandleFunc("/hello", handler.TaskHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("could not listen on port 8080 %v", err)
	}
}
