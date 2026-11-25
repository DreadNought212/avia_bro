package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/DreadNought212/avia_bro/internal/domain"
	"github.com/DreadNought212/avia_bro/internal/service"
)

type TripHandler struct {
	service *service.TripService
}

func NewTripHandler(s *service.TripService) *TripHandler {
	return &TripHandler{service: s}
}

type searchRequest struct {
	Origin          string   `json:"origin"`
	Destinations    []string `json:"destinations"`
	ReturnOrigins   []string `json:"return_origins"`
	DepartureFrom   string   `json:"departure_from"`
	DepartureTo     string   `json:"departure_to"`
	MinTripDuration int      `json:"min_trip_duration"`
	MaxTripDuration int      `json:"max_trip_duration"`
	MaxTransfers    int      `json:"max_transfers"`
	Page            int      `json:"page"`
	Limit           int      `json:"limit"`
}

func (h *TripHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req searchRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Origin = strings.ToUpper(strings.TrimSpace(req.Origin))
	for i := range req.Destinations {
		req.Destinations[i] = strings.ToUpper(strings.TrimSpace(req.Destinations[i]))
	}
	for i := range req.ReturnOrigins {
		req.ReturnOrigins[i] = strings.ToUpper(strings.TrimSpace(req.ReturnOrigins[i]))
	}

	from, err := time.Parse("2006-01-02", req.DepartureFrom)
	if err != nil {
		writeError(w, http.StatusBadRequest, "departure_from must be YYYY-MM-DD")
		return
	}

	to, err := time.Parse("2006-01-02", req.DepartureTo)
	if err != nil || to.Before(from) {
		writeError(w, http.StatusBadRequest, "invalid departure_to")
		return
	}

	if req.Origin == "" || len(req.Destinations) == 0 || len(req.ReturnOrigins) == 0 {
		writeError(w, http.StatusBadRequest, "origin, destinations and return_origins are required")
		return
	}

	if req.MinTripDuration < 1 || req.MaxTripDuration < req.MinTripDuration ||
		req.MaxTripDuration > 60 {
		writeError(w, http.StatusBadRequest, "invalid trip duration")
		return
	}

	if req.MaxTransfers < 0 || req.MaxTransfers > 5 {
		writeError(w, http.StatusBadRequest, "max_transfers must be between 0 and 5")
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 100
	}
	if req.Limit < 1 || req.Limit > 100 {
		writeError(w, http.StatusBadRequest, "limit must be between 1 and 100")
		return
	}

	result, err := h.service.Search(r.Context(), domain.SearchParams{
		Origin:          req.Origin,
		Destinations:    unique(req.Destinations),
		ReturnOrigins:   unique(req.ReturnOrigins),
		DepartureFrom:   from,
		DepartureTo:     to,
		MinTripDuration: req.MinTripDuration,
		MaxTripDuration: req.MaxTripDuration,
		MaxTransfers:    req.MaxTransfers,
		Page:            req.Page,
		Limit:           req.Limit,
	})
	if err != nil {
		log.Printf("Trip search error: %v", err)
		writeError(w, http.StatusInternalServerError, "could not search trips")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *TripHandler) Details(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	outboundKey := r.URL.Query().Get("outbound_key")
	inboundKey := r.URL.Query().Get("inbound_key")

	if outboundKey == "" || inboundKey == "" {
		writeError(w, http.StatusBadRequest, "outbound_key and inbound_key are required")
		return
	}

	result, err := h.service.Details(r.Context(), outboundKey, inboundKey)
	if err != nil {
		if errors.Is(err, service.ErrTripNotFound) {
			writeError(w, http.StatusNotFound, "trip not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not get trip details")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func unique(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// Keep strconv imported out of generated API contract changes.
var _ = strconv.Itoa
