# 🏗️ Архитектура Aviasales Telegram Bot (Go Backend)

## 📐 Высокоуровневая архитектура

```
┌─────────────────────────────────────────────────────────────────┐
│                         EXTERNAL SERVICES                        │
├─────────────────────────────────────────────────────────────────┤
│  Aviasales API │ OpenAI/Claude API │ Telegram Bot API │ Datadog   │
└────────┬────────────────┬─────────────────┬──────────────┬────────┘
         │                │                 │              │
         ▼                ▼                 ▼              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     GO BACKEND SERVICES                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────┐  ┌──────────────────┐ ┌──────────────────┐ │
│  │  Parser Service  │  │ Ranker Service   │ │ Publisher Service │ │
│  │  (Aviasales)     │  │  (LLM based)     │ │ (Telegram)       │ │
│  └────────┬─────────┘  └────────┬─────────┘ └────────┬─────────┘ │
│           │                     │                    │           │
│  ┌────────▼─────────────────────▼────────────────────▼─────────┐ │
│  │         Scheduler (APScheduler / Temporal)                   │ │
│  │  - Каждые 30 мин: парсим билеты                             │ │
│  │  - Каждые 3 часа: LLM ранжирует + публикует                 │ │
│  │  - Каждые 12 часов: очистка кэша                            │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌──────────────────┐  ┌──────────────────┐ ┌──────────────────┐ │
│  │  HTTP API Server │  │ Webhook Handler  │ │ Admin Dashboard  │ │
│  │  (REST)          │  │ (Telegram events)│ │ (Internal)       │ │
│  └──────────────────┘  └──────────────────┘ └──────────────────┘ │
│                                                                   │
└────────┬───────────────────────────────────────────────────────┬─┘
         │                                                         │
         ▼                                                         ▼
┌──────────────────────┐   ┌──────────────────┐  ┌──────────────────┐
│   PostgreSQL 15      │   │  Redis Cache     │  │  S3 (для логов)  │
│                      │   │                  │  │                  │
│ ├─ Flights           │   │ ├─ Cache flights │  │ ├─ User logs     │
│ ├─ UserProfiles      │   │ ├─ User prefs    │  │ ├─ Analytics     │
│ ├─ PurchaseHistory   │   │ └─ API rate limiting │ └─ Backups      │
│ ├─ Analytics         │   │                  │  │                  │
│ └─ AuditLog          │   └──────────────────┘  └──────────────────┘
└──────────────────────┘
```

---

## 🔧 Компоненты системы (микросервисная архитектура)

### 1. **Parser Service** 🔍
**Назначение:** Периодически парсит Aviasales API и сохраняет билеты

**Функции:**
- Запрашивает данные у Aviasales каждые 30 минут
- Валидирует и очищает данные
- Де-дублирует билеты
- Сохраняет в PostgreSQL
- Кэширует в Redis для быстрого доступа

**Технологии:**
- Go (goroutines для параллельных запросов)
- HTTP client с retry + circuit breaker
- Protobuf для структур данных

**Входные данные:**
```go
type FlightRequest struct {
    Origin      string    // "MOW"
    Destination string    // "BCN"
    DepartDate  time.Time
    ReturnDate  *time.Time // опционально
    Adults      int
}
```

**Выходные данные:**
```go
type Flight struct {
    ID              string
    Origin          string
    Destination     string
    DepartDate      time.Time
    ReturnDate      *time.Time
    Price           int64  // в копейках
    Currency        string // "RUB"
    Airline         string
    DepartTime      time.Time
    ArrivalTime     time.Time
    Duration        int       // минут
    Transfers       int
    BookingLink     string    // Aviasales реферальная ссылка
    ParsedAt        time.Time
    Score           float32   // для ранжирования (вычисляется потом)
}
```

---

### 2. **Ranker Service** 🤖 (LLM-powered)
**Назначение:** Ранжирует билеты и выбирает топ для публикации

**Алгоритм:**
1. Берёт последние N билетов из БД
2. Фильтрует по правилам (не дублировать маршруты за 24 часа, и т.д.)
3. Запрашивает у LLM анализ (OpenAI / Claude):
   - Почему этот билет выгодный?
   - На какой сегмент пользователей ориентирован?
   - Привлекательное описание (200 символов max)
