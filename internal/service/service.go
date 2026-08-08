package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/joaquimrafael/go-task-api/internal/model"
)

type TaskRepository interface {
	List(ctx context.Context) ([]model.Task, error)
	GetByID(ctx context.Context, id int64) (*model.Task, error)
	Create(ctx context.Context, task model.TaskInput) (*model.Task, error)
	Update(ctx context.Context, task model.Task) (*model.Task, error)
	Delete(ctx context.Context, id int64) error
}

type TaskService struct {
	repository TaskRepository
}

func NewTaskService(repository TaskRepository) (*TaskService, error) {
	if repository == nil {
		return nil, fmt.Errorf("repository must not be nil")
	}
	return &TaskService{repository: repository}, nil
}

func validateTitle(title string) (string, error) {
	title = strings.TrimSpace(title)

	if title == "" {
		return "", fmt.Errorf("%w: title is required", model.ErrInvalidTask)
	}

	if utf8.RuneCountInString(title) > 120 {
		return "", fmt.Errorf(
			"%w: title must not exceed 120 characters",
			model.ErrInvalidTask,
		)
	}

	return title, nil
}

func (ts *TaskService) Create(ctx context.Context, input model.TaskInput) (*model.Task, error) {
	title, err := validateTitle(input.Title)
	if err != nil {
		return nil, err
	}

	input.Title = title

	created, err := ts.repository.Create(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	return created, nil
}

func (ts *TaskService) Update(ctx context.Context, id int64, input model.TaskInput) (*model.Task, error) {
	title, err := validateTitle(input.Title)
	if err != nil {
		return nil, err
	}

	task := model.Task{
		ID:          id,
		Title:       title,
		Description: input.Description,
		Completed:   input.Completed,
	}

	updated, err := ts.repository.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("update task %d: %w", id, err)
	}

	return updated, nil
}

func (ts *TaskService) GetByID(ctx context.Context, id int64) (*model.Task, error) {
	task, err := ts.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get task %d: %w", id, err)
	}
	return task, nil
}

func (ts *TaskService) List(ctx context.Context) ([]model.Task, error) {
	tasks, err := ts.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

func (ts *TaskService) Delete(ctx context.Context, id int64) error {
	if err := ts.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete task %d: %w", id, err)
	}

	return nil
}
