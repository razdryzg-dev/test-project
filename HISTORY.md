# История взаимодействия с LLM при выполнении тестового задания

## Проектирование архитектуры

**User:**  
Есть REST API на Go со структурой: cmd, internal (domain, usecase, repository, transport). Нужно добавить периодические задачи: daily (каждые N дней), monthly (в конкретное число месяца), specific_dates (на указанные даты), odd_even (чётные/нечётные дни). Опиши архитектуру: как хранить настройки, как генерировать экземпляры.

**Assistant:**  
Для реализации периодичности предлагаю хранить все данные в одной таблице `tasks`. Добавим следующие поля:
- `recurrence_type` (VARCHAR) – тип повторения (`daily`, `monthly`, `specific_dates`, `odd_even`)
- `recurrence_params` (JSONB) – параметры в зависимости от типа: для daily – `interval` (int), для monthly – `day_of_month` (int 1-30), для specific_dates – `dates` ([]string YYYY-MM-DD), для odd_even – `parity` ("odd"/"even")
- `parent_task_id` (BIGINT REFERENCES tasks(id) ON DELETE CASCADE) – связь экземпляра с шаблоном
- `is_template` (BOOLEAN DEFAULT FALSE) – флаг, указывающий, является ли запись шаблоном
- `due_date` (TIMESTAMPTZ) – дата выполнения (для экземпляров)

Задача-шаблон создаётся при указании `recurrence_type`. В момент создания генерируются экземпляры на ближайшие 30 дней и сохраняются в БД с `parent_task_id`, ссылающимся на шаблон. При обновлении шаблона старые будущие экземпляры удаляются, и генерируются новые. При удалении шаблона каскадно удаляются все экземпляры.

---

## Миграция базы данных

**User:**  
Напиши SQL-миграцию для добавления этих полей и соответствующий down-файл.

**Assistant:**  
`migrations/0002_add_recurrence_fields.up.sql`:
```sql
ALTER TABLE tasks 
    ADD COLUMN IF NOT EXISTS recurrence_type VARCHAR(20),
    ADD COLUMN IF NOT EXISTS recurrence_params JSONB,
    ADD COLUMN IF NOT EXISTS parent_task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS is_template BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS due_date TIMESTAMPTZ;
```

`migrations/0002_add_recurrence_fields.down.sql`:
```sql
ALTER TABLE tasks 
    DROP COLUMN IF EXISTS recurrence_type,
    DROP COLUMN IF EXISTS recurrence_params,
    DROP COLUMN IF EXISTS parent_task_id,
    DROP COLUMN IF EXISTS is_template,
    DROP COLUMN IF EXISTS due_date;
```

Применил миграцию через `docker exec -i postgres-1 psql ... < migrations/0002_...up.sql`.

---

## Доменная модель и генерация дат

**User:**  
Дополни структуру Task новыми полями и создай функцию генерации дат выполнения на основе параметров периодичности.

**Assistant:**  
Обновил `internal/domain/task/task.go`:
```go
type Task struct {
    ID               int64              `json:"id"`
    Title            string             `json:"title"`
    Description      string             `json:"description"`
    Status           Status             `json:"status"`
    CreatedAt        time.Time          `json:"created_at"`
    UpdatedAt        time.Time          `json:"updated_at"`
    DueDate          *time.Time         `json:"due_date,omitempty"`
    RecurrenceType   *string            `json:"recurrence_type,omitempty"`
    RecurrenceParams *RecurrenceParams  `json:"recurrence_params,omitempty"`
    ParentTaskID     *int64             `json:"parent_task_id,omitempty"`
    IsTemplate       bool               `json:"is_template"`
}

type RecurrenceParams struct {
    Interval   *int     `json:"interval,omitempty"`
    DayOfMonth *int     `json:"day_of_month,omitempty"`
    Dates      []string `json:"dates,omitempty"`
    Parity     *string  `json:"parity,omitempty"`
}
```

В новом файле `internal/domain/task/recurrence.go` реализовал метод `GenerateOccurrences(from time.Time, limitDays int) []time.Time`. Логика по типам:
- `daily`: генерируется каждые N дней (по умолчанию 1), начиная с `from`, на `limitDays` вперёд.
- `monthly`: каждая указанная дата месяца (1-30), пропуская несуществующие дни в коротких месяцах.
- `specific_dates`: только даты из массива, которые попадают в диапазон.
- `odd_even`: только чётные или только нечётные числа месяца.

---

## Репозиторий

**User:**  
Адаптируй методы репозитория под новые поля и добавь BulkCreate, DeleteFutureByParent.

