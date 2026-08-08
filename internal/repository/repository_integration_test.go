package repository_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaquimrafael/go-task-api/internal/model"
	"github.com/joaquimrafael/go-task-api/internal/repository"
)

func newTestRepository(t *testing.T) *repository.SQLiteTaskRepository {
	t.Helper()

	path := filepath.Join(t.TempDir(), "tasks.db")

	db, err := repository.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	repo, err := repository.NewSQLiteTaskRepository(db)
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	return repo
}

func assertTaskMatchesInput(t *testing.T, got *model.Task, want model.TaskInput) {
	t.Helper()

	if got == nil {
		t.Fatal("expected task, got nil")
	}
	if got.Title != want.Title {
		t.Errorf("title: got %q, want %q", got.Title, want.Title)
	}
	if got.Description != want.Description {
		t.Errorf("description: got %q, want %q", got.Description, want.Description)
	}
	if got.Completed != want.Completed {
		t.Errorf("completed: got %t, want %t", got.Completed, want.Completed)
	}
}

func assertStoredTask(t *testing.T, task *model.Task) {
	t.Helper()

	if task == nil {
		t.Fatal("expected task, got nil")
	}
	if task.ID <= 0 {
		t.Errorf("expected a positive ID, got %d", task.ID)
	}
	if task.CreatedAt.IsZero() {
		t.Error("expected created_at to be populated")
	}
	if task.UpdatedAt.IsZero() {
		t.Error("expected updated_at to be populated")
	}
}

func assertTasksEqual(t *testing.T, got, want *model.Task) {
	t.Helper()

	if got == nil || want == nil {
		t.Fatalf("cannot compare nil tasks: got %v, want %v", got, want)
	}
	if got.ID != want.ID {
		t.Errorf("ID: got %d, want %d", got.ID, want.ID)
	}
	if got.Title != want.Title {
		t.Errorf("title: got %q, want %q", got.Title, want.Title)
	}
	if got.Description != want.Description {
		t.Errorf("description: got %q, want %q", got.Description, want.Description)
	}
	if got.Completed != want.Completed {
		t.Errorf("completed: got %t, want %t", got.Completed, want.Completed)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("created_at: got %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	if !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Errorf("updated_at: got %v, want %v", got.UpdatedAt, want.UpdatedAt)
	}
}

func assertErrorIs(t *testing.T, got, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Fatalf("error: got %v, want %v", got, want)
	}
}

func createTask(t *testing.T, repo *repository.SQLiteTaskRepository, input model.TaskInput) *model.Task {
	t.Helper()

	task, err := repo.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	return task
}

func TestOpenSQLiteCanReopenExistingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.db")

	for attempt := 1; attempt <= 2; attempt++ {
		db, err := repository.OpenSQLite(path)
		if err != nil {
			t.Fatalf("OpenSQLite() attempt %d error: %v", attempt, err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("close database after attempt %d: %v", attempt, err)
		}
	}
}

func TestOpenSQLiteRejectsInvalidPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "tasks.db")

	db, err := repository.OpenSQLite(path)
	if err == nil {
		db.Close()
		t.Fatal("OpenSQLite() expected an error for an invalid path")
	}
}

func TestNewSQLiteTaskRepositoryRejectsNilDatabase(t *testing.T) {
	repo, err := repository.NewSQLiteTaskRepository(nil)
	if err == nil {
		t.Fatal("NewSQLiteTaskRepository() expected an error")
	}
	if repo != nil {
		t.Errorf("NewSQLiteTaskRepository() got %v, want nil", repo)
	}
}

func TestSQLiteTaskRepositoryCreate(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	input := model.TaskInput{
		Title:       "Study Go",
		Description: "Write repository tests",
		Completed:   false,
	}
	task, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	assertTaskMatchesInput(t, task, input)
	assertStoredTask(t, task)
}

func TestSQLiteTaskRepositoryCreateRejectsInvalidTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{name: "empty", title: ""},
		{name: "whitespace", title: "   "},
		{name: "too long", title: strings.Repeat("a", 121)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newTestRepository(t)

			task, err := repo.Create(context.Background(), model.TaskInput{Title: tt.title})
			if err == nil {
				t.Fatal("Create() expected an error")
			}
			if task != nil {
				t.Errorf("Create() got task %v, want nil", task)
			}
		})
	}
}

