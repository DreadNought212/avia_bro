package domain

import (
	"time"
)

// Flight представляет авиабилет
type Flight struct {
	ID              string    `db:"id"`
	Origin          string    `db:"origin"`        // MOW, SPB, BCN, etc.
	Destination     string    `db:"destination"`
	DepartDate      time.Time `db:"depart_date"`
	Price           int64     `db:"price"`         // в копейках!
	Airline         string    `db:"airline"`
	DepartTime      time.Time `db:"depart_time"`
	ArrivalTime     time.Time `db:"arrival_time"`
	DurationMinutes int       `db:"duration_minutes"`
	Transfers       int       `db:"transfers"`
	BookingLink     string    `db:"booking_link"`
	ParsedAt        time.Time `db:"parsed_at"`
	CreatedAt       time.Time `db:"created_at"`
}

// NewFlight создаёт новый объект полёта
func NewFlight(
	origin, destination string,
	departDate time.Time,
	price int64,
	airline string,
) *Flight {
	return &Flight{
		Origin:      origin,
		Destination: destination,
		DepartDate:  departDate,
		Price:       price,
		Airline:     airline,
		ParsedAt:    time.Now(),
		CreatedAt:   time.Now(),
	}
}

// String представляет полёт как строку (для логирования)
func (f *Flight) String() string {
	return f.Origin + " → " + f.Destination + " | " +
		f.DepartDate.Format("2006-01-02") + " | " +
		f.Airline + " | ₽" + string(rune(f.Price/100))
}
