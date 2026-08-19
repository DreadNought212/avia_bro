# ✈️ Aviasales Telegram Bot

Автоматический сервис для поиска выгодных авиабилетов, использующий AI для анализа и публикации лучших предложений в Telegram канал с реферальными ссылками.

![Status](https://img.shields.io/badge/status-alpha-yellow)
![Go](https://img.shields.io/badge/Go-1.21%2B-blue)
![License](https://img.shields.io/badge/license-MIT-green)

---

## 🎯 Основные возможности

- ✅ **Автоматический парсинг** Aviasales API каждые 30 минут
- ✅ **AI-powered анализ** с OpenAI/Claude для выбора лучших билетов
- ✅ **Умная фильтрация** (скидка >= 20%, исключение дублей, и т.д.)
- ✅ **Публикация в Telegram** каждые 3 часа с красивым форматированием
- ✅ **Реферальные ссылки** Aviasales для монетизации
- ✅ **Персонализация** профилей пользователей
- ✅ **Analytics & Monitoring** через Prometheus + Grafana
- ✅ **REST API** для доступа к данным и статистике
- ✅ **Production-ready** Docker & Kubernetes поддержка

---

## 🏗️ Архитектура

```
Parser (30 мин)  →  PostgreSQL  ↘
                       ↑         → Ranker (LLM, 3 часа)  →  Publisher  →  Telegram Channel
                      ↓         ↗
Cache (Redis)  ←  Analytics  

REST API (8080)  ←→  DB / Cache
```

**Компоненты:**
1. **Parser Service** — парсит Aviasales API, сохраняет билеты
2. **Ranker Service** — анализирует билеты с помощью LLM, выбирает топ 10
3. **Publisher Service** — публикует в Telegram с красивым форматом
4. **API Server** — REST endpoint для веб-приложения
5. **PostgreSQL** — основная БД с историей и аналитикой
6. **Redis** — кэш и rate limiting

---

## 🚀 Быстрый старт

### Требования

- **Go 1.21+**
- **Docker & Docker Compose**
- **PostgreSQL 15** (или в контейнере)
- **Redis 7** (или в контейнере)

### 1️⃣ Клонирование и настройка

```bash
git clone https://github.com/yourusername/aviasales-bot.git
cd aviasales-bot

# Инициальная настройка проекта
make setup

# Скопировать и заполнить .env
cp config/.env.example .env
# Редактируем .env с реальными API ключами
nano .env
```

### 2️⃣ Запуск через Docker Compose (Recommended)

```bash
# Запустить все сервисы
make docker-up

# Проверить статус
make ps

# Применить миграции БД
make migrate

# Просмотреть логи
make docker-logs
```

После запуска доступно:
- **API Server**: http://localhost:8080
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin / admin)
- **pgAdmin**: http://localhost:5050 (admin@example.com / admin)

### 3️⃣ Development режим (локально)

```bash
# Запустить только PostgreSQL и Redis
make db

# В отдельных терминалах запустить сервисы:
make run-parser
make run-ranker
make run-publisher
make run-api
```

---

## ⚙️ Конфигурация

### Обязательные переменные окружения

```bash
# Aviasales API ключ (получить на https://www.aviasales.com/api)
AVIASALES_API_KEY=your_key_here

# OpenAI API ключ (https://platform.openai.com/api-keys)
OPENAI_API_KEY=sk-...

# Telegram Bot Token (получить у @BotFather)
TELEGRAM_TOKEN=123456:ABC-DEF1234...

# ID Telegram канала (отрицательное число)
TELEGRAM_CHANNEL_ID=-1001234567890

# PostgreSQL
DATABASE_URL=postgres://bot:password@localhost:5432/aviasales_db
```

Все переменные описаны в `config/.env.example`

---

## 📊 REST API endpoints

### Просмотр билетов

```bash
# Получить билеты с фильтрами
curl "http://localhost:8080/api/v1/flights?origin=MOW&destination=BCN&limit=20"

# Получить конкретный билет
curl "http://localhost:8080/api/v1/flights/abc-123-uuid"

# Топ популярные маршруты
curl "http://localhost:8080/api/v1/analytics/top-routes"
```

### Профиль пользователя

```bash
# Установить предпочтения
curl -X POST "http://localhost:8080/api/v1/user/preferences" \
  -H "Content-Type: application/json" \
  -d '{
    "preferred_destinations": ["BCN", "PAR", "DUB"],
    "max_budget": 50000
  }'

# Получить профиль
curl "http://localhost:8080/api/v1/user/profile"
```

### Health Check

```bash
curl "http://localhost:8080/health"
```

Полная документация: [API.md](./docs/API.md)

---

## 🧪 Тестирование

```bash
# Unit тесты
make test

# Integration тесты (требует БД)
make test-int

# Coverage report
make coverage

# Lint
make lint
```

---

## 📈 Мониторинг

### Prometheus метрики

Доступны на `http://localhost:9090`:

```
aviasales_flights_parsed_total{origin="MOW"}
aviasales_parse_duration_seconds
aviasales_ranker_llm_requests_total
aviasales_publish_errors_total
aviasales_telegram_clicks_total
aviasales_conversion_rate{segment="budget_travelers"}
```

### Grafana Dashboard

Dashboard автоматически создаётся при запуске. Доступен на `http://localhost:3000`:

- Количество отпарсено билетов
- Топ маршруты
- Conversion rate по сегментам
- Ошибки и задержки
- Telegram клики

---

## 📁 Project Structure

```
aviasales-bot/
├── cmd/
│   ├── parser/main.go           # Parser Service entry point
│   ├── ranker/main.go           # Ranker Service entry point
│   ├── publisher/main.go        # Publisher Service entry point
│   └── api/main.go              # API Server entry point
│
├── internal/
│   ├── domain/                  # Доменные модели (Flight, User, etc.)
│   ├── repository/              # Слой доступа к данным (PostgreSQL, Redis)
│   ├── service/                 # Бизнес-логика (Parser, Ranker, Publisher)
│   ├── external/                # Интеграции с внешними API
│   └── middleware/              # Logging, Metrics, Auth
│
├── migrations/                  # SQL миграции БД
├── tests/                       # Unit & Integration тесты
├── scripts/                     # Вспомогательные скрипты
├── config/
│   ├── docker-compose.yml
│   ├── prometheus.yml
│   ├── grafana/
│   └── .env.example
│
├── Dockerfile.parser
├── Dockerfile.ranker
├── Dockerfile.publisher
├── Dockerfile.api
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 🔄 Development Workflow

### Запуск в development

```bash
# 1. Запустить БД
make db

# 2. Применить миграции
make migrate

# 3. Запустить сервисы (в разных терминалах)
make run-parser   # Terminal 1
make run-ranker   # Terminal 2
make run-publisher # Terminal 3
make run-api      # Terminal 4

# 4. Проверить логи
tail -f logs/parser.log
tail -f logs/ranker.log
```

### Добавление новой фичи

```bash
# 1. Создать новый коммит
git checkout -b feature/my-feature

# 2. Написать код + тесты
# 3. Запустить тесты
make test

# 4. Запустить linter
make lint

# 5. Commit и push
git add .
git commit -m "feat: description"
git push origin feature/my-feature
```

---

## 🐳 Docker

### Собрать images

```bash
make docker-build
```

### Production-like запуск

```bash
docker-compose -f config/docker-compose.yml up -d
```

### Логи

```bash
docker-compose logs -f parser
docker-compose logs -f ranker
docker-compose logs -f publisher
docker-compose logs -f api
```

### Подключиться к контейнеру

```bash
# PostgreSQL
make debug-db

# Redis
make debug-redis

# API сервис
docker-compose exec api sh
```

---

## 🚀 Deployment

### На VPS (простейший способ)

```bash
# 1. SSH на сервер
ssh user@your-vps.com

# 2. Клонировать проект
git clone https://github.com/yourusername/aviasales-bot.git
cd aviasales-bot

# 3. Скопировать .env
cp config/.env.example .env
nano .env  # Заполнить с production ключами

# 4. Запустить docker-compose
docker-compose -f config/docker-compose.yml up -d

# 5. Проверить
docker-compose ps
curl http://localhost:8080/health
```

### На Kubernetes (опционально)

```bash
# Применить manifests
kubectl apply -f k8s/

# Проверить поды
kubectl get pods -n aviasales

# Логи
kubectl logs -f deployment/parser -n aviasales
```

---

## 📚 Документация

- [API Documentation](./docs/API.md) — REST API endpoints
- [Database Schema](./docs/DB_SCHEMA.md) — Структура БД
- [Architecture](./docs/ARCHITECTURE.md) — Детальное описание архитектуры
- [LLM Prompts](./docs/LLM_PROMPTS.md) — Примеры промптов для анализа
- [Deployment Guide](./docs/DEPLOYMENT.md) — Production деплой

---

## 🐛 Troubleshooting

### Parser не парсит билеты

```bash
# Проверить логи
docker-compose logs parser

# Проверить API ключ в .env
echo $AVIASALES_API_KEY

# Проверить подключение к БД
make debug-db
SELECT COUNT(*) FROM flights;
```

### Ranker не ранжирует

```bash
# Проверить OpenAI API ключ
echo $OPENAI_API_KEY

# Проверить лимиты OpenAI
# https://platform.openai.com/account/rate-limits

# Проверить логи
docker-compose logs ranker
```

### Telegram не получает сообщения

```bash
# Проверить token и channel ID
docker-compose logs publisher

# Проверить, что бот администратор канала
# Проверить права доступа

# Отправить тестовое сообщение
curl -X POST https://api.telegram.org/bot${TELEGRAM_TOKEN}/sendMessage \
  -d chat_id=${TELEGRAM_CHANNEL_ID} \
  -d text="Test message"
```

---

## 💡 Рекомендации для расширения

### Фаза 1 (MVP) ✅
- [x] Parser Service
- [x] Basic DB schema
- [x] Ranker with LLM
- [x] Telegram Publisher
- [x] REST API

### Фаза 2 (User Features)
- [ ] User registration & authentication
- [ ] Personal preferences & favorites
- [ ] Email notifications
- [ ] Admin dashboard

### Фаза 3 (Monetization)
- [ ] Affiliate tracking & analytics
- [ ] Premium features
- [ ] A/B testing for offers
- [ ] Multi-channel publishing (Email, SMS)

### Фаза 4 (Scale)
- [ ] Kubernetes deployment
- [ ] Multi-region support
- [ ] Database sharding
- [ ] Real-time WebSocket updates

---

## 📝 License

MIT License — см. [LICENSE](./LICENSE)

---

## 👨‍💻 Contributing

Contributions приветствуются! Пожалуйста:

1. Fork репозиторий
2. Создайте feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit изменения (`git commit -m 'Add some AmazingFeature'`)
4. Push в branch (`git push origin feature/AmazingFeature`)
5. Откройте Pull Request

---

## 📧 Support

Вопросы? Проблемы?

- Создайте Issue на GitHub
- Обсудите в Discussions
- Напишите в Telegram

---

## 🎓 Resources

- [Aviasales API Docs](https://www.aviasales.com/api)
- [Telegram Bot API](https://core.telegram.org/bots/api)
- [OpenAI API](https://platform.openai.com/docs/api-reference)
- [PostgreSQL Docs](https://www.postgresql.org/docs/)
- [Go Best Practices](https://golang.org/doc/effective_go)

---

Made with ❤️ for flight deal hunters
