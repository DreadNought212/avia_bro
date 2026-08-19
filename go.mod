module github.com/yourusername/aviasales-bot

go 1.21

require (
	// Web Framework & HTTP
	github.com/gin-gonic/gin v1.9.1
	github.com/gorilla/mux v1.8.1
	
	// Database
	github.com/jackc/pgx/v5 v5.4.3
	github.com/jackc/pgconn v1.14.0
	
	// Cache & Redis
	github.com/redis/go-redis/v9 v9.0.5
	
	// Scheduling
	github.com/go-co-op/gocron/v2 v2.0.0
	
	// Telegram Bot
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	
	// LLM APIs
	github.com/sashabaranov/go-openai v1.15.4
	github.com/anthropics/anthropic-sdk-go v0.0.0 // optional
	
	// Configuration
	github.com/joho/godotenv v1.5.1
	github.com/spf13/viper v1.17.0
	github.com/spf13/cobra v1.7.0
	
	// Utilities
	github.com/google/uuid v1.3.0
	github.com/oklog/ulid/v2 v2.1.0
	
	// Logging
	github.com/sirupsen/logrus v1.9.3
	go.uber.org/zap v1.26.0
	
	// Monitoring & Metrics
	github.com/prometheus/client_golang v1.17.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/uber/jaeger-client-go v2.30.0
	
	// Error Handling
	github.com/pkg/errors v0.9.1
	
	// Testing
	github.com/stretchr/testify v1.8.4
	github.com/testcontainers/testcontainers-go v0.26.0
	github.com/golang/mock v1.6.0
	
	// JSON & Serialization
	github.com/json-iterator/go v1.1.12
	
	// Rate Limiting & Circuit Breaker
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	
	// Time & Timezone
	github.com/google/go-cmp v0.6.0
	
	// CLI
	github.com/urfave/cli/v2 v2.25.1
)

require (
	// Transitive dependencies
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/sync v0.4.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
)
