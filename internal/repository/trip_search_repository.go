package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/DreadNought212/avia_bro/internal/domain"
	"github.com/lib/pq"
)

type TripSearchRepository struct {
	db *sql.DB
}

func NewTripSearchRepository(db *sql.DB) *TripSearchRepository {
	return &TripSearchRepository{db: db}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTrip(scanner rowScanner) (*domain.TripOption, error) {
	var trip domain.TripOption

	err := scanner.Scan(
		// Flight IDs
		&trip.Outbound.ID,
		&trip.Inbound.ID,

		// Outbound flight
		&trip.Outbound.FlightKey,
		&trip.Outbound.Origin,
		&trip.Outbound.Destination,
		&trip.Outbound.OriginAirport,
		&trip.Outbound.DestinationAirport,
		&trip.Outbound.DepartureAt,
		&trip.Outbound.Price,
		&trip.Outbound.Currency,
		&trip.Outbound.Airline,
		&trip.Outbound.FlightNumber,
		&trip.Outbound.Transfers,
		&trip.Outbound.Duration,
		&trip.Outbound.Link,
		&trip.Outbound.BookingLink,

		// Inbound flight
		&trip.Inbound.FlightKey,
		&trip.Inbound.Origin,
		&trip.Inbound.Destination,
		&trip.Inbound.OriginAirport,
		&trip.Inbound.DestinationAirport,
		&trip.Inbound.DepartureAt,
		&trip.Inbound.Price,
		&trip.Inbound.Currency,
		&trip.Inbound.Airline,
		&trip.Inbound.FlightNumber,
		&trip.Inbound.Transfers,
		&trip.Inbound.Duration,
		&trip.Inbound.Link,
		&trip.Inbound.BookingLink,

		// Trip data
		&trip.TotalPrice,
		&trip.Currency,
		&trip.TripDuration,
	)

	if err != nil {
		return nil, err
	}

	return &trip, nil
}

const tripSelect = `
	SELECT
		o.id, i.id,

		o.flight_key, o.origin, o.destination,
		COALESCE(o.origin_airport, ''),
		COALESCE(o.destination_airport, ''),
		o.departure_at, o.price, o.currency, o.airline,
		COALESCE(o.flight_number, ''),
		o.transfers, COALESCE(o.duration, 0),
		COALESCE(o.link, ''), COALESCE(o.booking_link, ''),

		i.flight_key, i.origin, i.destination,
		COALESCE(i.origin_airport, ''),
		COALESCE(i.destination_airport, ''),
		i.departure_at, i.price, i.currency, i.airline,
		COALESCE(i.flight_number, ''),
		i.transfers, COALESCE(i.duration, 0),
		COALESCE(i.link, ''), COALESCE(i.booking_link, ''),

		o.price + i.price AS total_price,
		o.currency,
		(i.departure_at::date - o.departure_at::date) AS trip_duration
`

func (r *TripSearchRepository) Search(
	ctx context.Context,
	p domain.SearchParams,
) ([]domain.TripOption, error) {
	query := tripSelect + `
		FROM common.flights o

		JOIN common.flights i
			ON i.is_one_way = TRUE
			AND i.is_active = TRUE
			AND i.destination = o.origin
			AND i.origin = ANY($3::varchar[])
			AND i.departure_at > o.departure_at

		WHERE
			o.is_one_way = TRUE
			AND o.is_active = TRUE
			AND o.origin = $1
			AND o.destination = ANY($2::varchar[])
			AND o.departure_at::date >= $4::date
			AND o.departure_at::date <= $5::date
			AND (
				i.departure_at::date - o.departure_at::date
			) BETWEEN $6::integer AND $7::integer
			AND o.transfers <= $8
			AND i.transfers <= $8
			AND o.airline NOT IN ('S7', 'DV')
			AND i.airline NOT IN ('S7', 'DV')
			AND o.currency = i.currency

		ORDER BY
			total_price ASC,
			trip_duration ASC,
			o.departure_at ASC

		LIMIT $9 OFFSET $10
	`

	offset := (p.Page - 1) * p.Limit

	rows, err := r.db.QueryContext(
		ctx,
		query,
		p.Origin,
		pq.Array(p.Destinations),
		pq.Array(p.ReturnOrigins),
		p.DepartureFrom,
		p.DepartureTo,
		p.MinTripDuration,
		p.MaxTripDuration,
		p.MaxTransfers,
		p.Limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("search trips: %w", err)
	}
	defer rows.Close()

	var result []domain.TripOption

	for rows.Next() {
		trip, err := scanTrip(rows)
		if err != nil {
			return nil, fmt.Errorf("scan trip: %w", err)
		}

		result = append(result, *trip)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trips: %w", err)
	}

	return result, nil
}

func (r *TripSearchRepository) Details(
	ctx context.Context,
	outboundKey string,
	inboundKey string,
) (*domain.TripOption, error) {
	query := tripSelect + `
		FROM common.flights o
		JOIN common.flights i ON TRUE
		WHERE
			o.flight_key = $1
			AND i.flight_key = $2
			AND o.is_one_way = TRUE
			AND i.is_one_way = TRUE
			AND o.currency = i.currency
			AND i.departure_at::date > o.departure_at::date
	`

	row := r.db.QueryRowContext(ctx, query, outboundKey, inboundKey)

	trip, err := scanTrip(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get trip details: %w", err)
	}

	return trip, nil
}
