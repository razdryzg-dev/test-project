package task

import "time"

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Task struct {
	ID               int64             `json:"id"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	Status           Status            `json:"status"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	DueDate          *time.Time        `json:"due_date,omitempty"`
	RecurrenceType   *string           `json:"recurrence_type,omitempty"`
	RecurrenceParams *RecurrenceParams `json:"recurrence_params,omitempty"`
	ParentTaskID     *int64            `json:"parent_task_id,omitempty"`
	IsTemplate       bool              `json:"is_template"`
}

type RecurrenceParams struct {
	Interval   *int     `json:"interval,omitempty"`
	DayOfMonth *int     `json:"day_of_month,omitempty"`
	Dates      []string `json:"dates,omitempty"`
	Parity     *string  `json:"parity,omitempty"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}
