package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type SyncLog struct {
	ID                 int64
	RouteID            int
	StartedAt          time.Time
	FinishedAt         *time.Time
	Status             string
	FlightsFound       int
	FlightsNew         int
	FlightsUpdated     int
	FlightsDeactivated int
	ErrorMessage       string
	CreatedAt          time.Time
}

type SyncLogRepository struct {
	db *sql.DB
}

func NewSyncLogRepository(db *sql.DB) *SyncLogRepository {
	return &SyncLogRepository{db: db}
}

// CreateSyncLog создаёт новую запись о синхронизации
func (r *SyncLogRepository) CreateSyncLog(
	ctx context.Context,
	routeID int,
) (int64, error) {
	query := `
		INSERT INTO common.sync_logs (route_id, started_at, status)
		VALUES ($1, NOW(), 'running')
		RETURNING id
	`

	var id int64
	err := r.db.QueryRowContext(ctx, query, routeID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create sync log: %w", err)
	}

	return id, nil
}

// UpdateSyncLogSuccess обновляет лог при успешной синхронизации
func (r *SyncLogRepository) UpdateSyncLogSuccess(
	ctx context.Context,
	logID int64,
	flightsFound, flightsNew, flightsUpdated, flightsDeactivated int,
) error {
	query := `
		UPDATE common.sync_logs
		SET finished_at = NOW(),
		    status = 'success',
		    flights_found = $1,
		    flights_new = $2,
		    flights_updated = $3,
		    flights_deactivated = $4
		WHERE id = $5
	`

	_, err := r.db.ExecContext(
		ctx, query,
		flightsFound, flightsNew, flightsUpdated, flightsDeactivated, logID,
	)
	if err != nil {
		return fmt.Errorf("failed to update sync log: %w", err)
	}

	return nil
}

// UpdateSyncLogFailure обновляет лог при ошибке
func (r *SyncLogRepository) UpdateSyncLogFailure(
	ctx context.Context,
	logID int64,
	errorMessage string,
) error {
	query := `
		UPDATE common.sync_logs
		SET finished_at = NOW(),
		    status = 'failed',
		    error_message = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, errorMessage, logID)
	if err != nil {
		return fmt.Errorf("failed to update sync log failure: %w", err)
	}

	return nil
}

// GetRecentLogs получает последние логи синхронизации
func (r *SyncLogRepository) GetRecentLogs(
	ctx context.Context,
	routeID int,
	limit int,
) ([]*SyncLog, error) {
	query := `
		SELECT 
			id, route_id, started_at, finished_at, status,
			flights_found, flights_new, flights_updated, flights_deactivated,
			error_message, created_at
		FROM common.sync_logs
		WHERE route_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, routeID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query sync logs: %w", err)
	}
	defer rows.Close()

	var logs []*SyncLog
	for rows.Next() {
		log := &SyncLog{}
		var finishedAt sql.NullTime
		var errorMessage sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.RouteID,
			&log.StartedAt,
			&finishedAt,
			&log.Status,
			&log.FlightsFound,
			&log.FlightsNew,
			&log.FlightsUpdated,
			&log.FlightsDeactivated,
			&errorMessage,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sync log: %w", err)
		}

		if finishedAt.Valid {
			log.FinishedAt = &finishedAt.Time
		}
		if errorMessage.Valid {
			log.ErrorMessage = errorMessage.String
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}
