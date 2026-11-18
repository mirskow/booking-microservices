# API

Все сервисы принимают и возвращают JSON. Для запросов с телом нужен заголовок `Content-Type: application/json`; неизвестные поля в теле отклоняются, максимальный размер тела — 1 МиБ. Идентификаторы — UUID, время — RFC 3339.

Формат ошибки у всех сервисов:

```json
{ "error": "описание ошибки" }
```

| Сервис | Базовый URL (с хоста) | Внутри Docker-сети |
|---|---|---|
| profile-service | `http://localhost:8081` | `http://profile-service:8080` |
| notification-service | `http://localhost:8082` | `http://notification-service:8080` |
| booking-service | `http://localhost:8083` | `http://booking-service:8080` |

## Общие эндпоинты (все сервисы)

| Метод | Путь | Описание | Ответ |
|---|---|---|---|
| GET | `/health/live` | Процесс жив | `200 {"status":"alive"}` |
| GET | `/health/ready` | Готовность, проверка БД | `200 {"status":"ready","database":"connected"}` или `503` |
| GET | `/metrics` | Метрики Prometheus | `200` text |

---

## profile-service

### Профили

| Метод | Путь | Описание | Успех |
|---|---|---|---|
| POST | `/profiles` | Создать профиль | `201` |
| GET | `/profiles/{id}` | Получить профиль | `200` |
| PUT | `/profiles/{id}` | Обновить профиль | `200` |

Тело `POST` / `PUT`:

```json
{
  "name": "Иван Петров",
  "email": "ivan@example.com",
  "timezone": "Europe/Moscow"
}
```

Правила: `name` обязателен, до 100 символов; `email` обязателен, корректный, до 255 символов, уникален без учёта регистра; `timezone` обязателен, IANA-название.

Ответ:

```json
{
  "id": "uuid",
  "name": "Иван Петров",
  "email": "ivan@example.com",
  "timezone": "Europe/Moscow",
  "created_at": "2026-01-01T10:00:00Z",
  "updated_at": "2026-01-01T10:00:00Z"
}
```

Коды ошибок: `400` (неверные данные или ID), `404` (профиль не найден), `409` (email уже существует), `500`.

### Расписание

| Метод | Путь | Описание | Успех |
|---|---|---|---|
| PUT | `/profiles/{id}/schedule` | Полностью заменить недельное расписание | `200` |
| GET | `/profiles/{id}/schedule` | Получить расписание | `200` |
| GET | `/profiles/{id}/slots?date=YYYY-MM-DD` | Свободные слоты на дату (в часовом поясе пользователя) | `200` |

Тело `PUT /schedule` (`weekday`: 0 — понедельник … 6 — воскресенье; время `HH:MM`; не более 100 правил; пересечения в один день запрещены):

```json
{
  "rules": [
    { "weekday": 0, "start_time": "09:00", "end_time": "12:00" },
    { "weekday": 2, "start_time": "14:00", "end_time": "18:00" }
  ]
}
```

Ответ `PUT` / `GET /schedule`:

```json
{
  "user_id": "uuid",
  "rules": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "weekday": 0,
      "start_time": "09:00:00",
      "end_time": "12:00:00",
      "created_at": "2026-01-01T10:00:00Z"
    }
  ]
}
```

Ответ `GET /slots` (длина слота задаётся `SLOT_DURATION`, по умолчанию 60 минут):

```json
{
  "user_id": "uuid",
  "date": "2026-02-02",
  "timezone": "Europe/Moscow",
  "slots": [
    { "start": "2026-02-02T09:00:00+03:00", "end": "2026-02-02T10:00:00+03:00" }
  ]
}
```

Коды ошибок: `400` (неверный ID, формат даты, отсутствует `date`, некорректное расписание), `404` (пользователь не найден), `500`.

---

## booking-service

| Метод | Путь | Описание | Успех |
|---|---|---|---|
| POST | `/bookings` | Создать встречу | `201` |
| GET | `/bookings/{id}` | Получить встречу | `200` |
| GET | `/bookings?user_id=...&limit=20&offset=0` | Список встреч пользователя | `200` |
| DELETE | `/bookings/{id}?requester_id=...` | Отменить встречу | `200` |

### POST /bookings

```json
{
  "organizer_id": "uuid",
  "invitee_id": "uuid",
  "title": "Планирование",
  "description": "Обсуждение задач",
  "starts_at": "2026-02-02T09:00:00+03:00",
  "ends_at": "2026-02-02T10:00:00+03:00"
}
```

Правила:

- `organizer_id` и `invitee_id` обязательны и различны.
- `title` обязателен, до 255 символов; `description` до 5000 символов.
- `starts_at` < `ends_at`, встреча не в прошлом.
- Интервал должен точно совпадать со слотом обоих участников (по данным profile-service).
- Пересечение с другой подтверждённой встречей участника запрещено.

Ответ (также для `GET /bookings/{id}` и `DELETE`):

