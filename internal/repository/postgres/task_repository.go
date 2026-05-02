package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
        INSERT INTO tasks (title, description, status, created_at, updated_at,
                           due_date, recurrence_type, recurrence_params, parent_task_id, is_template)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
        RETURNING id, title, description, status, created_at, updated_at,
                  due_date, recurrence_type, recurrence_params, parent_task_id, is_template
    `
	row := r.pool.QueryRow(ctx, query,
		task.Title, task.Description, task.Status,
		task.CreatedAt, task.UpdatedAt,
		task.DueDate, task.RecurrenceType, task.RecurrenceParams,
		task.ParentTaskID, task.IsTemplate,
	)
	return scanTaskFull(row)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
        SELECT id, title, description, status, created_at, updated_at,
               due_date, recurrence_type, recurrence_params, parent_task_id, is_template
        FROM tasks
        WHERE id = $1
    `
	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTaskFull(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}
	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
        UPDATE tasks
        SET title = $1, description = $2, status = $3, updated_at = $4,
            due_date = $5, recurrence_type = $6, recurrence_params = $7,
            parent_task_id = $8, is_template = $9
        WHERE id = $10
        RETURNING id, title, description, status, created_at, updated_at,
                  due_date, recurrence_type, recurrence_params, parent_task_id, is_template
    `
	row := r.pool.QueryRow(ctx, query,
		task.Title, task.Description, task.Status, task.UpdatedAt,
		task.DueDate, task.RecurrenceType, task.RecurrenceParams,
		task.ParentTaskID, task.IsTemplate,
		task.ID,
	)
	updated, err := scanTaskFull(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
        SELECT id, title, description, status, created_at, updated_at,
               due_date, recurrence_type, recurrence_params, parent_task_id, is_template
        FROM tasks
        WHERE is_template = FALSE
        ORDER BY id DESC
    `
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTaskFull(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTaskFull(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task   taskdomain.Task
		status string
	)
	err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.DueDate,
		&task.RecurrenceType,
		&task.RecurrenceParams,
		&task.ParentTaskID,
		&task.IsTemplate,
	)
	if err != nil {
		return nil, err
	}
	task.Status = taskdomain.Status(status)
	return &task, nil
}

func (r *Repository) BulkCreate(ctx context.Context, tasks []taskdomain.Task) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, t := range tasks {
		_, err := tx.Exec(ctx,
			`INSERT INTO tasks (title, description, status, created_at, updated_at,
                                due_date, recurrence_type, recurrence_params, parent_task_id, is_template)
             VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			t.Title, t.Description, t.Status, t.CreatedAt, t.UpdatedAt,
			t.DueDate, t.RecurrenceType, t.RecurrenceParams, t.ParentTaskID, t.IsTemplate,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) DeleteFutureByParent(ctx context.Context, parentID int64, from time.Time) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM tasks WHERE parent_task_id = $1 AND due_date >= $2`,
		parentID, from,
	)
	return err
}