4. Вычисляет скор: `(старая_цена - новая_цена) / старая_цена`
5. Берёт топ 5-10 билетов
6. Сохраняет выбранные билеты в таблицу `PublishedFlights`

**Prompt для LLM (пример):**
```
Ты - эксперт в путешествиях. Проанализируй авиабилет:
- Маршрут: Москва → Барселона
- Цена: 25,500 RUB (была 55,000 RUB)
- Дата вылета: 15 мая 2024
- Пересадки: 2
- Время в пути: 14ч 30м

Ответь JSON:
{
  "appeal": "Отличная цена для весеннего отпуска в Европу!",
  "segment": "budget_travelers", // family, luxury, business, budget_travelers
  "discount_percent": 54,
  "why_good": "Цена упала на 54% - редкий случай"
}
```

**Правила фильтрации:**
- Скидка должна быть >= 20%
- Не постим один маршрут чаще, чем раз в 48 часов
- Только рейсы на вылет в будущем (не раньше, чем через 1 день)
- Исключить непопулярные направления (если нет истории букингов)

---

### 3. **Publisher Service** 📢
**Назначение:** Отправляет выбранные билеты в Telegram канал

**Функции:**
- Форматирует сообщение (красиво и кратко)
- Добавляет реферальную ссылку Aviasales
- Отправляет в Telegram канал
- Отслеживает клики (через URL shortener с tracking)
- Логирует результат отправки

**Формат сообщения:**
```
✈️ ВЫГОДНЫЙ БИЛЕТ

🛫 Москва → Барселона
📅 15 мая - 22 мая (7 дней)
💰 25 500 ₽ (-54%, было 55 000 ₽)
⏱ Вылет: 14:20 → Прибытие: 18:45+2
⭐ Рейтинг авиакомпании: 4.8/5
🪑 2 пересадки

Для бюджетных путешественников в Европу — отличный вариант!

[🔗 Купить билет](https://aviasales.ru/ref/abc123)

#полеты #барселона #весна #скидка #путешествия
```

---

### 4. **Scheduler** ⏰
**Назначение:** Управляет периодическими задачами

**Расписание:**
```
┌─ Каждые 30 минут (06:00 - 23:00):
│  └─ Parser Service запрашивает новые билеты
│
├─ Каждые 3 часа (08:00, 11:00, 14:00, 17:00, 20:00, 23:00):
│  └─ Ranker Service выбирает топ и публикует
│
├─ Каждые 12 часов (00:00, 12:00):
│  └─ Analytics: пересчитываем статистику
│
├─ Каждые 24 часа (02:00):
│  └─ Cleanup: удаляем старые билеты (старше 30 дней)
│
└─ Каждый понедельник (09:00):
   └─ Weekly digest для пользователей
```

**Реализация:**
- Используем `robfig/cron` или `go-co-op/gocron`
- Распределённый лок через Redis (чтобы не запускалось на всех инстансах)

---

### 5. **REST API Server** 🌐
**Назначение:** Внешний интерфейс для веб-приложения и статистики

**Endpoints:**

```
GET  /api/v1/flights
     ?origin=MOW&destination=BCN&sort=price&limit=20
     → Список билетов с фильтрами

GET  /api/v1/flights/:id
     → Подробная информация о билете

POST /api/v1/user/preferences
     Body: { preferred_destinations: ["BCN", "PAR"], budget: 50000 }
     → Сохранить предпочтения пользователя

GET  /api/v1/user/profile
     → Профиль пользователя в Telegram

GET  /api/v1/analytics/top-routes
     → Топ популярные маршруты (для dashboard)

GET  /api/v1/health
     → Health check для balancer
```

**Фреймворк:**
- `gin-gonic/gin` или `gorilla/mux` (легче, чем большие фреймворки)
- JWT auth для пользователей (если нужна приватная статистика)

---

### 6. **Telegram Webhook Handler** 🤖
**Назначение:** Слушает события от Telegram (нажатия кнопок, сообщения)

**Функции:**
- Обрабатывает callback_query (когда пользователь нажимает кнопку)
- Сохраняет действия в БД для analytics
- Отправляет реферальную ссылку в приватном сообщении

