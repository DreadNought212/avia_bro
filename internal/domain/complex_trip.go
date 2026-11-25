package domain

import "time"

type ComplexSearchStatus string

const (
	ComplexSearchStatusRunning   ComplexSearchStatus = "running"
	ComplexSearchStatusCompleted ComplexSearchStatus = "completed"
	ComplexSearchStatusFailed    ComplexSearchStatus = "failed"
)

// Параметры поискового запроса.
type ComplexSearchParams struct {
	Origin            string            `json:"origin"`
	Stops             []ComplexTripStop `json:"stops"`
	FinalDestination  string            `json:"final_destination"`
	FinalMaxTransfers int               `json:"final_max_transfers"`
	DepartureDateFrom time.Time         `json:"departure_date_from"`
	DepartureDateTo   time.Time         `json:"departure_date_to"`
	Currency          string            `json:"currency"`
}

// Промежуточный город и допустимое количество дней пребывания.
type ComplexTripStop struct {
	City         string `json:"city"`
	MinDays      int    `json:"min_days"`
	MaxDays      int    `json:"max_days"`
	MaxTransfers int    `json:"max_transfers"`
}

// Отдельный запуск поиска.
type ComplexSearchSession struct {
	ID           int64               `json:"id"`
	Status       ComplexSearchStatus `json:"status"`
	SearchParams ComplexSearchParams `json:"search_params"`
	CreatedAt    time.Time           `json:"created_at"`
	FinishedAt   *time.Time          `json:"finished_at,omitempty"`
	ErrorMessage string              `json:"error_message,omitempty"`
}

// Снимок билета, полученного во время конкретного поиска.
type ComplexSearchFlight struct {
	ID        int64  `json:"id"`
	SessionID int64  `json:"session_id"`
	FlightKey string `json:"flight_key"`

	Origin             string `json:"origin"`
	Destination        string `json:"destination"`
	OriginAirport      string `json:"origin_airport,omitempty"`
	DestinationAirport string `json:"destination_airport,omitempty"`

	DepartureAt time.Time `json:"departure_at"`

	Price    int64  `json:"price"`
	Currency string `json:"currency"`

	Airline      string `json:"airline,omitempty"`
	FlightNumber string `json:"flight_number,omitempty"`

	Transfers int `json:"transfers"`
	Duration  int `json:"duration"`

	Link        string `json:"link,omitempty"`
	BookingLink string `json:"booking_link,omitempty"`

	SavedAt time.Time `json:"saved_at"`
}
