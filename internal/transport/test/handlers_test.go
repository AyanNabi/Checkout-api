package test

//
//import (
//	"checkout-api/internal/domain"
//	jwt_validator "checkout-api/internal/jwt-validator"
//	"checkout-api/internal/repository"
//	services2 "checkout-api/internal/services/cart"
//	item2 "checkout-api/internal/services/item"
//	order2 "checkout-api/internal/services/order"
//	user3 "checkout-api/internal/services/user"
//	"checkout-api/internal/transport/cart"
//	"checkout-api/internal/transport/item"
//	"checkout-api/internal/transport/order"
//	user2 "checkout-api/internal/transport/user"
//	"context"
//	"encoding/json"
//	"net/http"
//	"net/http/httptest"
//	"strings"
//	"testing"
//	"time"
//
//	"github.com/golang-jwt/jwt/v5"
//)
//
//func generateTestToken(userID int, username string, secret []byte) string {
//	claims := jwt.MapClaims{
//		"user_id":  userID,
//		"username": username,
//		"exp":      time.Now().Add(time.Hour).Unix(),
//	}
//	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
//	tokenString, _ := token.SignedString(secret)
//	return tokenString
//}
//
//func TestGetItems(t *testing.T) {
//	repository.ClearInMemStore()
//	s := repository.NewInMemStore()
//	itemService := item2.NewItemService(s)
//	h := item.NewItemHandler(itemService)
//
//	tests := []struct {
//		name       string
//		method     string
//		wantStatus int
//		wantBody   string
//	}{
//		{
//			name:       "GET returns items",
//			method:     http.MethodGet,
//			wantStatus: http.StatusOK,
//			wantBody:   "Laptop",
//		},
//		{
//			name:       "POST not allowed",
//			method:     http.MethodPost,
//			wantStatus: http.StatusMethodNotAllowed,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			req := httptest.NewRequest(tt.method, "/items", nil)
//			w := httptest.NewRecorder()
//
//			h.GetItems(w, req)
//
//			if w.Code != tt.wantStatus {
//				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
//			}
//			if tt.wantBody != "" && !strings.Contains(w.Body.String(), tt.wantBody) {
//				t.Errorf("body missing %q, got: %s", tt.wantBody, w.Body.String())
//			}
//		})
//	}
//}
//
//func TestGetItemByID(t *testing.T) {
//	repository.ClearInMemStore()
//	s := repository.NewInMemStore()
//	itemService := item2.NewItemService(s)
//	h := item.NewItemHandler(itemService)
//
//	tests := []struct {
//		name       string
//		method     string
//		path       string
//		wantStatus int
//		wantBody   string
//	}{
//		{
//			name:       "valid item",
//			method:     http.MethodGet,
//			path:       "/items/1",
//			wantStatus: http.StatusOK,
//			wantBody:   "Laptop",
//		},
//		{
//			name:       "item not found",
//			method:     http.MethodGet,
//			path:       "/items/999",
//			wantStatus: http.StatusNotFound,
//		},
//		{
//			name:       "invalid ID",
//			method:     http.MethodGet,
//			path:       "/items/abc",
//			wantStatus: http.StatusBadRequest,
//		},
//		{
//			name:       "wrong method",
//			method:     http.MethodPost,
//			path:       "/items/1",
//			wantStatus: http.StatusMethodNotAllowed,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			req := httptest.NewRequest(tt.method, tt.path, nil)
//			w := httptest.NewRecorder()
//
//			h.GetItemByID(w, req)
//
//			if w.Code != tt.wantStatus {
//				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
//			}
//			if tt.wantBody != "" && !strings.Contains(w.Body.String(), tt.wantBody) {
//				t.Errorf("body missing %q, got: %s", tt.wantBody, w.Body.String())
//			}
//		})
//	}
//}
//
//func TestGetUserCart(t *testing.T) {
//	secret := []byte("secret")
//	tests := []struct {
//		name       string
//		method     string
//		userID     int
//		setupCart  bool
//		wantStatus int
//		wantBody   string
//		useToken   bool
//	}{
//		{
//			name:       "cart does not exist - returns empty cart",
//			method:     http.MethodGet,
//			userID:     1,
//			setupCart:  false,
//			wantStatus: http.StatusOK,
//			wantBody:   `"id":"","user_id":1,"items":[]`,
//			useToken:   true,
//		},
//		{
//			name:       "cart exists - returns cart with items",
//			method:     http.MethodGet,
//			userID:     1,
//			setupCart:  true,
//			wantStatus: http.StatusOK,
//			wantBody:   `"user_id":1,"items":[{"item_id":1`,
//			useToken:   true,
//		},
//		{
//			name:       "missing token",
//			method:     http.MethodGet,
//			userID:     1,
//			setupCart:  false,
//			wantStatus: http.StatusUnauthorized,
//			useToken:   false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			repository.ClearInMemStore()
//			s := repository.NewInMemStore()
//			cartService := services2.NewCartService(s)
//			h := cart.NewCartHandler(cartService)
//
//			if tt.setupCart {
//				s.AddToCart(context.Background(), tt.userID, domain.LineItem{ItemID: 1, Quantity: 2, Price: 1000})
//			}
//
//			req := httptest.NewRequest(tt.method, "/user/cart", nil)
//			if tt.useToken {
//				token := generateTestToken(tt.userID, "testuser", secret)
//				req.Header.Set("Authorization", "Bearer "+token)
//			}
//
//			w := httptest.NewRecorder()
//
//			auth := jwt_validator.JWTAuth(secret)
//			auth(http.HandlerFunc(h.GetUserCart)).ServeHTTP(w, req)
//
//			if w.Code != tt.wantStatus {
//				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
//			}
//			if tt.wantBody != "" && !strings.Contains(w.Body.String(), tt.wantBody) {
//				t.Errorf("body missing %q, got: %s", tt.wantBody, w.Body.String())
//			}
//		})
//	}
//}
//
//func TestCreateOrder(t *testing.T) {
//	secret := []byte("secret")
//	tests := []struct {
//		name       string
//		method     string
//		body       string
//		wantStatus int
//		wantBody   string
//		useToken   bool
//		userID     int
//	}{
//		{
//			name:       "valid order",
//			method:     http.MethodPost,
//			body:       `{"user_id":1,"items":[{"item_id":1,"quantity":1}]}`,
//			wantStatus: http.StatusCreated,
//			wantBody:   `"status":"paid"`,
//			useToken:   true,
//			userID:     1,
//		},
//		{
//			name:       "invalid item",
//			method:     http.MethodPost,
//			body:       `{"user_id":1,"items":[{"item_id":999,"quantity":1}]}`,
//			wantStatus: http.StatusBadRequest,
//			wantBody:   "not found",
//			useToken:   true,
//			userID:     1,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			repository.ClearInMemStore()
//			s := repository.NewInMemStore()
//			cartService := services2.NewCartService(s)
//			orderService := order2.NewOrderService(s)
//			h := order.NewOrderHandler(orderService, cartService)
//
//			req := httptest.NewRequest(tt.method, "/orders", strings.NewReader(tt.body))
//			req.Header.Set("Content-Type", "application/json")
//			if tt.useToken {
//				token := generateTestToken(tt.userID, "testuser", secret)
//				req.Header.Set("Authorization", "Bearer "+token)
//			}
//			w := httptest.NewRecorder()
//
//			// CreateOrder doesn't use authMiddleware in main.go, but it does use userID from body.
//			// Wait, I check main.go again.
//			h.CreateOrder(w, req)
//
//			if w.Code != tt.wantStatus {
//				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
//			}
//			if tt.wantBody != "" && !strings.Contains(w.Body.String(), tt.wantBody) {
//				t.Errorf("body missing %q, got: %s", tt.wantBody, w.Body.String())
//			}
//		})
//	}
//}
//
//func TestSignUp(t *testing.T) {
//	repository.ClearInMemStore()
//	s := repository.NewInMemStore()
//	svc := user3.NewUserService(s, []byte("secret"))
//	h := user2.NewUserHandler(svc)
//
//	reqBody := `{"username": "testuser", "email": "test@example.com", "password": "password123"}`
//	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(reqBody))
//	rr := httptest.NewRecorder()
//
//	h.SignUp(rr, req)
//
//	if rr.Code != http.StatusCreated {
//		t.Errorf("expected status 201, got %d, body: %s", rr.Code, rr.Body.String())
//	}
//
//	var user domain.User
//	if err := json.NewDecoder(rr.Body).Decode(&user); err != nil {
//		t.Fatalf("failed to decode response: %v", err)
//	}
//
//	if user.Username != "testuser" {
//		t.Errorf("expected username testuser, got %s", user.Username)
//	}
//	if user.Email != "test@example.com" {
//		t.Errorf("expected email test@example.com, got %s", user.Email)
//	}
//}
//
//func TestLogin(t *testing.T) {
//	repository.ClearInMemStore()
//	s := repository.NewInMemStore()
//	svc := user3.NewUserService(s, []byte("secret"))
//	h := user2.NewUserHandler(svc)
//
//	// Setup: create a user
//	ctx := context.Background()
//	svc.SignUp(ctx, "testuser", "test@example.com", "password123")
//
//	// Test login
//	reqBody := `{"username": "testuser", "password": "password123"}`
//	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(reqBody))
//	rr := httptest.NewRecorder()
//
//	h.Login(rr, req)
//
//	if rr.Code != http.StatusOK {
//		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
//	}
//
//	var resp map[string]string
//	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
//		t.Fatalf("failed to decode response: %v", err)
//	}
//
//	if resp["access_token"] == "" {
//		t.Error("expected access_token, got empty")
//	}
//	if resp["refresh_token"] == "" {
//		t.Error("expected refresh_token, got empty")
//	}
//}
