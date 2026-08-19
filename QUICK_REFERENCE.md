# 📋 ПОЛНАЯ АРХИТЕКТУРА AVIASALES BOT - КРАТКАЯ СПРАВКА

---

## 🎯 Что это?

Автоматический сервис на Go, который:
1. **Каждые 30 мин** парсит Aviasales API
2. **Сохраняет** авиабилеты в PostgreSQL
3. **Каждые 3 часа** анализирует билеты через LLM (ChatGPT)
4. **Выбирает** топ 10 самых выгодных
5. **Публикует** в Telegram канал с реф. ссылками
6. **Отслеживает** клики и конверсию
7. **Предоставляет** REST API для доступа к данным

---

## 🏗️ АРХИТЕКТУРНАЯ ДИАГРАММА

```
┌─────────────────────────────────────────────────────────────────┐
│                        EXTERNAL SERVICES                         │
├─────────────────────────────────────────────────────────────────┤
│  Aviasales API  │  OpenAI API  │  Telegram API  │  S3 (логи)    │
└────────┬────────────────┬────────────────┬──────────┬────────────┘
         │                │                │          │
         ▼                ▼                ▼          ▼
┌──────────────────────────────────────────────────────────────────┐
│                    GO BACKEND (Microservices)                    │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │   PARSER     │  │    RANKER    │  │  PUBLISHER  │           │
│  │  ▼ 30 мин   │  │  ▼ 3 часа    │  │  ▼ Async    │           │
│  │             │  │  (LLM based) │  │             │           │
│  └──────┬───────┘  └──────┬───────┘  └──────┬──────┘           │
│         │                 │                  │                   │
│  ┌──────▼─────────────────▼──────────────────▼─────────────┐    │
│  │            Scheduler (gocron / APScheduler)             │    │
│  │  Управляет всеми периодическими задачами              │    │
│  └──────────────────────────────────────────────────────┘    │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │  REST API    │  │   Telegram   │  │  Admin       │           │
│  │  (Gin)       │  │   Webhook    │  │  Dashboard   │           │
│  │  Port: 8080  │  │   Handler    │  │  (Internal)  │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
│                                                                   │
└─────────┬──────────────────────────────────────────────────────┬─┘
          │                                                      │
          ▼                                                      ▼
┌──────────────────────┐  ┌──────────────┐  ┌─────────────────────┐
│   PostgreSQL 15      │  │  Redis 7     │  │  S3 (Backups)       │
│                      │  │              │  │                     │
│  ├─ flights          │  │  ├─ Cache    │  │  ├─ Error logs      │
│  ├─ published_       │  │  ├─ Rate-    │  │  ├─ Analytics       │
│  │  flights          │  │  │  limiting │  │  └─ Backups         │
│  ├─ user_profiles    │  │  └─ Sessions│  │                     │
│  ├─ user_actions     │  │             │  │                     │
│  └─ analytics_cache  │  └─────────────┘  └─────────────────────┘
└──────────────────────┘

МОНИТОРИНГ:
└─ Prometheus (metrics) → Grafana (dashboards)
```

---

## 📊 DATA FLOW (Поток данных)

```
Aviasales API                     LLM (OpenAI)              Telegram Channel
      │                               │                            ▲
      │                               │                            │
      ▼                               ▼                            │
   PARSER ──────────┬─────────→ RANKER ─────────┬────────→ PUBLISHER
      │             │              │             │             │
      │             ▼              ▼             ▼             │
   Raw Data ──→ PostgreSQL ──→ Analyze & ──→ Format & ──→ Send Message
                              Rank          Beautify        + Track
                              
   Кэш в Redis ←─────────────────────────────────────────────────

USER ACTIONS:
Пользователь видит пост → Нажимает кнопку → Webhook → Логируем в БД
```

---

## 🔧 КОМПОНЕНТЫ (Детально)

### 1. PARSER SERVICE ✈️

**Что делает:**
- Запрашивает данные у Aviasales API каждые 30 минут
- Парсит JSON ответ
- Фильтрует дублипликаты
- Сохраняет в PostgreSQL
- Кэширует в Redis

**Входные данные:**
```json
{
  "origin": "MOW",
  "destination": "BCN",
  "depart_date": "2024-05-15"
}
```

