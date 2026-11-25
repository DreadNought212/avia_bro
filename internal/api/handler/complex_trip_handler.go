package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/DreadNought212/avia_bro/internal/domain"
	"github.com/DreadNought212/avia_bro/internal/service"
)

type ComplexTripService interface {
	Search(
		ctx context.Context,
		params domain.ComplexSearchParams,
	) (*domain.ComplexSearchSession, error)

	Combine(
		ctx context.Context,
		sessionID int64,
	) ([]service.ComplexTripResult, error)
}

type ComplexTripHandler struct {
	service ComplexTripService
}

func NewComplexTripHandler(
	service ComplexTripService,
) *ComplexTripHandler {
	return &ComplexTripHandler{
		service: service,
	}
}

// POST /api/v1/complex-trips/search
func (h *ComplexTripHandler) Search(
	w http.ResponseWriter,
	r *http.Request,
) {
	var params domain.ComplexSearchParams

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	session, err := h.service.Search(r.Context(), params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error":   err.Error(),
			"session": session,
		})
		return
	}

	writeJSON(w, http.StatusCreated, session)
}

// GET /api/v1/complex-trips/{id}/results
func (h *ComplexTripHandler) Results(
	w http.ResponseWriter,
	r *http.Request,
) {
	parts := strings.Split(
		strings.Trim(r.URL.Path, "/"),
		"/",
	)

	if len(parts) != 5 || parts[4] != "results" {
		http.NotFound(w, r)
		return
	}

	sessionID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || sessionID <= 0 {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	results, err := h.service.Combine(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_id": sessionID,
		"results":    results,
	})
}
