package order

import (
	"context"
	"errors"
	"testing"
	"time"

	models "checkout-api/internal/domain"
	cartrepo "checkout-api/internal/repository/cart"
	itemrepo "checkout-api/internal/repository/item"
	orderrepo "checkout-api/internal/repository/order"

	cartmocks "checkout-api/internal/services/cart/mocks"
	itemmocks "checkout-api/internal/services/item/mocks"
	ordermocks "checkout-api/internal/services/order/mocks"

	"go.uber.org/mock/gomock"
)

var (
	errDatabaseFailure      = errors.New("database failure")
	errItemStoreFailure     = errors.New("item store failure")
	errOrderDatabaseFailure = errors.New("order database failure")
	errStockDatabaseFailure = errors.New("stock database failure")
	errItemDatabaseFailure  = errors.New("item database failure")
	errCreateOrderDatabase  = errors.New("create order database failure")
	errDeleteCartDatabase   = errors.New("delete cart database failure")
)

func TestNewOrderService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	os := ordermocks.NewMockOrderStore(ctrl)
	is := itemmocks.NewMockItemStore(ctrl)
	cs := cartmocks.NewMockCartStore(ctrl)

	svc := NewOrderService(os, is, cs)

	if svc == nil {
		t.Fatal("expected service, got nil")
	}

	if svc.orderStore != os {
		t.Fatal("expected order store to be assigned")
	}

	if svc.itemStore != is {
		t.Fatal("expected item store to be assigned")
	}

	if svc.cartStore != cs {
		t.Fatal("expected cart store to be assigned")
	}

	if svc.idempotencyCache == nil {
		t.Fatal("expected idempotency cache to be initialized")
	}
}

func TestOrderService_GetOrderByID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		orderID int
		wantErr error
		setup   func(*ordermocks.MockOrderStore)
	}{
		{
			name:    "success",
			orderID: 1,
			setup: func(os *ordermocks.MockOrderStore) {
				expectedOrder := &models.Order{
					ID:     1,
					UserID: 10,
					Total:  500,
					Status: "paid",
				}

				os.EXPECT().
					GetOrderByID(ctx, 1).
					Return(expectedOrder, nil).
					Times(1)
			},
		},
		{
			name:    "order not found",
			orderID: 999,
			wantErr: ErrItemNotFound,
			setup: func(os *ordermocks.MockOrderStore) {
				os.EXPECT().
					GetOrderByID(ctx, 999).
					Return(nil, orderrepo.ErrOrderNotFound).
					Times(1)
			},
		},
		{
			name:    "store error",
			orderID: 1,
			wantErr: errDatabaseFailure,
			setup: func(os *ordermocks.MockOrderStore) {
				os.EXPECT().
					GetOrderByID(ctx, 1).
					Return(nil, errDatabaseFailure).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			os := ordermocks.NewMockOrderStore(ctrl)
			is := itemmocks.NewMockItemStore(ctrl)
			cs := cartmocks.NewMockCartStore(ctrl)

			tt.setup(os)

			svc := NewOrderService(os, is, cs)

			got, err := svc.GetOrderByID(ctx, tt.orderID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"expected %v, got %v",
						tt.wantErr,
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got == nil {
				t.Fatal("expected order, got nil")
			}

			if got.ID != 1 {
				t.Fatalf("expected ID 1, got %d", got.ID)
			}
		})
	}
}

