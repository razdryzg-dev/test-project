package task

import (
	"context"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
	BulkCreate(ctx context.Context, tasks []taskdomain.Task) error
	DeleteFutureByParent(ctx context.Context, parentID int64, from time.Time) error
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
}

type CreateInput struct {
	Title            string                       `json:"title" validate:"required"`
	Description      string                       `json:"description"`
	Status           taskdomain.Status            `json:"status"`
	DueDate          *time.Time                   `json:"due_date,omitempty"`
	RecurrenceType   *string                      `json:"recurrence_type,omitempty"`
	RecurrenceParams *taskdomain.RecurrenceParams `json:"recurrence_params,omitempty"`
}

type UpdateInput struct {
	Title            string                       `json:"title,omitempty"`
	Description      string                       `json:"description,omitempty"`
	Status           taskdomain.Status            `json:"status,omitempty"`
	DueDate          *time.Time                   `json:"due_date,omitempty"`
	RecurrenceType   *string                      `json:"recurrence_type,omitempty"`
	RecurrenceParams *taskdomain.RecurrenceParams `json:"recurrence_params,omitempty"`
}