**Выходные данные:**
```json
{
  "id": "uuid-123",
  "origin": "MOW",
  "destination": "BCN",
  "price": 2550000,  // копейки!
  "airline": "Aeroflot",
  "depart_date": "2024-05-15",
  "booking_link": "https://aviasales.ru/...",
  "score": 0.0
}
```

**Интервал:** 30 минут (парллельно 5 маршрутов)

---

### 2. RANKER SERVICE 🤖

**Что делает:**
- Берёт последние N билетов из БД
- Применяет фильтры (скидка >= 20%, не дублировать, и т.д.)
- Отправляет в OpenAI/Claude для анализа
- Вычисляет score (привлекательность)
- Выбирает топ 10
- Сохраняет в таблицу `published_flights`

**Промпт для LLM:**
```
Ты — эксперт в путешествиях. Проанализируй билет:
- Москва → Барселона
- 25,500 RUB (было 55,000 RUB, -54%)
- 15 мая, 2 пересадки

Ответь JSON:
{
  "appeal": "Отличное предложение для весеннего отпуска!",
  "segment": "budget_travelers",
  "discount_percent": 54,
  "why_good": "Цена упала на половину!"
}
```

**Интервал:** 3 часа

---

### 3. PUBLISHER SERVICE 📢

**Что делает:**
- Форматирует красивое сообщение
- Отправляет в Telegram канал
- Сохраняет ID сообщения
- Отслеживает клики

**Пример сообщения:**
```
✈️ ВЫГОДНЫЙ БИЛЕТ

🛫 Москва → Барселона
📅 15 мая - 22 мая (7 дней)
💰 25 500 ₽ (-54%, было 55 000 ₽)
⏱ Вылет: 14:20 → Прибытие: 18:45+2
🪑 2 пересадки

Отличное предложение для весеннего отпуска в Европу!

[✈️ Купить билет]
```

**Интервал:** Синхронизирован с Ranker (каждые 3 часа)

---

### 4. REST API 🌐

**Endpoints:**

| Метод | Endpoint | Описание |
|-------|----------|---------|
| GET | `/api/v1/flights` | Получить билеты с фильтрами |
| GET | `/api/v1/flights/:id` | Детали билета |
| GET | `/api/v1/analytics/top-routes` | Топ маршруты |
| POST | `/api/v1/user/preferences` | Сохранить предпочтения |
| GET | `/api/v1/user/profile` | Профиль пользователя |
| GET | `/health` | Health check |

**Порт:** 8080

---

## 🗄️ DATABASE SCHEMA

### flights
```sql
id (UUID)              -- уникальный ID
origin (VARCHAR 3)     -- MOW, SPB, etc.
destination            -- BCN, PAR, etc.
depart_date (DATE)
return_date (DATE)
price (BIGINT)         -- в копейках!
currency               -- RUB
airline                -- Aeroflot
depart_time (TIME)
arrival_time           -- TIME
duration_minutes       -- INT
transfers              -- INT (количество пересадок)
booking_link           -- TEXT (с реф параметром)
score (FLOAT)          -- для ранжирования
parsed_at (TIMESTAMP)
created_at / updated_at
```

### published_flights
```sql
id (UUID)
flight_id (FK)         -- ссылка на flights
telegram_message_id    -- ID сообщения в Telegram
appeal (TEXT)          -- почему это хорошо
segment (VARCHAR)      -- budget_travelers, family, luxury
published_at
clicks (INT)           -- количество кликов
conversions (INT)      -- количество покупок
```

### user_profiles
```sql
telegram_user_id (BIGINT, PK)
username (VARCHAR)
preferred_origins (ARRAY)      -- ["MOW", "SPB"]
preferred_destinations (ARRAY) -- ["BCN", "PAR"]
max_budget (INT)
notification_enabled (BOOL)
language (VARCHAR)
created_at / updated_at
```

### user_actions
```sql
id (BIGSERIAL)
telegram_user_id (FK)
action_type (VARCHAR)  -- view, click_buy, share
flight_id (FK)
published_flight_id (FK)
metadata (JSONB)       -- любые доп данные
created_at
```

---

## 🚀 QUICK START

### Минимальная настройка (5 минут):

