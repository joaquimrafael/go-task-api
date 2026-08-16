package service_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/joaquimrafael/go-task-api/internal/model"
	"github.com/joaquimrafael/go-task-api/internal/service"
)

type fakeTaskRepository struct {
	listFn    func(context.Context) ([]model.Task, error)
	getByIDFn func(context.Context, int64) (*model.Task, error)
	createFn  func(context.Context, model.TaskInput) (*model.Task, error)
	updateFn  func(context.Context, model.Task) (*model.Task, error)
	deleteFn  func(context.Context, int64) error
}

func (f *fakeTaskRepository) List(ctx context.Context) ([]model.Task, error) {
	return f.listFn(ctx)
}

func (f *fakeTaskRepository) GetByID(ctx context.Context, id int64) (*model.Task, error) {
	return f.getByIDFn(ctx, id)
}

func (f *fakeTaskRepository) Create(ctx context.Context, input model.TaskInput) (*model.Task, error) {
	return f.createFn(ctx, input)
}

func (f *fakeTaskRepository) Update(ctx context.Context, task model.Task) (*model.Task, error) {
	return f.updateFn(ctx, task)
}

func (f *fakeTaskRepository) Delete(ctx context.Context, id int64) error {
	return f.deleteFn(ctx, id)
}

func newFakeTaskRepository(t *testing.T) *fakeTaskRepository {
	t.Helper()

	unexpected := func(method string) {
		t.Helper()
		t.Fatalf("unexpected repository call: %s", method)
	}

	return &fakeTaskRepository{
		listFn: func(context.Context) ([]model.Task, error) {
			unexpected("List")
			return nil, nil
		},
		getByIDFn: func(context.Context, int64) (*model.Task, error) {
			unexpected("GetByID")
			return nil, nil
		},
		createFn: func(context.Context, model.TaskInput) (*model.Task, error) {
			unexpected("Create")
			return nil, nil
		},
		updateFn: func(context.Context, model.Task) (*model.Task, error) {
			unexpected("Update")
			return nil, nil
		},
		deleteFn: func(context.Context, int64) error {
			unexpected("Delete")
			return nil
		},
	}
}

func newTestTaskService(t *testing.T, repository service.TaskRepository) *service.TaskService {
	t.Helper()

	service, err := service.NewTaskService(repository)
	if err != nil {
		t.Fatalf("NewTaskService() error = %v", err)
	}

	return service
}

func assertEqual[T any](t *testing.T, got, want T) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func assertError(t *testing.T, got, want error, contains string) {
	t.Helper()

	if want == nil {
		if got != nil {
			t.Fatalf("unexpected error: %v", got)
		}
		return
	}

	if got == nil {
		t.Fatalf("expected error wrapping %v, got nil", want)
	}
	if !errors.Is(got, want) {
		t.Errorf("error = %v, want errors.Is(error, %v)", got, want)
	}
	if contains != "" && !strings.Contains(got.Error(), contains) {
		t.Errorf("error = %q, want it to contain %q", got, contains)
	}
}

func assertContext(t *testing.T, got, want context.Context) {
	t.Helper()

	if got != want {
		t.Errorf("repository received a different context")
	}
}

