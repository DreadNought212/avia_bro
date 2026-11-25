package service

import (
	"testing"
	"time"

	"github.com/DreadNought212/avia_bro/internal/domain"
)

func TestCombineFlights_TransferLimits(t *testing.T) {
	tests := []struct {
		name            string
		maxTransfers    int
		flightTransfers int
		wantResults     int
	}{
		{
			name:            "direct flight allowed",
			maxTransfers:    0,
			flightTransfers: 0,
			wantResults:     1,
		},
		{
			name:            "one transfer allowed",
			maxTransfers:    1,
			flightTransfers: 1,
			wantResults:     1,
		},
		{
			name:            "too many transfers rejected",
			maxTransfers:    1,
			flightTransfers: 2,
			wantResults:     0,
		},
		{
			name:            "direct flight rejected when limit is zero",
			maxTransfers:    0,
			flightTransfers: 1,
			wantResults:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			departure := time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC)

			params := domain.ComplexSearchParams{
				Origin:            "MOW",
				FinalDestination:  "SHA",
				FinalMaxTransfers: tt.maxTransfers,
				DepartureDateFrom: departure.Add(-time.Hour),
				DepartureDateTo:   departure.Add(time.Hour),
			}

			flights := []*domain.ComplexSearchFlight{
				{
					Origin:      "MOW",
					Destination: "SHA",
					DepartureAt: departure,
					Price:       30000,
					Currency:    "RUB",
					Transfers:   tt.flightTransfers,
					Duration:    600,
				},
			}

			results := combineFlights(params, flights)

			if len(results) != tt.wantResults {
				t.Fatalf(
					"expected %d results, got %d",
					tt.wantResults,
					len(results),
				)
			}
		})
	}
}

func TestCombineFlights_TransferLimitsAreIndependent(t *testing.T) {
	departure := time.Date(2026, 12, 1, 6, 0, 0, 0, time.UTC)

	params := domain.ComplexSearchParams{
		Origin: "MOW",
		Stops: []domain.ComplexTripStop{
			{
				City:         "SHA",
				MinDays:      1,
				MaxDays:      5,
				MaxTransfers: 1,
			},
		},
		FinalDestination:  "TYO",
		FinalMaxTransfers: 0,
		DepartureDateFrom: departure.Add(-time.Hour),
		DepartureDateTo:   departure.Add(time.Hour),
	}

	flights := []*domain.ComplexSearchFlight{
		{
			Origin:      "MOW",
			Destination: "SHA",
			DepartureAt: departure,
			Price:       30000,
			Currency:    "RUB",
			Transfers:   1,
			Duration:    600,
		},
		{
			Origin:      "SHA",
			Destination: "TYO",
			DepartureAt: time.Date(2026, 12, 4, 2, 0, 0, 0, time.UTC),
			Price:       15000,
			Currency:    "RUB",
			Transfers:   0,
			Duration:    180,
		},
		{
			Origin:      "SHA",
			Destination: "TYO",
			DepartureAt: time.Date(2026, 12, 4, 3, 0, 0, 0, time.UTC),
			Price:       12000,
			Currency:    "RUB",
			Transfers:   1,
			Duration:    180,
		},
	}

	results := combineFlights(params, flights)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Flights[0].Transfers != 1 {
		t.Errorf("first segment should allow 1 transfer")
	}

	if results[0].Flights[1].Transfers != 0 {
		t.Errorf("final segment should allow only direct flights")
	}
}

func TestCombineFlights_Top1000CheapestResults(t *testing.T) {
	tests := []struct {
		name        string
		flightCount int
		descending  bool
	}{
		{
			name:        "less than limit",
			flightCount: 999,
		},
		{
			name:        "exactly limit",
			flightCount: 1000,
		},
		{
			name:        "more than limit",
			flightCount: 1001,
		},
		{
			name:        "replace expensive results with cheaper ones",
			flightCount: 1001,
			descending:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			departure := time.Date(
				2026, 12, 1, 10, 0, 0, 0, time.UTC,
			)

			params := domain.ComplexSearchParams{
				Origin:            "MOW",
				FinalDestination:  "SHA",
				FinalMaxTransfers: 0,
				DepartureDateFrom: departure.Add(-time.Hour),
				DepartureDateTo:   departure.Add(time.Duration(tt.flightCount+1) * time.Hour),
			}

			flights := make([]*domain.ComplexSearchFlight, 0, tt.flightCount)

			for i := 0; i < tt.flightCount; i++ {
				price := int64(i + 1)

				if tt.descending {
					price = int64(tt.flightCount - i)
				}

				flights = append(flights, &domain.ComplexSearchFlight{
					Origin:      "MOW",
					Destination: "SHA",
					DepartureAt: departure.Add(time.Duration(i) * time.Minute),
					Price:       price,
					Currency:    "RUB",
					Transfers:   0,
					Duration:    60,
				})
			}

			results := combineFlights(params, flights)

			expectedCount := tt.flightCount
			if expectedCount > maxComplexTripResults {
				expectedCount = maxComplexTripResults
			}

			if len(results) != expectedCount {
				t.Fatalf(
					"expected %d results, got %d",
					expectedCount,
					len(results),
				)
			}

			for i, result := range results {
				expectedPrice := int64(i + 1)

				if result.TotalPrice != expectedPrice {
					t.Errorf(
						"result %d: expected price %d, got %d",
						i,
						expectedPrice,
						result.TotalPrice,
					)
				}
			}
		})
	}
}

func BenchmarkCombineFlights_MultiSegment(b *testing.B) {
	baseDate := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)

	params := domain.ComplexSearchParams{
		Origin: "MOW",
		Stops: []domain.ComplexTripStop{
			{
				City:         "SHA",
				MinDays:      1,
				MaxDays:      5,
				MaxTransfers: 0,
			},
			{
				City:         "TYO",
				MinDays:      1,
				MaxDays:      5,
				MaxTransfers: 0,
			},
		},
		FinalDestination:  "MOW",
		FinalMaxTransfers: 0,
		DepartureDateFrom: baseDate,
		DepartureDateTo:   baseDate.AddDate(0, 0, 999),
	}

	routes := [][2]string{
		{"MOW", "SHA"},
		{"SHA", "TYO"},
		{"TYO", "MOW"},
	}

	var flights []*domain.ComplexSearchFlight

	for _, route := range routes {
		for i := 0; i < 1000; i++ {
			flights = append(flights, &domain.ComplexSearchFlight{
				Origin:      route[0],
				Destination: route[1],
				DepartureAt: baseDate.AddDate(0, 0, i),
				Price:       int64(10000 + i*10),
				Currency:    "RUB",
				Transfers:   0,
				Duration:    60,
			})
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		combineFlights(params, flights)
	}
}
