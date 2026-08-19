# 🚀 IMPLEMENTATION CHECKLIST - Aviasales Bot на Go

---

## НЕДЕЛЯ 1: FOUNDATION (Project Setup)

### ✅ Day 1-2: Project Initialization
- [ ] Создать GitHub репозиторий
- [ ] Инициализировать Go проект (`go mod init`)
- [ ] Скопировать структуру папок:
  ```
  cmd/
  internal/
  migrations/
  tests/
  config/
  ```
- [ ] Создать Makefile с основными командами
- [ ] Создать docker-compose.yml (PostgreSQL + Redis)
- [ ] Создать .env.example с переменными

### ✅ Day 3-4: Database Setup
- [ ] Создать миграции (001_initial.sql)
- [ ] Создать schema для flights, published_flights, users, etc.
- [ ] Создать индексы для быстрого поиска
- [ ] Протестировать подключение PostgreSQL
- [ ] Создать views для аналитики
- [ ] Заполнить тестовые данные

### ✅ Day 5-7: Domain Models & Repository Layer
- [ ] Написать domain models:
  - [ ] Flight struct
  - [ ] PublishedFlight struct
  - [ ] UserProfile struct
  - [ ] UserAction struct
- [ ] Написать repository interfaces
- [ ] Реализовать PostgreSQL repository
  - [ ] SaveFlight()
  - [ ] GetRecentFlights()
  - [ ] FindFlights()
  - [ ] GetAveragePrice()
- [ ] Написать unit тесты для repository

---

## НЕДЕЛЯ 2: PARSER SERVICE

### ✅ Day 8-9: Aviasales API Client
- [ ] Создать external/aviasales/client.go
- [ ] Реализовать SearchFlights() метод
- [ ] Обработать Aviasales API response
- [ ] Добавить retry logic & circuit breaker
- [ ] Обработать ошибки и rate limiting
- [ ] Написать тесты с mock'ом

### ✅ Day 10-11: Parser Service
- [ ] Создать service/parser.go
- [ ] Реализовать parseAllRoutes() (параллельно горутины)
- [ ] Добавить фильтрацию дублей
- [ ] Реализовать convertToFlight()
- [ ] Добавить логирование

### ✅ Day 12-13: Parser Scheduling & Redis Caching
- [ ] Добавить Redis client в cmd/parser
- [ ] Реализовать кэширование в Redis
- [ ] Создать scheduled job (каждые 30 мин)
- [ ] Написать graceful shutdown
- [ ] Протестировать end-to-end
- [ ] Проверить, что билеты сохраняются в БД

### ✅ Day 14: Parser Docker & Testing
- [ ] Создать Dockerfile.parser
- [ ] Написать integration тесты
- [ ] Протестировать с docker-compose up
- [ ] Проверить логи
- [ ] Оптимизировать performance

---

## НЕДЕЛЯ 3: RANKER SERVICE (LLM)

### ✅ Day 15-16: OpenAI Integration
- [ ] Создать external/llm/client.go
- [ ] Реализовать Complete() метод для OpenAI
- [ ] Написать buildPrompt() для анализа билетов
- [ ] Обработать JSON response от LLM
- [ ] Добавить error handling & retry

### ✅ Day 17-18: Ranker Service Logic
- [ ] Создать service/ranker.go
- [ ] Реализовать filterFlights() с бизнес-правилами:
  - [ ] Скидка >= 20%
  - [ ] Не дублировать маршруты 48 часов
  - [ ] Вылет не раньше 1 дня
  - [ ] Популярные направления только
- [ ] Реализовать rankWithLLM()
- [ ] Вычислить score для каждого билета
- [ ] Выбрать топ 10 билетов

### ✅ Day 19-20: Ranker Publishing & Scheduling
- [ ] Создать PublishedFlightRepository
- [ ] Сохранять выбранные билеты
- [ ] Создать scheduled job (каждые 3 часа)
- [ ] Написать тесты для фильтрации
- [ ] Оптимизировать LLM запросы (batching)

### ✅ Day 21: Ranker Docker & Integration
- [ ] Создать Dockerfile.ranker
- [ ] Протестировать с docker-compose
- [ ] Проверить, что LLM правильно анализирует
- [ ] Оптимизировать запросы
- [ ] Настроить логирование

---

## НЕДЕЛЯ 4: PUBLISHER & API

### ✅ Day 22-23: Telegram Publisher
- [ ] Создать service/publisher.go
- [ ] Реализовать PublishFlights()
- [ ] Написать formatMessage() (красивый формат)
- [ ] Добавить inline keyboard buttons
- [ ] Реализовать отслеживание сообщений
- [ ] Добавить rate limiting (не спамить)

### ✅ Day 24: Telegram Webhook Handler
- [ ] Создать webhook handler для callback_query
- [ ] Логировать пользовательские действия
- [ ] Отправлять реф. ссылку в PM
- [ ] Считать conversion rate

### ✅ Day 25-26: REST API Server
- [ ] Создать cmd/api/main.go (Gin)
- [ ] Реализовать endpoints:
  - [ ] GET /api/v1/flights
  - [ ] GET /api/v1/flights/:id
  - [ ] POST /api/v1/user/preferences
  - [ ] GET /api/v1/user/profile
  - [ ] GET /api/v1/analytics/top-routes
  - [ ] GET /health
- [ ] Добавить JWT auth (опционально)
- [ ] Написать тесты для API

### ✅ Day 27: Docker & Integration
- [ ] Создать Dockerfile.api
- [ ] Создать Dockerfile.publisher
- [ ] Протестировать docker-compose с всеми сервисами
- [ ] Убедиться, что данные проходят от Parser → Ranker → Publisher