func TestOrderService_GetOrdersByUserID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		want    int
		wantErr error
		setup   func(*ordermocks.MockOrderStore)
	}{
		{
			name: "success",
			want: 2,
			setup: func(os *ordermocks.MockOrderStore) {
				expectedOrders := []*models.Order{
					{
						ID:     1,
						UserID: 10,
					},
					{
						ID:     2,
						UserID: 10,
					},
				}

				os.EXPECT().
					GetOrdersByUserID(ctx, 10, 10, 0, "").
					Return(expectedOrders, nil).
					Times(1)
			},
		},
		{
			name:    "order not found",
			wantErr: ErrOrderNotFound,
			setup: func(os *ordermocks.MockOrderStore) {
				os.EXPECT().
					GetOrdersByUserID(ctx, 10, 10, 0, "").
					Return(nil, orderrepo.ErrOrderNotFound).
					Times(1)
			},
		},
		{
			name:    "store error",
			wantErr: errDatabaseFailure,
			setup: func(os *ordermocks.MockOrderStore) {
				os.EXPECT().
					GetOrdersByUserID(ctx, 10, 10, 0, "").
					Return(nil, errDatabaseFailure).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			os := ordermocks.NewMockOrderStore(ctrl)
			is := itemmocks.NewMockItemStore(ctrl)
			cs := cartmocks.NewMockCartStore(ctrl)

			tt.setup(os)

			svc := NewOrderService(os, is, cs)

			orders, err := svc.GetOrdersByUserID(
				ctx,
				10,
				10,
				0,
				"",
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"expected %v, got %v",
						tt.wantErr,
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(orders) != tt.want {
				t.Fatalf(
					"expected %d orders, got %d",
					tt.want,
					len(orders),
				)
			}
		})
	}
}

