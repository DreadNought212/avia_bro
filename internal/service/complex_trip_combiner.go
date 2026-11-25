package service

import (
	"container/heap"
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/DreadNought212/avia_bro/internal/domain"
)

// ComplexTripResult — один полностью собранный маршрут.
type ComplexTripResult struct {
	Flights     []*domain.ComplexSearchFlight `json:"flights"`
	TotalPrice  int64                         `json:"total_price"`
	Currency    string                        `json:"currency"`
	DepartureAt time.Time                     `json:"departure_at"`
	ArrivalAt   time.Time                     `json:"arrival_at"`
}

// Combine получает билеты одной сессии и строит допустимые маршруты.
func (s *ComplexTripService) Combine(
	ctx context.Context,
	sessionID int64,
) ([]ComplexTripResult, error) {
	session, err := s.repository.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get search session: %w", err)
	}

	if session.Status != domain.ComplexSearchStatusCompleted {
		return nil, fmt.Errorf(
			"search session %d is not completed",
			sessionID,
		)
	}

	flights, err := s.repository.GetFlightsBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session flights: %w", err)
	}

	return combineFlights(session.SearchParams, flights), nil
}

const maxComplexTripResults = 1000

type complexTripResultMaxHeap []ComplexTripResult

func (h complexTripResultMaxHeap) Len() int {
	return len(h)
}

func (h complexTripResultMaxHeap) Less(i, j int) bool {
	return h[i].TotalPrice > h[j].TotalPrice
}

func (h complexTripResultMaxHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *complexTripResultMaxHeap) Push(value any) {
	*h = append(*h, value.(ComplexTripResult))
}

func (h *complexTripResultMaxHeap) Pop() any {
	old := *h
	n := len(old)

	value := old[n-1]
	*h = old[:n-1]

	return value
}

func combineFlights(
	params domain.ComplexSearchParams,
	flights []*domain.ComplexSearchFlight,
) []ComplexTripResult {
	cities := make([]string, 0, len(params.Stops)+2)
	cities = append(cities, params.Origin)

	for _, stop := range params.Stops {
		cities = append(cities, stop.City)
	}

	cities = append(cities, params.FinalDestination)

	// Группируем билеты по направлениям.
	flightsByRoute := make(map[string][]*domain.ComplexSearchFlight)

	for _, flight := range flights {
		if flight == nil {
			continue
		}

		key := routeKey(flight.Origin, flight.Destination)
		flightsByRoute[key] = append(flightsByRoute[key], flight)
	}

	// Сортируем билеты по времени вылета.
	for key := range flightsByRoute {
		sort.Slice(flightsByRoute[key], func(i, j int) bool {
			return flightsByRoute[key][i].DepartureAt.Before(
				flightsByRoute[key][j].DepartureAt,
			)
		})
	}

	segmentCount := len(cities) - 1

	// Вычисляем минимальную стоимость билета для каждого сегмента.
	// Это нижняя граница стоимости оставшейся части маршрута.
	minRemainingCost := make([]int64, segmentCount+1)

	for i := segmentCount - 1; i >= 0; i-- {
		candidates := flightsByRoute[routeKey(cities[i], cities[i+1])]

		if len(candidates) == 0 {
			return nil
		}

		minPrice := candidates[0].Price

		for _, candidate := range candidates {
			if candidate.Price < minPrice {
				minPrice = candidate.Price
			}
		}

		minRemainingCost[i] = minPrice + minRemainingCost[i+1]
	}

	// Храним только maxComplexTripResults самых дешёвых маршрутов.
	results := make(complexTripResultMaxHeap, 0, maxComplexTripResults)
	heap.Init(&results)

	var selected []*domain.ComplexSearchFlight

	var search func(segmentIndex int, totalPrice int64)

	search = func(segmentIndex int, totalPrice int64) {
		// Если даже минимально возможная стоимость оставшихся
		// билетов не позволит улучшить результат, прекращаем ветку.
		if len(results) == maxComplexTripResults &&
			totalPrice+minRemainingCost[segmentIndex] >
				results[0].TotalPrice {
			return
		}

		// Все сегменты успешно соединены.
		if segmentIndex == segmentCount {
			first := selected[0]
			last := selected[len(selected)-1]

			arrivalAt, ok := flightArrivalTime(last)
			if !ok {
				return
			}

			result := ComplexTripResult{
				Flights:     append([]*domain.ComplexSearchFlight(nil), selected...),
				TotalPrice:  totalPrice,
				Currency:    first.Currency,
				DepartureAt: first.DepartureAt,
				ArrivalAt:   arrivalAt,
			}

			if len(results) < maxComplexTripResults {
				heap.Push(&results, result)
			} else if totalPrice < results[0].TotalPrice {
				heap.Pop(&results)
				heap.Push(&results, result)
			}

			return
		}

		origin := cities[segmentIndex]
		destination := cities[segmentIndex+1]

		candidates := flightsByRoute[routeKey(origin, destination)]

		for _, candidate := range candidates {
			// Проверяем максимальное количество пересадок
			// для текущего сегмента маршрута.
			maxTransfers := params.FinalMaxTransfers

			if segmentIndex < len(params.Stops) {
				maxTransfers = params.Stops[segmentIndex].MaxTransfers
			}

			if candidate.Transfers > maxTransfers {
				continue
			}
			// Первый перелёт должен попадать в заданный диапазон.
			if segmentIndex == 0 {
				if candidate.DepartureAt.Before(params.DepartureDateFrom) ||
					candidate.DepartureAt.After(params.DepartureDateTo) {
					continue
				}
			}

			if len(selected) > 0 {
				previous := selected[len(selected)-1]

				previousArrival, ok := flightArrivalTime(previous)
				if !ok {
					continue
				}

				// Следующий рейс должен вылетать строго после прилёта.
				if !candidate.DepartureAt.After(previousArrival) {
					continue
				}

				// Проверяем продолжительность пребывания
				// в промежуточном городе.
				stop := params.Stops[segmentIndex-1]

				days, ok := calendarDayDifference(
					previousArrival,
					candidate.DepartureAt,
					stop.City,
				)

				if !ok {
					continue
				}

				if days < stop.MinDays || days > stop.MaxDays {
					continue
				}

				// Все билеты должны быть в одной валюте.
				if candidate.Currency != selected[0].Currency {
					continue
				}
			}

			selected = append(selected, candidate)

			search(
				segmentIndex+1,
				totalPrice+candidate.Price,
			)

			selected = selected[:len(selected)-1]
		}
	}

	search(0, 0)

	// Сортируем оставшиеся результаты по возрастанию цены.
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalPrice < results[j].TotalPrice
	})

	return results
}

