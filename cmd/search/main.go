package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/DreadNought212/avia_bro/internal/external/travelpayouts"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

// main — точка входа в программу.
//
// Функция:
// 1. Загружает конфигурацию из .env.
// 2. Получает API-ключ Travelpayouts.
// 3. Создаёт API-клиент.
// 4. Выполняет поиск авиабилетов за указанный период.
// 5. Сортирует найденные билеты.
// 6. Выводит подробную информацию о билетах.
// 7. Выводит общую статистику по результатам поиска.
func main() {
	// Загружаем переменные окружения из файла config/.env.
	godotenv.Load("config/.env")

	// Получаем API-ключ Travelpayouts из переменной окружения.
	apiKey := os.Getenv("TRAVELPAYOUTS_API_KEY")

	// Если API-ключ не найден, дальнейшая работа невозможна.
	if apiKey == "" {
		log.Fatal("❌ TRAVELPAYOUTS_API_KEY не установлена")
	}

	fmt.Println("🚀 Запуск парсера Travelpayouts")
	fmt.Println(strings.Repeat("=", 50))

	// Создаём контекст, который передаём во внешние запросы.
	ctx := context.Background()

	// Создаём клиент для работы с Travelpayouts API.
	fmt.Println("\n📡 Запрос к Travelpayouts API...")
	apiClient := travelpayouts.NewClient(apiKey)

	// Параметры поиска.
	origin := "MOW"
	destination := "SEL"
	direct := false
	minTripDuration := 7
	maxTripDuration := 10

	// Тип сортировки:
	// 1 — цена по возрастанию
	// 2 — дата вылета по возрастанию
	// 3 — дата возвращения по возрастанию
	// 4 — длительность поездки по возрастанию
	// 5 — количество пересадок по возрастанию
	// 6 — цена по убыванию
	sort := 1

	// Определяем период, за который будем искать билеты.
	startMonth := time.Date(
		2026,
		time.September,
		1,
		0, 0, 0, 0,
		time.UTC,
	)

	endMonth := time.Date(
		2027,
		time.February,
		1,
		0, 0, 0, 0,
		time.UTC,
	)

	// Выполняем поиск билетов через Travelpayouts API.
	flights, err := apiClient.SearchFlightsForDateRange(
		ctx,
		origin,
		destination,
		startMonth,
		endMonth,
		direct,
		minTripDuration,
		maxTripDuration,
		"rub",
	)

	// Если API вернул ошибку, завершаем программу.
	if err != nil {
		log.Fatalf(
			"❌ Ошибка при запросе %s → %s: %v\n",
			origin,
			destination,
			err,
		)
	}

	fmt.Printf("\n📥 Всего получено %d билетов\n", len(flights))
	fmt.Println(strings.Repeat("=", 50))

	// Если билетов не найдено, выводим подсказки и завершаем работу.
	if len(flights) == 0 {
		fmt.Println("⚠️  Билеты не найдены")
		fmt.Println("\n💡 Попробуй:")
		fmt.Println("  - Изменить даты")
		fmt.Println("  - Убрать фильтры")
		fmt.Println("  - Проверить что маршрут существует")
		return
	}

	// Сортируем найденные билеты в соответствии с выбранным типом.
	sortFlights(flights, sort)

	// Выводим информацию о каждом найденном билете.
	for i, flight := range flights {
		fmt.Printf("\n✈️  БИЛЕТ #%d\n", i+1)
		fmt.Println(strings.Repeat("-", 50))

		// Выводим маршрут.
		fmt.Printf(
			"🛫 Маршрут:         %s → %s\n",
			flight.Origin,
			flight.Destination,
		)

		// Выводим дату и время вылета.
		fmt.Printf(
			"📅 Дата вылета:     %s\n",
			flight.DepartureAt.Format("2006-01-02 15:04"),
		)

		// Выводим дату возвращения.
		// Если ReturnAt == nil, билет является билетом в одну сторону.
		if flight.ReturnAt != nil {
			fmt.Printf(
				"🔙 Дата возврата:   %s\n",
				flight.ReturnAt.Format("2006-01-02 15:04"),
			)
		} else {
			fmt.Printf("🔙 Дата возврата:   В один конец\n")
		}

		// Если есть обратный рейс, рассчитываем продолжительность поездки.
		if flight.ReturnAt != nil {
			tripDuration := flight.ReturnAt.Sub(flight.DepartureAt)
			days := int(tripDuration.Hours() / 24)

			fmt.Printf("📆 Длительность:    %d дней\n", days)
		}

		// Цена хранится в минимальных денежных единицах.
		// Например, 150000 = 1500.00 RUB.
		priceRub := float64(flight.Price) / 100

		fmt.Printf(
			"💰 Цена:            %.0f %s\n",
			priceRub,
			strings.ToUpper(flight.Currency),
		)

		// Выводим название авиакомпании.
		fmt.Printf("🏢 Авиакомпания:    %s\n", flight.Airline)

		// Если номер рейса известен, выводим его.
		if flight.FlightNumber != "" {
			fmt.Printf("✈️  Рейс:           %s\n", flight.FlightNumber)
		}

		// Выводим количество пересадок.
		// Отдельно показываем пересадки туда и обратно.
		if flight.Transfers == 0 && flight.ReturnTransfers == 0 {
			fmt.Printf("🔄 Пересадки:       Прямой рейс ✨\n")
		} else {
			fmt.Printf(
				"🔄 Пересадки:       туда: %d, обратно: %d\n",
				flight.Transfers,
				flight.ReturnTransfers,
			)
		}

		// Выводим общую длительность перелёта в часах и минутах.
		if flight.Duration > 0 {
			hours := flight.Duration / 60
			minutes := flight.Duration % 60

			fmt.Printf(
				"⏱️  Время в пути:   %dч %dм (%d минут)\n",
				hours,
				minutes,
				flight.Duration,
			)
		}

		// Выводим ссылку, по которой можно перейти к бронированию.
		fmt.Printf("🔗 Ссылка: %s\n", flight.BookingLink)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))

	// Выводим статистику по найденным билетам.
	fmt.Println("\n📊 СТАТИСТИКА:")
	fmt.Printf(
		"  📅 Период:        %s - %s\n",
		startMonth,
		endMonth,
	)

	fmt.Printf(
		"  🎫 Всего билетов: %d\n",
		len(flights),
	)

	if len(flights) > 0 {
		// Рассчитываем среднюю стоимость билета.
		var totalPrice int64

		for _, f := range flights {
			totalPrice += f.Price
		}

		avgPrice := float64(totalPrice) / float64(len(flights)) / 100

		fmt.Printf(
			"  💰 Средняя цена:  %.0f RUB\n",
			avgPrice,
		)

		// Ищем самый дешёвый билет.
		minPrice := flights[0].Price
		minFlight := flights[0]

		for _, f := range flights {
			if f.Price < minPrice {
				minPrice = f.Price
				minFlight = f
			}
		}

		fmt.Printf(
			"  💎 Самый дешёвый: %.0f RUB (%s, %s)\n",
			float64(minPrice)/100,
			minFlight.Airline,
			minFlight.DepartureAt.Format("02 Jan"),
		)

		// Ищем самый дорогой билет.
		maxPrice := flights[0].Price

		for _, f := range flights {
			if f.Price > maxPrice {
				maxPrice = f.Price
			}
		}

		fmt.Printf(
			"  💸 Самый дорогой: %.0f RUB\n",
			float64(maxPrice)/100,
		)

		// Подсчитываем количество билетов без пересадок.
		directCount := 0

		for _, f := range flights {
			if f.Transfers == 0 && f.ReturnTransfers == 0 {
				directCount++
			}
		}

		fmt.Printf(
			"  ✈️ Прямых рейсов: %d из %d\n",
			directCount,
			len(flights),
		)

		// Создаём map для получения списка уникальных авиакомпаний.
		airlines := make(map[string]bool)

		for _, f := range flights {
			airlines[f.Airline] = true
		}

		fmt.Printf(
			"  🏢 Авиакомпаний:  %d\n",
			len(airlines),
		)
	}

	fmt.Println("\n✨ Готово!")
}