```json
{
  "id": "uuid",
  "title": "Планирование",
  "description": "Обсуждение задач",
  "starts_at": "2026-02-02T09:00:00+03:00",
  "ends_at": "2026-02-02T10:00:00+03:00",
  "status": "confirmed",
  "organizer_id": "uuid",
  "invitee_id": "uuid",
  "created_at": "2026-01-01T10:00:00Z",
  "updated_at": "2026-01-01T10:00:00Z"
}
```

`status`: `confirmed` | `cancelled`.

### GET /bookings

Параметры: `user_id` (обязателен), `limit` (1–100, по умолчанию 20), `offset` (≥ 0, по умолчанию 0).

```json
{ "bookings": [ { "...": "объект встречи" } ] }
```

### DELETE /bookings/{id}

Параметр `requester_id` (обязателен) — отменить может только организатор или приглашённый. Повторная отмена возвращает встречу без повторного уведомления.

### Коды ответов booking-service

| Код | Причина |
|---|---|
| 400 | Неверные данные, ID, `limit`, `offset`, отсутствуют `user_id` / `requester_id` |
| 403 | Отмена не участником встречи |
| 404 | Встреча или пользователь не найдены |
| 409 | Слот недоступен или конфликт с другой встречей |
| 503 | profile-service недоступен (`dependent service unavailable`) |
| 500 | Внутренняя ошибка |

### Вызовы к другим сервисам

| Куда | Запрос | Когда |
|---|---|---|
| profile-service | `GET /profiles/{id}` | Создание (проверка участников), отмена (данные для письма) |
| profile-service | `GET /profiles/{id}/slots?date=...` | Создание (проверка слота) |
| notification-service | `POST /events` | После создания и отмены встречи (ошибка только логируется) |

---

## notification-service

| Метод | Путь | Описание | Успех |
|---|---|---|---|
| POST | `/events` | Принять событие и разослать уведомления | `200` |
| GET | `/notifications/{id}` | Получить уведомление | `200` |
| GET | `/notifications?recipient_id=...&limit=20&offset=0` | Уведомления получателя | `200` |

### POST /events

```json
{
  "event_id": "uuid",
  "booking_id": "uuid",
  "type": "booking.created",
  "title": "Планирование",
  "description": "Обсуждение задач",
  "starts_at": "2026-02-02T09:00:00+03:00",
  "ends_at": "2026-02-02T10:00:00+03:00",
  "recipients": [
    { "user_id": "uuid", "name": "Иван Петров", "email": "ivan@example.com" }
  ]
}
```

Правила: `type` — `booking.created` или `booking.cancelled`; `event_id`, `booking_id`, `title` (до 255 символов) обязательны; `starts_at` < `ends_at`; получателей от 1 до 100, без повторов, у каждого `user_id` и `name`. Пара `(event_id, recipient_id)` уникальна, уже отправленные уведомления повторно не отправляются.

Ответ:

```json
{
  "event_id": "uuid",
  "notifications": [
    {
      "id": "uuid",
      "event_id": "uuid",
      "recipient": { "user_id": "uuid", "name": "Иван Петров", "email": "ivan@example.com" },
      "booking_id": "uuid",
      "type": "booking.created",
      "channel": "email",
      "status": "sent",
      "attempts": 1,
      "created_at": "2026-01-01T10:00:00Z",
      "sent_at": "2026-01-01T10:00:01Z"
    }
  ]
}
```

`status`: `pending` | `sent` | `failed`. `channel`: `email`. Письма отправляются через SMTP (Mailpit, UI на `http://localhost:8025`).

### GET /notifications

Параметры: `recipient_id` (обязателен), `limit` (по умолчанию 20, максимум 100), `offset`. Возвращает массив объектов уведомления.

### Коды ответов notification-service

| Код | Причина |
|---|---|
| 400 | Неверный JSON, данные события, ID, `limit`, `offset` |
| 404 | Уведомление не найдено |
| 502 | Не все письма доставлены (в теле — `error`, `event_id`, `notifications`) |
| 500 | Внутренняя ошибка |

---

## Пример сквозного сценария

```bash
# 1. Создать двух пользователей
curl -X POST localhost:8081/profiles -H 'Content-Type: application/json' \
  -d '{"name":"Анна","email":"anna@example.com","timezone":"Europe/Moscow"}'
curl -X POST localhost:8081/profiles -H 'Content-Type: application/json' \
  -d '{"name":"Борис","email":"boris@example.com","timezone":"Europe/Moscow"}'

# 2. Задать расписание обоим (пример: понедельник 09:00-12:00)
curl -X PUT localhost:8081/profiles/<ID>/schedule -H 'Content-Type: application/json' \
  -d '{"rules":[{"weekday":0,"start_time":"09:00","end_time":"12:00"}]}'

# 3. Получить слоты на будущий понедельник
curl 'localhost:8081/profiles/<ID>/slots?date=<YYYY-MM-DD>'

# 4. Создать встречу по одному из слотов
curl -X POST localhost:8083/bookings -H 'Content-Type: application/json' \
  -d '{"organizer_id":"<ID1>","invitee_id":"<ID2>","title":"Встреча","description":"",
       "starts_at":"<slot.start>","ends_at":"<slot.end>"}'

# 5. Проверить уведомления и письма
curl 'localhost:8082/notifications?recipient_id=<ID1>'
# письма: http://localhost:8025
```