**Пример взаимодействия:**
```
Пользователь видит в канале сообщение с кнопкой [✈️ Купить]
         ↓
Нажимает кнопку
         ↓
Telegram отправляет webhook на наш сервер
         ↓
Мы логируем: user_id, flight_id, action="click_buy"
         ↓
Отправляем в PM пользователю: прямую ссылку на Aviasales
```

---

## 🗄️ Схема базы данных PostgreSQL

```sql
-- Основная таблица авиабилетов
CREATE TABLE flights (
    id UUID PRIMARY KEY,
    origin VARCHAR(3) NOT NULL,
    destination VARCHAR(3) NOT NULL,
    depart_date DATE NOT NULL,
    return_date DATE,
    price BIGINT NOT NULL, -- копейки
    currency VARCHAR(3) DEFAULT 'RUB',
    airline VARCHAR(50),
    depart_time TIME,
    arrival_time TIME,
    duration_minutes INT,
    transfers INT,
    booking_link TEXT,
    parsed_at TIMESTAMP DEFAULT NOW(),
    score FLOAT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_origin_dest (origin, destination),
    INDEX idx_depart_date (depart_date),
    INDEX idx_price (price),
    INDEX idx_parsed_at (parsed_at)
);

-- Опубликованные билеты (история)
CREATE TABLE published_flights (
    id UUID PRIMARY KEY,
    flight_id UUID NOT NULL REFERENCES flights(id),
    telegram_message_id BIGINT,
    appeal TEXT, -- текст, который генерировала LLM
    segment VARCHAR(50), -- budget_travelers, family, luxury, etc.
    published_at TIMESTAMP DEFAULT NOW(),
    clicks INT DEFAULT 0,
    conversions INT DEFAULT 0,
    
    INDEX idx_published_at (published_at),
    INDEX idx_flight_id (flight_id)
);

-- Профили пользователей
CREATE TABLE user_profiles (
    telegram_user_id BIGINT PRIMARY KEY,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    username VARCHAR(255),
    preferred_origins VARCHAR(3)[] DEFAULT ARRAY[]::VARCHAR(3)[],
    preferred_destinations VARCHAR(3)[] DEFAULT ARRAY[]::VARCHAR(3)[],
    max_budget INT,
    min_rating FLOAT DEFAULT 4.0,
    notification_enabled BOOLEAN DEFAULT TRUE,
    language VARCHAR(10) DEFAULT 'ru',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- История действий для аналитики
CREATE TABLE user_actions (
    id BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT REFERENCES user_profiles(telegram_user_id),
    action_type VARCHAR(50), -- view, click_buy, share, etc.
    flight_id UUID REFERENCES flights(id),
    published_flight_id UUID REFERENCES published_flights(id),
    metadata JSONB, -- любые доп данные
    created_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_user_action (telegram_user_id, action_type),
    INDEX idx_created_at (created_at)
);

-- Кэш статистики (для быстрого доступа)
CREATE TABLE analytics_cache (
    id SERIAL PRIMARY KEY,
    metric_type VARCHAR(50), -- top_routes, conversion_rate, etc.
    data JSONB,
    calculated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE (metric_type)
);

-- Лог ошибок
CREATE TABLE error_logs (
    id BIGSERIAL PRIMARY KEY,
    service VARCHAR(50),
    error_message TEXT,
    stack_trace TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_service (service),
    INDEX idx_created_at (created_at)
);
```

---

## 🚀 Go Project Structure