func TestNewTaskService(t *testing.T) {
	tests := []struct {
		name       string
		repository service.TaskRepository
		wantErr    bool
	}{
		{
			name:       "repository is accepted",
			repository: newFakeTaskRepository(t),
		},
		{
			name:    "nil repository is rejected",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := service.NewTaskService(test.repository)
			if test.wantErr {
				if err == nil {
					t.Fatal("NewTaskService() error = nil, want an error")
				}
				if got != nil {
					t.Errorf("NewTaskService() service = %#v, want nil", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewTaskService() error = %v", err)
			}
			if got == nil {
				t.Fatal("NewTaskService() service = nil, want a service")
			}
		})
	}
}

func TestTaskServiceCreate(t *testing.T) {
	repositoryErr := errors.New("repository unavailable")
	createdAt := time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC)
	createdTask := &model.Task{
		ID:          7,
		Title:       "Learn Go",
		Description: "Practice interfaces",
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}

	tests := []struct {
		name            string
		input           model.TaskInput
		repositoryTask  *model.Task
		repositoryErr   error
		wantInput       *model.TaskInput
		wantTask        *model.Task
		wantErr         error
		wantErrContains string
	}{
		{
			name: "creates task and trims title",
			input: model.TaskInput{
				Title:       "  Learn Go  ",
				Description: "Practice interfaces",
			},
			repositoryTask: createdTask,
			wantInput: &model.TaskInput{
				Title:       "Learn Go",
				Description: "Practice interfaces",
			},
			wantTask: createdTask,
		},
		{
			name:           "accepts title with exactly 120 characters",
			input:          model.TaskInput{Title: strings.Repeat("界", 120)},
			repositoryTask: &model.Task{ID: 8, Title: strings.Repeat("界", 120)},
			wantInput:      &model.TaskInput{Title: strings.Repeat("界", 120)},
			wantTask:       &model.Task{ID: 8, Title: strings.Repeat("界", 120)},
		},
		{
			name:            "rejects empty title",
			input:           model.TaskInput{},
			wantErr:         model.ErrInvalidTask,
			wantErrContains: "title is required",
		},
		{
			name:            "rejects whitespace-only title",
			input:           model.TaskInput{Title: " \t\n "},
			wantErr:         model.ErrInvalidTask,
			wantErrContains: "title is required",
		},
		{
			name:            "rejects title longer than 120 Unicode characters",
			input:           model.TaskInput{Title: strings.Repeat("界", 121)},
			wantErr:         model.ErrInvalidTask,
			wantErrContains: "title must not exceed 120 characters",
		},
		{
			name:            "wraps repository error",
			input:           model.TaskInput{Title: "Learn Go"},
			repositoryErr:   repositoryErr,
			wantInput:       &model.TaskInput{Title: "Learn Go"},
			wantErr:         repositoryErr,
			wantErrContains: "create task",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newFakeTaskRepository(t)
			ctx := context.WithValue(context.Background(), testContextKey{}, test.name)
			calls := 0
			if test.wantInput != nil {
				repository.createFn = func(gotCtx context.Context, gotInput model.TaskInput) (*model.Task, error) {
					calls++
					assertContext(t, gotCtx, ctx)
					assertEqual(t, gotInput, *test.wantInput)
					return test.repositoryTask, test.repositoryErr
				}
			}

			got, err := newTestTaskService(t, repository).Create(ctx, test.input)

			assertError(t, err, test.wantErr, test.wantErrContains)
			assertEqual(t, got, test.wantTask)
			if test.wantInput != nil && calls != 1 {
				t.Errorf("repository Create() calls = %d, want 1", calls)
			}
		})
	}
}

func TestTaskServiceUpdate(t *testing.T) {
	repositoryErr := errors.New("write failed")
	updatedAt := time.Date(2026, time.August, 15, 13, 0, 0, 0, time.UTC)
	updatedTask := &model.Task{
		ID:          12,
		Title:       "Updated title",
		Description: "Updated description",
		Completed:   true,
		UpdatedAt:   updatedAt,
	}

	tests := []struct {
		name            string
		id              int64
		input           model.TaskInput
		repositoryTask  *model.Task
		repositoryErr   error
		wantRepository  *model.Task
		wantTask        *model.Task
		wantErr         error
		wantErrContains string
	}{
		{
			name: "updates task and trims title",
			id:   12,
			input: model.TaskInput{
				Title:       "  Updated title  ",
				Description: "Updated description",
				Completed:   true,
			},
			repositoryTask: updatedTask,
			wantRepository: &model.Task{
				ID:          12,
				Title:       "Updated title",
				Description: "Updated description",
				Completed:   true,
			},
			wantTask: updatedTask,
		},
		{
			name:           "accepts title with exactly 120 characters",
			id:             13,
			input:          model.TaskInput{Title: strings.Repeat("a", 120)},
			repositoryTask: &model.Task{ID: 13, Title: strings.Repeat("a", 120)},
			wantRepository: &model.Task{ID: 13, Title: strings.Repeat("a", 120)},
			wantTask:       &model.Task{ID: 13, Title: strings.Repeat("a", 120)},
		},
		{
			name:            "rejects empty title",
			id:              12,
			input:           model.TaskInput{},
			wantErr:         model.ErrInvalidTask,
			wantErrContains: "title is required",
		},
		{
			name:            "rejects whitespace-only title",
			id:              12,
			input:           model.TaskInput{Title: " \t\n "},
			wantErr:         model.ErrInvalidTask,
			wantErrContains: "title is required",
		},
		{
			name:            "rejects title longer than 120 characters",
			id:              12,
			input:           model.TaskInput{Title: strings.Repeat("a", 121)},
			wantErr:         model.ErrInvalidTask,
			wantErrContains: "title must not exceed 120 characters",
		},
		{
			name:          "preserves not-found error",
			id:            99,
			input:         model.TaskInput{Title: "Missing task"},
			repositoryErr: model.ErrTaskNotFound,
			wantRepository: &model.Task{
				ID:    99,
				Title: "Missing task",
			},
			wantErr:         model.ErrTaskNotFound,
			wantErrContains: "update task 99",
		},
		{
			name:          "wraps unexpected repository error",
			id:            12,
			input:         model.TaskInput{Title: "Updated title"},
			repositoryErr: repositoryErr,
			wantRepository: &model.Task{
				ID:    12,
				Title: "Updated title",
			},
			wantErr:         repositoryErr,
			wantErrContains: "update task 12",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newFakeTaskRepository(t)
			ctx := context.WithValue(context.Background(), testContextKey{}, test.name)
			calls := 0
			if test.wantRepository != nil {
				repository.updateFn = func(gotCtx context.Context, gotTask model.Task) (*model.Task, error) {
					calls++
					assertContext(t, gotCtx, ctx)
					assertEqual(t, gotTask, *test.wantRepository)
					return test.repositoryTask, test.repositoryErr
				}
			}

			got, err := newTestTaskService(t, repository).Update(ctx, test.id, test.input)

			assertError(t, err, test.wantErr, test.wantErrContains)
			assertEqual(t, got, test.wantTask)
			if test.wantRepository != nil && calls != 1 {
				t.Errorf("repository Update() calls = %d, want 1", calls)
			}
		})
	}
}

