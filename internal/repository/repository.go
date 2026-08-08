package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/joaquimrafael/go-task-api/internal/model"
)

type SQLiteTaskRepository struct {
	db *sql.DB
}

func NewSQLiteTaskRepository(db *sql.DB) (*SQLiteTaskRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("sql.DB must not be nil")
	}
	return &SQLiteTaskRepository{db: db}, nil
}

func (r *SQLiteTaskRepository) Create(ctx context.Context, input model.TaskInput) (*model.Task, error) {
	query := `INSERT INTO tasks (title, description, completed) VALUES (?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query, input.Title, input.Description, input.Completed)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last inserted task: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *SQLiteTaskRepository) GetByID(ctx context.Context, id int64) (*model.Task, error) {
	query := `SELECT id, title, description, completed, created_at, updated_at FROM tasks WHERE id = ?;`
	row := r.db.QueryRowContext(ctx, query, id)
	task := &model.Task{}
	err := row.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Completed,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan task %d: %w", id, err)
	}

	return task, nil
}

func (r *SQLiteTaskRepository) List(ctx context.Context) ([]model.Task, error) {
	query := `SELECT id, title, description, completed, created_at, updated_at FROM tasks ORDER BY id`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]model.Task, 0)
	for rows.Next() {
		var task model.Task
		err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Completed,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}

func (r *SQLiteTaskRepository) Update(ctx context.Context, task model.Task) (*model.Task, error) {
	query := `
      UPDATE tasks
      SET title = ?,
          description = ?,
          completed = ?,
          updated_at = CURRENT_TIMESTAMP
      WHERE id = ?
  	`
	result, err := r.db.ExecContext(ctx, query, task.Title, task.Description, task.Completed, task.ID)
	if err != nil {
		return nil, fmt.Errorf("update task %d: %w", task.ID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return nil, model.ErrTaskNotFound
	}
	return r.GetByID(ctx, task.ID)
}

func (r *SQLiteTaskRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM tasks WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete task %d: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if rows == 0 {
		return model.ErrTaskNotFound
	}

	return nil
}
