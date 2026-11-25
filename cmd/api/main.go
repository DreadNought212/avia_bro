package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/DreadNought212/avia_bro/internal/api"
	"github.com/DreadNought212/avia_bro/internal/external/travelpayouts"
	"github.com/DreadNought212/avia_bro/internal/repository"
	"github.com/DreadNought212/avia_bro/internal/service"
)

func main() {
	_ = godotenv.Load("config/.env")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	apiKey := os.Getenv("TRAVELPAYOUTS_API_KEY")
	if apiKey == "" {
		log.Fatal("TRAVELPAYOUTS_API_KEY is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	excludedAirlinesEnv := os.Getenv("EXCLUDED_AIRLINES")
	if excludedAirlinesEnv == "" {
		excludedAirlinesEnv = "S7,DV"
	}

	excludedAirlines := strings.Split(excludedAirlinesEnv, ",")

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("connect to database: %v", err)
	}

	// Обычный поиск билетов
	tripRepository := repository.NewTripSearchRepository(db)
	tripService := service.NewTripService(tripRepository)

	// Сложный многосегментный поиск
	complexSearchRepository := repository.NewComplexSearchRepository(db)
	travelpayoutsClient := travelpayouts.NewClient(apiKey)
	complexTripService := service.NewComplexTripService(
		complexSearchRepository,
		travelpayoutsClient,
		excludedAirlines,
	)

	router := api.NewRouter(tripService, complexTripService)
	server := api.NewHTTPServer(port, router)

	go func() {
		log.Printf("API server listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
			log.Fatalf("HTTP server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
}
