package aviasales

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client для работы с Aviasales API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient создаёт новый клиент Aviasales
func NewClient(apiKey string) *Client {
	return &Client{
		baseURL: "https://api.aviasales.ru/v2",
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchResponse структура ответа от API
type SearchResponse struct {
	Data []struct {
		Value struct {
			GroupParameters struct {
				TripClass string `json:"tripClass"`
				Cabin     string `json:"cabin"`
			} `json:"groupParameters"`

			Legs []struct {
				DepartureDate   string `json:"departureDate"` // 2024-05-15T14:20:00
				ArrivalDate     string `json:"arrivalDate"`
				DurationSeconds int    `json:"durationSeconds"`
				Stops           int    `json:"stops"`
				AirlineID       int    `json:"airlineId"`
				Operating       string `json:"operating"`
			} `json:"legs"`

			Price struct {
				Minfare      int    `json:"minfare"`      // в рублях
				CurrencyCode string `json:"currencyCode"` // RUB
				Discount     int    `json:"discount"`
			} `json:"price"`
		} `json:"value"`
	} `json:"data"`

	Dictionaries struct {
		Airlines map[string]struct {
			Name string `json:"name"`
		} `json:"airlines"`

		Airports map[string]struct {
			Name string `json:"name"`
			City struct {
				Name string `json:"name"`
			} `json:"city"`
		} `json:"airports"`
	} `json:"dictionaries"`
}

// Flight простая модель для результата
type Flight struct {
	Origin          string
	Destination     string
	DepartDate      time.Time
	Price           int64 // в копейках (минfare * 100)
	Airline         string
	DepartTime      time.Time
	ArrivalTime     time.Time
	DurationMinutes int
	Transfers       int
	BookingLink     string
}

// SearchFlights ищет авиабилеты
// origin: MOW, destination: BCN, departDate: 2024-05-15
func (c *Client) SearchFlights(
	ctx context.Context,
	origin, destination string,
	departDate time.Time,
) ([]Flight, error) {

	// Строим URL
	url := fmt.Sprintf(
		"%s/search?origin=%s&destination=%s&depart_date=%s&one_way=1&token=%s",
		c.baseURL,
		origin,
		destination,
		departDate.Format("2006-01-02"),
		c.apiKey,
	)

	fmt.Printf("🔍 Запрашиваю API: %s → %s на %s\n",
		origin, destination, departDate.Format("2006-01-02"))

	// Создаём request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Отправляем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Парсим JSON
	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Конвертируем в наш формат
	flights := c.convertToFlights(result, origin, destination)

	//fmt.Printf("✅ Получено %d билетов\n", len(flights))

	return flights, nil
}

// convertToFlights конвертирует API ответ в наш формат
func (c *Client) convertToFlights(result SearchResponse, origin, destination string) []Flight {
	var flights []Flight

	// Берём первые 20 билетов из ответа
	maxFlights := 20
	if len(result.Data) < maxFlights {
		maxFlights = len(result.Data)
	}

	for i := 0; i < maxFlights; i++ {
		data := result.Data[i]
		value := data.Value

		// Пропускаем если нет цены или ног
		if value.Price.Minfare == 0 || len(value.Legs) == 0 {
			continue
		}

		// Получаем первую лег (вылет)
		outboundLeg := value.Legs[0]

		// Парсим даты вылета и прилёта
		departTime, _ := time.Parse("2006-01-02T15:04:05", outboundLeg.DepartureDate)
		arrivalTime, _ := time.Parse("2006-01-02T15:04:05", outboundLeg.ArrivalDate)

		// Получаем имя авиакомпании
		airlineName := "Unknown"
		if airline, ok := result.Dictionaries.Airlines[fmt.Sprintf("%d", outboundLeg.AirlineID)]; ok {
			airlineName = airline.Name
		}

		// Создаём объект полёта
		flight := Flight{
			Origin:          origin,
			Destination:     destination,
			DepartDate:      departTime,
			Price:           int64(value.Price.Minfare) * 100, // конвертируем в копейки
			Airline:         airlineName,
			DepartTime:      departTime,
			ArrivalTime:     arrivalTime,
			DurationMinutes: outboundLeg.DurationSeconds / 60,
			Transfers:       outboundLeg.Stops,
			BookingLink: fmt.Sprintf(
				"https://www.aviasales.ru/search/%s%s%s?params=",
				origin, destination, departTime.Format("020106"),
			),
		}

		flights = append(flights, flight)
	}

	return flights
}
