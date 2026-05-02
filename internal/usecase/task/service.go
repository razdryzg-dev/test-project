package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	now := s.now()
	model := &taskdomain.Task{
		Title:            normalized.Title,
		Description:      normalized.Description,
		Status:           normalized.Status,
		DueDate:          normalized.DueDate,
		RecurrenceType:   normalized.RecurrenceType,
		RecurrenceParams: normalized.RecurrenceParams,
		IsTemplate:       normalized.RecurrenceType != nil,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	// Если это шаблон — генерируем экземпляры на 30 дней
	if created.IsTemplate {
		startDate := now
		if created.DueDate != nil {
			startDate = *created.DueDate
		}
		instances := make([]taskdomain.Task, 0)
		for _, dt := range created.GenerateOccurrences(startDate, 30) {
			inst := taskdomain.Task{
				Title:        created.Title,
				Description:  created.Description,
				Status:       taskdomain.StatusNew, // или можно копировать статус
				DueDate:      &dt,
				ParentTaskID: &created.ID,
				IsTemplate:   false,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			instances = append(instances, inst)
		}
		if len(instances) > 0 {
			if err := s.repo.BulkCreate(ctx, instances); err != nil {
				return nil, err
			}
		}
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	// Получаем текущую задачу
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Применяем изменения из нормализованного ввода
	if normalized.Title != "" {
		current.Title = normalized.Title
	}
	if normalized.Description != "" {
		current.Description = normalized.Description
	}
	if normalized.Status != "" {
		current.Status = normalized.Status
	}
	if normalized.DueDate != nil {
		current.DueDate = normalized.DueDate
	}

	// Обработка периодичности
	now := s.now()

	// Если в запросе передан recurrence_type (может быть nil или строка)
	if normalized.RecurrenceType != nil && *normalized.RecurrenceType != "" {
		// Удаляем будущие экземпляры старого шаблона, если был
		if current.IsTemplate {
			_ = s.repo.DeleteFutureByParent(ctx, current.ID, now)
		}
		// Применяем новые параметры
		current.RecurrenceType = normalized.RecurrenceType
		current.RecurrenceParams = normalized.RecurrenceParams
		current.IsTemplate = true
	} else if normalized.RecurrenceType == nil || *normalized.RecurrenceType == "" {
		// Тип не передан или пуст → убираем периодичность
		if current.IsTemplate {
			_ = s.repo.DeleteFutureByParent(ctx, current.ID, now)
		}
		current.RecurrenceType = nil
		current.RecurrenceParams = nil
		current.IsTemplate = false
	}

	current.UpdatedAt = now

	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		return nil, err
	}

	// Генерируем новые экземпляры, если теперь это шаблон
	if updated.IsTemplate {
		startDate := now
		if updated.DueDate != nil {
			startDate = *updated.DueDate
		}
		instances := make([]taskdomain.Task, 0)
		for _, dt := range updated.GenerateOccurrences(startDate, 30) {
			inst := taskdomain.Task{
				Title:        updated.Title,
				Description:  updated.Description,
				Status:       taskdomain.StatusNew,
				DueDate:      &dt,
				ParentTaskID: &updated.ID,
				IsTemplate:   false,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			instances = append(instances, inst)
		}
		if len(instances) > 0 {
			if err := s.repo.BulkCreate(ctx, instances); err != nil {
				return nil, err
			}
		}
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}
	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}
	if input.RecurrenceType != nil {
		if err := validateRecurrenceParams(*input.RecurrenceType, input.RecurrenceParams); err != nil {
			return CreateInput{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
	}
	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	// Обрезаем пробелы, только если поле не пустое
	if input.Title != "" {
		input.Title = strings.TrimSpace(input.Title)
		if input.Title == "" {
			return UpdateInput{}, fmt.Errorf("%w: title must not be empty after trimming", ErrInvalidInput)
		}
	}
	if input.Description != "" {
		input.Description = strings.TrimSpace(input.Description)
	}
	// Статус: если передан, проверяем валидность
	if input.Status != "" {
		if !input.Status.Valid() {
			return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
		}
	}
	// Проверка параметров периодичности, если задан тип
	if input.RecurrenceType != nil && *input.RecurrenceType != "" {
		if err := validateRecurrenceParams(*input.RecurrenceType, input.RecurrenceParams); err != nil {
			return UpdateInput{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
	}
	return input, nil
}

func validateRecurrenceParams(recurType string, params *taskdomain.RecurrenceParams) error {
	if params == nil {
		return fmt.Errorf("recurrence_params is required for type %s", recurType)
	}
	switch recurType {
	case "daily":
		if params.Interval != nil && *params.Interval <= 0 {
			return fmt.Errorf("interval must be > 0")
		}
	case "monthly":
		if params.DayOfMonth == nil || *params.DayOfMonth < 1 || *params.DayOfMonth > 30 {
			return fmt.Errorf("day_of_month must be 1..30")
		}
	case "specific_dates":
		if len(params.Dates) == 0 {
			return fmt.Errorf("dates cannot be empty")
		}
		// можно проверить формат дат
	case "odd_even":
		if params.Parity == nil || (*params.Parity != "odd" && *params.Parity != "even") {
			return fmt.Errorf("parity must be 'odd' or 'even'")
		}
	default:
		return fmt.Errorf("unknown recurrence type: %s", recurType)
	}
	return nil
}
