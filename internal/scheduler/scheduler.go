package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DreadNought212/avia_bro/internal/repository"
	"github.com/DreadNought212/avia_bro/internal/service"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron        *cron.Cron
	syncService *service.FlightSyncService
	routeRepo   *repository.RouteRepository
	mu          sync.Mutex
	running     map[int]bool // routeID -> is running
}

func NewScheduler(
	syncService *service.FlightSyncService,
	routeRepo *repository.RouteRepository,
) *Scheduler {
	return &Scheduler{
		cron:        cron.New(),
		syncService: syncService,
		routeRepo:   routeRepo,
		running:     make(map[int]bool),
	}
}

// Start запускает scheduler для всех активных маршрутов
func (s *Scheduler) Start(ctx context.Context) error {
	fmt.Println("🚀 Запуск scheduler...")

	// Получаем все активные маршруты
	routes, err := s.routeRepo.GetActiveRoutes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active routes: %w", err)
	}

	if len(routes) == 0 {
		fmt.Println("⚠️  Нет активных маршрутов для мониторинга")
		return nil
	}

	// Добавляем задачу для каждого маршрута
	for _, route := range routes {
		err := s.AddRoute(ctx, route)
		if err != nil {
			fmt.Printf("⚠️  Ошибка добавления маршрута %s → %s: %v\n",
				route.Origin, route.Destination, err)
			continue
		}
	}

	// Запускаем scheduler
	s.cron.Start()
	fmt.Printf("✅ Scheduler запущен, мониторим %d маршрутов\n", len(routes))

	return nil
}

// AddRoute добавляет маршрут в scheduler
func (s *Scheduler) AddRoute(ctx context.Context, route *repository.MonitoredRoute) error {
	// Создаём cron выражение: каждые N минут
	cronExpr := fmt.Sprintf("@every %dm", route.SyncIntervalMinutes)

	// Создаём копию route для замыкания
	r := route

	_, err := s.cron.AddFunc(cronExpr, func() {
		s.syncRouteWithLock(r)
	})

	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	fmt.Printf("➕ Добавлен маршрут: %s → %s (каждые %d минут)\n",
		route.Origin, route.Destination, route.SyncIntervalMinutes)

	return nil
}

// syncRouteWithLock синхронизирует маршрут с защитой от параллельного запуска
func (s *Scheduler) syncRouteWithLock(route *repository.MonitoredRoute) {
	s.mu.Lock()
	if s.running[route.ID] {
		fmt.Printf("⏭️  Пропускаю %s → %s: предыдущая синхронизация ещё идёт\n",
			route.Origin, route.Destination)
		s.mu.Unlock()
		return
	}
	s.running[route.ID] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running[route.ID] = false
		s.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err := s.syncService.SyncRoute(ctx, route)
	if err != nil {
		fmt.Printf("❌ Ошибка синхронизации %s → %s: %v\n",
			route.Origin, route.Destination, err)
	}
}

// Stop останавливает scheduler
func (s *Scheduler) Stop() {
	fmt.Println("🛑 Остановка scheduler...")
	ctx := s.cron.Stop()
	<-ctx.Done()
	fmt.Println("✅ Scheduler остановлен")
}

// RunOnce запускает синхронизацию всех маршрутов один раз (для тестирования)
func (s *Scheduler) RunOnce(ctx context.Context) error {
	return s.syncService.SyncAllActiveRoutes(ctx)
}
