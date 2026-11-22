package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type ComplexRoute struct {
	ID              int
	Name            string
	Origin          string
	MinTripDuration int
	MaxTripDuration int
	Active          bool
}

type ComplexRouteDestination struct {
	ID             int
	ComplexRouteID int
	Destination    string
}

type ComplexRouteRepository struct {
	db *sql.DB
}

func NewComplexRouteRepository(db *sql.DB) *ComplexRouteRepository {
	return &ComplexRouteRepository{
		db: db,
	}
}

func (r *ComplexRouteRepository) GetActiveRoutes(
	ctx context.Context,
) ([]ComplexRoute, error) {

	query := `
		SELECT
			id,
			name,
			origin,
			min_trip_duration,
			max_trip_duration,
			active
		FROM common.complex_routes
		WHERE active = TRUE
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active complex routes: %w", err)
	}
	defer rows.Close()

	var routes []ComplexRoute

	for rows.Next() {
		var route ComplexRoute

		err := rows.Scan(
			&route.ID,
			&route.Name,
			&route.Origin,
			&route.MinTripDuration,
			&route.MaxTripDuration,
			&route.Active,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to scan complex route: %w",
				err,
			)
		}

		routes = append(routes, route)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error while iterating complex routes: %w",
			err,
		)
	}

	return routes, nil
}

func (r *ComplexRouteRepository) GetDestinations(
	ctx context.Context,
	complexRouteID int,
) ([]string, error) {

	query := `
		SELECT destination
		FROM common.complex_route_destinations
		WHERE complex_route_id = $1
		ORDER BY destination
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		complexRouteID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get destinations for complex route %d: %w",
			complexRouteID,
			err,
		)
	}
	defer rows.Close()

	var destinations []string

	for rows.Next() {
		var destination string

		if err := rows.Scan(&destination); err != nil {
			return nil, fmt.Errorf(
				"failed to scan destination: %w",
				err,
			)
		}

		destinations = append(destinations, destination)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error while iterating destinations: %w",
			err,
		)
	}

	return destinations, nil
}

func (r *ComplexRouteRepository) GetByID(
	ctx context.Context,
	id int,
) (*ComplexRoute, error) {

	query := `
		SELECT
			id,
			name,
			origin,
			min_trip_duration,
			max_trip_duration,
			active
		FROM common.complex_routes
		WHERE id = $1
	`

	var route ComplexRoute

	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&route.ID,
		&route.Name,
		&route.Origin,
		&route.MinTripDuration,
		&route.MaxTripDuration,
		&route.Active,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get complex route %d: %w",
			id,
			err,
		)
	}

	return &route, nil
}