```bash
# 1. Клонировать
git clone https://github.com/yourusername/aviasales-bot
cd aviasales-bot

# 2. Скопировать .env
cp config/.env.example .env

# 3. Заполнить .env
nano .env
# AVIASALES_API_KEY=...
# OPENAI_API_KEY=sk-...
# TELEGRAM_TOKEN=...
# TELEGRAM_CHANNEL_ID=-1001234567890

# 4. Запустить (Docker)
make docker-up

# 5. Проверить
curl http://localhost:8080/health
```

### Логирование:
```bash
docker-compose logs -f parser
docker-compose logs -f ranker
docker-compose logs -f publisher
```

---

## 📈 METRICS & MONITORING

**Prometheus метрики доступны на** `http://localhost:9090`:

```
aviasales_flights_parsed_total{origin="MOW"}
aviasales_parse_duration_seconds
aviasales_ranker_llm_requests_total
aviasales_publish_errors_total
aviasales_telegram_clicks_total
aviasales_conversion_rate{segment="budget_travelers"}
```

**Grafana Dashboard:** `http://localhost:3000`

---

## 🔒 SECURITY & PERFORMANCE

### Performance Optimization:
- Redis кэш для частых запросов
- Индексы на всех часто используемых полях
- Параллельные goroutines для парсинга (5 одновременно)
- Connection pooling для БД
- Rate limiting для API

### Security:
- Переменные окружения для sensitive данных
- SQL injection protection (parameterized queries)
- Rate limiting на API endpoints
- Validation всех входных данных
- Error logs без exposure sensitive info

---

## 🎯 ROADMAP

### ✅ MVP (Фаза 1)
- Parser Service
- Basic DB schema
- Ranker with LLM
- Telegram Publisher
- REST API

### 📋 Phase 2 (User Features)
- [ ] User registration
- [ ] Personalized preferences
- [ ] Email notifications
- [ ] Admin dashboard

### 💰 Phase 3 (Monetization)
- [ ] Affiliate tracking
- [ ] Premium features
- [ ] A/B testing
- [ ] Multi-channel publishing

### 📈 Phase 4 (Scale)
- [ ] Kubernetes
- [ ] Multi-region support
- [ ] Database sharding
- [ ] WebSocket real-time updates

---

## 📚 KEY FILES

| File | Описание |
|------|----------|
| `cmd/parser/main.go` | Entry point для Parser Service |
| `cmd/ranker/main.go` | Entry point для Ranker Service |
| `cmd/publisher/main.go` | Entry point для Publisher Service |
| `cmd/api/main.go` | Entry point для API Server |
| `internal/domain/` | Domain models (Flight, User, etc.) |
| `internal/repository/` | Database layer |
| `internal/service/` | Business logic |
| `internal/external/` | External API clients |
| `config/docker-compose.yml` | Docker configuration |
| `migrations/001_initial.sql` | Database schema |
| `Makefile` | Build & deployment commands |

---

## 🛠️ TECH STACK

| Слой | Технология |
|------|-----------|
| **Language** | Go 1.21+ |
| **Web Framework** | Gin Gonic |
| **Database** | PostgreSQL 15 |
| **Cache** | Redis 7 |
| **Scheduler** | gocron |
| **Telegram** | telegram-bot-api |
| **LLM** | OpenAI API (ChatGPT-4) |
| **Monitoring** | Prometheus + Grafana |
| **Logging** | Zap + Logrus |
| **Containers** | Docker + Docker Compose |

---

## 📞 SUPPORT

Вопросы? Проблемы?

- GitHub Issues
- GitHub Discussions
- Telegram: @yourbot

---

## 📝 NOTES

1. **Price в копейках!** 25500₽ = 2550000 в БД
2. **Реф ссылка** добавляется в booking_link при парсинге
3. **LLM стоит денег** (~$0.01 за запрос на GPT-4)
4. **Telegram rate limiting** - макс 30 сообщений в секунду
5. **PostgreSQL индексы** критичны для производительности

---

Made with ❤️ for flight deal hunters

**Время на разработку MVP:** ~80 часов
**Сложность:** Medium
**Потенциальный доход:** Зависит от реф. rate Aviasales (обычно 5-15%)
