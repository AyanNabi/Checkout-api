package cart_test

import (
	"context"
	"errors"
	"testing"

	models "checkout-api/internal/domain"
	cartrepo "checkout-api/internal/repository/cart"
	"checkout-api/internal/services/cart"
	cartmocks "checkout-api/internal/services/cart/mocks"
	itemmocks "checkout-api/internal/services/item/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var errStoreFailure = errors.New("store failure")

func TestGetCart(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid user", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.GetCart(ctx, 0)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrCartNotFound) {
			t.Fatalf("expected ErrCartNotFound, got %v", err)
		}
	})

	t.Run("cart not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(nil, cartrepo.ErrCartNotFound)

		service := cart.NewCartService(cartStore, itemStore)

		result, err := service.GetCart(ctx, 1)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if result == nil {
			t.Fatal("expected cart")
		}

		if result.UserID != 1 {
			t.Fatalf("expected user ID 1, got %d", result.UserID)
		}

		if len(result.Items) != 0 {
			t.Fatalf("expected empty cart, got %d items", len(result.Items))
		}
	})

	t.Run("store error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(nil, errStoreFailure)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.GetCart(ctx, 1)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, errStoreFailure) {
			t.Fatalf("expected store failure, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		expectedCart := &models.Cart{
			ID:     "cart_1",
			UserID: 1,
			Items: []models.LineItem{
				{
					ItemID:   1,
					Name:     "Test Item",
					Quantity: 2,
					Price:    100,
				},
			},
		}

		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(expectedCart, nil)

		service := cart.NewCartService(cartStore, itemStore)

		result, err := service.GetCart(ctx, 1)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if result == nil {
			t.Fatal("expected cart")
		}

		if result.ID != "cart_1" {
			t.Fatalf("expected cart_1, got %s", result.ID)
		}
	})
}

func TestCreateCart(t *testing.T) {
	ctx := context.Background()

	validItem := &models.Item{
		ID:    1,
		Name:  "Test Item",
		Price: 100,
		Stock: 10,
	}

	t.Run("empty items", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.CreateCart(ctx, 1, nil)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrCartEmpty) {
			t.Fatalf("expected ErrCartEmpty, got %v", err)
		}
	})

	t.Run("cart already exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		existingCart := &models.Cart{
			ID:     "cart_1",
			UserID: 1,
			Items: []models.LineItem{
				{
					ItemID:   1,
					Quantity: 1,
				},
			},
		}

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(existingCart, nil)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.CreateCart(
			ctx,
			1,
			[]models.LineItemRequest{
				{
					ItemID:   1,
					Quantity: 1,
				},
			},
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrCartAlreadyExists) {
			t.Fatalf("expected ErrCartAlreadyExists, got %v", err)
		}
	})

	t.Run("item not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(nil, cartrepo.ErrCartNotFound)

		itemStore.EXPECT().
			GetItemByID(ctx, 10).
			Return(nil, cartrepo.ErrCartItemNotFound)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.CreateCart(
			ctx,
			1,
			[]models.LineItemRequest{
				{
					ItemID:   10,
					Quantity: 1,
				},
			},
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cartrepo.ErrCartItemNotFound) {
			t.Fatalf("expected ErrCartItemNotFound, got %v", err)
		}
	})

	t.Run("item store error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(nil, cartrepo.ErrCartNotFound)

		itemStore.EXPECT().
			GetItemByID(ctx, 1).
			Return(nil, errStoreFailure)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.CreateCart(
			ctx,
			1,
			[]models.LineItemRequest{
				{
					ItemID:   1,
					Quantity: 1,
				},
			},
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, errStoreFailure) {
			t.Fatalf("expected store failure, got %v", err)
		}
	})

	t.Run("nil item", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(nil, cartrepo.ErrCartNotFound)

		itemStore.EXPECT().
			GetItemByID(ctx, 1).
			Return(nil, nil)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.CreateCart(
			ctx,
			1,
			[]models.LineItemRequest{
				{
					ItemID:   1,
					Quantity: 1,
				},
			},
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrItemNotFound) {
			t.Fatalf("expected ErrItemNotFound, got %v", err)
		}
	})

	t.Run("create cart store error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(nil, cartrepo.ErrCartNotFound)

		itemStore.EXPECT().
			GetItemByID(ctx, 1).
			Return(validItem, nil)

		cartStore.EXPECT().
			CreateUserCart(ctx, gomock.Any()).
			Return(errStoreFailure)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.CreateCart(
			ctx,
			1,
			[]models.LineItemRequest{
				{
					ItemID:   1,
					Quantity: 1,
				},
			},
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, errStoreFailure) {
			t.Fatalf("expected store failure, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(nil, cartrepo.ErrCartNotFound)

		itemStore.EXPECT().
			GetItemByID(ctx, 1).
			Return(validItem, nil)

		cartStore.EXPECT().
			CreateUserCart(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, createdCart *models.Cart) error {
				assert.Equal(t, 1, createdCart.UserID)
				assert.Equal(t, 1, len(createdCart.Items))
				assert.Equal(t, 1, createdCart.Items[0].ItemID)
				assert.Equal(t, 2, createdCart.Items[0].Quantity)
				return nil
			})

		service := cart.NewCartService(cartStore, itemStore)

		result, err := service.CreateCart(
			ctx,
			1,
			[]models.LineItemRequest{
				{
					ItemID:   1,
					Quantity: 2,
				},
			},
		)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if result == nil {
			t.Fatal("expected cart")
		}
	})
}

