package domain

import "time"

type TripFlight struct {
	ID                 int64         `json:"id"`
	FlightKey          string        `json:"flight_key"`
	Origin             string        `json:"origin"`
	Destination        string        `json:"destination"`
	OriginAirport      string        `json:"origin_airport,omitempty"`
	DestinationAirport string        `json:"destination_airport,omitempty"`
	DepartureAt        LocalDateTime `json:"departure_at"`
	Price              int64         `json:"price"`
	Currency           string        `json:"currency"`
	Airline            string        `json:"airline"`
	FlightNumber       string        `json:"flight_number,omitempty"`
	Transfers          int           `json:"transfers"`
	Duration           int           `json:"duration,omitempty"`
	Link               string        `json:"link,omitempty"`
	BookingLink        string        `json:"booking_link,omitempty"`
}

type TripOption struct {
	Outbound     TripFlight `json:"outbound"`
	Inbound      TripFlight `json:"inbound"`
	TotalPrice   int64      `json:"total_price"`
	Currency     string     `json:"currency"`
	TripDuration int        `json:"trip_duration"`
}

type SearchParams struct {
	Origin          string
	Destinations    []string
	ReturnOrigins   []string
	DepartureFrom   time.Time
	DepartureTo     time.Time
	MinTripDuration int
	MaxTripDuration int
	MaxTransfers    int
	Page            int
	Limit           int
}