func TestTaskServiceGetByID(t *testing.T) {
	repositoryErr := errors.New("read failed")
	task := &model.Task{ID: 3, Title: "Test services"}

	tests := []struct {
		name            string
		id              int64
		repositoryTask  *model.Task
		repositoryErr   error
		wantTask        *model.Task
		wantErr         error
		wantErrContains string
	}{
		{
			name:           "returns task",
			id:             3,
			repositoryTask: task,
			wantTask:       task,
		},
		{
			name:            "preserves not-found error",
			id:              404,
			repositoryErr:   model.ErrTaskNotFound,
			wantErr:         model.ErrTaskNotFound,
			wantErrContains: "get task 404",
		},
		{
			name:            "wraps unexpected repository error",
			id:              3,
			repositoryErr:   repositoryErr,
			wantErr:         repositoryErr,
			wantErrContains: "get task 3",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newFakeTaskRepository(t)
			ctx := context.WithValue(context.Background(), testContextKey{}, test.name)
			calls := 0
			repository.getByIDFn = func(gotCtx context.Context, gotID int64) (*model.Task, error) {
				calls++
				assertContext(t, gotCtx, ctx)
				assertEqual(t, gotID, test.id)
				return test.repositoryTask, test.repositoryErr
			}

			got, err := newTestTaskService(t, repository).GetByID(ctx, test.id)

			assertError(t, err, test.wantErr, test.wantErrContains)
			assertEqual(t, got, test.wantTask)
			assertEqual(t, calls, 1)
		})
	}
}

func TestTaskServiceList(t *testing.T) {
	repositoryErr := errors.New("read failed")
	tasks := []model.Task{
		{ID: 1, Title: "First"},
		{ID: 2, Title: "Second", Completed: true},
	}

	tests := []struct {
		name            string
		repositoryTasks []model.Task
		repositoryErr   error
		wantTasks       []model.Task
		wantErr         error
		wantErrContains string
	}{
		{
			name:            "returns tasks",
			repositoryTasks: tasks,
			wantTasks:       tasks,
		},
		{
			name:            "returns empty list",
			repositoryTasks: []model.Task{},
			wantTasks:       []model.Task{},
		},
		{
			name:            "wraps repository error",
			repositoryErr:   repositoryErr,
			wantErr:         repositoryErr,
			wantErrContains: "list tasks",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newFakeTaskRepository(t)
			ctx := context.WithValue(context.Background(), testContextKey{}, test.name)
			calls := 0
			repository.listFn = func(gotCtx context.Context) ([]model.Task, error) {
				calls++
				assertContext(t, gotCtx, ctx)
				return test.repositoryTasks, test.repositoryErr
			}

			got, err := newTestTaskService(t, repository).List(ctx)

			assertError(t, err, test.wantErr, test.wantErrContains)
			assertEqual(t, got, test.wantTasks)
			assertEqual(t, calls, 1)
		})
	}
}

func TestTaskServiceDelete(t *testing.T) {
	repositoryErr := errors.New("delete failed")

	tests := []struct {
		name            string
		id              int64
		repositoryErr   error
		wantErr         error
		wantErrContains string
	}{
		{
			name: "deletes task",
			id:   5,
		},
		{
			name:            "preserves not-found error",
			id:              404,
			repositoryErr:   model.ErrTaskNotFound,
			wantErr:         model.ErrTaskNotFound,
			wantErrContains: "delete task 404",
		},
		{
			name:            "wraps unexpected repository error",
			id:              5,
			repositoryErr:   repositoryErr,
			wantErr:         repositoryErr,
			wantErrContains: "delete task 5",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newFakeTaskRepository(t)
			ctx := context.WithValue(context.Background(), testContextKey{}, test.name)
			calls := 0
			repository.deleteFn = func(gotCtx context.Context, gotID int64) error {
				calls++
				assertContext(t, gotCtx, ctx)
				assertEqual(t, gotID, test.id)
				return test.repositoryErr
			}

			err := newTestTaskService(t, repository).Delete(ctx, test.id)

			assertError(t, err, test.wantErr, test.wantErrContains)
			assertEqual(t, calls, 1)
		})
	}
}

type testContextKey struct{}