func TestOrderService_UpdateOrderStatus(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		wantErr error
		setup   func(*ordermocks.MockOrderStore)
	}{
		{
			name: "success",
			setup: func(os *ordermocks.MockOrderStore) {
				os.EXPECT().
					UpdateOrderStatus(ctx, 1, "paid").
					Return(nil).
					Times(1)
			},
		},
		{
			name:    "store error",
			wantErr: errDatabaseFailure,
			setup: func(os *ordermocks.MockOrderStore) {
				os.EXPECT().
					UpdateOrderStatus(ctx, 1, "paid").
					Return(errDatabaseFailure).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			os := ordermocks.NewMockOrderStore(ctrl)
			is := itemmocks.NewMockItemStore(ctrl)
			cs := cartmocks.NewMockCartStore(ctrl)

			tt.setup(os)

			svc := NewOrderService(os, is, cs)

			err := svc.UpdateOrderStatus(
				ctx,
				1,
				"paid",
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"expected %v, got %v",
						tt.wantErr,
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestOrderService_CreateOrder(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		items   map[int]*models.Item
		itemErr map[int]error

		returnNil bool

		createOrderErr error
		decrementErr   error

		req models.CreateOrderRequest

		wantErr error

		wantTotal  int
		wantStatus string

		wantDecrement map[int]int
	}{
		{
			name: "zero items",

			items: map[int]*models.Item{},

			req: models.CreateOrderRequest{
				UserID: 1,
				Items:  []models.LineItemRequest{},
			},

			wantTotal:  0,
			wantStatus: "failed",
		},

		{
			name: "one item",

			items: map[int]*models.Item{
				10: {
					ID:    10,
					Name:  "Book",
					Price: 500,
					Stock: 10,
				},
			},

			req: models.CreateOrderRequest{
				UserID: 1,
				Items: []models.LineItemRequest{
					{
						ItemID:   10,
						Quantity: 1,
					},
				},
			},

			wantTotal:  500,
			wantStatus: "paid",

			wantDecrement: map[int]int{
				10: 1,
			},
		},

		{
			name: "many items",

			items: map[int]*models.Item{
				10: {
					ID:    10,
					Name:  "Book",
					Price: 500,
					Stock: 10,
				},
				20: {
					ID:    20,
					Name:  "Pen",
					Price: 200,
					Stock: 10,
				},
			},

			req: models.CreateOrderRequest{
				UserID: 1,
				Items: []models.LineItemRequest{
					{
						ItemID:   10,
						Quantity: 2,
					},
					{
						ItemID:   20,
						Quantity: 3,
					},
				},
			},

			wantTotal:  1600,
			wantStatus: "paid",

			wantDecrement: map[int]int{
				10: 2,
				20: 3,
			},
		},

		{
			name: "boundary quantity equals stock",

			items: map[int]*models.Item{
				10: {
					ID:    10,
					Name:  "Book",
					Price: 500,
					Stock: 2,
				},
			},

			req: models.CreateOrderRequest{
				UserID: 1,
				Items: []models.LineItemRequest{
					{
						ItemID:   10,
						Quantity: 2,
					},
				},
			},

			wantTotal:  1000,
			wantStatus: "paid",

			wantDecrement: map[int]int{
				10: 2,
			},
		},

		{
			name: "boundary quantity exceeds stock",

			items: map[int]*models.Item{
				10: {
					ID:    10,
					Name:  "Book",
					Price: 500,
					Stock: 2,
				},
			},

			req: models.CreateOrderRequest{
				UserID: 1,
				Items: []models.LineItemRequest{
					{
						ItemID:   10,
						Quantity: 3,
					},
				},
			},

			wantErr: ErrInsufficientStock,
		},

		{
			name: "interface item store error",

			itemErr: map[int]error{
				10: errItemStoreFailure,
			},

			req: models.CreateOrderRequest{
				UserID: 1,
				Items: []models.LineItemRequest{
					{
						ItemID:   10,
						Quantity: 1,
					},
				},
			},

			wantErr: errItemStoreFailure,
		},

		{
			name: "item not found",

			itemErr: map[int]error{
				10: itemrepo.ErrItemNotFound,
			},

			req: models.CreateOrderRequest{
				UserID: 1,
				Items: []models.LineItemRequest{
					{
						ItemID:   10,
						Quantity: 1,
					},
				},
			},

			wantErr: itemrepo.ErrItemNotFound,
		},

		{
			name:      "nil item",
			returnNil: true,

			req: models.CreateOrderRequest{
				UserID: 1,
				Items: []models.LineItemRequest{
					{
						ItemID:   10,
						Quantity: 1,
					},
				},
			},

			wantErr: ErrItemNotFound,
		},

		{
			name: "exceptional create order error",

			items: map[int]*models.Item{
				10: {
					ID:    10,
					Name:  "Book",
					Price: 500,
					Stock: 10,
				},
			},

			createOrderErr: errOrderDatabaseFailure,

			req: models.CreateOrderRequest{
				UserID: 1,
				Items: []models.LineItemRequest{
					{
						ItemID:   10,
						Quantity: 1,
					},
				},
			},

			wantErr:    errOrderDatabaseFailure,
			wantTotal:  500,
			wantStatus: "paid",
		},

		{
			name: "decrement stock insufficient",

			items: map[int]*models.Item{
				10: {
					ID:    10,
					Name:  "Book",
					Price: 500,
					Stock: 10,
				},
			},

			decrementErr: orderrepo.ErrInsufficientStock,

			req: models.CreateOrderRequest{
				UserID: 1,
				Items: []models.LineItemRequest{
					{
						ItemID:   10,
						Quantity: 1,
					},
				},
			},

			wantErr:    ErrInsufficientStock,
			wantTotal:  500,
			wantStatus: "paid",

			wantDecrement: map[int]int{
				10: 1,
			},
		},

		{
			name: "decrement stock item not found",

			items: map[int]*models.Item{
				10: {
					ID:    10,
					Name:  "Book",
					Price: 500,
					Stock: 10,
				},
			},

			decrementErr: orderrepo.ErrItemNotFound,

			req: models.CreateOrderRequest{
				UserID: 1,
				Items: []models.LineItemRequest{
					{
						ItemID:   10,
						Quantity: 1,
					},
				},
			},

			wantErr:    ErrItemNotFound,
			wantTotal:  500,
			wantStatus: "paid",

			wantDecrement: map[int]int{
				10: 1,
			},
		},

		{
			name: "decrement stock generic error",

			items: map[int]*models.Item{
				10: {
					ID:    10,
					Name:  "Book",
					Price: 500,
					Stock: 10,
				},
			},

			decrementErr: errStockDatabaseFailure,

			req: models.CreateOrderRequest{
				UserID: 1,
				Items: []models.LineItemRequest{
					{
						ItemID:   10,
						Quantity: 1,
					},
				},
			},

			wantErr:    errStockDatabaseFailure,
			wantTotal:  500,
			wantStatus: "paid",

			wantDecrement: map[int]int{
				10: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			os := ordermocks.NewMockOrderStore(ctrl)
			is := itemmocks.NewMockItemStore(ctrl)
			cs := cartmocks.NewMockCartStore(ctrl)

			svc := NewOrderService(os, is, cs)

			// ---------------------------------------------------------
			// Item store expectations
			// ---------------------------------------------------------

			for itemID, itemErr := range tt.itemErr {
				is.EXPECT().
					GetItemByID(ctx, itemID).
					Return(nil, itemErr).
					Times(1)
			}

			if tt.returnNil {
				is.EXPECT().
					GetItemByID(ctx, 10).
					Return(nil, nil).
					Times(1)
			}

			for itemID, item := range tt.items {
				is.EXPECT().
					GetItemByID(ctx, itemID).
					Return(item, nil).
					Times(1)
			}

			// ---------------------------------------------------------
			// CreateOrder expectations
			// ---------------------------------------------------------

			if tt.createOrderErr != nil {

				// Payment succeeds because total = 500.
				// Therefore service calls:
				//
				// CreateOrder(..., 500, "paid")
				//
				os.EXPECT().
					CreateOrder(
						ctx,
						tt.req.UserID,
						gomock.Any(),
						tt.wantTotal,
						tt.wantStatus,
					).
					Return(nil, tt.createOrderErr).
					Times(1)

			} else if tt.wantErr == nil {

				expectedOrder := &models.Order{
					UserID: tt.req.UserID,
					Total:  tt.wantTotal,
					Status: tt.wantStatus,
				}

				os.EXPECT().
					CreateOrder(
						ctx,
						tt.req.UserID,
						gomock.Any(),
						tt.wantTotal,
						tt.wantStatus,
					).
					Return(expectedOrder, nil).
					Times(1)

			} else if tt.decrementErr != nil {

				// IMPORTANT:
				// Even though DecrementStock will fail later,
				// CreateOrder itself succeeds first.
				//
				// total = 500
				// status = paid
				//
				expectedOrder := &models.Order{
					ID:     1,
					UserID: tt.req.UserID,
					Total:  tt.wantTotal,
					Status: tt.wantStatus,
				}

				os.EXPECT().
					CreateOrder(
						ctx,
						tt.req.UserID,
						gomock.Any(),
						tt.wantTotal,
						tt.wantStatus,
					).
					Return(expectedOrder, nil).
					Times(1)
			}

			// ---------------------------------------------------------
			// Decrement stock expectations
			// ---------------------------------------------------------

			if tt.decrementErr != nil {

				for itemID, quantity := range tt.wantDecrement {
					os.EXPECT().
						DecrementStock(
							ctx,
							itemID,
							quantity,
						).
						Return(tt.decrementErr).
						Times(1)
				}

			} else if tt.wantErr == nil {

				for itemID, quantity := range tt.wantDecrement {
					os.EXPECT().
						DecrementStock(
							ctx,
							itemID,
							quantity,
						).
						Return(nil).
						Times(1)
				}
			}

			// ---------------------------------------------------------
			// Execute
			// ---------------------------------------------------------

			got, payment, err := svc.CreateOrder(
				ctx,
				tt.req,
			)

			// ---------------------------------------------------------
			// Error assertions
			// ---------------------------------------------------------

			if tt.wantErr != nil {

				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"expected %v, got %v",
						tt.wantErr,
						err,
					)
				}

				return
			}

			// ---------------------------------------------------------
			// Success assertions
			// ---------------------------------------------------------

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got == nil {
				t.Fatal("expected order, got nil")
			}

			if got.Total != tt.wantTotal {
				t.Fatalf(
					"expected total %d, got %d",
					tt.wantTotal,
					got.Total,
				)
			}

			if tt.wantStatus != "" && got.Status != tt.wantStatus {
				t.Fatalf(
					"expected status %s, got %s",
					tt.wantStatus,
					got.Status,
				)
			}

			if tt.wantTotal > 0 {

				if !payment.Success {
					t.Fatal("expected payment to succeed")
				}

				if payment.TransactionID == "" {
					t.Fatal("expected transaction ID")
				}

			} else {

				if payment.Success {
					t.Fatal("expected payment to fail for zero amount")
				}

				if payment.Error != "Payment declined" {
					t.Fatalf(
						"expected Payment declined, got %s",
						payment.Error,
					)
				}
			}
		})
	}
}

func TestOrderService_CreateOrderFromCart(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		cart *models.Cart

		itemErr        error
		returnNil      bool
		createOrderErr error
		deleteCartErr  error

		wantErr error
	}{
		{
			name:    "nil cart",
			cart:    nil,
			wantErr: ErrCartEmpty,
		},

		{
			name: "empty cart",
			cart: &models.Cart{
				UserID: 1,
				Items:  []models.LineItem{},
			},
			wantErr: ErrCartEmpty,
		},

		{
			name: "item not found",
			cart: &models.Cart{
				UserID: 1,
				Items: []models.LineItem{
					{
						ItemID:   10,
						Quantity: 1,
						Price:    100,
					},
				},
			},
			itemErr: itemrepo.ErrItemNotFound,
			wantErr: ErrItemNotFound,
		},

		{
			name: "generic item store error",
			cart: &models.Cart{
				UserID: 1,
				Items: []models.LineItem{
					{
						ItemID:   10,
						Quantity: 1,
						Price:    100,
					},
				},
			},
			itemErr: errItemDatabaseFailure,
			wantErr: errItemDatabaseFailure,
		},

		{
			name: "nil item",
			cart: &models.Cart{
				UserID: 1,
				Items: []models.LineItem{
					{
						ItemID:   10,
						Quantity: 1,
						Price:    100,
					},
				},
			},
			returnNil: true,
			wantErr:   ErrItemNotFound,
		},

		{
			name: "insufficient stock",
			cart: &models.Cart{
				UserID: 1,
				Items: []models.LineItem{
					{
						ItemID:   10,
						Quantity: 5,
						Price:    500,
					},
				},
			},
			wantErr: ErrInsufficientStock,
		},

		{
			name: "create order error",
			cart: &models.Cart{
				UserID: 1,
				Items: []models.LineItem{
					{
						ItemID:   10,
						Quantity: 1,
						Price:    500,
					},
				},
			},
			createOrderErr: errCreateOrderDatabase,
			wantErr:        errCreateOrderDatabase,
		},

		{
			name: "clear cart not found",
			cart: &models.Cart{
				UserID: 1,
				Items: []models.LineItem{
					{
						ItemID:   10,
						Quantity: 1,
						Price:    500,
					},
				},
			},
			deleteCartErr: cartrepo.ErrCartNotFound,
			wantErr:       cartrepo.ErrCartNotFound,
		},

		{
			name: "clear cart generic error",
			cart: &models.Cart{
				UserID: 1,
				Items: []models.LineItem{
					{
						ItemID:   10,
						Quantity: 1,
						Price:    500,
					},
				},
			},
			deleteCartErr: errDeleteCartDatabase,
			wantErr:       errDeleteCartDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			os := ordermocks.NewMockOrderStore(ctrl)
			is := itemmocks.NewMockItemStore(ctrl)
			cs := cartmocks.NewMockCartStore(ctrl)

			svc := NewOrderService(os, is, cs)

			// ---------------------------------------------------------
			// Nil / empty cart
			// ---------------------------------------------------------

			if tt.cart == nil || len(tt.cart.Items) == 0 {
				_, _, err := svc.CreateOrderFromCart(
					ctx,
					models.CreateOrderFromCartRequest{
						UserID: 1,
					},
					tt.cart,
				)

				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"expected %v, got %v",
						tt.wantErr,
						err,
					)
				}

				return
			}

			// ---------------------------------------------------------
			// Item validation
			// ---------------------------------------------------------

			if tt.itemErr != nil {

				is.EXPECT().
					GetItemByID(ctx, 10).
					Return(nil, tt.itemErr).
					Times(1)

			} else if tt.returnNil {

				is.EXPECT().
					GetItemByID(ctx, 10).
					Return(nil, nil).
					Times(1)

			} else if tt.name == "insufficient stock" {

				is.EXPECT().
					GetItemByID(ctx, 10).
					Return(
						&models.Item{
							ID:    10,
							Name:  "Book",
							Price: 500,
							Stock: 1,
						},
						nil,
					).
					Times(1)

			} else {

				is.EXPECT().
					GetItemByID(ctx, 10).
					Return(
						&models.Item{
							ID:    10,
							Name:  "Book",
							Price: 500,
							Stock: 10,
						},
						nil,
					).
					Times(1)
			}

			// ---------------------------------------------------------
			// Create order error
			// ---------------------------------------------------------

			if tt.createOrderErr != nil {

				os.EXPECT().
					CreateOrder(
						ctx,
						1,
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					Return(nil, tt.createOrderErr).
					Times(1)
			}

			// ---------------------------------------------------------
			// Cart delete error
			// ---------------------------------------------------------

			if tt.deleteCartErr != nil {

				os.EXPECT().
					CreateOrder(
						ctx,
						1,
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					Return(
						&models.Order{
							ID:     1,
							UserID: 1,
						},
						nil,
					).
					Times(1)

				// Service decrements stock before deleting cart.
				os.EXPECT().
					DecrementStock(
						ctx,
						10,
						1,
					).
					Return(nil).
					Times(1)

				cs.EXPECT().
					DeleteUserCart(ctx, 1).
					Return(tt.deleteCartErr).
					Times(1)
			}

			// ---------------------------------------------------------
			// Execute
			// ---------------------------------------------------------

			_, _, err := svc.CreateOrderFromCart(
				ctx,
				models.CreateOrderFromCartRequest{
					UserID: 1,
				},
				tt.cart,
			)

			// ---------------------------------------------------------
			// Assertions
			// ---------------------------------------------------------

			if tt.wantErr != nil {

				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"expected %v, got %v",
						tt.wantErr,
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestOrderService_Idempotency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := NewOrderService(
		ordermocks.NewMockOrderStore(ctrl),
		itemmocks.NewMockItemStore(ctrl),
		cartmocks.NewMockCartStore(ctrl),
	)

	t.Run("key does not exist", func(t *testing.T) {
		record, exists := svc.GetIdempotentResponse("missing-key")

		if exists {
			t.Fatal("expected key to not exist")
		}

		if record != nil {
			t.Fatal("expected nil record")
		}
	})

	t.Run("cache and retrieve response", func(t *testing.T) {
		response := []byte(`{"success":true}`)

		svc.CacheResponse(
			"key-1",
			response,
			201,
		)

		record, exists := svc.GetIdempotentResponse("key-1")

		if !exists {
			t.Fatal("expected cached response")
		}

		if record == nil {
			t.Fatal("expected record, got nil")
		}

		if string(record.Response) != string(response) {
			t.Fatalf(
				"expected response %s, got %s",
				string(response),
				string(record.Response),
			)
		}

		if record.StatusCode != 201 {
			t.Fatalf(
				"expected status code 201, got %d",
				record.StatusCode,
			)
		}

		if !record.Expiry.After(time.Now()) {
			t.Fatal("expected expiry to be in the future")
		}
	})

	t.Run("expired response", func(t *testing.T) {
		svc.CacheResponse(
			"expired-key",
			[]byte(`expired`),
			200,
		)

		record := svc.idempotencyCache["expired-key"]

		if record == nil {
			t.Fatal("expected record")
		}

		record.Expiry = time.Now().Add(-time.Hour)

		got, exists := svc.GetIdempotentResponse("expired-key")

		if exists {
			t.Fatal("expected expired record to be removed")
		}

		if got != nil {
			t.Fatal("expected nil record")
		}

		if _, exists := svc.idempotencyCache["expired-key"]; exists {
			t.Fatal("expected expired key to be deleted from cache")
		}
	})
}

func TestOrderService_MockProcessPayment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := NewOrderService(
		ordermocks.NewMockOrderStore(ctrl),
		itemmocks.NewMockItemStore(ctrl),
		cartmocks.NewMockCartStore(ctrl),
	)

	t.Run("successful payment", func(t *testing.T) {
		result := svc.MockProcessPayment(500)

		if !result.Success {
			t.Fatal("expected payment to succeed")
		}

		if result.TransactionID == "" {
			t.Fatal("expected transaction ID")
		}

		if result.Error != "" {
			t.Fatalf(
				"expected empty error, got %s",
				result.Error,
			)
		}
	})

	t.Run("zero amount", func(t *testing.T) {
		result := svc.MockProcessPayment(0)

		if result.Success {
			t.Fatal("expected payment to fail")
		}

		if result.Error != "Payment declined" {
			t.Fatalf(
				"expected Payment declined, got %s",
				result.Error,
			)
		}
	})

	t.Run("amount too large", func(t *testing.T) {
		result := svc.MockProcessPayment(1000000)

		if result.Success {
			t.Fatal("expected payment to fail")
		}

		if result.Error != "Payment declined" {
			t.Fatalf(
				"expected Payment declined, got %s",
				result.Error,
			)
		}
	})
}