```
aviasales-bot/
├── cmd/
│   ├── parser/
│   │   └── main.go          # Parser Service
│   ├── ranker/
│   │   └── main.go          # Ranker Service
│   ├── publisher/
│   │   └── main.go          # Publisher Service
│   └── api/
│       └── main.go          # REST API Server
│
├── internal/
│   ├── domain/
│   │   ├── flight.go        # Domain models
│   │   ├── user.go
│   │   └── analytics.go
│   │
│   ├── repository/
│   │   ├── postgres/
│   │   │   ├── flight.go    # SQL queries for flights
│   │   │   ├── user.go
│   │   │   └── analytics.go
│   │   └── cache/
│   │       └── redis.go     # Cache layer
│   │
│   ├── service/
│   │   ├── parser.go        # Parser business logic
│   │   ├── ranker.go        # Ranker with LLM
│   │   ├── publisher.go     # Telegram publisher
│   │   └── analytics.go     # Analytics
│   │
│   ├── external/
│   │   ├── aviasales/
│   │   │   └── client.go    # Aviasales API client
│   │   ├── llm/
│   │   │   └── client.go    # OpenAI / Claude client
│   │   ├── telegram/
│   │   │   └── client.go    # Telegram Bot API
│   │   └── config/
│   │       └── config.go    # Environment variables
│   │
│   └── middleware/
│       ├── logger.go        # Logging
│       ├── metrics.go       # Prometheus metrics
│       └── auth.go          # JWT verification
│
├── migrations/
│   ├── 001_initial.sql
│   ├── 002_add_analytics.sql
│   └── ...
│
├── tests/
│   ├── unit/
│   │   ├── service_test.go
│   │   └── ranker_test.go
│   └── integration/
│       ├── aviasales_test.go
│       └── postgres_test.go
│
├── scripts/
│   ├── seed_data.go         # Для тестирования
│   └── backfill_analytics.go
│
├── config/
│   ├── .env.example
│   ├── docker-compose.yml   # PostgreSQL, Redis, etc.
│   └── kubernetes/          # k8s manifests (опционально)
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 📦 Go Dependencies (go.mod)

```go
require (
    github.com/gin-gonic/gin v1.9.1
    github.com/jackc/pgx/v5 v5.4.3
    github.com/redis/go-redis/v9 v9.0.5
    github.com/go-co-op/gocron/v2 v2.0.0
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
    github.com/sashabaranov/go-openai v1.15.4
    github.com/joho/godotenv v1.5.1
    github.com/google/uuid v1.3.0
    github.com/sirupsen/logrus v1.9.3
    github.com/prometheus/client_golang v1.17.0
    go.uber.org/zap v1.26.0
    github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.0 // опционально, если нужен gRPC
)
```

---

## 🔄 Deployment & Scaling

### Production Setup (Docker Compose):
```yaml
version: '3.9'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: bot
      POSTGRES_PASSWORD: secure_password
      POSTGRES_DB: aviasales_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  parser:
    build:
      context: .
      dockerfile: Dockerfile.parser
    environment:
      - DATABASE_URL=postgres://bot:secure_password@postgres:5432/aviasales_db
      - REDIS_URL=redis://redis:6379
      - AVIASALES_API_KEY=${AVIASALES_API_KEY}
    depends_on:
      - postgres
      - redis

  ranker:
    build:
      context: .
      dockerfile: Dockerfile.ranker
    environment:
      - DATABASE_URL=postgres://bot:secure_password@postgres:5432/aviasales_db
      - REDIS_URL=redis://redis:6379
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    depends_on:
      - postgres
      - redis

  publisher:
    build:
      context: .
      dockerfile: Dockerfile.publisher
    environment:
      - DATABASE_URL=postgres://bot:secure_password@postgres:5432/aviasales_db
      - TELEGRAM_TOKEN=${TELEGRAM_TOKEN}
      - TELEGRAM_CHANNEL_ID=${TELEGRAM_CHANNEL_ID}
    depends_on:
      - postgres

  api:
    build:
      context: .
      dockerfile: Dockerfile.api
    environment:
      - DATABASE_URL=postgres://bot:secure_password@postgres:5432/aviasales_db
      - REDIS_URL=redis://redis:6379
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis

volumes:
  postgres_data:
```

---

## 🔐 Окружающие переменные (.env)

```
# Database
DATABASE_URL=postgres://user:password@localhost:5432/aviasales_db
DB_MAX_CONNECTIONS=20
DB_QUERY_TIMEOUT=30s

# Redis
REDIS_URL=redis://localhost:6379/0

# External APIs
AVIASALES_API_KEY=your_api_key_here
OPENAI_API_KEY=sk-...
TELEGRAM_TOKEN=123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11

# Telegram
TELEGRAM_CHANNEL_ID=-1001234567890  # отрицательное число для приватного канала
TELEGRAM_WEBHOOK_URL=https://yourdomain.com/webhook/telegram

