package main

// ============================================================================
// ПРИМЕРЫ КОДА ДЛЯ AVIASALES BOT НА GO
// ============================================================================

// ============================================================================
// 1. DOMAIN MODELS
// ============================================================================

package domain

import (
	"time"
	"github.com/google/uuid"
)

// Flight представляет авиабилет
type Flight struct {
	ID          string    `db:"id"`
	Origin      string    `db:"origin"`        // MOW, SPB, etc.
	Destination string    `db:"destination"`
	DepartDate  time.Time `db:"depart_date"`
	ReturnDate  *time.Time `db:"return_date"`
	Price       int64     `db:"price"`         // в копейках
	Currency    string    `db:"currency"`
	Airline     string    `db:"airline"`
	DepartTime  time.Time `db:"depart_time"`
	ArrivalTime time.Time `db:"arrival_time"`
	Duration    int       `db:"duration_minutes"`
	Transfers   int       `db:"transfers"`
	BookingLink string    `db:"booking_link"`
	Score       float32   `db:"score"`
	ParsedAt    time.Time `db:"parsed_at"`
	CreatedAt   time.Time `db:"created_at"`
}

// PublishedFlight представляет опубликованный билет
type PublishedFlight struct {
	ID              string
	FlightID        string
	TelegramMsgID   int64
	Appeal          string // текст для привлечения внимания
	Segment         string // budget_travelers, family, luxury, business
	PublishedAt     time.Time
	Clicks          int
	Conversions     int
}

// UserProfile профиль пользователя
type UserProfile struct {
	TelegramUserID      int64
	FirstName           string
	LastName            string
	Username            string
	PreferredOrigins    []string // ["MOW", "SPB"]
	PreferredDestinations []string // ["BCN", "PAR", "MIL"]
	MaxBudget           int      // в рублях
	MinRating           float32  // минимальный рейтинг авиакомпании
	NotificationEnabled bool
	Language            string
	CreatedAt           time.Time
}

// UserAction логирует действия пользователя
type UserAction struct {
	ID            int64
	TelegramUserID int64
	ActionType    string // "view", "click_buy", "share"
	FlightID      string
	PublishedID   string
	Metadata      map[string]interface{}
	CreatedAt     time.Time
}

// ============================================================================
// 2. AVIASALES API CLIENT
// ============================================================================

package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AviasalesClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// AviasalesResponse это структура ответа от Aviasales API
type AviasalesResponse struct {
	Data []struct {
		Value struct {
			GroupParameters struct {
				TripClass string `json:"tripClass"` // economy, business, etc.
				Cabin     string `json:"cabin"`
			} `json:"groupParameters"`
			Legs []struct {
				DepartureDate   string `json:"departureDate"` // 2024-05-15T14:20:00`
				ArrivalDate     string `json:"arrivalDate"`
				DurationSeconds int    `json:"durationSeconds"`
				Stops           int    `json:"stops"`
				AirlineID       int    `json:"airlineId"`
			} `json:"legs"`
			Price struct {
				Minfare int    `json:"minfare"` // в копейках
				CurrencyCode string `json:"currencyCode"` // RUB
			} `json:"price"`
		} `json:"value"`
	} `json:"data"`
}