// sortFlights сортирует список авиабилетов.
//
// sortType определяет критерий сортировки:
// 1 — цена по возрастанию
// 2 — дата вылета
// 3 — дата возвращения
// 4 — длительность поездки
// 5 — количество пересадок
// 6 — цена по убыванию
//
// Функция изменяет исходный массив flights,
// так как sort.Slice сортирует срез на месте.
func sortFlights(flights []travelpayouts.Flight, sortType int) {
	switch sortType {

	case 1:
		// Сортировка по цене: от дешёвых к дорогим.
		sort.Slice(flights, func(i, j int) bool {
			return flights[i].Price < flights[j].Price
		})

	case 2:
		// Сортировка по дате вылета: от ранних дат к поздним.
		sort.Slice(flights, func(i, j int) bool {
			return flights[i].DepartureAt.Before(flights[j].DepartureAt)
		})

	case 3:
		// Сортировка по дате возвращения.
		sort.Slice(flights, func(i, j int) bool {
			// Билет без обратного рейса считаем более поздним.
			if flights[i].ReturnAt == nil {
				return false
			}

			if flights[j].ReturnAt == nil {
				return true
			}

			return flights[i].ReturnAt.Before(*flights[j].ReturnAt)
		})

	case 4:
		// Сортировка по длительности поездки:
		// от самых коротких поездок к самым длинным.
		sort.Slice(flights, func(i, j int) bool {
			// Билет без обратного рейса нельзя корректно
			// использовать для расчёта длительности поездки.
			if flights[i].ReturnAt == nil {
				return false
			}

			if flights[j].ReturnAt == nil {
				return true
			}

			// Рассчитываем длительность обеих поездок.
			durationI := flights[i].ReturnAt.Sub(flights[i].DepartureAt)
			durationJ := flights[j].ReturnAt.Sub(flights[j].DepartureAt)

			return durationI < durationJ
		})

	case 5:
		// Сортировка по общему количеству пересадок:
		// сначала билеты с меньшим количеством пересадок.
		sort.Slice(flights, func(i, j int) bool {
			transfersI := flights[i].Transfers + flights[i].ReturnTransfers
			transfersJ := flights[j].Transfers + flights[j].ReturnTransfers

			return transfersI < transfersJ
		})

	case 6:
		// Сортировка по цене: от дорогих к дешёвым.
		sort.Slice(flights, func(i, j int) bool {
			return flights[i].Price > flights[j].Price
		})

	default:
		// Если передан неизвестный тип сортировки,
		// используем сортировку по цене по возрастанию.
		fmt.Printf(
			"⚠️ Неизвестный тип сортировки: %d. Используется сортировка по цене.\n",
			sortType,
		)

		sort.Slice(flights, func(i, j int) bool {
			return flights[i].Price < flights[j].Price
		})
	}
}

