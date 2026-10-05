package cart

import (
	models "checkout-api/internal/domain"
	cartrepo "checkout-api/internal/repository/cart"
	"checkout-api/internal/services/item"
	"context"
	"errors"
	"fmt"
	"time"
)

//go:generate mockgen -source=service.go -destination=mocks/mock_cart_store.go -package=mocks
type CartStore interface {
	GetUserCart(ctx context.Context, userID int) (*models.Cart, error)
	CreateUserCart(ctx context.Context, cart *models.Cart) error
	UpdateCartItem(ctx context.Context, userID int, itemID int, quantity int) (bool, error)
	AddToCart(ctx context.Context, userID int, item models.LineItem) error
	RemoveCartItem(ctx context.Context, userID int, itemID int) (bool, error)
	DeleteUserCart(ctx context.Context, userID int) error
}

type CartService struct {
	cartStore CartStore
	itemStore item.ItemStore
}

func NewCartService(
	cartStore CartStore,
	itemStore item.ItemStore,
) *CartService {
	return &CartService{
		cartStore: cartStore,
		itemStore: itemStore,
	}
}

func (s *CartService) GetCart(ctx context.Context, userID int) (*models.Cart, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("get cart: %w", ErrCartNotFound)
	}

	cart, err := s.cartStore.GetUserCart(ctx, userID)
	if err != nil {
		if errors.Is(err, cartrepo.ErrCartNotFound) {
			return &models.Cart{
				UserID: userID,
				Items:  []models.LineItem{},
			}, nil
		}
		return nil, fmt.Errorf("get cart: %w", err)
	}

	return cart, nil
}

func (s *CartService) CreateCart(ctx context.Context, userID int, items []models.LineItemRequest) (*models.Cart, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("create cart: %w", ErrCartEmpty)
	}

	existingCart, err := s.cartStore.GetUserCart(ctx, userID)
	if err == nil && existingCart != nil && len(existingCart.Items) > 0 {
		return nil, fmt.Errorf("create cart: %w", ErrCartAlreadyExists)
	}

	lineItems := make([]models.LineItem, 0, len(items))
	for _, item := range items {
		storeItem, err := s.itemStore.GetItemByID(ctx, item.ItemID)
		if err != nil {
			if errors.Is(err, cartrepo.ErrCartItemNotFound) {
				return nil, fmt.Errorf("create cart: get item %d: %w", item.ItemID, err)
			}
			return nil, fmt.Errorf("create cart: get item %d: %w", item.ItemID, err)
		}
		if storeItem == nil {
			return nil, fmt.Errorf("create cart: get item %d: %w", item.ItemID, ErrItemNotFound)
		}

		lineItems = append(lineItems, models.LineItem{
			ItemID:   item.ItemID,
			Name:     storeItem.Name,
			Quantity: item.Quantity,
			Price:    storeItem.Price,
		})
	}

	userCart := &models.Cart{
		ID:     fmt.Sprintf("cart_%d", time.Now().UnixNano()),
		UserID: userID,
		Items:  lineItems,
	}

	if err := s.cartStore.CreateUserCart(ctx, userCart); err != nil {
		return nil, fmt.Errorf("create cart: %w", err)
	}
	return userCart, nil
}

func (s *CartService) AddItem(ctx context.Context, req models.AddCartItemRequest) (*models.Cart, error) {
	if req.Quantity <= 0 {
		return nil, fmt.Errorf("add item to cart: %w", ErrInvalidQuantity)
	}
	storeItem, err := s.itemStore.GetItemByID(ctx, req.ItemID)
	if err != nil {
		if errors.Is(err, ErrItemNotFound) {
			return nil, fmt.Errorf("add item to cart: %w", ErrItemNotFound)
		}
		return nil, fmt.Errorf("add item to cart: get item %d: %w", req.ItemID, err)
	}
	if storeItem == nil {
		return nil, fmt.Errorf("add item to cart: %w", ErrItemNotFound)
	}

	if storeItem.Stock < req.Quantity {
		return nil, fmt.Errorf("add item to cart: %w", ErrInsufficientStock)
	}

	lineItem := models.LineItem{
		ItemID:   req.ItemID,
		Name:     storeItem.Name,
		Quantity: req.Quantity,
		Price:    storeItem.Price,
	}

	if err := s.cartStore.AddToCart(ctx, req.UserID, lineItem); err != nil {
		return nil, fmt.Errorf("add item to cart: %w", err)
	}

	cart, err := s.cartStore.GetUserCart(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("add item to cart: get updated cart: %w", err)
	}
	return cart, nil
}

func (s *CartService) UpdateItem(ctx context.Context, req models.UpdateCartItemRequest, itemID int) (*models.Cart, error) {
	if req.Quantity <= 0 {
		return nil, fmt.Errorf("update cart item: %w", ErrInvalidQuantity)
	}
	success, err := s.cartStore.UpdateCartItem(ctx, req.UserID, itemID, req.Quantity)
	if err != nil {
		if errors.Is(err, cartrepo.ErrCartNotFound) {
			return nil, fmt.Errorf("update cart item: %w", ErrCartNotFound)
		}
		return nil, fmt.Errorf("update cart item: %w", err)
	}
	if !success {
		return nil, fmt.Errorf("update cart item: %w", ErrCartItemNotFound)
	}

	cart, err := s.cartStore.GetUserCart(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("update cart item: get updated cart: %w", err)
	}
	return cart, nil
}

func (s *CartService) RemoveItem(ctx context.Context, req models.RemoveCartItemRequest, itemID int) (*models.Cart, error) {
	success, err := s.cartStore.RemoveCartItem(ctx, req.UserID, itemID)
	if err != nil {
		if errors.Is(err, cartrepo.ErrCartNotFound) {
			return nil, fmt.Errorf("remove cart item: %w", ErrCartNotFound)
		}
		return nil, fmt.Errorf("remove cart item: %w", err)
	}
	if !success {
		return nil, fmt.Errorf("remove cart item: %w", ErrCartItemNotFound)
	}

	cart, err := s.cartStore.GetUserCart(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, cartrepo.ErrCartNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("remove cart item: get updated cart: %w", err)
	}
	if cart != nil && len(cart.Items) == 0 {
		if err := s.cartStore.DeleteUserCart(ctx, req.UserID); err != nil {
			return nil, fmt.Errorf("remove cart item: delete empty cart: %w", err)
		}
		return nil, nil
	}
	return cart, nil
}

func (s *CartService) ClearCart(ctx context.Context, userID int) error {
	if userID <= 0 {
		return fmt.Errorf("clear cart: %w", ErrCartNotFound)
	}

	_, err := s.cartStore.GetUserCart(ctx, userID)
	if err != nil {
		if errors.Is(err, cartrepo.ErrCartNotFound) {
			return fmt.Errorf("clear cart: %w", ErrCartNotFound)
		}
		return fmt.Errorf("clear cart: %w", err)
	}

	if err := s.cartStore.DeleteUserCart(ctx, userID); err != nil {
		return fmt.Errorf("clear cart: %w", err)
	}
	return nil
}
