package inventory

import (
	"checkout-api/internal/domain"
	"context"
	"fmt"
)

type Store interface {
	GetInventory(ctx context.Context, userID int) ([]*domain.InventoryItem, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) GetInventory(ctx context.Context, userID int) ([]*domain.InventoryItem, error) {
	items, err := s.store.GetInventory(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get inventory: %w", err)
	}
	return items, nil
}
