package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/DreadNought212/avia_bro/internal/external/travelpayouts"
)

type FlightRepository struct {
	db *sql.DB
}

type UpsertResult struct {
	IsNew        bool
	OldPrice     *int
	Price        int
	PriceDropped bool
}

func NewFlightRepository(db *sql.DB) *FlightRepository {
	return &FlightRepository{db: db}
}

// UpsertFlight сохраняет или обновляет билет
// Возвращает true если билет новый, false если обновлён
func (r *FlightRepository) UpsertFlight(ctx context.Context, flight *travelpayouts.Flight, flightKey string) (result UpsertResult, err error) {

	query := `
		SELECT
			is_new,
			old_price,
			price,
			price_dropped
		FROM common.upsert_flight(
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	err = r.db.QueryRowContext(
		ctx,
		query,
		flightKey,
		flight.Origin,
		flight.Destination,
		flight.OriginAirport,
		flight.DestinationAirport,
		flight.DepartureAt,
		flight.ReturnAt,
		flight.Price,
		flight.Currency,
		flight.Airline,
		flight.FlightNumber,
		flight.Transfers,
		flight.ReturnTransfers,
		flight.Duration,
		flight.Link,
		flight.BookingLink,
	).Scan(
		&result.IsNew,
		&result.OldPrice,
		&result.Price,
		&result.PriceDropped,
	)

	if err != nil {
		return UpsertResult{}, fmt.Errorf("failed to upsert flight: %w", err)
	}

	return result, nil
}

// DeactivateFlightsNotInKeys помечает билеты как неактивные если их нет в списке ключей
func (r *FlightRepository) DeactivateFlightsNotInKeys(
	ctx context.Context,
	origin, destination string,
	flightKeys []string,
) (int, error) {
	query := `
		UPDATE common.flights
		SET is_active = FALSE, updated_at = NOW()
		WHERE origin = $1 AND destination = $2
		  AND is_active = TRUE
		  AND flight_key != ALL($3)
	`

	result, err := r.db.ExecContext(ctx, query, origin, destination, flightKeys)
	if err != nil {
		return 0, fmt.Errorf("failed to deactivate flights: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// GetNewFlights получает новые билеты за последний период
func (r *FlightRepository) GetNewFlights(
	ctx context.Context,
	origin, destination string,
	since time.Time,
) ([]*travelpayouts.Flight, error) {
	query := `
		SELECT 
			origin, destination, origin_airport, destination_airport,
			departure_at, return_at, price, currency, airline, flight_number,
			transfers, return_transfers, duration, link, booking_link,
			first_seen_at, last_seen_at
		FROM common.flights
		WHERE origin = $1 AND destination = $2
		  AND is_active = TRUE
		  AND first_seen_at >= $3
		ORDER BY price ASC
	`

	rows, err := r.db.QueryContext(ctx, query, origin, destination, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query new flights: %w", err)
	}
	defer rows.Close()

	var flights []*travelpayouts.Flight
	for rows.Next() {
		flight := &travelpayouts.Flight{}
		var returnAt sql.NullTime
		var firstSeenAt, lastSeenAt time.Time

		err := rows.Scan(
			&flight.Origin,
			&flight.Destination,
			&flight.OriginAirport,
			&flight.DestinationAirport,
			&flight.DepartureAt,
			&returnAt,
			&flight.Price,
			&flight.Currency,
			&flight.Airline,
			&flight.FlightNumber,
			&flight.Transfers,
			&flight.ReturnTransfers,
			&flight.Duration,
			&flight.Link,
			&flight.BookingLink,
			&firstSeenAt,
			&lastSeenAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan flight: %w", err)
		}

		if returnAt.Valid {
			flight.ReturnAt = &returnAt.Time
		}

		flights = append(flights, flight)
	}

	return flights, rows.Err()
}

// GetFlightByKey получает билет по ключу
func (r *FlightRepository) GetFlightByKey(ctx context.Context, flightKey string) (*travelpayouts.Flight, error) {
	query := `
		SELECT 
			origin, destination, origin_airport, destination_airport,
			departure_at, return_at, price, currency, airline, flight_number,
			transfers, return_transfers, duration, link, booking_link
		FROM common.flights
		WHERE flight_key = $1
	`

	flight := &travelpayouts.Flight{}
	var returnAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, flightKey).Scan(
		&flight.Origin,
		&flight.Destination,
		&flight.OriginAirport,
		&flight.DestinationAirport,
		&flight.DepartureAt,
		&returnAt,
		&flight.Price,
		&flight.Currency,
		&flight.Airline,
		&flight.FlightNumber,
		&flight.Transfers,
		&flight.ReturnTransfers,
		&flight.Duration,
		&flight.Link,
		&flight.BookingLink,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get flight: %w", err)
	}

	if returnAt.Valid {
		flight.ReturnAt = &returnAt.Time
	}

	return flight, nil
}