---

## НЕДЕЛЯ 5: MONITORING & DEPLOYMENT

### ✅ Day 28-29: Monitoring
- [ ] Добавить Prometheus метрики:
  - [ ] aviasales_flights_parsed_total
  - [ ] aviasales_ranker_llm_requests_total
  - [ ] aviasales_publish_errors_total
  - [ ] aviasales_telegram_clicks_total
- [ ] Добавить Prometheus service в docker-compose
- [ ] Создать Grafana dashboard
- [ ] Добавить Grafana сервис в docker-compose

### ✅ Day 30-31: Logging & Error Handling
- [ ] Добавить structured logging (zap)
- [ ] Создать error_logs таблицу
- [ ] Логировать все ошибки
- [ ] Добавить Sentry integration (опционально)
- [ ] Настроить log rotation

### ✅ Day 32-33: Testing & Documentation
- [ ] Написать полный integration test
- [ ] Написать README с инструкциями
- [ ] Создать API документацию (OpenAPI/Swagger?)
- [ ] Создать диаграммы архитектуры
- [ ] Протестировать end-to-end

### ✅ Day 34-35: Production Readiness
- [ ] Добавить graceful shutdown для всех сервисов
- [ ] Добавить healthcheck endpoints
- [ ] Убедиться, что нет memory leaks
- [ ] Оптимизировать database queries
- [ ] Настроить логирование для production

---

## ФИНАЛЬНАЯ CHECKLIST

### Code Quality
- [ ] Все тесты проходят (`make test`)
- [ ] Lint успешен (`make lint`)
- [ ] Coverage >= 80% для критичных путей
- [ ] Нет TODO/FIXME в коде (или они задокументированы)
- [ ] Code review сделан

### Database
- [ ] Миграции работают с нуля
- [ ] Индексы созданы правильно
- [ ] Views работают
- [ ] Резервная копия schema готова

### Docker
- [ ] docker-compose up -d работает без ошибок
- [ ] Все контейнеры healthy
- [ ] Пробы health check работают
- [ ] Логи читаемы и полезны
- [ ] Volume binding работает правильно

### API
- [ ] Все endpoints работают
- [ ] Error responses правильного формата
- [ ] Rate limiting работает
- [ ] CORS настроен (если нужно)

### Telegram
- [ ] Бот правильно находит канал
- [ ] Сообщения форматируются красиво
- [ ] Кнопки работают
- [ ] Emoji отображаются правильно

### Monitoring
- [ ] Prometheus собирает метрики
- [ ] Grafana dashboard отображает данные
- [ ] Алерты настроены (опционально)

### Documentation
- [ ] README заполнен
- [ ] .env.example актуален
- [ ] Architecture.md понятен
- [ ] API документирован

### Security
- [ ] Нет hardcoded secrets
- [ ] SQL injections невозможны
- [ ] .env не коммитится
- [ ] API ключи в переменных окружения

---

## 📊 TIMELINE

```
┌─────────────────────────────────────────────────────┐
│  НЕДЕЛЯ 1 | НЕДЕЛЯ 2 | НЕДЕЛЯ 3 | НЕДЕЛЯ 4 | НЕДЕЛЯ 5 │
│ Foundation│ Parser  │ Ranker  │ Publisher│ Monitoring│
│           │         │         │  & API   │           │
│  ████████ │ ████    │ ████    │  ████    │  ████     │
└─────────────────────────────────────────────────────┘

Total: ~35 дней (80 часов)
```

---

## 🎓 DEPENDENCIES TO INSTALL

```bash
# Web & HTTP
go get github.com/gin-gonic/gin

# Database
go get github.com/jackc/pgx/v5

# Cache
go get github.com/redis/go-redis/v9

# Scheduling
go get github.com/go-co-op/gocron/v2

# Telegram
go get github.com/go-telegram-bot-api/telegram-bot-api/v5

# LLM
go get github.com/sashabaranov/go-openai

# Config & Logging
go get github.com/joho/godotenv
go get go.uber.org/zap

# Monitoring
go get github.com/prometheus/client_golang

# Utilities
go get github.com/google/uuid
```

---

## 🔗 IMPORTANT LINKS

- [Aviasales API Docs](https://www.aviasales.com/api)
- [Telegram Bot API](https://core.telegram.org/bots/api)
- [OpenAI API Reference](https://platform.openai.com/docs/api-reference)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Redis Documentation](https://redis.io/documentation)
- [Go Best Practices](https://golang.org/doc/effective_go)
- [Docker Documentation](https://docs.docker.com/)

---

## 💡 TIPS FOR SUCCESS

1. **Начни с Parser Service** — самая простая часть
2. **Тестируй на каждом шаге** — не жди конца недели
3. **Используй Docker с начала** — экономит время на setup
4. **Логирование критично** — добавь с первого дня
5. **Запусти dry-run перед публикацией** — не спамь сразу
6. **Мониторинг с начала** — поймёшь, где узкие места
7. **API ключи в .env** — никогда не коммить в репо!

---

## 🎁 BONUS: Quick Commands Reference

```bash
# Setup
make setup
make db
make migrate

# Development
make run-parser
make run-ranker
make run-publisher
make run-api

# Testing
make test
make coverage
make lint

# Docker
make docker-up
make docker-down
make docker-logs

# Database
make debug-db
make debug-redis
make seed

# Deployment
make build
make docker-build
```

---

Ready to build? Let's go! 🚀

P.S. Если застрял на каком-то компоненте — посмотри на example_code.go, там готовые примеры всех основных частей.