func TestAddItem(t *testing.T) {
	ctx := context.Background()

	validItem := &models.Item{
		ID:    1,
		Name:  "Test Item",
		Price: 100,
		Stock: 10,
	}

	t.Run("invalid quantity", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.AddItem(
			ctx,
			models.AddCartItemRequest{
				UserID:   1,
				ItemID:   1,
				Quantity: 0,
			},
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrInvalidQuantity) {
			t.Fatalf("expected ErrInvalidQuantity, got %v", err)
		}
	})

	t.Run("item not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		itemStore.EXPECT().
			GetItemByID(ctx, 10).
			Return(nil, cart.ErrItemNotFound)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.AddItem(
			ctx,
			models.AddCartItemRequest{
				UserID:   1,
				ItemID:   10,
				Quantity: 1,
			},
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrItemNotFound) {
			t.Fatalf("expected ErrItemNotFound, got %v", err)
		}
	})

	t.Run("item store error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		itemStore.EXPECT().
			GetItemByID(ctx, 10).
			Return(nil, errStoreFailure)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.AddItem(
			ctx,
			models.AddCartItemRequest{
				UserID:   1,
				ItemID:   10,
				Quantity: 1,
			},
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, errStoreFailure) {
			t.Fatalf("expected store failure, got %v", err)
		}
	})

	t.Run("nil item", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		itemStore.EXPECT().
			GetItemByID(ctx, 1).
			Return(nil, nil)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.AddItem(
			ctx,
			models.AddCartItemRequest{
				UserID:   1,
				ItemID:   1,
				Quantity: 1,
			},
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrItemNotFound) {
			t.Fatalf("expected ErrItemNotFound, got %v", err)
		}
	})

	t.Run("insufficient stock", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		item := &models.Item{
			ID:    1,
			Name:  "Test Item",
			Price: 100,
			Stock: 2,
		}

		itemStore.EXPECT().
			GetItemByID(ctx, 1).
			Return(item, nil)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.AddItem(
			ctx,
			models.AddCartItemRequest{
				UserID:   1,
				ItemID:   1,
				Quantity: 5,
			},
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrInsufficientStock) {
			t.Fatalf("expected ErrInsufficientStock, got %v", err)
		}
	})

	t.Run("add to cart error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		itemStore.EXPECT().
			GetItemByID(ctx, 1).
			Return(validItem, nil)

		cartStore.EXPECT().
			AddToCart(
				ctx,
				1,
				gomock.Any(),
			).
			Return(errStoreFailure)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.AddItem(
			ctx,
			models.AddCartItemRequest{
				UserID:   1,
				ItemID:   1,
				Quantity: 1,
			},
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, errStoreFailure) {
			t.Fatalf("expected store failure, got %v", err)
		}
	})

	t.Run("get updated cart error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		itemStore.EXPECT().
			GetItemByID(ctx, 1).
			Return(validItem, nil)

		cartStore.EXPECT().
			AddToCart(ctx, 1, gomock.Any()).
			Return(nil)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(nil, errStoreFailure)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.AddItem(
			ctx,
			models.AddCartItemRequest{
				UserID:   1,
				ItemID:   1,
				Quantity: 1,
			},
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, errStoreFailure) {
			t.Fatalf("expected store failure, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		expectedCart := &models.Cart{
			ID:     "cart_1",
			UserID: 1,
			Items: []models.LineItem{
				{
					ItemID:   1,
					Name:     "Test Item",
					Quantity: 1,
					Price:    100,
				},
			},
		}

		itemStore.EXPECT().
			GetItemByID(ctx, 1).
			Return(validItem, nil)

		cartStore.EXPECT().
			AddToCart(ctx, 1, gomock.Any()).
			Return(nil)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(expectedCart, nil)

		service := cart.NewCartService(cartStore, itemStore)

		result, err := service.AddItem(
			ctx,
			models.AddCartItemRequest{
				UserID:   1,
				ItemID:   1,
				Quantity: 1,
			},
		)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if result == nil {
			t.Fatal("expected cart")
		}

		if len(result.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result.Items))
		}
	})
}

