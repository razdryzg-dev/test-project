package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title            string                       `json:"title"`
	Description      string                       `json:"description"`
	Status           taskdomain.Status            `json:"status"`
	DueDate          *time.Time                   `json:"due_date,omitempty"`
	RecurrenceType   *string                      `json:"recurrence_type,omitempty"`
	RecurrenceParams *taskdomain.RecurrenceParams `json:"recurrence_params,omitempty"`
}

type taskDTO struct {
	ID               int64                        `json:"id"`
	Title            string                       `json:"title"`
	Description      string                       `json:"description"`
	Status           taskdomain.Status            `json:"status"`
	CreatedAt        time.Time                    `json:"created_at"`
	UpdatedAt        time.Time                    `json:"updated_at"`
	DueDate          *time.Time                   `json:"due_date,omitempty"`
	RecurrenceType   *string                      `json:"recurrence_type,omitempty"`
	RecurrenceParams *taskdomain.RecurrenceParams `json:"recurrence_params,omitempty"`
	ParentTaskID     *int64                       `json:"parent_task_id,omitempty"`
	IsTemplate       bool                         `json:"is_template"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	return taskDTO{
		ID:               task.ID,
		Title:            task.Title,
		Description:      task.Description,
		Status:           task.Status,
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
		DueDate:          task.DueDate,
		RecurrenceType:   task.RecurrenceType,
		RecurrenceParams: task.RecurrenceParams,
		ParentTaskID:     task.ParentTaskID,
		IsTemplate:       task.IsTemplate,
	}
}
