package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"

	"github.com/DreadNought212/avia_bro/internal/external/travelpayouts"
	"github.com/DreadNought212/avia_bro/internal/repository"
	"github.com/DreadNought212/avia_bro/internal/scheduler"
	"github.com/DreadNought212/avia_bro/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	// Загружаем .env
	godotenv.Load("config/.env")

	databaseURL := os.Getenv("DATABASE_URL")
	apiKey := os.Getenv("TRAVELPAYOUTS_API_KEY")

	if databaseURL == "" {
		log.Fatal("❌ DATABASE_URL не установлена")
	}
	if apiKey == "" {
		log.Fatal("❌ TRAVELPAYOUTS_API_KEY не установлена")
	}

	fmt.Println("🚀 Запуск Flight Sync Service")
	fmt.Println("=================================================================================================")

	// Подключаемся к БД
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("❌ Не удалось подключиться к БД: %v", err)
	}
	// Отложить вызов db.Close() до выхода из функции
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("❌ БД недоступна: %v", err)
	}
	fmt.Println("✅ Подключение к БД установлено")

	// Инициализируем зависимости
	apiClient := travelpayouts.NewClient(apiKey)
	flightRepo := repository.NewFlightRepository(db)
	routeRepo := repository.NewRouteRepository(db)
	syncLogRepo := repository.NewSyncLogRepository(db)
	notificationRuleRepo := repository.NewNotificationRuleRepository(db)

	syncService := service.NewFlightSyncService(
		apiClient,
		flightRepo,
		routeRepo,
		syncLogRepo,
		notificationRuleRepo,
	)

	scheduler := scheduler.NewScheduler(syncService, routeRepo)

	ctx := context.Background()

	// Первая синхронизация сразу после запуска
	fmt.Println("\n🚀 Запускаем первоначальную синхронизацию...")

	err = scheduler.RunOnce(ctx)
	if err != nil {
		log.Printf("❌ Ошибка первоначальной синхронизации: %v", err)
	}

	// Запускаем scheduler
	err = scheduler.Start(ctx)
	if err != nil {
		log.Fatalf("❌ Не удалось запустить scheduler: %v", err)
	}

	fmt.Println("\n✅ Сервис запущен и работает")
	fmt.Println("📡 Мониторинг маршрутов активен")
	fmt.Println("🛑 Для остановки нажмите Ctrl+C")

	// Ждём сигнала остановки
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	fmt.Println("\n\n🛑 Получен сигнал остановки...")
	scheduler.Stop()
	fmt.Println("👋 Сервис остановлен")
}
