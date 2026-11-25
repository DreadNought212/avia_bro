package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/DreadNought212/avia_bro/internal/domain"
	"github.com/DreadNought212/avia_bro/internal/external/travelpayouts"
)

type ComplexSearchRepository interface {
	CreateSession(
		ctx context.Context,
		params domain.ComplexSearchParams,
	) (*domain.ComplexSearchSession, error)

	GetSession(
		ctx context.Context,
		sessionID int64,
	) (*domain.ComplexSearchSession, error)

	SaveFlights(
		ctx context.Context,
		sessionID int64,
		flights []*domain.ComplexSearchFlight,
	) error

	UpdateSessionStatus(
		ctx context.Context,
		sessionID int64,
		status domain.ComplexSearchStatus,
		errorMessage string,
	) error

	GetFlightsBySession(
		ctx context.Context,
		sessionID int64,
	) ([]*domain.ComplexSearchFlight, error)
}

type ComplexTripService struct {
	repository       ComplexSearchRepository
	apiClient        *travelpayouts.Client
	excludedAirlines map[string]struct{}
}

func NewComplexTripService(
	repository ComplexSearchRepository,
	apiClient *travelpayouts.Client,
	excludedAirlines []string,
) *ComplexTripService {
	excluded := make(map[string]struct{}, len(excludedAirlines))

	for _, airline := range excludedAirlines {
		code := strings.ToUpper(strings.TrimSpace(airline))
		if code != "" {
			excluded[code] = struct{}{}
		}
	}

	return &ComplexTripService{
		repository:       repository,
		apiClient:        apiClient,
		excludedAirlines: excluded,
	}
}

// Search запускает сбор билетов для сложного маршрута.
func (s *ComplexTripService) Search(
	ctx context.Context,
	params domain.ComplexSearchParams,
) (*domain.ComplexSearchSession, error) {
	if err := validateComplexSearchParams(params); err != nil {
		return nil, err
	}

	if params.Currency == "" {
		params.Currency = "rub"
	}

	session, err := s.repository.CreateSession(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create search session: %w", err)
	}

	if err := s.fetchFlights(ctx, session.ID, params); err != nil {
		statusErr := s.repository.UpdateSessionStatus(
			context.Background(),
			session.ID,
			domain.ComplexSearchStatusFailed,
			err.Error(),
		)

		if statusErr != nil {
			return session, fmt.Errorf(
				"search failed: %w; additionally failed to update session status: %v",
				err,
				statusErr,
			)
		}

		session.Status = domain.ComplexSearchStatusFailed
		session.ErrorMessage = err.Error()

		return session, fmt.Errorf("search failed: %w", err)
	}

	if err := s.repository.UpdateSessionStatus(
		ctx,
		session.ID,
		domain.ComplexSearchStatusCompleted,
		"",
	); err != nil {
		return session, fmt.Errorf("complete search session: %w", err)
	}

	session.Status = domain.ComplexSearchStatusCompleted
	now := time.Now()
	session.FinishedAt = &now

	return session, nil
}

// fetchFlights загружает билеты по всем уникальным направлениям.
func (s *ComplexTripService) fetchFlights(
	ctx context.Context,
	sessionID int64,
	params domain.ComplexSearchParams,
) error {
	routes := buildUniqueRoutes(params)

	maxTotalStayDays := 0
	for _, stop := range params.Stops {
		maxTotalStayDays += stop.MaxDays
	}

	// Последующие сегменты могут вылетать позже даты последнего
	// возможного вылета из начального города.
	extendedEndDate := params.DepartureDateTo.AddDate(
		0,
		0,
		maxTotalStayDays,
	)

	startMonth := firstDayOfMonth(params.DepartureDateFrom)
	endMonth := firstDayOfMonth(extendedEndDate)

	for _, route := range routes {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("search context cancelled: %w", err)
		}

		flights, err := s.apiClient.SearchFlightsForOneWayDateRange(
			ctx,
			route.origin,
			route.destination,
			startMonth,
			endMonth,
			params.Currency,
		)
		if err != nil {
			return fmt.Errorf(
				"fetch flights %s → %s: %w",
				route.origin,
				route.destination,
				err,
			)
		}

		domainFlights := make([]*domain.ComplexSearchFlight, 0, len(flights))

		for _, flight := range flights {
			airline := strings.ToUpper(strings.TrimSpace(flight.Airline))

			if _, excluded := s.excludedAirlines[airline]; excluded {
				continue
			}

			domainFlights = append(
				domainFlights,
				convertFlightToDomain(flight),
			)
		}

		if err := s.repository.SaveFlights(
			ctx,
			sessionID,
			domainFlights,
		); err != nil {
			return fmt.Errorf(
				"save flights %s → %s: %w",
				route.origin,
				route.destination,
				err,
			)
		}
	}

	return nil
}

