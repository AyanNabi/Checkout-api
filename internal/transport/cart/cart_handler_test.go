package cart

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	models "checkout-api/internal/domain"
	"checkout-api/internal/jwt-validator"
	cartservice "checkout-api/internal/services/cart"
)

type mockCartService struct {
	err    error
	called bool
}

func (m *mockCartService) AddItem(
	ctx context.Context,
	req models.AddCartItemRequest,
) (*models.Cart, error) {
	m.called = true

	if m.err != nil {
		return nil, m.err
	}

	return &models.Cart{
		UserID: req.UserID,
	}, nil
}

func (m *mockCartService) UpdateItem(
	ctx context.Context,
	req models.UpdateCartItemRequest,
	itemID int,
) (*models.Cart, error) {
	m.called = true

	if m.err != nil {
		return nil, m.err
	}

	return &models.Cart{
		UserID: req.UserID,
	}, nil
}

func (m *mockCartService) RemoveItem(
	ctx context.Context,
	req models.RemoveCartItemRequest,
	itemID int,
) (*models.Cart, error) {
	m.called = true

	if m.err != nil {
		return nil, m.err
	}

	return &models.Cart{
		UserID: req.UserID,
	}, nil
}

func (m *mockCartService) GetCart(
	ctx context.Context,
	userID int,
) (*models.Cart, error) {
	m.called = true

	if m.err != nil {
		return nil, m.err
	}

	return &models.Cart{
		UserID: userID,
	}, nil
}

func TestAddCartItem(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		method string
		body   string
		status int
	}{
		{
			name:   "success",
			status: http.StatusOK,
			method: http.MethodPost,
			body:   `{"item_id":1,"quantity":2}`,
		},
		{
			name:   "invalid quantity",
			err:    cartservice.ErrInvalidQuantity,
			status: http.StatusBadRequest,
			method: http.MethodPost,
			body:   `{"item_id":1,"quantity":0}`,
		},
		{
			name:   "item not found",
			err:    cartservice.ErrItemNotFound,
			status: http.StatusNotFound,
			method: http.MethodPost,
			body:   `{"item_id":1,"quantity":2}`,
		},
		{
			name:   "insufficient stock",
			err:    cartservice.ErrInsufficientStock,
			status: http.StatusConflict,
			method: http.MethodPost,
			body:   `{"item_id":1,"quantity":2}`,
		},
		{
			name:   "internal error",
			err:    errors.New("database error"),
			status: http.StatusInternalServerError,
			method: http.MethodPost,
			body:   `{"item_id":1,"quantity":2}`,
		},
		{
			name:   "method not allowed",
			status: http.StatusMethodNotAllowed,
			method: http.MethodGet,
			body:   `{"item_id":1,"quantity":2}`,
		},
		{
			name:   "invalid json",
			status: http.StatusBadRequest,
			method: http.MethodPost,
			body:   `invalid json`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockCartService{
				err: tt.err,
			}

			handler := NewCartHandler(service)

			req := httptest.NewRequest(
				tt.method,
				"/user/cart",
				bytes.NewBufferString(tt.body),
			)

			ctx := context.WithValue(
				req.Context(),
				jwt_validator.UserIDKey,
				1,
			)

			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			handler.AddCartItem(rec, req)

			if rec.Code != tt.status {
				t.Fatalf(
					"got %d, want %d",
					rec.Code,
					tt.status,
				)
			}
		})
	}
}

func TestAddCartItem_Unauthorized(t *testing.T) {
	service := &mockCartService{}
	handler := NewCartHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/user/cart",
		bytes.NewBufferString(`{"item_id":1,"quantity":2}`),
	)

	rec := httptest.NewRecorder()

	handler.AddCartItem(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"got %d, want %d",
			rec.Code,
			http.StatusUnauthorized,
		)
	}

	if service.called {
		t.Fatal("service should not be called")
	}
}

