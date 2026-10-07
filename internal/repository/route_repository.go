package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type MonitoredRoute struct {
	ID                  int
	Origin              string
	Destination         string
	StartMonth          time.Time
	EndMonth            time.Time
	Direct              bool
	MinTripDuration     int
	MaxTripDuration     int
	Currency            string
	SyncIntervalMinutes int
	IsActive            bool
	LastSyncAt          *time.Time
	LastSyncStatus      string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type RouteRepository struct {
	db *sql.DB
}

func NewRouteRepository(db *sql.DB) *RouteRepository {
	return &RouteRepository{db: db}
}

// GetActiveRoutes возвращает все активные маршруты
func (r *RouteRepository) GetActiveRoutes(ctx context.Context) ([]*MonitoredRoute, error) {
	query := `SELECT * FROM common.get_active_routes()`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active routes: %w", err)
	}
	defer rows.Close()

	var routes []*MonitoredRoute

	for rows.Next() {
		route := &MonitoredRoute{}

		var lastSyncAt sql.NullTime
		var lastSyncStatus sql.NullString

		err := rows.Scan(
			&route.ID,
			&route.Origin,
			&route.Destination,
			&route.StartMonth,
			&route.EndMonth,
			&route.Direct,
			&route.MinTripDuration,
			&route.MaxTripDuration,
			&route.Currency,
			&route.SyncIntervalMinutes,
			&route.IsActive,
			&lastSyncAt,
			&lastSyncStatus,
			&route.CreatedAt,
			&route.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan route: %w", err)
		}

		if lastSyncAt.Valid {
			route.LastSyncAt = &lastSyncAt.Time
		}

		if lastSyncStatus.Valid {
			route.LastSyncStatus = lastSyncStatus.String
		}

		routes = append(routes, route)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate routes: %w", err)
	}

	return routes, nil
}

// UpdateLastSync обновляет время и статус последней синхронизации
func (r *RouteRepository) UpdateLastSync(
	ctx context.Context,
	routeID int,
	status string,
) error {
	query := `
		UPDATE common.monitored_routes
		SET last_sync_at = NOW(),
		    last_sync_status = $1,
		    updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, status, routeID)
	if err != nil {
		return fmt.Errorf("failed to update last sync: %w", err)
	}

	return nil
}

// CreateRoute создаёт новый маршрут для мониторинга
func (r *RouteRepository) CreateRoute(ctx context.Context, route *MonitoredRoute) (int, error) {
	query := `
		INSERT INTO common.monitored_routes (
			origin, destination, start_month, end_month,
			direct, min_trip_duration, max_trip_duration, currency,
			sync_interval_minutes, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx, query,
		route.Origin,
		route.Destination,
		route.StartMonth,
		route.EndMonth,
		route.Direct,
		route.MinTripDuration,
		route.MaxTripDuration,
		route.Currency,
		route.SyncIntervalMinutes,
		route.IsActive,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create route: %w", err)
	}

	return id, nil
}
