package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DreadNought212/avia_bro/internal/external/travelpayouts"
	"github.com/DreadNought212/avia_bro/internal/repository"
)

type FlightSyncService struct {
	apiClient            *travelpayouts.Client
	flightRepo           *repository.FlightRepository
	routeRepo            *repository.RouteRepository
	syncLogRepo          *repository.SyncLogRepository
	notificationRuleRepo *repository.NotificationRuleRepository
}

func NewFlightSyncService(
	apiClient *travelpayouts.Client,
	flightRepo *repository.FlightRepository,
	routeRepo *repository.RouteRepository,
	syncLogRepo *repository.SyncLogRepository,
	notificationRuleRepo *repository.NotificationRuleRepository,
) *FlightSyncService {
	return &FlightSyncService{
		apiClient:            apiClient,
		flightRepo:           flightRepo,
		routeRepo:            routeRepo,
		syncLogRepo:          syncLogRepo,
		notificationRuleRepo: notificationRuleRepo,
	}
}

// GenerateFlightKey создаёт уникальный ключ для билета
func GenerateFlightKey(flight *travelpayouts.Flight) string {
	returnAt := ""
	if flight.ReturnAt != nil {
		returnAt = flight.ReturnAt.Format(time.RFC3339)
	}

	return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		flight.Origin,
		flight.Destination,
		flight.DepartureAt.Format(time.RFC3339),
		returnAt,
		flight.Airline,
		flight.FlightNumber,
	)
}

// SyncRoute синхронизирует билеты для одного маршрута
func (s *FlightSyncService) SyncRoute(ctx context.Context, route *repository.MonitoredRoute) error {
	fmt.Printf("\n🔄 Синхронизация: %s → %s [%s]\n", route.Origin, route.Destination, time.Now().Format("15:04 02-01-2006"))
	//fmt.Println("==================================================================================")

	rule, err := s.notificationRuleRepo.GetByRouteID(ctx, route.ID)
	if err != nil {
		return fmt.Errorf(
			"failed to get notification rule for route %d: %w",
			route.ID,
			err,
		)
	}

	// Создаём лог синхронизации
	logID, err := s.syncLogRepo.CreateSyncLog(ctx, route.ID)
	if err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}

	// Обновляем статус маршрута
	err = s.routeRepo.UpdateLastSync(ctx, route.ID, "running")
	if err != nil {
		return fmt.Errorf("failed to update route status: %w", err)
	}

	// Запрашиваем билеты через API
	flights, err := s.apiClient.SearchFlightsForDateRange(
		ctx,
		route.Origin,
		route.Destination,
		route.StartMonth,
		route.EndMonth,
		route.Direct,
		route.MinTripDuration,
		route.MaxTripDuration,
		route.Currency,
	)

	if err != nil {
		// Логируем ошибку
		s.syncLogRepo.UpdateSyncLogFailure(ctx, logID, err.Error())
		s.routeRepo.UpdateLastSync(ctx, route.ID, "failed")
		return fmt.Errorf("API request failed: %w", err)
	}

	//fmt.Printf("\n📊 Обработка %d билетов...\n", len(flights))

	// Обрабатываем каждый билет
	var flightKeys []string

	newCount := 0
	updatedCount := 0

	for _, flight := range flights {
		flightKey := GenerateFlightKey(&flight)
		flightKeys = append(flightKeys, flightKey)

		result, err := s.flightRepo.UpsertFlight(ctx, &flight, flightKey)
		if err != nil {
			fmt.Printf("⚠️  Ошибка при сохранении билета %s: %v\n", flightKey, err)
			continue
		}

		if result.IsNew {
			newCount++

			if rule != nil && result.Price <= rule.MaxPrice {
				discountPercent := float64(rule.MaxPrice-result.Price) /
					float64(rule.MaxPrice) * 100

				fmt.Printf(
					"🔔 [%s → %s] Новый дешёвый билет: %.0f %s (на %.1f%% ниже лимита)\n",
					route.Origin,
					route.Destination,
					float64(result.Price)/100,
					flight.Currency,
					discountPercent,
				)
			}
		} else {
			updatedCount++

			if result.PriceDropped && rule != nil && result.Price <= rule.MaxPrice {
				discountPercent := float64(rule.MaxPrice-result.Price) /
					float64(rule.MaxPrice) * 100

				fmt.Printf(
					"📉 [%s → %s] Цена упала: %.0f → %.0f %s (на %.1f%% ниже лимита)\n",
					route.Origin,
					route.Destination,
					float64(*result.OldPrice)/100,
					float64(result.Price)/100,
					flight.Currency,
					discountPercent,
				)
			}
		}
	}

	// Деактивируем билеты которых нет в текущей выборке
	deactivatedCount, err := s.flightRepo.DeactivateFlightsNotInKeys(
		ctx,
		route.Origin,
		route.Destination,
		flightKeys,
	)
	if err != nil {
		fmt.Printf("⚠️  Ошибка при деактивации билетов: %v\n", err)
	}

	// Обновляем лог успехом
	err = s.syncLogRepo.UpdateSyncLogSuccess(
		ctx,
		logID,
		len(flights),
		newCount,
		updatedCount,
		deactivatedCount,
	)
	if err != nil {
		return fmt.Errorf("failed to update sync log: %w", err)
	}

	// Обновляем статус маршрута
	err = s.routeRepo.UpdateLastSync(ctx, route.ID, "success")
	if err != nil {
		return fmt.Errorf("failed to update route status: %w", err)
	}

	//fmt.Printf("\n📈 РЕЗУЛЬТАТЫ [%s → %s]:\n",
	//	route.Origin,
	//	route.Destination,
	//)
	//fmt.Printf("  ✅ Всего найдено:    %d\n", len(flights))
	//fmt.Printf("  ✨ Новых:            %d\n", newCount)
	//fmt.Printf("  🔄 Обновлено:        %d\n", updatedCount)
	//fmt.Printf("  ❌ Деактивировано:   %d\n", deactivatedCount)

	return nil
}

// SyncAllActiveRoutes синхронизирует все активные маршруты
func (s *FlightSyncService) SyncAllActiveRoutes(ctx context.Context) error {
	routes, err := s.routeRepo.GetActiveRoutes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active routes: %w", err)
	}

	fmt.Printf("\n🚀 Запуск синхронизации %d маршрутов [%s]\n", len(routes), time.Now().Format("15:04 02-01-2006"))

	for _, route := range routes {
		err := s.SyncRoute(ctx, route)
		if err != nil {
			fmt.Printf("❌ Ошибка синхронизации %s → %s: %v\n",
				route.Origin, route.Destination, err)
			// Продолжаем следующий маршрут
			continue
		}

		// Пауза между маршрутами
		time.Sleep(2 * time.Second)
	}

	fmt.Println("\n✅ Синхронизация завершена")
	return nil
}

// GetNewFlights возвращает новые билеты за последний период
func (s *FlightSyncService) GetNewFlights(
	ctx context.Context,
	origin, destination string,
	sinceDuration time.Duration,
) ([]*travelpayouts.Flight, error) {
	since := time.Now().Add(-sinceDuration)
	return s.flightRepo.GetNewFlights(ctx, origin, destination, since)
}