func TestUpdateCartItem(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		path   string
		method string
		body   string
		status int
	}{
		{
			name:   "success",
			path:   "/user/cart/items/1",
			method: http.MethodPatch,
			body:   `{"quantity":2}`,
			status: http.StatusOK,
		},
		{
			name:   "invalid quantity",
			err:    cartservice.ErrInvalidQuantity,
			path:   "/user/cart/items/1",
			method: http.MethodPatch,
			body:   `{"quantity":0}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "cart not found",
			err:    cartservice.ErrCartNotFound,
			path:   "/user/cart/items/1",
			method: http.MethodPatch,
			body:   `{"quantity":2}`,
			status: http.StatusNotFound,
		},
		{
			name:   "cart item not found",
			err:    cartservice.ErrCartItemNotFound,
			path:   "/user/cart/items/1",
			method: http.MethodPatch,
			body:   `{"quantity":2}`,
			status: http.StatusNotFound,
		},
		{
			name:   "internal error",
			err:    errors.New("database error"),
			path:   "/user/cart/items/1",
			method: http.MethodPatch,
			body:   `{"quantity":2}`,
			status: http.StatusInternalServerError,
		},
		{
			name:   "invalid id",
			path:   "/user/cart/items/abc",
			method: http.MethodPatch,
			body:   `{"quantity":2}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "invalid path",
			path:   "/user/cart",
			method: http.MethodPatch,
			body:   `{"quantity":2}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "method not allowed",
			path:   "/user/cart/items/1",
			method: http.MethodGet,
			body:   `{"quantity":2}`,
			status: http.StatusMethodNotAllowed,
		},
		{
			name:   "invalid json",
			path:   "/user/cart/items/1",
			method: http.MethodPatch,
			body:   `invalid json`,
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockCartService{
				err: tt.err,
			}

			handler := NewCartHandler(service)

			req := httptest.NewRequest(
				tt.method,
				tt.path,
				bytes.NewBufferString(tt.body),
			)

			ctx := context.WithValue(
				req.Context(),
				jwt_validator.UserIDKey,
				1,
			)

			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			handler.UpdateCartItem(rec, req)

			if rec.Code != tt.status {
				t.Fatalf(
					"got %d, want %d",
					rec.Code,
					tt.status,
				)
			}
		})
	}
}

func TestUpdateCartItem_Unauthorized(t *testing.T) {
	service := &mockCartService{}
	handler := NewCartHandler(service)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/user/cart/items/1",
		bytes.NewBufferString(`{"quantity":2}`),
	)

	rec := httptest.NewRecorder()

	handler.UpdateCartItem(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"got %d, want %d",
			rec.Code,
			http.StatusUnauthorized,
		)
	}

	if service.called {
		t.Fatal("service should not be called")
	}
}

func TestRemoveCartItem(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		path   string
		method string
		status int
	}{
		{
			name:   "success",
			path:   "/user/cart/items/1",
			method: http.MethodDelete,
			status: http.StatusNoContent,
		},
		{
			name:   "cart not found",
			err:    cartservice.ErrCartNotFound,
			path:   "/user/cart/items/1",
			method: http.MethodDelete,
			status: http.StatusNotFound,
		},
		{
			name:   "cart item not found",
			err:    cartservice.ErrCartItemNotFound,
			path:   "/user/cart/items/1",
			method: http.MethodDelete,
			status: http.StatusNotFound,
		},
		{
			name:   "internal error",
			err:    errors.New("database error"),
			path:   "/user/cart/items/1",
			method: http.MethodDelete,
			status: http.StatusInternalServerError,
		},
		{
			name:   "invalid id",
			path:   "/user/cart/items/abc",
			method: http.MethodDelete,
			status: http.StatusBadRequest,
		},
		{
			name:   "invalid path",
			path:   "/user/cart",
			method: http.MethodDelete,
			status: http.StatusBadRequest,
		},
		{
			name:   "method not allowed",
			path:   "/user/cart/items/1",
			method: http.MethodGet,
			status: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockCartService{
				err: tt.err,
			}

			handler := NewCartHandler(service)

			req := httptest.NewRequest(
				tt.method,
				tt.path,
				nil,
			)

			ctx := context.WithValue(
				req.Context(),
				jwt_validator.UserIDKey,
				1,
			)

			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			handler.RemoveCartItem(rec, req)

			if rec.Code != tt.status {
				t.Fatalf(
					"got %d, want %d",
					rec.Code,
					tt.status,
				)
			}
		})
	}
}

func TestRemoveCartItem_Unauthorized(t *testing.T) {
	service := &mockCartService{}
	handler := NewCartHandler(service)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/user/cart/items/1",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.RemoveCartItem(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"got %d, want %d",
			rec.Code,
			http.StatusUnauthorized,
		)
	}

	if service.called {
		t.Fatal("service should not be called")
	}
}

func TestGetUserCart(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		method string
		status int
	}{
		{
			name:   "success",
			method: http.MethodGet,
			status: http.StatusOK,
		},
		{
			name:   "cart not found",
			err:    cartservice.ErrCartNotFound,
			method: http.MethodGet,
			status: http.StatusNotFound,
		},
		{
			name:   "internal error",
			err:    errors.New("database error"),
			method: http.MethodGet,
			status: http.StatusInternalServerError,
		},
		{
			name:   "method not allowed",
			method: http.MethodPost,
			status: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockCartService{
				err: tt.err,
			}

			handler := NewCartHandler(service)

			req := httptest.NewRequest(
				tt.method,
				"/user/cart",
				nil,
			)

			ctx := context.WithValue(
				req.Context(),
				jwt_validator.UserIDKey,
				1,
			)

			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			handler.GetUserCart(rec, req)

			if rec.Code != tt.status {
				t.Fatalf(
					"got %d, want %d",
					rec.Code,
					tt.status,
				)
			}
		})
	}
}

func TestGetUserCart_Unauthorized(t *testing.T) {
	service := &mockCartService{}
	handler := NewCartHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/user/cart",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.GetUserCart(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"got %d, want %d",
			rec.Code,
			http.StatusUnauthorized,
		)
	}

	if service.called {
		t.Fatal("service should not be called")
	}
}
