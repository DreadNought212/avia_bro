package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DreadNought212/avia_bro/internal/domain"
)

type ComplexSearchRepository struct {
	db *sql.DB
}

func NewComplexSearchRepository(db *sql.DB) *ComplexSearchRepository {
	return &ComplexSearchRepository{db: db}
}

// CreateSession создаёт новую поисковую сессию.
func (r *ComplexSearchRepository) CreateSession(
	ctx context.Context,
	params domain.ComplexSearchParams,
) (*domain.ComplexSearchSession, error) {
	searchParams, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal search params: %w", err)
	}

	query := `
		INSERT INTO common.complex_search_sessions (
			search_params,
			departure_date_from,
			departure_date_to,
			currency
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, status, created_at
	`

	session := &domain.ComplexSearchSession{
		SearchParams: params,
	}

	err = r.db.QueryRowContext(
		ctx,
		query,
		searchParams,
		params.DepartureDateFrom,
		params.DepartureDateTo,
		params.Currency,
	).Scan(
		&session.ID,
		&session.Status,
		&session.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create search session: %w", err)
	}

	return session, nil
}

// SaveFlights сохраняет билеты в рамках одной поисковой сессии.
// В случае ошибки все изменения откатываются.
func (r *ComplexSearchRepository) SaveFlights(
	ctx context.Context,
	sessionID int64,
	flights []*domain.ComplexSearchFlight,
) error {
	if len(flights) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO common.complex_search_flights (
			session_id,
			flight_key,
			origin,
			destination,
			origin_airport,
			destination_airport,
			departure_at,
			price,
			currency,
			airline,
			flight_number,
			transfers,
			duration,
			link,
			booking_link
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15
		)
		ON CONFLICT (session_id, flight_key) DO NOTHING
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare flight insert: %w", err)
	}
	defer stmt.Close()

	for _, flight := range flights {
		_, err := stmt.ExecContext(
			ctx,
			sessionID,
			flight.FlightKey,
			flight.Origin,
			flight.Destination,
			flight.OriginAirport,
			flight.DestinationAirport,
			flight.DepartureAt,
			flight.Price,
			flight.Currency,
			flight.Airline,
			flight.FlightNumber,
			flight.Transfers,
			flight.Duration,
			flight.Link,
			flight.BookingLink,
		)
		if err != nil {
			return fmt.Errorf(
				"save flight %s: %w",
				flight.FlightKey,
				err,
			)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit flight transaction: %w", err)
	}

	return nil
}

// UpdateSessionStatus обновляет статус поисковой сессии.
func (r *ComplexSearchRepository) UpdateSessionStatus(
	ctx context.Context,
	sessionID int64,
	status domain.ComplexSearchStatus,
	errorMessage string,
) error {
	query := `
		UPDATE common.complex_search_sessions
		SET
			status = $2,
			error_message = NULLIF($3, ''),
			finished_at = $4
		WHERE id = $1
	`

	var finishedAt *time.Time
	if status == domain.ComplexSearchStatusCompleted ||
		status == domain.ComplexSearchStatusFailed {
		now := time.Now()
		finishedAt = &now
	}

	result, err := r.db.ExecContext(
		ctx,
		query,
		sessionID,
		status,
		errorMessage,
		finishedAt,
	)
	if err != nil {
		return fmt.Errorf("update search session status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("search session %d not found", sessionID)
	}

	return nil
}

// GetFlightsBySession получает все билеты конкретной сессии.
func (r *ComplexSearchRepository) GetFlightsBySession(
	ctx context.Context,
	sessionID int64,
) ([]*domain.ComplexSearchFlight, error) {
	query := `
		SELECT
			id,
			session_id,
			flight_key,
			origin,
			destination,
			origin_airport,
			destination_airport,
			departure_at,
			price,
			currency,
			airline,
			flight_number,
			transfers,
			duration,
			link,
			booking_link,
			saved_at
		FROM common.complex_search_flights
		WHERE session_id = $1
		ORDER BY departure_at ASC, price ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query flights by session: %w", err)
	}
	defer rows.Close()

	var flights []*domain.ComplexSearchFlight

	for rows.Next() {
		flight := &domain.ComplexSearchFlight{}

		err := rows.Scan(
			&flight.ID,
			&flight.SessionID,
			&flight.FlightKey,
			&flight.Origin,
			&flight.Destination,
			&flight.OriginAirport,
			&flight.DestinationAirport,
			&flight.DepartureAt,
			&flight.Price,
			&flight.Currency,
			&flight.Airline,
			&flight.FlightNumber,
			&flight.Transfers,
			&flight.Duration,
			&flight.Link,
			&flight.BookingLink,
			&flight.SavedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan flight: %w", err)
		}

		flights = append(flights, flight)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate flights: %w", err)
	}

	return flights, nil
}

func (r *ComplexSearchRepository) GetSession(
	ctx context.Context,
	sessionID int64,
) (*domain.ComplexSearchSession, error) {
	query := `
		SELECT
			id,
			status,
			search_params,
			created_at,
			finished_at,
			error_message
		FROM common.complex_search_sessions
		WHERE id = $1
	`

	session := &domain.ComplexSearchSession{}

	var searchParams []byte
	var finishedAt sql.NullTime
	var errorMessage sql.NullString

	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID,
		&session.Status,
		&searchParams,
		&session.CreatedAt,
		&finishedAt,
		&errorMessage,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("search session %d not found", sessionID)
		}

		return nil, fmt.Errorf("get search session: %w", err)
	}

	if err := json.Unmarshal(searchParams, &session.SearchParams); err != nil {
		return nil, fmt.Errorf("unmarshal search params: %w", err)
	}

	if finishedAt.Valid {
		session.FinishedAt = &finishedAt.Time
	}

	if errorMessage.Valid {
		session.ErrorMessage = errorMessage.String
	}

	return session, nil
}