# Service Config
LOG_LEVEL=info
PARSER_INTERVAL=30m
RANKER_INTERVAL=3h
ENVIRONMENT=production

# LLM Config
LLM_PROVIDER=openai  # или anthropic
LLM_MODEL=gpt-4-turbo

# Monitoring
SENTRY_DSN=https://...@sentry.io/...
PROMETHEUS_PORT=9090
```

---

## 🎯 Основной flow выполнения

```
[PARSER Service]
    ↓ Каждые 30 мин запрашивает Aviasales API
    ↓ Парсит JSON ответ
    ↓ Вставляет в PostgreSQL
    ↓ Кэширует в Redis
    ↓
[RANKER Service]
    ↓ Каждые 3 часа просыпается
    ↓ Берёт последние N билетов из БД
    ↓ Фильтрует (скидка >= 20%, не дублировать 48h, и т.д.)
    ↓ Отправляет в OpenAI/Claude для анализа
    ↓ Вычисляет score и рейтинг
    ↓ Отбирает топ 5-10 билетов
    ↓ Сохраняет в published_flights
    ↓
[PUBLISHER Service]
    ↓ Форматирует сообщение для Telegram
    ↓ Добавляет реферальную ссылку
    ↓ Отправляет в канал
    ↓ Логирует message_id для tracking кликов
    ↓
[USER clicks button in Telegram]
    ↓ Telegram отправляет webhook
    ↓ API обрабатывает callback_query
    ↓ Логируем в user_actions
    ↓ Отправляем в PM ссылку
    ↓ Считаем conversion rate
```

---

## 📊 Monitoring & Observability

**Metrics (Prometheus):**
```
aviasales_flights_parsed_total{origin="MOW"} 1234
aviasales_parse_duration_seconds{quantile="0.95"} 0.52
aviasales_ranker_llm_requests_total 456
aviasales_publish_errors_total 2
aviasales_telegram_clicks_total 567
aviasales_conversion_rate{segment="budget_travelers"} 0.08
```

**Logging (structured):**
- Используем `zap` или `logrus`
- Логируем в stdout (для Docker logs)
- Отправляем в ELK / Datadog для аналитики

---

## ✅ Checklist для MVP

- [ ] Структура проекта на Go
- [ ] Aviasales API client (парсер)
- [ ] PostgreSQL migration scripts
- [ ] Parser Service (сохранение билетов)
- [ ] Redis кэширование
- [ ] OpenAI интеграция для LLM
- [ ] Ranker Service (выбор топ билетов)
- [ ] Telegram Bot API интеграция
- [ ] Publisher Service (отправка в канал)
- [ ] REST API endpoints
- [ ] Docker Compose для локального разворота
- [ ] Unit тесты для critical paths
- [ ] Graceful shutdown & error handling
- [ ] Monitoring (Prometheus)
- [ ] Деплой на VPS / K8s

---

## 🎓 Рекомендуемые библиотеки для Go

| Функция | Библиотека | Причина |
|---------|-----------|---------|
| HTTP клиент | `net/http` (стд) | Встроено, достаточно мощно |
| БД драйвер | `jackc/pgx` | Самый быстрый, асинхронный |
| ORM (опционально) | `sqlc` или `ent` | Типобезопасные queries |
| REST фреймворк | `gin-gonic/gin` | Быстро, минималистично |
| Scheduling | `go-co-op/gocron` | Простой API для cron jobs |
| Telegram | `go-telegram-bot-api` | Официально поддерживается |
| LLM | `sashabaranov/go-openai` | Официальный SDK OpenAI |
| Logging | `uber-go/zap` | Структурированное логирование |
| Config | `joho/godotenv` | Загрузка .env файлов |
| UUID | `google/uuid` | Стандартная библиотека |

---

## 🚀 Примерная timeline для MVP

| Неделя | Задачи | Часов |
|--------|--------|-------|
| 1 | Проект setup, БД schema, Parser Service | 20 |
| 2 | LLM интеграция, Ranker Service | 20 |
| 3 | Telegram Publisher, REST API | 20 |
| 4 | Тестирование, Docker, Мониторинг | 20 |

**Итого:** ~80 часов для полностью рабочего MVP

