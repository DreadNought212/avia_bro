package service

import (
	"context"
	"errors"

	"github.com/DreadNought212/avia_bro/internal/domain"
)

var ErrTripNotFound = errors.New("trip not found")

type SearchResult struct {
	Items []domain.TripOption `json:"items"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
}

type TripRepository interface {
	Search(ctx context.Context, params domain.SearchParams) ([]domain.TripOption, error)
	Details(ctx context.Context, outboundKey, inboundKey string) (*domain.TripOption, error)
}

type TripService struct {
	repository TripRepository
}

func NewTripService(repository TripRepository) *TripService {
	return &TripService{repository: repository}
}

func (s *TripService) Search(
	ctx context.Context,
	params domain.SearchParams,
) (*SearchResult, error) {
	items, err := s.repository.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	return &SearchResult{
		Items: items,
		Page:  params.Page,
		Limit: params.Limit,
	}, nil
}

func (s *TripService) Details(
	ctx context.Context,
	outboundKey, inboundKey string,
) (*domain.TripOption, error) {
	result, err := s.repository.Details(ctx, outboundKey, inboundKey)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, ErrTripNotFound
	}

	return result, nil
}