// showStats выводит статистику по билетам,
// сохранённым в PostgreSQL.
//
// Функция получает подключение к БД и выполняет несколько SQL-запросов:
// 1. Общее количество билетов.
// 2. Количество билетов и минимальную цену по маршрутам.
// 3. Среднюю цену всех билетов.
// 4. Последние добавленные билеты.
func showStats(ctx context.Context, conn *pgx.Conn) {

	// Получаем общее количество билетов в таблице flights.
	var count int

	conn.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM flights",
	).Scan(&count)

	fmt.Printf(
		"  📍 Всего билетов: %d\n",
		count,
	)

	// Получаем статистику по маршрутам.
	// Для каждого маршрута считаем количество билетов
	// и минимальную найденную цену.
	rows, _ := conn.Query(ctx, `
		SELECT origin, destination, COUNT(*) as cnt, MIN(price) as min_price
		FROM flights
		GROUP BY origin, destination
		ORDER BY cnt DESC
		LIMIT 10
	`)

	// После завершения работы освобождаем ресурсы,
	// связанные с результатом SQL-запроса.
	defer rows.Close()

	fmt.Println("\n  🛫 Маршруты:")

	// Читаем результаты запроса построчно.
	for rows.Next() {
		var origin, dest string
		var cnt int
		var minPrice int64

		// Извлекаем значения текущей строки.
		if err := rows.Scan(
			&origin,
			&dest,
			&cnt,
			&minPrice,
		); err == nil {
			fmt.Printf(
				"    • %s → %s | %d билетов | мин цена: ₽%d\n",
				origin,
				dest,
				cnt,
				minPrice/100,
			)
		}
	}

	// Получаем среднюю цену всех билетов.
	var avgPrice float64

	conn.QueryRow(
		ctx,
		"SELECT AVG(price) FROM flights",
	).Scan(&avgPrice)

	fmt.Printf(
		"\n  💰 Средняя цена: ₽%.0f\n",
		avgPrice/100,
	)

	// Получаем последние добавленные билеты.
	fmt.Println("\n  📋 Последние добавленные билеты:")

	flightRows, _ := conn.Query(ctx, `
		SELECT origin, destination, airline, departure_at, price
		FROM flights
		ORDER BY created_at DESC
		LIMIT 5
	`)

	// Освобождаем ресурсы после завершения работы с результатом.
	defer flightRows.Close()

	// Читаем последние билеты построчно.
	for flightRows.Next() {
		var origin, dest, airline string
		var departAt time.Time
		var price int64

		// Извлекаем данные текущего билета.
		if err := flightRows.Scan(
			&origin,
			&dest,
			&airline,
			&departAt,
			&price,
		); err == nil {
			fmt.Printf(
				"    • %s → %s | %s | %s | ₽%d\n",
				origin,
				dest,
				airline,
				departAt.Format("2006-01-02"),
				price/100,
			)
		}
	}
}