type complexRoute struct {
	origin      string
	destination string
}

// buildUniqueRoutes формирует последовательность направлений
// и удаляет повторяющиеся пары городов.
func buildUniqueRoutes(
	params domain.ComplexSearchParams,
) []complexRoute {
	cities := make([]string, 0, len(params.Stops)+2)

	cities = append(cities, params.Origin)

	for _, stop := range params.Stops {
		cities = append(cities, stop.City)
	}

	cities = append(cities, params.FinalDestination)

	seen := make(map[string]struct{})
	routes := make([]complexRoute, 0, len(cities)-1)

	for i := 0; i < len(cities)-1; i++ {
		origin := cities[i]
		destination := cities[i+1]

		key := origin + "|" + destination

		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}

		routes = append(routes, complexRoute{
			origin:      origin,
			destination: destination,
		})
	}

	return routes
}

// convertFlightToDomain преобразует ответ Travelpayouts
// в нашу доменную модель.
func convertFlightToDomain(
	flight travelpayouts.Flight,
) *domain.ComplexSearchFlight {
	keyData := fmt.Sprintf(
		"%s|%s|%s|%s|%s|%d|%d",
		flight.Origin,
		flight.Destination,
		flight.DepartureAt.Format(time.RFC3339),
		flight.Airline,
		flight.FlightNumber,
		flight.Price,
		flight.Transfers,
	)

	hash := sha256.Sum256([]byte(keyData))

	return &domain.ComplexSearchFlight{
		FlightKey:          hex.EncodeToString(hash[:]),
		Origin:             flight.Origin,
		Destination:        flight.Destination,
		OriginAirport:      flight.OriginAirport,
		DestinationAirport: flight.DestinationAirport,
		DepartureAt:        flight.DepartureAt,
		Price:              flight.Price,
		Currency:           flight.Currency,
		Airline:            flight.Airline,
		FlightNumber:       flight.FlightNumber,
		Transfers:          flight.Transfers,
		Duration:           flight.Duration,
		Link:               flight.Link,
		BookingLink:        flight.BookingLink,
	}
}

func firstDayOfMonth(date time.Time) time.Time {
	return time.Date(
		date.Year(),
		date.Month(),
		1,
		0,
		0,
		0,
		0,
		date.Location(),
	)
}

func validateComplexSearchParams(
	params domain.ComplexSearchParams,
) error {
	if strings.TrimSpace(params.Origin) == "" {
		return fmt.Errorf("origin is required")
	}

	if strings.TrimSpace(params.FinalDestination) == "" {
		return fmt.Errorf("final destination is required")
	}

	if params.DepartureDateFrom.IsZero() ||
		params.DepartureDateTo.IsZero() {
		return fmt.Errorf("departure date range is required")
	}

	if params.DepartureDateFrom.After(params.DepartureDateTo) {
		return fmt.Errorf("departure date from must not be after departure date to")
	}

	if params.FinalMaxTransfers < 0 {
		return fmt.Errorf("final max transfers cannot be negative")
	}

	for i, stop := range params.Stops {
		if strings.TrimSpace(stop.City) == "" {
			return fmt.Errorf("stop %d: city is required", i+1)
		}

		if stop.MinDays < 0 || stop.MaxDays < 0 {
			return fmt.Errorf("stop %d: stay duration cannot be negative", i+1)
		}

		if stop.MinDays > stop.MaxDays {
			return fmt.Errorf(
				"stop %d: min days cannot be greater than max days",
				i+1,
			)
		}

		if stop.MaxTransfers < 0 {
			return fmt.Errorf(
				"stop %d: max transfers cannot be negative",
				i+1,
			)
		}
	}

	return nil
}
