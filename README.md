# Task Service

Сервис для управления задачами с HTTP API на Go.

## Требования

- Go `1.23+`
- Docker и Docker Compose

## Быстрый запуск через Docker Compose

```bash
docker compose up --build
```

После запуска сервис будет доступен по адресу `http://localhost:8080`.

Если `postgres` уже запускался ранее со старой схемой, пересоздай volume:

```bash
docker compose down -v
docker compose up --build
```

Причина в том, что SQL-файл из `migrations/0001_create_tasks.up.sql` монтируется в `docker-entrypoint-initdb.d` и применяется только при инициализации пустого data volume.

## Swagger

Swagger UI:

```text
http://localhost:8080/swagger/
```

OpenAPI JSON:

```text
http://localhost:8080/swagger/openapi.json
```

## API

Базовый префикс API:

```text
/api/v1
```

Основные маршруты:

- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}`

## Периодичность задач

Реализована возможность задавать повторяющиеся задачи (шаблоны) с автоматической генерацией экземпляров.

### Типы периодичности
- **daily** – каждые N дней (параметр `interval`, целое > 0).
- **monthly** – каждый месяц в указанное число (`day_of_month`, 1-30).
- **specific_dates** – только на конкретные даты (массив строк `dates` в формате YYYY-MM-DD).
- **odd_even** – только по чётным или нечётным дням месяца (`parity`: "odd"/"even").

### Принцип работы
- При создании задачи с `recurrence_type` она помечается как шаблон (`is_template: true`) и в БД сразу создаются экземпляры на ближайшие 30 дней.
- Список задач (`GET /api/v1/tasks`) возвращает только экземпляры (`is_template: false`).
- При обновлении шаблона (PUT) старые будущие экземпляры удаляются и генерируются новые согласно обновлённым параметрам.
- При удалении шаблона каскадно удаляются все связанные экземпляры.

### Предположения и ограничения
- Горизонт генерации – 30 дней.
- Параметры периодичности хранятся в JSONB.
- Статус новых экземпляров – `new`.
- Схема БД дополнена миграцией 0002_add_recurrence_fields

_Выполнено в рамках тестового задания._
