# Makefile для Aviasales Telegram Bot

.PHONY: help setup dev db migrate test docker-up docker-down deploy

help:
	@echo "Aviasales Telegram Bot - Available commands:"
	@echo ""
	@echo "Setup & Development:"
	@echo "  make setup         - Первоначальная настройка проекта"
	@echo "  make dev           - Запуск проекта в development режиме"
	@echo "  make build         - Собрать все сервисы"
	@echo ""
	@echo "Database:"
	@echo "  make db            - Запустить PostgreSQL и Redis"
	@echo "  make migrate       - Применить миграции БД"
	@echo "  make seed          - Заполнить test данные"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  make test          - Запустить unit тесты"
	@echo "  make test-int      - Запустить integration тесты"
	@echo "  make coverage      - Coverage report"
	@echo "  make lint          - Запустить linter"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up     - Запустить docker-compose"
	@echo "  make docker-down   - Остановить контейнеры"
	@echo "  make docker-logs   - Показать логи"
	@echo ""
	@echo "Deployment:"
	@echo "  make deploy        - Деплой на production"
	@echo "  make logs-prod     - Логи production"

# Setup
setup:
	@echo "🔧 Setting up project..."
	go mod download
	go mod tidy
	cp config/.env.example .env
	@echo "✅ Setup complete. Edit .env file with your API keys"

dev:
	@echo "🚀 Starting development services..."
	docker-compose -f config/docker-compose.yml up -d postgres redis
	@sleep 5
	@echo "✅ PostgreSQL and Redis are running"
	@echo ""
	@echo "Running services (use separate terminals):"
	@echo "  Terminal 1: make run-parser"
	@echo "  Terminal 2: make run-ranker"
	@echo "  Terminal 3: make run-publisher"
	@echo "  Terminal 4: make run-api"

run-parser:
	cd cmd/parser && go run main.go

run-ranker:
	cd cmd/ranker && go run main.go

run-publisher:
	cd cmd/publisher && go run main.go

run-api:
	cd cmd/api && go run main.go

# Build
build:
	@echo "🏗️ Building services..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/parser ./cmd/parser
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/ranker ./cmd/ranker
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/publisher ./cmd/publisher
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/api ./cmd/api
	@echo "✅ Build complete"

# Database
db:
	@echo "🗄️ Starting PostgreSQL and Redis..."
	docker-compose -f config/docker-compose.yml up -d postgres redis

migrate:
	@echo "🔄 Running migrations..."
	@cd migrations && \
	for file in *.sql; do \
		echo "  → $$file"; \
		psql $(DATABASE_URL) -f $$file; \
	done
	@echo "✅ Migrations complete"

seed:
	@echo "🌱 Seeding test data..."
	go run scripts/seed_data.go
	@echo "✅ Test data loaded"

# Testing
test:
	@echo "🧪 Running unit tests..."
	go test -v -race -timeout 30s ./internal/...

test-int:
	@echo "🧪 Running integration tests..."
	go test -v -tags=integration -timeout 120s ./tests/integration/...

coverage:
	@echo "📊 Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage: coverage.html"

lint:
	@echo "🔍 Linting code..."
	golangci-lint run ./...

# Docker
docker-up:
	@echo "🐳 Starting docker-compose..."
	docker-compose -f config/docker-compose.yml up -d
	@echo "✅ All services are running"
	@echo ""
	@echo "Services:"
	@echo "  API:        http://localhost:8080"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Grafana:    http://localhost:3000"

docker-down:
	@echo "⛔ Stopping docker-compose..."
	docker-compose -f config/docker-compose.yml down
	@echo "✅ Services stopped"

docker-logs:
	docker-compose -f config/docker-compose.yml logs -f

docker-build:
	@echo "🏗️ Building Docker images..."
	docker-compose -f config/docker-compose.yml build
	@echo "✅ Images built"

# Deployment
deploy:
	@echo "🚀 Deploying to production..."
	@echo "Run this on your server:"
	@echo "  git pull origin main"
	@echo "  make build"
	@echo "  docker-compose -f config/docker-compose.yml up -d"
	@echo "✅ Deployment instructions printed"

logs-prod:
	@echo "📋 Production logs:"
	@docker-compose -f config/docker-compose.yml logs -f --tail=100

# Utilities
fmt:
	@echo "💅 Formatting code..."
	go fmt ./...

clean:
	@echo "🧹 Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -testcache
	@echo "✅ Clean complete"

install-tools:
	@echo "🛠️ Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest
	@echo "✅ Tools installed"

ps:
	@echo "🐳 Running containers:"
	docker-compose -f config/docker-compose.yml ps

restart:
	@echo "♻️ Restarting services..."
	docker-compose -f config/docker-compose.yml restart
	@echo "✅ Services restarted"

debug-db:
	@echo "💻 Connecting to PostgreSQL..."
	docker-compose -f config/docker-compose.yml exec postgres psql -U bot -d aviasales_db

debug-redis:
	@echo "💻 Connecting to Redis..."
	docker-compose -f config/docker-compose.yml exec redis redis-cli
