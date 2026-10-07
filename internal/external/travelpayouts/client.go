package travelpayouts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Client для работы с Travelpayouts API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient создаёт новый клиент
func NewClient(apiKey string) *Client {
	return &Client{
		baseURL: "https://api.travelpayouts.com/aviasales/v3",
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchResponse структура ответа от API
type SearchResponse struct {
	Success  bool                  `json:"success"`
	Data     map[string]FlightData `json:"data"`
	Currency string                `json:"currency"`
}

// FlightData данные одного билета
type FlightData struct {
	Origin             string `json:"origin"`
	Destination        string `json:"destination"`
	OriginAirport      string `json:"origin_airport"`
	DestinationAirport string `json:"destination_airport"`
	Price              int    `json:"price"`
	Airline            string `json:"airline"`
	FlightNumber       string `json:"flight_number"`
	DepartureAt        string `json:"departure_at"`
	ReturnAt           string `json:"return_at"`
	Transfers          int    `json:"transfers"`
	ReturnTransfers    int    `json:"return_transfers"`
	Duration           int    `json:"duration"`
	Link               string `json:"link"`
}

// Flight модель для сохранения в БД
type Flight struct {
	Origin             string
	Destination        string
	OriginAirport      string
	DestinationAirport string
	Price              int64 // в копейках
	Currency           string
	Airline            string
	FlightNumber       string
	DepartureAt        time.Time
	ReturnAt           *time.Time
	Transfers          int
	ReturnTransfers    int
	Duration           int
	Link               string
	BookingLink        string
}

// SearchFlights ищет дешевые билеты (origin + destination)
func (c *Client) SearchFlights(
	ctx context.Context,
	origin string,
	departureDate string, // формат: 2024-05 или 2024-05-15
	currency string,
) ([]Flight, error) {

	url := fmt.Sprintf(
		"%s/grouped_prices?origin=%s&currency=%s&departure_at=%s&group_by=departure_at&token=%s",
		c.baseURL,
		origin,
		currency,
		departureDate,
		c.apiKey,
	)

	fmt.Printf("🔍 Запрашиваю Travelpayouts API: %s\n", origin)

	return c.doRequest(ctx, url)
}

// SearchFlightsFromOrigin ищет билеты только по origin (без destination)
func (c *Client) SearchFlightsFromOrigin(
	ctx context.Context,
	origin string,
	departureDate string,
	currency string,
) ([]Flight, error) {

	url := fmt.Sprintf(
		"%s/grouped_prices?origin=%s&currency=%s&departure_at=%s&group_by=departure_at&token=%s",
		c.baseURL,
		origin,
		currency,
		departureDate,
		c.apiKey,
	)

	fmt.Printf("🔍 Запрашиваю Travelpayouts API: только из %s\n", origin)

	return c.doRequest(ctx, url)
}

// SearchFlightsFromOriginToDestination ищет билеты по origin и destination
func (c *Client) SearchFlightsFromOriginToDestination(
	ctx context.Context,
	origin string,
	destination string,
	departureAt string,
	returnAt string,
	direct bool,
	minTripDuration int,
	maxTripDuration int,
	currency string,
) ([]Flight, error) {

	url := fmt.Sprintf(
		"%s/grouped_prices?origin=%s&destination=%s&currency=%s&departure_at=%s&return_at=%s&group_by=departure_at&direct=%s&min_trip_duration=%s&max_trip_duration=%s&token=%s",
		c.baseURL,
		origin,
		destination,
		currency,
		departureAt,
		returnAt,
		strconv.FormatBool(direct),
		strconv.Itoa(minTripDuration),
		strconv.Itoa(maxTripDuration),
		c.apiKey,
	)

	//fmt.Printf("🔍 Запрашиваю Travelpayouts API: из %s в %s\n", origin, destination)

	return c.doRequest(ctx, url)
}

// doRequest выполняет HTTP запрос и парсит ответ
func (c *Client) doRequest(ctx context.Context, url string) ([]Flight, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	flights := c.convertToFlights(result)

	//fmt.Printf("✅ Получено %d билетов\n", len(flights))

	return flights, nil
}

// convertToFlights конвертирует API ответ в наш формат
func (c *Client) convertToFlights(result SearchResponse) []Flight {
	var flights []Flight

	for dateKey, data := range result.Data {
		departureTime, err := time.Parse(time.RFC3339, data.DepartureAt)
		if err != nil {
			departureTime, _ = time.Parse("2006-01-02", dateKey)
		}

		var returnTime *time.Time
		if data.ReturnAt != "" {
			if rt, err := time.Parse(time.RFC3339, data.ReturnAt); err == nil {
				returnTime = &rt
			}
		}

		flight := Flight{
			Origin:             data.Origin,
			Destination:        data.Destination,
			OriginAirport:      data.OriginAirport,
			DestinationAirport: data.DestinationAirport,
			Price:              int64(data.Price) * 100, // конвертируем в копейки
			Currency:           result.Currency,
			Airline:            data.Airline,
			FlightNumber:       data.FlightNumber,
			DepartureAt:        departureTime,
			ReturnAt:           returnTime,
			Transfers:          data.Transfers,
			ReturnTransfers:    data.ReturnTransfers,
			Duration:           data.Duration,
			Link:               data.Link,
			BookingLink: fmt.Sprintf(
				"https://www.aviasales.ru%s",
				data.Link,
			),
		}

		flights = append(flights, flight)
	}

	return flights
}

func (c *Client) SearchFlightsForDateRange(
	ctx context.Context,
	origin string,
	destination string,
	startMonth time.Time,
	endMonth time.Time,
	direct bool,
	minTripDuration int,
	maxTripDuration int,
	currency string,
) ([]Flight, error) {

	var allFlights []Flight

	//fmt.Printf("🔍 Запрашиваю Travelpayouts API: из %s в %s\n", origin, destination)

	for month := startMonth; !month.After(endMonth); month = month.AddDate(0, 1, 0) {
		departureAt := month.Format("2006-01")

		// Вызов 1: вылет и возврат в одном месяце
		returnAtSame := month.Format("2006-01")

		flights1, err := c.SearchFlightsFromOriginToDestination(
			ctx, origin, destination,
			departureAt, returnAtSame,
			direct, minTripDuration, maxTripDuration, currency,
		)
		if err != nil {
			fmt.Printf("⚠️  Ошибка для %s → %s: %v\n", departureAt, returnAtSame, err)
		} else {
			allFlights = append(allFlights, flights1...)
			//fmt.Printf("✅ %s → %s: найдено %d билетов\n", departureAt, returnAtSame, len(flights1))
		}

		time.Sleep(500 * time.Millisecond)

		// Вызов 2: вылет в текущем месяце, возврат в следующем
		// (покрывает случай типа 30 сен → 5 окт)
		nextMonth := month.AddDate(0, 1, 0)
		returnAtNext := nextMonth.Format("2006-01")

		flights2, err := c.SearchFlightsFromOriginToDestination(
			ctx, origin, destination,
			departureAt, returnAtNext,
			direct, minTripDuration, maxTripDuration, currency,
		)
		if err != nil {
			fmt.Printf("⚠️  Ошибка для %s → %s: %v\n", departureAt, returnAtNext, err)
		} else {
			allFlights = append(allFlights, flights2...)
			//fmt.Printf("✅ %s → %s: найдено %d билетов\n", departureAt, returnAtNext, len(flights2))
		}

		time.Sleep(500 * time.Millisecond)
	}

	return allFlights, nil
}