func TestSQLiteTaskRepositoryGetByID(t *testing.T) {
	repo := newTestRepository(t)
	want := createTask(t, repo, model.TaskInput{
		Title:       "Read task",
		Description: "Load a complete row",
		Completed:   true,
	})

	got, err := repo.GetByID(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	assertTasksEqual(t, got, want)
}

func TestSQLiteTaskRepositoryGetByIDNotFound(t *testing.T) {
	repo := newTestRepository(t)

	task, err := repo.GetByID(context.Background(), 999)
	assertErrorIs(t, err, model.ErrTaskNotFound)
	if task != nil {
		t.Errorf("GetByID() got task %v, want nil", task)
	}
}

func TestSQLiteTaskRepositoryListEmpty(t *testing.T) {
	repo := newTestRepository(t)

	tasks, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if tasks == nil {
		t.Fatal("List() got nil, want an empty slice")
	}
	if len(tasks) != 0 {
		t.Errorf("List() returned %d tasks, want 0", len(tasks))
	}
}

func TestSQLiteTaskRepositoryListOrdersByID(t *testing.T) {
	repo := newTestRepository(t)
	inputs := []model.TaskInput{
		{Title: "First"},
		{Title: "Second", Description: "middle", Completed: true},
		{Title: "Third"},
	}

	for _, input := range inputs {
		createTask(t, repo, input)
	}

	tasks, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(tasks) != len(inputs) {
		t.Fatalf("List() returned %d tasks, want %d", len(tasks), len(inputs))
	}
	for i := range tasks {
		assertTaskMatchesInput(t, &tasks[i], inputs[i])
		assertStoredTask(t, &tasks[i])
		if i > 0 && tasks[i-1].ID >= tasks[i].ID {
			t.Errorf("tasks are not ordered by ID: %d before %d", tasks[i-1].ID, tasks[i].ID)
		}
	}
}

func TestSQLiteTaskRepositoryUpdate(t *testing.T) {
	repo := newTestRepository(t)
	created := createTask(t, repo, model.TaskInput{Title: "Before update"})

	update := *created
	update.Title = "After update"
	update.Description = "Changed description"
	update.Completed = true

	got, err := repo.Update(context.Background(), update)
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	assertTaskMatchesInput(t, got, model.TaskInput{
		Title:       update.Title,
		Description: update.Description,
		Completed:   update.Completed,
	})
	assertStoredTask(t, got)
	if got.ID != created.ID {
		t.Errorf("ID changed: got %d, want %d", got.ID, created.ID)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("created_at changed: got %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if got.UpdatedAt.Before(got.CreatedAt) {
		t.Errorf("updated_at %v is before created_at %v", got.UpdatedAt, got.CreatedAt)
	}
}

func TestSQLiteTaskRepositoryUpdateNotFound(t *testing.T) {
	repo := newTestRepository(t)

	task, err := repo.Update(context.Background(), model.Task{ID: 999, Title: "Missing"})
	assertErrorIs(t, err, model.ErrTaskNotFound)
	if task != nil {
		t.Errorf("Update() got task %v, want nil", task)
	}
}

func TestSQLiteTaskRepositoryUpdateRejectsInvalidTitle(t *testing.T) {
	repo := newTestRepository(t)
	created := createTask(t, repo, model.TaskInput{Title: "Valid title"})
	update := *created
	update.Title = "   "

	task, err := repo.Update(context.Background(), update)
	if err == nil {
		t.Fatal("Update() expected an error")
	}
	if task != nil {
		t.Errorf("Update() got task %v, want nil", task)
	}

	stored, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID() after failed update error: %v", err)
	}
	assertTasksEqual(t, stored, created)
}

func TestSQLiteTaskRepositoryDelete(t *testing.T) {
	repo := newTestRepository(t)
	created := createTask(t, repo, model.TaskInput{Title: "Delete me"})

	if err := repo.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	task, err := repo.GetByID(context.Background(), created.ID)
	assertErrorIs(t, err, model.ErrTaskNotFound)
	if task != nil {
		t.Errorf("GetByID() after delete got task %v, want nil", task)
	}
}

func TestSQLiteTaskRepositoryDeleteNotFound(t *testing.T) {
	repo := newTestRepository(t)

	err := repo.Delete(context.Background(), 999)
	assertErrorIs(t, err, model.ErrTaskNotFound)
}

func TestSQLiteTaskRepositoryHonorsCanceledContext(t *testing.T) {
	repo := newTestRepository(t)
	created := createTask(t, repo, model.TaskInput{Title: "Existing task"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	t.Run("create", func(t *testing.T) {
		_, err := repo.Create(ctx, model.TaskInput{Title: "Canceled"})
		assertErrorIs(t, err, context.Canceled)
	})

	t.Run("get by ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, created.ID)
		assertErrorIs(t, err, context.Canceled)
	})

	t.Run("list", func(t *testing.T) {
		_, err := repo.List(ctx)
		assertErrorIs(t, err, context.Canceled)
	})

	t.Run("update", func(t *testing.T) {
		_, err := repo.Update(ctx, *created)
		assertErrorIs(t, err, context.Canceled)
	})

	t.Run("delete", func(t *testing.T) {
		err := repo.Delete(ctx, created.ID)
		assertErrorIs(t, err, context.Canceled)
	})
}
