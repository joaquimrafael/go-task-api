package handler

import (
	"fmt"
	"net/http"
)

type TaskHandler struct {
	http.Handler
}

func NewTaskHandler() *TaskHandler {
	return &TaskHandler{}
}

func (t *TaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello World!\n")
}
