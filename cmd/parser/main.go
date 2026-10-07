package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/DreadNought212/avia_bro/internal/external/travelpayouts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	fmt.Println("🚀 Запуск парсера Travelpayouts")
	fmt.Println(strings.Repeat("=", 50))

	// Подключаемся к БД
	fmt.Println("🔌 Подключение к БД...")
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к БД: %v", err)
	}
	defer conn.Close(ctx)
	fmt.Println("✅ Подключение успешно")

	// Создаём таблицу
	fmt.Println("📋 Создание таблицы flights...")
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS flights (
			id UUID PRIMARY KEY,
			origin VARCHAR(3) NOT NULL,
			destination VARCHAR(3) NOT NULL,
			origin_airport VARCHAR(3),
			destination_airport VARCHAR(3),
			depart_date DATE NOT NULL,
			return_date DATE,
			price BIGINT NOT NULL,
			currency VARCHAR(3) DEFAULT 'RUB',
			airline VARCHAR(100),
			flight_number VARCHAR(50),
			departure_at TIMESTAMP,
			return_at TIMESTAMP,
			transfers INT DEFAULT 0,
			return_transfers INT DEFAULT 0,
			duration INT,
			link TEXT,
			booking_link TEXT,
			parsed_at TIMESTAMP,
			created_at TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_flights_destination ON flights(destination);
		CREATE INDEX IF NOT EXISTS idx_flights_origin_dest ON flights(origin, destination);
		CREATE INDEX IF NOT EXISTS idx_flights_depart_date ON flights(depart_date);
		CREATE INDEX IF NOT EXISTS idx_flights_price ON flights(price);
	`
	_, err = conn.Exec(ctx, createTableSQL)
	if err != nil {
		log.Fatalf("❌ Ошибка создания таблицы: %v", err)
	}
	fmt.Println("✅ Таблица готова")

	// Парсим билеты через API
	fmt.Println("\n📡 Запрос к Travelpayouts API...")
	apiClient := travelpayouts.NewClient(apiKey)

	origin := "MOW"

	// Запрашиваем на месяц вперёд
	currentMonth := time.Now().Format("2006-01")

	flights, err := apiClient.SearchFlights(
		ctx,
		origin,
		currentMonth,
		"rub", // валюта
	)

	if err != nil {
		fmt.Printf("⚠️  Ошибка при запросе %s → %s: %v\n", origin, err)
	}

	allFlights := []travelpayouts.Flight{}

	allFlights = append(allFlights, flights...)
	time.Sleep(500 * time.Millisecond) // Пауза между запросами

	fmt.Printf("\n📥 Всего получено %d билетов\n", len(allFlights))

	// Сохраняем в БД
	fmt.Println("💾 Сохранение билетов в БД...")
	savedCount := 0
	for _, flight := range allFlights {
		flightID := uuid.New().String()

		// Парсим дату вылета
		departDate := flight.DepartureAt.Format("2006-01-02")

		// Парсим дату возврата (если есть)
		var returnDate *string
		if flight.ReturnAt != nil {
			rd := flight.ReturnAt.Format("2006-01-02")
			returnDate = &rd
		}

		insertSQL := `
			INSERT INTO flights (
				id, origin, destination, origin_airport, destination_airport,
				depart_date, return_date, price, currency, airline,
				flight_number, departure_at, return_at, transfers, return_transfers,
				duration, link, booking_link, parsed_at, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16, $17, $18, $19, $20
			)
			ON CONFLICT (id) DO NOTHING
		`

		_, err := conn.Exec(
			ctx, insertSQL,
			flightID,
			flight.Origin,
			flight.Destination,
			flight.OriginAirport,
			flight.DestinationAirport,
			departDate,
			returnDate,
			flight.Price,
			flight.Currency,
			flight.Airline,
			flight.FlightNumber,
			flight.DepartureAt,
			flight.ReturnAt,
			flight.Transfers,
			flight.ReturnTransfers,
			flight.Duration,
			flight.Link,
			flight.BookingLink,
			time.Now(),
			time.Now(),
		)

		if err == nil {
			savedCount++
		} else {
			fmt.Printf("⚠️  Ошибка при сохранении: %v\n", err)
		}
	}

	fmt.Printf("✅ Сохранено %d билетов\n", savedCount)

	// Выводим статистику
	fmt.Println("\n📊 Статистика:")
	showStats(ctx, conn)

	fmt.Println("\n✨ Готово!")
}

func showStats(ctx context.Context, conn *pgx.Conn) {
	// Общее количество
	var count int
	conn.QueryRow(ctx, "SELECT COUNT(*) FROM flights").Scan(&count)
	fmt.Printf("  📍 Всего билетов: %d\n", count)

	// По маршрутам
	rows, _ := conn.Query(ctx, `
		SELECT origin, destination, COUNT(*) as cnt, MIN(price) as min_price
		FROM flights
		GROUP BY origin, destination
		ORDER BY cnt DESC
		LIMIT 10
	`)
	defer rows.Close()

	fmt.Println("\n  🛫 Маршруты:")
	for rows.Next() {
		var origin, dest string
		var cnt int
		var minPrice int64
		if err := rows.Scan(&origin, &dest, &cnt, &minPrice); err == nil {
			fmt.Printf("    • %s → %s | %d билетов | мин цена: ₽%d\n",
				origin, dest, cnt, minPrice/100)
		}
	}

	// Средняя цена
	var avgPrice float64
	conn.QueryRow(ctx, "SELECT AVG(price) FROM flights").Scan(&avgPrice)
	fmt.Printf("\n  💰 Средняя цена: ₽%.0f\n", avgPrice/100)

	// Последние добавленные билеты
	fmt.Println("\n  📋 Последние добавленные билеты:")
	flightRows, _ := conn.Query(ctx, `
		SELECT origin, destination, airline, departure_at, price
		FROM flights
		ORDER BY created_at DESC
		LIMIT 5
	`)
	defer flightRows.Close()

	for flightRows.Next() {
		var origin, dest, airline string
		var departAt time.Time
		var price int64

		if err := flightRows.Scan(&origin, &dest, &airline, &departAt, &price); err == nil {
			fmt.Printf("    • %s → %s | %s | %s | ₽%d\n",
				origin, dest, airline,
				departAt.Format("2006-01-02"),
				price/100,
			)
		}
	}
}