func TestUpdateItem(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid quantity", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.UpdateItem(
			ctx,
			models.UpdateCartItemRequest{
				UserID:   1,
				Quantity: 0,
			},
			1,
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrInvalidQuantity) {
			t.Fatalf("expected ErrInvalidQuantity, got %v", err)
		}
	})

	t.Run("cart not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			UpdateCartItem(ctx, 1, 10, 2).
			Return(false, cartrepo.ErrCartNotFound)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.UpdateItem(
			ctx,
			models.UpdateCartItemRequest{
				UserID:   1,
				Quantity: 2,
			},
			10,
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrCartNotFound) {
			t.Fatalf("expected ErrCartNotFound, got %v", err)
		}
	})

	t.Run("store error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			UpdateCartItem(ctx, 1, 10, 2).
			Return(false, errStoreFailure)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.UpdateItem(
			ctx,
			models.UpdateCartItemRequest{
				UserID:   1,
				Quantity: 2,
			},
			10,
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, errStoreFailure) {
			t.Fatalf("expected store failure, got %v", err)
		}
	})

	t.Run("item not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			UpdateCartItem(ctx, 1, 10, 2).
			Return(false, nil)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.UpdateItem(
			ctx,
			models.UpdateCartItemRequest{
				UserID:   1,
				Quantity: 2,
			},
			10,
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrCartItemNotFound) {
			t.Fatalf("expected ErrCartItemNotFound, got %v", err)
		}
	})

	t.Run("get updated cart error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			UpdateCartItem(ctx, 1, 10, 2).
			Return(true, nil)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(nil, errStoreFailure)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.UpdateItem(
			ctx,
			models.UpdateCartItemRequest{
				UserID:   1,
				Quantity: 2,
			},
			10,
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, errStoreFailure) {
			t.Fatalf("expected store failure, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		expectedCart := &models.Cart{
			ID:     "cart_1",
			UserID: 1,
			Items: []models.LineItem{
				{
					ItemID:   10,
					Name:     "Test Item",
					Quantity: 2,
					Price:    100,
				},
			},
		}

		cartStore.EXPECT().
			UpdateCartItem(ctx, 1, 10, 2).
			Return(true, nil)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(expectedCart, nil)

		service := cart.NewCartService(cartStore, itemStore)

		result, err := service.UpdateItem(
			ctx,
			models.UpdateCartItemRequest{
				UserID:   1,
				Quantity: 2,
			},
			10,
		)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if result == nil {
			t.Fatal("expected cart")
		}
	})
}

