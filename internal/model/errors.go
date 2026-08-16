package model

import "errors"

// ErrTaskNotFound indicates that a requested task does not exist.
var ErrTaskNotFound = errors.New("task not found")

// ErrInvalidTask indicates that task input violates a domain validation rule.
var ErrInvalidTask = errors.New("invalid task")
