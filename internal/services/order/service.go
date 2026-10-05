package order

import (
	models "checkout-api/internal/domain"
	cartrepo "checkout-api/internal/repository/cart"
	itemrepo "checkout-api/internal/repository/item"
	orderrepo "checkout-api/internal/repository/order"
	"checkout-api/internal/services/balance"
	"checkout-api/internal/services/cart"
	"checkout-api/internal/services/item"
	"context"
	"errors"
	"fmt"
	"time"
)

//go:generate mockgen -source=service.go -destination=mocks/mock_order_store.go -package=mocks
type OrderStore interface {
	CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error)
	GetOrderByID(ctx context.Context, id int) (*models.Order, error)
	GetOrdersByUserID(ctx context.Context,
		userID int,
		limit int,
		offset int,
		cursor string) ([]*models.Order, error)
	UpdateOrderStatus(ctx context.Context, id int, status string) error
	DecrementStock(ctx context.Context, itemID int, quantity int) error
}

type IdempotencyRecord struct {
	Response   []byte
	StatusCode int
	Expiry     time.Time
}

type balanceService interface {
	GetBalance(ctx context.Context, userID int) (int64, error)
	DebitForPurchase(ctx context.Context, userID int, amount int64) (int64, error)
}

type OrderService struct {
	orderStore       OrderStore
	idempotencyCache map[string]*IdempotencyRecord
	itemStore        item.ItemStore
	cartStore        cart.CartStore
	balanceService   balanceService
}

func NewOrderService(
	orderStore OrderStore,
	itemStore item.ItemStore,
	cartStore cart.CartStore,
	balanceSvc ...balanceService,
) *OrderService {
	var bs balanceService
	if len(balanceSvc) > 0 {
		bs = balanceSvc[0]
	}
	return &OrderService{
		orderStore:       orderStore,
		itemStore:        itemStore,
		cartStore:        cartStore,
		balanceService:   bs,
		idempotencyCache: make(map[string]*IdempotencyRecord),
	}
}

func (s *OrderService) GetOrderByID(ctx context.Context, id int) (*models.Order, error) {
	order, err := s.orderStore.GetOrderByID(ctx, id)
	if err != nil {
		if errors.Is(err, orderrepo.ErrOrderNotFound) {
			return nil, fmt.Errorf("get order by id: %w", ErrItemNotFound)
		}
		return nil, fmt.Errorf("get order by id: %w", err)
	}
	return order, nil
}

func (s *OrderService) GetOrdersByUserID(ctx context.Context, userID int, limit int, offset int, cursor string) ([]*models.Order, error) {

	orders, err := s.orderStore.GetOrdersByUserID(
		ctx,
		userID,
		limit,
		offset,
		cursor,
	)

	if err != nil {
		if errors.Is(err, orderrepo.ErrOrderNotFound) {
			return nil, fmt.Errorf(
				"get orders by user id: %w",
				ErrOrderNotFound,
			)
		}

		return nil, fmt.Errorf(
			"get orders by user id: %w",
			err,
		)
	}

	return orders, nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, id int, status string) error {
	if err := s.orderStore.UpdateOrderStatus(ctx, id, status); err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	return nil
}

func (s *OrderService) createOrderInternal(ctx context.Context, req models.CreateOrderRequest, lineItems []models.LineItem, total int) (*models.Order, models.PaymentResult, error) {
	paymentResult := s.MockProcessPayment(total)
	status := "paid"
	if !paymentResult.Success {
		status = "failed"
	}

	order, err := s.orderStore.CreateOrder(ctx, req.UserID, lineItems, total, status)
	if err != nil {
		if err != nil {
			return nil, paymentResult, fmt.Errorf(
				"create order internal: %w",
				err,
			)
		}
	}
	return order, paymentResult, nil
}

func (s *OrderService) CreateOrder(ctx context.Context, req models.CreateOrderRequest) (*models.Order, models.PaymentResult, error) {
	total := 0
	lineItems := make([]models.LineItem, 0, len(req.Items))
	for _, item := range req.Items {
		storeItem, err := s.itemStore.GetItemByID(ctx, item.ItemID)
		if err != nil {
			if errors.Is(err, orderrepo.ErrItemNotFound) {
				return nil, models.PaymentResult{}, fmt.Errorf("create order: get item %d: %w", item.ItemID, ErrItemNotFound)
			}
			return nil, models.PaymentResult{}, fmt.Errorf("create order: get item %d: %w", item.ItemID, err)
		}
		if storeItem == nil {
			return nil, models.PaymentResult{}, fmt.Errorf("create order: get item %d: %w", item.ItemID, ErrItemNotFound)
		}

		if storeItem.Stock < item.Quantity {
			return nil, models.PaymentResult{}, fmt.Errorf("create order: item %d: %w", item.ItemID, ErrInsufficientStock)
		}
		total += storeItem.Price * item.Quantity
		lineItems = append(lineItems, models.LineItem{
			ItemID:   item.ItemID,
			Name:     storeItem.Name,
			Quantity: item.Quantity,
			Price:    storeItem.Price,
		})
	}

	order, paymentResult, err := s.createOrderInternal(ctx, req, lineItems, total)
	if err != nil {
		return nil, paymentResult, fmt.Errorf("create order: %w", err)
	}

	if paymentResult.Success {
		if s.balanceService != nil {
			if _, err := s.balanceService.DebitForPurchase(ctx, req.UserID, int64(total)); err != nil {
				if errors.Is(err, balance.ErrInsufficientBalance) {
					return nil, paymentResult, fmt.Errorf("create order: %w", ErrInsufficientBalance)
				}
				return nil, paymentResult, fmt.Errorf("create order: debit balance: %w", err)
			}
		}
		for _, item := range lineItems {
			if err := s.orderStore.DecrementStock(ctx, item.ItemID, item.Quantity); err != nil {
				if errors.Is(err, orderrepo.ErrInsufficientStock) {
					return nil, paymentResult, fmt.Errorf(
						"create order: decrement stock for item %d: %w",
						item.ItemID,
						ErrInsufficientStock,
					)
				}

				if errors.Is(err, orderrepo.ErrItemNotFound) {
					return nil, paymentResult, fmt.Errorf(
						"create order: decrement stock for item %d: %w",
						item.ItemID,
						ErrItemNotFound,
					)
				}

				return nil, paymentResult, fmt.Errorf(
					"create order: decrement stock for item %d: %w",
					item.ItemID,
					err,
				)

			}
		}
	}

	return order, paymentResult, nil
}