func NewAviasalesClient(apiKey string) *AviasalesClient {
	return &AviasalesClient{
		baseURL: "https://api.aviasales.ru",
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchFlights ищет авиабилеты
func (ac *AviasalesClient) SearchFlights(ctx context.Context, origin, destination string, departDate time.Time) (*AviasalesResponse, error) {
	url := fmt.Sprintf(
		"%s/v2/search?origin=%s&destination=%s&depart_date=%s&one_way=1&token=%s",
		ac.baseURL,
		origin,
		destination,
		departDate.Format("2006-01-02"),
		ac.apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ac.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch flights: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result AviasalesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ============================================================================
// 3. PARSER SERVICE
// ============================================================================

package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type ParserService struct {
	aviasalesClient *AviasalesClient
	flightRepo      *FlightRepository
	cache           *RedisCache
	logger          *Logger
}

// PopularRoutes для парсинга (можно расширять)
var PopularRoutes = [][2]string{
	{"MOW", "BCN"}, // Москва → Барселона
	{"MOW", "PAR"}, // Москва → Париж
	{"SPB", "DUB"}, // СПб → Дублин
	{"MOW", "BKK"}, // Москва → Бангкок
	{"SVX", "IST"}, // Екатеринбург → Стамбул
	{"MOW", "MIL"}, // Москва → Милан
	{"MOW", "CDG"}, // Москва → Шарль де Голль
	// ...добавлять по мере расширения
}

func NewParserService(
	aviasalesClient *AviasalesClient,
	flightRepo *FlightRepository,
	cache *RedisCache,
	logger *Logger,
) *ParserService {
	return &ParserService{
		aviasalesClient: aviasalesClient,
		flightRepo:      flightRepo,
		cache:           cache,
		logger:          logger,
	}
}

// Start запускает парсер в фоновом режиме
func (ps *ParserService) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	ps.logger.Info("Parser Service started", map[string]interface{}{
		"interval": interval,
	})

	for {
		select {
		case <-ctx.Done():
			ps.logger.Info("Parser Service stopped")
			return
		case <-ticker.C:
			ps.parseAllRoutes(ctx)
		}
	}
}

// parseAllRoutes парсит все популярные маршруты параллельно
func (ps *ParserService) parseAllRoutes(ctx context.Context) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Ограничиваем 5 параллельными запросами

	for _, route := range PopularRoutes {
		wg.Add(1)
		go func(origin, destination string) {
			defer wg.Done()

			semaphore <- struct{}{}        // Занимаем слот
			defer func() { <-semaphore }() // Освобождаем

			ps.parseRoute(ctx, origin, destination)
		}(route[0], route[1])
	}

	wg.Wait()
	ps.logger.Info("Parsing round completed", map[string]interface{}{
		"timestamp": time.Now(),
	})
}

// parseRoute парсит один маршрут
func (ps *ParserService) parseRoute(ctx context.Context, origin, destination string) {
	// Парсим на неделю вперед
	for i := 1; i <= 30; i++ {
		departDate := time.Now().AddDate(0, 0, i)

		response, err := ps.aviasalesClient.SearchFlights(ctx, origin, destination, departDate)
		if err != nil {
			ps.logger.Error("Failed to search flights", map[string]interface{}{
				"origin":      origin,
				"destination": destination,
				"date":        departDate,
				"error":       err.Error(),
			})
			continue
		}

		// Обрабатываем результаты
		ps.processFlights(ctx, response, origin, destination)

		// Небольшая задержка, чтобы не заспамить API
		time.Sleep(100 * time.Millisecond)
	}
}

// processFlights сохраняет билеты в БД и кэш
func (ps *ParserService) processFlights(ctx context.Context, resp *AviasalesResponse, origin, destination string) {
	for _, data := range resp.Data {
		flight := ps.convertToFlight(data, origin, destination)

		// Сохраняем в БД (upsert)
		if err := ps.flightRepo.SaveFlight(ctx, flight); err != nil {
			ps.logger.Error("Failed to save flight", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		// Кэшируем для быстрого доступа
		cacheKey := fmt.Sprintf("flight:%s:%s", origin, destination)
		ps.cache.AddToList(ctx, cacheKey, flight, 2*time.Hour)
	}
}

// convertToFlight конвертирует ответ от API в доменную модель
func (ps *ParserService) convertToFlight(data interface{}, origin, destination string) *Flight {
	// Реализация конвертации...
	return &Flight{
		ID:          uuid.New().String(),
		Origin:      origin,
		Destination: destination,
		ParsedAt:    time.Now(),
	}
}

// ============================================================================
// 4. RANKER SERVICE (с LLM)
// ============================================================================

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type RankerService struct {
	flightRepo    *FlightRepository
	publishedRepo *PublishedFlightRepository
	llmClient     *LLMClient // OpenAI или Claude
	logger        *Logger
}

type RankerOutput struct {
	Appeal          string `json:"appeal"`           // текст для привлечения
	Segment         string `json:"segment"`          // целевой сегмент
	DiscountPercent int    `json:"discount_percent"` // процент скидки
	WhyGood         string `json:"why_good"`         // почему выгодно
}

func NewRankerService(
	flightRepo *FlightRepository,
	publishedRepo *PublishedFlightRepository,
	llmClient *LLMClient,
	logger *Logger,
) *RankerService {
	return &RankerService{
		flightRepo:    flightRepo,
		publishedRepo: publishedRepo,
		llmClient:     llmClient,
		logger:        logger,
	}
}

// Start запускает ранжировщик
func (rs *RankerService) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	rs.logger.Info("Ranker Service started", map[string]interface{}{
		"interval": interval,
	})

	for {
		select {
		case <-ctx.Done():
			rs.logger.Info("Ranker Service stopped")
			return
		case <-ticker.C:
			rs.rankAndPublish(ctx)
		}
	}
}

// rankAndPublish выбирает топ билеты и публикует их
func (rs *RankerService) rankAndPublish(ctx context.Context) {
	// 1. Получаем последние билеты из БД
	recentFlights, err := rs.flightRepo.GetRecentFlights(ctx, 1000)
	if err != nil {
		rs.logger.Error("Failed to get recent flights", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 2. Фильтруем по правилам
	filteredFlights := rs.filterFlights(ctx, recentFlights)

	// 3. Отправляем в LLM для ранжирования
	rankedFlights, err := rs.rankWithLLM(ctx, filteredFlights)
	if err != nil {
		rs.logger.Error("Failed to rank flights with LLM", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 4. Берём топ 5-10 билетов
	topFlights := rankedFlights[:min(10, len(rankedFlights))]

	// 5. Сохраняем для публикации
	for _, flight := range topFlights {
		published := &PublishedFlight{
			ID:        uuid.New().String(),
			FlightID:  flight.ID,
			Appeal:    flight.rankerOutput.Appeal,
			Segment:   flight.rankerOutput.Segment,
			PublishedAt: time.Now(),
		}

		if err := rs.publishedRepo.Save(ctx, published); err != nil {
			rs.logger.Error("Failed to save published flight", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	rs.logger.Info("Ranking completed", map[string]interface{}{
		"total_ranked":  len(rankedFlights),
		"top_published": len(topFlights),
	})
}

// filterFlights применяет бизнес-правила фильтрации
func (rs *RankerService) filterFlights(ctx context.Context, flights []*Flight) []*Flight {
	var filtered []*Flight

	for _, flight := range flights {
		// Правило 1: Скидка >= 20%
		discountPercent := rs.calculateDiscount(flight)
		if discountPercent < 20 {
			continue
		}

		// Правило 2: Не дублировать один маршрут в течение 48 часов
		if rs.isRouteDuplicate(ctx, flight) {
			continue
		}

		// Правило 3: Вылет не раньше, чем через 1 день
		if flight.DepartDate.Sub(time.Now()) < 24*time.Hour {
			continue
		}

		// Правило 4: Исключить непопулярные направления
		if !rs.isPopularDestination(flight.Destination) {
			continue
		}

		filtered = append(filtered, flight)
	}

	return filtered
}

// rankWithLLM отправляет билеты в LLM для анализа
func (rs *RankerService) rankWithLLM(ctx context.Context, flights []*Flight) ([]*Flight, error) {
	type FlightWithScore struct {
		Flight *Flight
		Score  float32
	}

	var results []*Flight

	for _, flight := range flights {
		prompt := rs.buildPrompt(flight)

		response, err := rs.llmClient.Complete(ctx, prompt)
		if err != nil {
			rs.logger.Error("LLM request failed", map[string]interface{}{
				"flight_id": flight.ID,
				"error":     err.Error(),
			})
			continue
		}

		// Парсим JSON ответ от LLM
		var output RankerOutput
		if err := json.Unmarshal([]byte(response), &output); err != nil {
			rs.logger.Error("Failed to parse LLM response", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		flight.rankerOutput = output
		flight.Score = float32(output.DiscountPercent) / 100.0

		results = append(results, flight)
	}

	// Сортируем по скору (descending)
	// ...

	return results, nil
}

// buildPrompt создаёт промпт для LLM
func (rs *RankerService) buildPrompt(flight *Flight) string {
	return fmt.Sprintf(`
Ты — эксперт в путешествиях. Проанализируй авиабилет и ответь JSON без дополнительного текста:

Маршрут: %s → %s
Цена: %d ₽
Дата вылета: %s
Пересадки: %d
Время в пути: %d часов

Ответь ТОЛЬКО JSON:
{
  "appeal": "Привлекательное описание для пользователя (max 200 символов)",
  "segment": "budget_travelers|family|luxury|business",
  "discount_percent": 54,
  "why_good": "Почему это хороший билет"
}`,
		flight.Origin,
		flight.Destination,
		flight.Price/100,
		flight.DepartDate.Format("02.01.2006"),
		flight.Transfers,
		flight.Duration/60,
	)
}

// calculateDiscount вычисляет процент скидки
func (rs *RankerService) calculateDiscount(flight *Flight) int {
	// Получаем среднюю цену для этого маршрута за последние 30 дней
	avgPrice := rs.flightRepo.GetAveragePrice(context.Background(), flight.Origin, flight.Destination, 30)
	if avgPrice == 0 {
		return 0
	}
	return int((float64(avgPrice-flight.Price) / float64(avgPrice)) * 100)
}

func (rs *RankerService) isRouteDuplicate(ctx context.Context, flight *Flight) bool {
	// Проверяем, был ли этот маршрут опубликован за последние 48 часов
	lastPublished, err := rs.publishedRepo.GetLastPublishedForRoute(ctx, flight.Origin, flight.Destination)
	if err != nil || lastPublished == nil {
		return false
	}
	return time.Since(lastPublished.PublishedAt) < 48*time.Hour
}

func (rs *RankerService) isPopularDestination(destination string) bool {
	popular := map[string]bool{
		"BCN": true, // Барселона
		"PAR": true, // Париж
		"CDG": true, // Шарль де Голль
		"DUB": true, // Дублин
		"BKK": true, // Бангкок
		"IST": true, // Стамбул
		"MIL": true, // Милан
		"FCO": true, // Рим
		"MAD": true, // Мадрид
		"AMS": true, // Амстердам
	}
	return popular[destination]
}

// ============================================================================
// 5. PUBLISHER SERVICE (Telegram)
// ============================================================================

package service

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type PublisherService struct {
	telegramClient    *tgbotapi.BotAPI
	publishedRepo     *PublishedFlightRepository
	userActionRepo    *UserActionRepository
	channelID         int64
	logger            *Logger
}

func NewPublisherService(
	telegramToken string,
	channelID int64,
	publishedRepo *PublishedFlightRepository,
	userActionRepo *UserActionRepository,
	logger *Logger,
) (*PublisherService, error) {
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	return &PublisherService{
		telegramClient: bot,
		publishedRepo:  publishedRepo,
		userActionRepo: userActionRepo,
		channelID:      channelID,
		logger:         logger,
	}, nil
}

// PublishFlights публикует выбранные билеты в Telegram
func (ps *PublisherService) PublishFlights(ctx context.Context, flights []*PublishedFlight) error {
	for _, published := range flights {
		// Получаем полную информацию о билете
		flight, err := ps.publishedRepo.GetFlightDetails(ctx, published.FlightID)
		if err != nil {
			ps.logger.Error("Failed to get flight details", map[string]interface{}{
				"flight_id": published.FlightID,
			})
			continue
		}

		// Форматируем сообщение
		message := ps.formatMessage(flight, published)

		// Отправляем в Telegram
		msg := tgbotapi.NewMessageToChannel(ps.channelID, message)
		msg.ParseMode = "HTML"

		// Добавляем inline-кнопки
		msg.ReplyMarkup = ps.createInlineKeyboard(published)

		sentMessage, err := ps.telegramClient.Send(msg)
		if err != nil {
			ps.logger.Error("Failed to send message to Telegram", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		// Сохраняем ID сообщения для отслеживания
		published.TelegramMsgID = int64(sentMessage.MessageID)
		if err := ps.publishedRepo.Update(ctx, published); err != nil {
			ps.logger.Error("Failed to update published flight", map[string]interface{}{
				"error": err.Error(),
			})
		}

		ps.logger.Info("Message published", map[string]interface{}{
			"message_id": sentMessage.MessageID,
			"route":      fmt.Sprintf("%s→%s", flight.Origin, flight.Destination),
		})

		time.Sleep(500 * time.Millisecond) // Избежать rate limiting
	}

	return nil
}

// formatMessage формирует красивое сообщение для Telegram
func (ps *PublisherService) formatMessage(flight *Flight, published *PublishedFlight) string {
	discount := ps.calculateDiscountPercent(flight)

	return fmt.Sprintf(`✈️ <b>ВЫГОДНЫЙ БИЛЕТ</b>

🛫 %s → %s
📅 %s - %s (%d дней)
💰 <b>%d ₽</b> (-<b>%d%%</b>, было %d ₽)
⏱ Вылет: %s → Прибытие: %s+%d
🪑 %d пересадки

%s

#полеты #путешествия #скидка`,
		flight.Origin,
		flight.Destination,
		flight.DepartDate.Format("2 Jan"),
		flight.ReturnDate.Format("2 Jan"),
		int(flight.ReturnDate.Sub(flight.DepartDate).Hours())/24,
		flight.Price/100,
		discount,
		ps.getAveragePrice(flight)*100,
		flight.DepartTime.Format("15:04"),
		flight.ArrivalTime.Format("15:04"),
		ps.calculateTimezoneOffset(flight),
		flight.Transfers,
		published.Appeal,
	)
}

// createInlineKeyboard создаёт кнопки под сообщением
func (ps *PublisherService) createInlineKeyboard(published *PublishedFlight) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(
				"✈️ Купить билет",
				published.Flight.BookingLink, // + реферальный параметр
			),
		),
	)
}

// ============================================================================
// 6. REST API SERVER
// ============================================================================

package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type APIServer struct {
	engine *gin.Engine
	flightRepo *FlightRepository
	userRepo   *UserProfileRepository
	logger     *Logger
}

func NewAPIServer(flightRepo *FlightRepository, userRepo *UserProfileRepository, logger *Logger) *APIServer {
	engine := gin.Default()

	server := &APIServer{
		engine:     engine,
		flightRepo: flightRepo,
		userRepo:   userRepo,
		logger:     logger,
	}

	server.setupRoutes()
	return server
}

func (s *APIServer) setupRoutes() {
	api := s.engine.Group("/api/v1")

	// Flights
	api.GET("/flights", s.getFlights)
	api.GET("/flights/:id", s.getFlightByID)

	// User
	api.POST("/user/preferences", s.setUserPreferences)
	api.GET("/user/profile", s.getUserProfile)

	// Analytics
	api.GET("/analytics/top-routes", s.getTopRoutes)

	// Health
	s.engine.GET("/health", s.health)
}

// Handlers

func (s *APIServer) getFlights(c *gin.Context) {
	origin := c.Query("origin")
	destination := c.Query("destination")
	limit := c.DefaultQuery("limit", "20")

	flights, err := s.flightRepo.FindFlights(c.Request.Context(), origin, destination, limit)
	if err != nil {
		s.logger.Error("Failed to get flights", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flights"})
		return
	}

	c.JSON(http.StatusOK, flights)
}

func (s *APIServer) getFlightByID(c *gin.Context) {
	id := c.Param("id")
	flight, err := s.flightRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Flight not found"})
		return
	}
	c.JSON(http.StatusOK, flight)
}

func (s *APIServer) setUserPreferences(c *gin.Context) {
	var prefs UserProfile
	if err := c.ShouldBindJSON(&prefs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.userRepo.Save(c.Request.Context(), &prefs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences saved"})
}

func (s *APIServer) getUserProfile(c *gin.Context) {
	userID := c.GetInt64("user_id") // из JWT токена

	profile, err := s.userRepo.FindByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (s *APIServer) getTopRoutes(c *gin.Context) {
	routes, err := s.flightRepo.GetTopRoutes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get top routes"})
		return
	}

	c.JSON(http.StatusOK, routes)
}

func (s *APIServer) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func (s *APIServer) Start(addr string) error {
	return s.engine.Run(addr)
}

// ============================================================================
// 7. MAIN SERVICE
// ============================================================================

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Загружаем переменные окружения
	godotenv.Load()

	logger := NewLogger(os.Getenv("LOG_LEVEL"))

	// Инициализируем БД
	db := InitPostgres(os.Getenv("DATABASE_URL"))
	defer db.Close()

	// Инициализируем Redis
	cache := InitRedis(os.Getenv("REDIS_URL"))
	defer cache.Close()

	// Создаём репозитории
	flightRepo := NewFlightRepository(db)
	publishedRepo := NewPublishedFlightRepository(db)
	userRepo := NewUserProfileRepository(db)

	// Создаём external clients
	aviasalesClient := NewAviasalesClient(os.Getenv("AVIASALES_API_KEY"))
	llmClient := NewLLMClient(os.Getenv("OPENAI_API_KEY"))
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	channelID := parseChannelID(os.Getenv("TELEGRAM_CHANNEL_ID"))

	// Инициализируем сервисы
	parserService := NewParserService(aviasalesClient, flightRepo, cache, logger)
	rankerService := NewRankerService(flightRepo, publishedRepo, llmClient, logger)
	publisherService, err := NewPublisherService(telegramToken, channelID, publishedRepo, logger)
	if err != nil {
		log.Fatalf("Failed to create publisher service: %v", err)
	}

	apiServer := NewAPIServer(flightRepo, userRepo, logger)

	// Контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем сервисы в горутинах
	go parserService.Start(ctx, 30*time.Minute)
	go rankerService.Start(ctx, 3*time.Hour)
	go apiServer.Start(":8080")

	logger.Info("All services started", nil)

	// Ожидаем сигнала для завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutting down...", nil)

	cancel()
	time.Sleep(2 * time.Second) // Даём время на graceful shutdown
}
