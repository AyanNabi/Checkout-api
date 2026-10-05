package item

import (
	"checkout-api/internal/domain"
	filter "checkout-api/internal/helper/filter"
	itemrepo "checkout-api/internal/repository/item"
	"context"
	"errors"
	"fmt"
)

//go:generate mockgen -source=service.go -destination=mocks/mock_item_store.go -package=mocks
type ItemStore interface {
	GetItems(ctx context.Context, query filter.Request) ([]*domain.Item, error)
	GetItemByID(ctx context.Context, id int) (*domain.Item, error)
}

type ItemService struct {
	store ItemStore
}

func NewItemService(s ItemStore) *ItemService {
	return &ItemService{store: s}
}

func (s *ItemService) GetAllItems(
	ctx context.Context,
	request filter.Request,
) ([]*domain.Item, error) {

	items, err := s.store.GetItems(
		ctx,
		request,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"get all items: %w",
			err,
		)
	}

	return items, nil
}

func (s *ItemService) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	if id <= 0 {
		return nil, fmt.Errorf("get item by id: %w", ErrItemNotFound)
	}

	item, err := s.store.GetItemByID(ctx, id)
	if err != nil {
		if errors.Is(err, itemrepo.ErrItemNotFound) {
			return nil, fmt.Errorf("get item by id: %w", ErrItemNotFound)
		}
		return nil, fmt.Errorf("get item by id: %w", err)
	}

	return item, nil
}
