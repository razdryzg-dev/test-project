package task

import (
	"sort"
	"time"
)

// GenerateOccurrences генерирует даты выполнения задачи-шаблона
// начиная с from, на limitDays дней вперёд.
func (t *Task) GenerateOccurrences(from time.Time, limitDays int) []time.Time {
	if t.RecurrenceType == nil || t.RecurrenceParams == nil {
		return nil
	}
	endDate := from.AddDate(0, 0, limitDays)
	var dates []time.Time

	switch *t.RecurrenceType {
	case "daily":
		interval := 1
		if t.RecurrenceParams.Interval != nil && *t.RecurrenceParams.Interval > 0 {
			interval = *t.RecurrenceParams.Interval
		}
		for d := from; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, interval) {
			dates = append(dates, d)
		}

	case "monthly":
		day := 1
		if t.RecurrenceParams.DayOfMonth != nil && *t.RecurrenceParams.DayOfMonth >= 1 && *t.RecurrenceParams.DayOfMonth <= 30 {
			day = *t.RecurrenceParams.DayOfMonth
		}
		// Найти первый подходящий месяц
		y, m, _ := from.Date()
		candidate := time.Date(y, m, day, 0, 0, 0, 0, from.Location())
		if candidate.Before(from) {
			candidate = candidate.AddDate(0, 1, 0)
		}
		for d := candidate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 1, 0) {
			if d.Day() == day { // пропуск несуществующих дней (29-30 февраля)
				dates = append(dates, d)
			}
		}

	case "specific_dates":
		for _, ds := range t.RecurrenceParams.Dates {
			parsed, err := time.Parse("2006-01-02", ds)
			if err != nil {
				continue
			}
			if (parsed.After(from) || parsed.Equal(from)) && (parsed.Before(endDate) || parsed.Equal(endDate)) {
				dates = append(dates, parsed)
			}
		}
		sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	case "odd_even":
		parity := "odd"
		if t.RecurrenceParams.Parity != nil {
			parity = *t.RecurrenceParams.Parity
		}
		for d := from; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
			dom := d.Day()
			if (parity == "odd" && dom%2 != 0) || (parity == "even" && dom%2 == 0) {
				dates = append(dates, d)
			}
		}
	}
	return dates
}