func (s *OrderService) CreateOrderFromCart(ctx context.Context, req models.CreateOrderFromCartRequest, cart *models.Cart) (*models.Order, models.PaymentResult, error) {
	if cart == nil || len(cart.Items) == 0 {
		return nil, models.PaymentResult{}, fmt.Errorf("create order from cart: %w", ErrCartEmpty)
	}

	total := 0
	for _, item := range cart.Items {
		storeItem, err := s.itemStore.GetItemByID(ctx, item.ItemID)
		if err != nil {
			//
			//			if storeItem.Stock < item.Quantity {
			//				fmt.Printf(
			//					"INSUFFICIENT STOCK: itemID=%d, stock=%d, requested=%d\n",
			//					item.ItemID,
			//					storeItem.Stock,
			//					item.Quantity,
			//				)
			//
			//				return nil, models.PaymentResult{}, fmt.Errorf(
			//					"create order from cart: item %d: %w",
			//					item.ItemID,
			//					ErrInsufficientStock,
			//				)
			//			}
			if errors.Is(err, itemrepo.ErrItemNotFound) {
				return nil, models.PaymentResult{}, fmt.Errorf(
					"create order from cart: get item %d: %w",
					item.ItemID,
					ErrItemNotFound,
				)
			}

			return nil, models.PaymentResult{}, fmt.Errorf(
				"create order from cart: get item %d: %w",
				item.ItemID,
				err,
			)
		}
		if storeItem == nil {
			return nil, models.PaymentResult{}, fmt.Errorf(
				"create order from cart: get item %d: %w",
				item.ItemID,
				ErrItemNotFound,
			)
		}

		if storeItem.Stock < item.Quantity {
			return nil, models.PaymentResult{}, fmt.Errorf(
				"create order from cart: item %d: %w",
				item.ItemID,
				ErrInsufficientStock,
			)
		}

		total += item.Price * item.Quantity
	}

	if s.balanceService != nil {
		if _, err := s.balanceService.DebitForPurchase(ctx, req.UserID, int64(total)); err != nil {
			if errors.Is(err, balance.ErrInsufficientBalance) {
				return nil, models.PaymentResult{}, fmt.Errorf("create order from cart: %w", ErrInsufficientBalance)
			}
			return nil, models.PaymentResult{}, fmt.Errorf("create order from cart: debit balance: %w", err)
		}
	}

	order, paymentResult, err := s.createOrderInternal(ctx, models.CreateOrderRequest{UserID: req.UserID}, cart.Items, total)
	if err != nil {
		return nil, paymentResult, fmt.Errorf("create order from cart: %w", err)
	}

	if paymentResult.Success {
		for _, item := range cart.Items {
			if err := s.orderStore.DecrementStock(ctx, item.ItemID, item.Quantity); err != nil {
				fmt.Printf("CRITICAL: Failed to decrement stock for item %d after payment: %v\n", item.ItemID, err)
			}
		}
		if err := s.cartStore.DeleteUserCart(ctx, req.UserID); err != nil {
			if errors.Is(err, cartrepo.ErrCartNotFound) {
				return order, paymentResult, fmt.Errorf(
					"create order from cart: clear user cart: %w",
					cartrepo.ErrCartNotFound,
				)
			}

			return order, paymentResult, fmt.Errorf(
				"create order from cart: clear user cart: %w",
				err,
			)
		}
	}

	return order, paymentResult, nil
}

func (s *OrderService) GetIdempotentResponse(key string) (*IdempotencyRecord, bool) {
	record, exists := s.idempotencyCache[key]
	if !exists {
		return nil, false
	}
	if time.Now().After(record.Expiry) {
		delete(s.idempotencyCache, key)
		return nil, false
	}
	return record, true
}

func (s *OrderService) CacheResponse(key string, response []byte, statusCode int) {
	s.idempotencyCache[key] = &IdempotencyRecord{
		Response:   response,
		StatusCode: statusCode,
		Expiry:     time.Now().Add(24 * time.Hour),
	}
}

func (s *OrderService) MockProcessPayment(amount int) models.PaymentResult {
	if amount > 0 && amount < 1000000 {
		return models.PaymentResult{
			Success:       true,
			TransactionID: fmt.Sprintf("txn_%d", time.Now().UnixNano()),
		}
	}
	return models.PaymentResult{
		Success: false,
		Error:   "Payment declined",
	}
}