func routeKey(origin, destination string) string {
	return origin + "|" + destination
}

// Вычисляет время прилёта на основе времени вылета
// и продолжительности рейса в минутах.
func flightArrivalTime(
	flight *domain.ComplexSearchFlight,
) (time.Time, bool) {
	if flight == nil || flight.Duration <= 0 {
		return time.Time{}, false
	}

	return flight.DepartureAt.Add(
		time.Duration(flight.Duration) * time.Minute,
	), true
}

// Разница между календарными датами, а не полными сутками.
// Разница между календарными датами в часовом поясе города пребывания.
func calendarDayDifference(
	from, to time.Time,
	city string,
) (int, bool) {
	timeZone, ok := cityTimeZone(city)
	if !ok {
		return 0, false
	}

	location, err := time.LoadLocation(timeZone)
	if err != nil {
		return 0, false
	}

	from = from.In(location)
	to = to.In(location)

	fromDate := time.Date(
		from.Year(), from.Month(), from.Day(),
		0, 0, 0, 0, time.UTC,
	)

	toDate := time.Date(
		to.Year(), to.Month(), to.Day(),
		0, 0, 0, 0, time.UTC,
	)

	return int(toDate.Sub(fromDate).Hours() / 24), true
}

// Часовые пояса поддерживаемых городов.
func cityTimeZone(city string) (string, bool) {
	timeZones := map[string]string{
		// Россия
		"MOW": "Europe/Moscow",
		"LED": "Europe/Moscow",
		"SVX": "Asia/Yekaterinburg",
		"OVB": "Asia/Novosibirsk",
		"KJA": "Asia/Krasnoyarsk",
		"IKT": "Asia/Irkutsk",
		"VVO": "Asia/Vladivostok",
		"KHV": "Asia/Vladivostok",
		"KGD": "Europe/Kaliningrad",
		"AER": "Europe/Moscow",
		"KZN": "Europe/Moscow",

		// Китай
		"SHA": "Asia/Shanghai",
		"PVG": "Asia/Shanghai",
		"PEK": "Asia/Shanghai",
		"PKX": "Asia/Shanghai",
		"CAN": "Asia/Shanghai",
		"SZX": "Asia/Shanghai",
		"CTU": "Asia/Shanghai",
		"CKG": "Asia/Shanghai",
		"HGH": "Asia/Shanghai",
		"NKG": "Asia/Shanghai",
		"XIY": "Asia/Shanghai",
		"WUH": "Asia/Shanghai",
		"XMN": "Asia/Shanghai",
		"CSX": "Asia/Shanghai",
		"KMG": "Asia/Shanghai",
		"SHE": "Asia/Shanghai",
		"HRB": "Asia/Shanghai",
		"TSN": "Asia/Shanghai",
		"TAO": "Asia/Shanghai",
		"FOC": "Asia/Shanghai",
		"SYX": "Asia/Shanghai",

		// Япония
		"TYO": "Asia/Tokyo",
		"NRT": "Asia/Tokyo",
		"HND": "Asia/Tokyo",
		"OSA": "Asia/Tokyo",
		"KIX": "Asia/Tokyo",
		"ITM": "Asia/Tokyo",
		"NGO": "Asia/Tokyo",
		"FUK": "Asia/Tokyo",
		"CTS": "Asia/Tokyo",
		"OKA": "Asia/Tokyo",
		"HIJ": "Asia/Tokyo",
		"KMQ": "Asia/Tokyo",

		// Южная Корея
		"SEL": "Asia/Seoul",
		"ICN": "Asia/Seoul",
		"GMP": "Asia/Seoul",
		"PUS": "Asia/Seoul",
		"CJU": "Asia/Seoul",
		"TAE": "Asia/Seoul",

		// Тайвань
		"TPE": "Asia/Taipei",
		"KHH": "Asia/Taipei",
		"RMQ": "Asia/Taipei",

		// Юго-Восточная Азия
		"KUL": "Asia/Kuala_Lumpur",
		"PEN": "Asia/Kuala_Lumpur",
		"LGK": "Asia/Kuala_Lumpur",
		"JHB": "Asia/Kuala_Lumpur",
		"BKK": "Asia/Bangkok",
		"DMK": "Asia/Bangkok",
		"HKT": "Asia/Bangkok",
		"CNX": "Asia/Bangkok",
		"USM": "Asia/Bangkok",
		"SGN": "Asia/Ho_Chi_Minh",
		"HAN": "Asia/Ho_Chi_Minh",
		"DAD": "Asia/Ho_Chi_Minh",
		"PQC": "Asia/Ho_Chi_Minh",
		"SIN": "Asia/Singapore",
		"MNL": "Asia/Manila",
		"CEB": "Asia/Manila",
		"CGK": "Asia/Jakarta",
		"DPS": "Asia/Makassar",
		"JOG": "Asia/Jakarta",
		"PNH": "Asia/Phnom_Penh",
		"REP": "Asia/Phnom_Penh",
		"VTE": "Asia/Vientiane",
		"RGN": "Asia/Yangon",

		// Ближний Восток
		"DXB": "Asia/Dubai",
		"AUH": "Asia/Dubai",
		"SHJ": "Asia/Dubai",
		"DOH": "Asia/Qatar",
		"RUH": "Asia/Riyadh",
		"JED": "Asia/Riyadh",
		"TLV": "Asia/Jerusalem",
		"AMM": "Asia/Amman",
		"IST": "Europe/Istanbul",
		"SAW": "Europe/Istanbul",
		"AYT": "Europe/Istanbul",

		// Европа
		"LHR": "Europe/London",
		"LGW": "Europe/London",
		"CDG": "Europe/Paris",
		"ORY": "Europe/Paris",
		"FRA": "Europe/Berlin",
		"MUC": "Europe/Berlin",
		"BER": "Europe/Berlin",
		"AMS": "Europe/Amsterdam",
		"BRU": "Europe/Brussels",
		"ZRH": "Europe/Zurich",
		"GVA": "Europe/Zurich",
		"FCO": "Europe/Rome",
		"MXP": "Europe/Rome",
		"BCN": "Europe/Madrid",
		"MAD": "Europe/Madrid",
		"LIS": "Europe/Lisbon",
		"ATH": "Europe/Athens",
		"PRG": "Europe/Prague",
		"WAW": "Europe/Warsaw",
		"BUD": "Europe/Budapest",
		"VIE": "Europe/Vienna",
		"CPH": "Europe/Copenhagen",
		"OSL": "Europe/Oslo",
		"ARN": "Europe/Stockholm",
		"HEL": "Europe/Helsinki",
		"RIX": "Europe/Riga",
		"VNO": "Europe/Vilnius",
		"TLL": "Europe/Tallinn",
		"BEG": "Europe/Belgrade",
		"OTP": "Europe/Bucharest",
		"SOF": "Europe/Sofia",
		"ZAG": "Europe/Zagreb",
		"SJJ": "Europe/Sarajevo",
		"TBS": "Asia/Tbilisi",
		"EVN": "Asia/Yerevan",
		"GYD": "Asia/Baku",
		"ALA": "Asia/Almaty",
		"NQZ": "Asia/Almaty",
		"TAS": "Asia/Tashkent",
		"FRU": "Asia/Bishkek",

		// Северная Америка
		"JFK": "America/New_York",
		"EWR": "America/New_York",
		"LAX": "America/Los_Angeles",
		"SFO": "America/Los_Angeles",
		"ORD": "America/Chicago",
		"MIA": "America/New_York",
		"YYZ": "America/Toronto",
		"YVR": "America/Vancouver",

		// Южная Америка
		"GRU": "America/Sao_Paulo",
		"GIG": "America/Sao_Paulo",
		"EZE": "America/Argentina/Buenos_Aires",
		"SCL": "America/Santiago",
		"LIM": "America/Lima",
		"BOG": "America/Bogota",

		// Австралия и Океания
		"SYD": "Australia/Sydney",
		"MEL": "Australia/Melbourne",
		"BNE": "Australia/Brisbane",
		"PER": "Australia/Perth",
		"AKL": "Pacific/Auckland",
		"NAN": "Pacific/Fiji",
	}

	timeZone, ok := timeZones[city]
	return timeZone, ok
}