**Assistant:**  
Обновил `internal/repository/postgres/task_repository.go`. Основные изменения:
- Функция `scanTask` заменена на `scanTaskFull`, считывающую все новые колонки.
- `Create`, `GetByID`, `Update`, `List` теперь включают поля `due_date`, `recurrence_type`, `recurrence_params`, `parent_task_id`, `is_template`.
- `List` возвращает только задачи с `is_template = FALSE`.
- Добавлен `BulkCreate` – транзакционная вставка нескольких экземпляров.
- Добавлен `DeleteFutureByParent(ctx, parentID, from)` – удаляет дочерние задачи с датой >= `from`, используется при обновлении шаблона для удаления будущих экземпляров.

---

## Сервисный слой и валидация

**User:**  
Как изменить сервис для создания/обновления шаблонов с генерацией экземпляров? Добавь валидацию параметров.

**Assistant:**  
В `internal/usecase/task/service.go` расширил `CreateInput` и `UpdateInput`, добавив поля `DueDate`, `RecurrenceType`, `RecurrenceParams`. В методе `Create`:
- Если задан `RecurrenceType`, задача помечается как шаблон (`IsTemplate = true`).
- После вставки шаблона вызывается `GenerateOccurrences(startDate, 30)`, где `startDate = DueDate` или текущее время.
- Экземпляры (со статусом `new`) массово сохраняются через `BulkCreate`.

В методе `Update`:
- Получаем существующую задачу, применяем изменения.
- Если передан новый `RecurrenceType`, удаляем будущие экземпляры (`DeleteFutureByParent`), обновляем параметры шаблона.
- Если `RecurrenceType` сброшен (пустая строка или null), шаблон превращается в обычную задачу, все дочерние удаляются.
- Если после обновления задача осталась шаблонной, заново генерируем экземпляры на 30 дней.

Добавлена функция `validateRecurrenceParams`, проверяющая корректность параметров в зависимости от типа:
- `daily`: если указан `interval`, то > 0.
- `monthly`: `day_of_month` от 1 до 30.
- `specific_dates`: массив не пуст, даты в формате YYYY-MM-DD.
- `odd_even`: `parity` равен "odd" или "even".

Обновлены `validateCreateInput` и `validateUpdateInput` для вызова этой проверки.

---

## HTTP-обработчики

**User:**  
Опиши изменения в обработчиках и DTO для работы с новыми полями.

**Assistant:**  
В `internal/transport/http/handlers/` файл с DTO и хендлерами. В `taskMutationDTO` и `taskDTO` добавлены поля:
```go
DueDate          *time.Time                `json:"due_date,omitempty"`
RecurrenceType   *string                   `json:"recurrence_type,omitempty"`
RecurrenceParams *taskdomain.RecurrenceParams `json:"recurrence_params,omitempty"`
```
В `taskDTO` также добавил `ParentTaskID` и `IsTemplate`. Функция `newTaskDTO` пробрасывает все поля из доменной модели.

В методах `Create` и `Update` хендлера новые поля маппятся в `CreateInput` / `UpdateInput` сервиса. Это позволило принимать JSON с периодичностью и возвращать полную информацию в ответах.

---

## Тестирование

**User:**  
Как протестировать работу периодичности через Swagger?

**Assistant:**  
После пересборки `docker compose up -d --build` открыл Swagger UI (`http://localhost:8080/swagger/`).

1. **POST /api/v1/tasks** с телом:
   ```json
   {
     "title": "Ежедневный обзвон",
     "status": "new",
     "due_date": "2026-05-03T10:00:00Z",
     "recurrence_type": "daily",
     "recurrence_params": {"interval": 1}
   }
   ```
   Получен ответ 201, шаблон с `is_template: true`. При запросе `GET /api/v1/tasks` появилось 30 экземпляров с датами от 3 мая до 2 июня с `parent_task_id = 1`.

2. **PUT /api/v1/tasks/1**, изменив интервал на 2. Старые экземпляры удалились, создались новые через день (15 задач).

3. Проверены другие типы:
    - `monthly` с `day_of_month: 15` – создались задачи на 15-е числа ближайших месяцев.
    - `odd_even` с `parity: "even"` – задачи на чётные дни.
    - `specific_dates` с массивом дат – ровно указанные даты.

4. **DELETE /api/v1/tasks/{id}** (шаблон) – каскадное удаление всех экземпляров.

Все операции отработали штатно, периодичность функционирует согласно требованиям.

---

## Выводы

При помощи LLM была спроектирована и реализована фича периодических задач для модуля трекера. В процессе были решены вопросы хранения настроек, генерации экземпляров, обновления и удаления шаблонов, а также протестирована работа API через Docker-окружение. Все изменения задокументированы в коде и README.
```