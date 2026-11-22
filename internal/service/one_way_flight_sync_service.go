package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DreadNought212/avia_bro/internal/external/travelpayouts"
	"github.com/DreadNought212/avia_bro/internal/repository"
)

type OneWayFlightSyncService struct {
	client           *travelpayouts.Client
	flightRepo       *repository.FlightRepository
	complexRouteRepo *repository.ComplexRouteRepository
}

func NewOneWayFlightSyncService(
	client *travelpayouts.Client,
	flightRepo *repository.FlightRepository,
	complexRouteRepo *repository.ComplexRouteRepository,
) *OneWayFlightSyncService {
	return &OneWayFlightSyncService{
		client:           client,
		flightRepo:       flightRepo,
		complexRouteRepo: complexRouteRepo,
	}
}

func (s *OneWayFlightSyncService) SyncRoute(
	ctx context.Context,
	routeID int,
	startMonth time.Time,
	endMonth time.Time,
	currency string,
) error {

	route, err := s.complexRouteRepo.GetByID(
		ctx,
		routeID,
	)
	if err != nil {
		return err
	}

	if !route.Active {
		return fmt.Errorf(
			"complex route %d is inactive",
			routeID,
		)
	}

	destinations, err := s.complexRouteRepo.GetDestinations(
		ctx,
		route.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to get destinations for complex route %d: %w",
			route.ID,
			err,
		)
	}

	for _, destination := range destinations {

		// MOW → SHA
		if err := s.syncDirection(
			ctx,
			route.Origin,
			destination,
			startMonth,
			endMonth,
			currency,
		); err != nil {
			return err
		}

		// SHA → MOW
		if err := s.syncDirection(
			ctx,
			destination,
			route.Origin,
			startMonth,
			endMonth,
			currency,
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *OneWayFlightSyncService) SyncAll(
	ctx context.Context,
	startMonth time.Time,
	endMonth time.Time,
	currency string,
) error {

	routes, err := s.complexRouteRepo.GetActiveRoutes(ctx)
	if err != nil {
		return fmt.Errorf(
			"failed to get active complex routes: %w",
			err,
		)
	}

	// key = "origin:destination"
	directions := make(map[string][2]string)

	for _, route := range routes {

		destinations, err := s.complexRouteRepo.GetDestinations(
			ctx,
			route.ID,
		)
		if err != nil {
			return fmt.Errorf(
				"failed to get destinations for complex route %d: %w",
				route.ID,
				err,
			)
		}

		for _, destination := range destinations {

			// origin → destination
			key := route.Origin + ":" + destination

			directions[key] = [2]string{
				route.Origin,
				destination,
			}

			// destination → origin
			key = destination + ":" + route.Origin

			directions[key] = [2]string{
				destination,
				route.Origin,
			}
		}
	}

	fmt.Printf(
		"🔄 One-way sync: найдено уникальных направлений: %d\n",
		len(directions),
	)

	for _, direction := range directions {

		origin := direction[0]
		destination := direction[1]

		if err := s.syncDirection(
			ctx,
			origin,
			destination,
			startMonth,
			endMonth,
			currency,
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *OneWayFlightSyncService) syncDirection(
	ctx context.Context,
	origin string,
	destination string,
	startMonth time.Time,
	endMonth time.Time,
	currency string,
) error {

	fmt.Printf(
		"🔄 One-way sync: %s → %s\n",
		origin,
		destination,
	)

	flights, err := s.client.SearchFlightsForOneWayDateRange(
		ctx,
		origin,
		destination,
		startMonth,
		endMonth,
		currency,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to search one-way flights %s → %s: %w",
			origin,
			destination,
			err,
		)
	}

	fmt.Printf(
		"✈️ %s → %s: найдено %d билетов\n",
		origin,
		destination,
		len(flights),
	)

	for i := range flights {

		flight := &flights[i]

		flightKey := GenerateFlightKey(flight)

		_, err := s.flightRepo.UpsertFlight(
			ctx,
			flight,
			flightKey,
		)
		if err != nil {
			return fmt.Errorf(
				"failed to save flight %s → %s: %w",
				origin,
				destination,
				err,
			)
		}
	}

	return nil
}