func TestRemoveItem(t *testing.T) {
	ctx := context.Background()

	t.Run("cart not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			RemoveCartItem(ctx, 1, 10).
			Return(false, cartrepo.ErrCartNotFound)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.RemoveItem(
			ctx,
			models.RemoveCartItemRequest{UserID: 1},
			10,
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrCartNotFound) {
			t.Fatalf("expected ErrCartNotFound, got %v", err)
		}
	})

	t.Run("store error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			RemoveCartItem(ctx, 1, 10).
			Return(false, errStoreFailure)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.RemoveItem(
			ctx,
			models.RemoveCartItemRequest{UserID: 1},
			10,
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, errStoreFailure) {
			t.Fatalf("expected store failure, got %v", err)
		}
	})

	t.Run("item not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			RemoveCartItem(ctx, 1, 10).
			Return(false, nil)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.RemoveItem(
			ctx,
			models.RemoveCartItemRequest{UserID: 1},
			10,
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrCartItemNotFound) {
			t.Fatalf("expected ErrCartItemNotFound, got %v", err)
		}
	})

	t.Run("updated cart not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			RemoveCartItem(ctx, 1, 10).
			Return(true, nil)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(nil, cartrepo.ErrCartNotFound)

		service := cart.NewCartService(cartStore, itemStore)

		result, err := service.RemoveItem(
			ctx,
			models.RemoveCartItemRequest{UserID: 1},
			10,
		)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if result != nil {
			t.Fatal("expected nil cart")
		}
	})

	t.Run("empty cart deletes cart", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		emptyCart := &models.Cart{
			ID:     "cart_1",
			UserID: 1,
			Items:  []models.LineItem{},
		}

		cartStore.EXPECT().
			RemoveCartItem(ctx, 1, 10).
			Return(true, nil)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(emptyCart, nil)

		cartStore.EXPECT().
			DeleteUserCart(ctx, 1).
			Return(nil)

		service := cart.NewCartService(cartStore, itemStore)

		result, err := service.RemoveItem(
			ctx,
			models.RemoveCartItemRequest{UserID: 1},
			10,
		)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if result != nil {
			t.Fatal("expected nil cart")
		}
	})

	t.Run("delete empty cart error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		emptyCart := &models.Cart{
			ID:     "cart_1",
			UserID: 1,
			Items:  []models.LineItem{},
		}

		cartStore.EXPECT().
			RemoveCartItem(ctx, 1, 10).
			Return(true, nil)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(emptyCart, nil)

		cartStore.EXPECT().
			DeleteUserCart(ctx, 1).
			Return(errStoreFailure)

		service := cart.NewCartService(cartStore, itemStore)

		_, err := service.RemoveItem(
			ctx,
			models.RemoveCartItemRequest{UserID: 1},
			10,
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, errStoreFailure) {
			t.Fatalf("expected store failure, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		expectedCart := &models.Cart{
			ID:     "cart_1",
			UserID: 1,
			Items: []models.LineItem{
				{
					ItemID:   2,
					Name:     "Another Item",
					Quantity: 1,
					Price:    200,
				},
			},
		}

		cartStore.EXPECT().
			RemoveCartItem(ctx, 1, 10).
			Return(true, nil)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(expectedCart, nil)

		service := cart.NewCartService(cartStore, itemStore)

		result, err := service.RemoveItem(
			ctx,
			models.RemoveCartItemRequest{UserID: 1},
			10,
		)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if result == nil {
			t.Fatal("expected cart")
		}

		if len(result.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result.Items))
		}
	})
}

func TestClearCart(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid user", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		service := cart.NewCartService(cartStore, itemStore)

		err := service.ClearCart(ctx, 0)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrCartNotFound) {
			t.Fatalf("expected ErrCartNotFound, got %v", err)
		}
	})

	t.Run("cart not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(nil, cartrepo.ErrCartNotFound)

		service := cart.NewCartService(cartStore, itemStore)

		err := service.ClearCart(ctx, 1)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, cart.ErrCartNotFound) {
			t.Fatalf("expected ErrCartNotFound, got %v", err)
		}
	})

	t.Run("get cart store error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(nil, errStoreFailure)

		service := cart.NewCartService(cartStore, itemStore)

		err := service.ClearCart(ctx, 1)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, errStoreFailure) {
			t.Fatalf("expected store failure, got %v", err)
		}
	})

	t.Run("delete cart error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		existingCart := &models.Cart{
			ID:     "cart_1",
			UserID: 1,
			Items: []models.LineItem{
				{
					ItemID:   1,
					Quantity: 1,
				},
			},
		}

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(existingCart, nil)

		cartStore.EXPECT().
			DeleteUserCart(ctx, 1).
			Return(errStoreFailure)

		service := cart.NewCartService(cartStore, itemStore)

		err := service.ClearCart(ctx, 1)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, errStoreFailure) {
			t.Fatalf("expected store failure, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cartStore := cartmocks.NewMockCartStore(ctrl)
		itemStore := itemmocks.NewMockItemStore(ctrl)

		existingCart := &models.Cart{
			ID:     "cart_1",
			UserID: 1,
			Items: []models.LineItem{
				{
					ItemID:   1,
					Quantity: 1,
				},
			},
		}

		cartStore.EXPECT().
			GetUserCart(ctx, 1).
			Return(existingCart, nil)

		cartStore.EXPECT().
			DeleteUserCart(ctx, 1).
			Return(nil)

		service := cart.NewCartService(cartStore, itemStore)

		err := service.ClearCart(ctx, 1)

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})
}
