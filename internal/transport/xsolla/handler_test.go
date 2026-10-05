package xsolla

import (
	"checkout-api/internal/domain"
	jwt_validator "checkout-api/internal/jwt-validator"
	purchaserepo "checkout-api/internal/repository/xsolla"
	service "checkout-api/internal/services/xsolla"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

type webhookPurchaseStore struct{}

func (webhookPurchaseStore) CreatePurchase(context.Context, domain.XsollaPurchase) error { return nil }
func (webhookPurchaseStore) ProcessPayment(context.Context, purchaserepo.Payment) (bool, error) {
	return true, nil
}
func (webhookPurchaseStore) UserExists(context.Context, int) (bool, error)    { return true, nil }
func (webhookPurchaseStore) MarkPurchaseFailed(context.Context, string) error { return nil }
func (webhookPurchaseStore) GetPurchasesByUserID(context.Context, int) ([]domain.XsollaPurchaseView, error) {
	return nil, nil
}

type purchaseHistoryStore struct {
	webhookPurchaseStore
	purchasesByUser map[int][]domain.XsollaPurchaseView
}

func (s purchaseHistoryStore) GetPurchasesByUserID(_ context.Context, userID int) ([]domain.XsollaPurchaseView, error) {
	return s.purchasesByUser[userID], nil
}

type webhookUserReader struct{}

func (webhookUserReader) GetUser(context.Context, int) (*domain.User, error) {
	return &domain.User{}, nil
}

type webhookItemReader struct{}

func (webhookItemReader) GetItemByID(context.Context, int) (*domain.Item, error) {
	return &domain.Item{}, nil
}

type redirectTokenClient struct{}

func (redirectTokenClient) CreatePaymentToken(context.Context, service.PaymentTokenRequest) (string, error) {
	return "token-123", nil
}

func signature(body, secret string) string {
	hash := sha1.Sum(append([]byte(body), []byte(secret)...))
	return "Signature " + hex.EncodeToString(hash[:])
}

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"notification_type":"user_validation"}`)
	if !verifySignature(body, "secret", signature(string(body), "secret")) {
		t.Fatal("expected valid signature")
	}
	if verifySignature(body, "secret", "Signature invalid") {
		t.Fatal("expected invalid signature")
	}
}

func TestWebhookRejectsInvalidSignature(t *testing.T) {
	handler := NewHandler(nil, "secret")
	request := httptest.NewRequest("POST", "/api/webhook", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Signature invalid")
	recorder := httptest.NewRecorder()

	if err := handler.Webhook(recorder, request); err == nil {
		t.Fatal("expected invalid signature error")
	}
}

func TestWebhookRejectsMalformedBodyAfterSignatureValidation(t *testing.T) {
	handler := NewHandler(nil, "secret")
	body := "{"
	request := httptest.NewRequest("POST", "/api/webhook", strings.NewReader(body))
	request.Header.Set("Authorization", signature(body, "secret"))
	recorder := httptest.NewRecorder()

	if err := handler.Webhook(recorder, request); err == nil {
		t.Fatal("expected malformed webhook error")
	}
}

func TestWebhookReturnsOKForValidUserValidation(t *testing.T) {
	webhookService := service.NewService(service.Config{}, nil, webhookPurchaseStore{}, webhookUserReader{}, webhookItemReader{})
	handler := NewHandler(webhookService, "secret")
	body := `{"notification_type":"user_validation","user":{"id":7}}`
	request := httptest.NewRequest("POST", "/api/webhook", strings.NewReader(body))
	request.Header.Set("Authorization", signature(body, "secret"))
	recorder := httptest.NewRecorder()

	if err := handler.Webhook(recorder, request); err != nil {
		t.Fatalf("valid webhook error = %v", err)
	}
	if recorder.Code != 200 {
		t.Fatalf("webhook status = %d, want 200", recorder.Code)
	}
}

func TestCreatePaymentReturnsPayStationToken(t *testing.T) {
	paymentService := service.NewService(service.Config{
		SKU: "fallback-sku", PayStationBaseURL: "https://sandbox-secure.xsolla.com/paystation4/",
	}, redirectTokenClient{}, webhookPurchaseStore{}, webhookUserReader{}, webhookItemReader{})
	handler := NewHandler(paymentService, "secret")
	request := httptest.NewRequest("POST", "/api/xsolla/purchase", strings.NewReader(`{"item_id":11,"quantity":1}`))
	request = request.WithContext(context.WithValue(request.Context(), jwt_validator.UserIDKey, 7))
	recorder := httptest.NewRecorder()

	if err := handler.CreatePayment(recorder, request); err != nil {
		t.Fatalf("create payment error = %v", err)
	}
	if recorder.Code != 200 {
		t.Fatalf("payment status = %d, want 200", recorder.Code)
	}
	var response domain.XsollaPaymentResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Token != "token-123" {
		t.Fatalf("payment token = %q, want token-123", response.Token)
	}
	if got := "https://sandbox-secure.xsolla.com/paystation4/?token=token-123"; response.PayStationURL != got {
		t.Fatalf("pay station url = %q, want %q", response.PayStationURL, got)
	}
}

type cartReader struct{ cart *domain.Cart }

func (c cartReader) GetUserCart(context.Context, int) (*domain.Cart, error) { return c.cart, nil }
func (c cartReader) DeleteUserCart(context.Context, int) error              { return nil }

type orderCreator struct{}

func (orderCreator) CreateOrder(_ context.Context, userID int, items []domain.LineItem, total int, status string) (*domain.Order, error) {
	return &domain.Order{ID: 77, UserID: userID, Items: items, Total: total, Status: status}, nil
}

type cartItemReader struct{}

func (cartItemReader) GetItemByID(_ context.Context, id int) (*domain.Item, error) {
	return &domain.Item{ID: id, Name: "Cart item", Price: 500, XsollaSKU: "cart-item-sku"}, nil
}

func TestCreatePaymentWithoutItemIDChecksOutWholeCart(t *testing.T) {
	cartSvc := service.NewService(service.Config{
		PayStationBaseURL: "https://sandbox-secure.xsolla.com/paystation4/",
	}, redirectTokenClient{}, webhookPurchaseStore{}, webhookUserReader{}, cartItemReader{},
		cartReader{cart: &domain.Cart{Items: []domain.LineItem{{ItemID: 1, Quantity: 2}, {ItemID: 2, Quantity: 1}}}},
		orderCreator{},
	)
	handler := NewHandler(cartSvc, "secret")
	request := httptest.NewRequest("POST", "/api/xsolla/purchase", strings.NewReader(`{}`))
	request = request.WithContext(context.WithValue(request.Context(), jwt_validator.UserIDKey, 7))
	recorder := httptest.NewRecorder()

	if err := handler.CreatePayment(recorder, request); err != nil {
		t.Fatalf("create cart payment error = %v", err)
	}
	var response domain.XsollaPaymentResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.OrderID != 77 {
		t.Fatalf("order id = %d, want 77", response.OrderID)
	}
}

func TestCreatePaymentRejectsEmptyCartWithBadRequest(t *testing.T) {
	cartSvc := service.NewService(service.Config{}, redirectTokenClient{}, webhookPurchaseStore{}, webhookUserReader{}, cartItemReader{},
		cartReader{cart: &domain.Cart{}}, orderCreator{},
	)
	handler := NewHandler(cartSvc, "secret")
	request := httptest.NewRequest("POST", "/api/xsolla/purchase", strings.NewReader(`{}`))
	request = request.WithContext(context.WithValue(request.Context(), jwt_validator.UserIDKey, 7))
	recorder := httptest.NewRecorder()

	err := handler.CreatePayment(recorder, request)
	if err == nil {
		t.Fatal("expected error for empty cart")
	}
}

func TestGetPurchasesReturnsOnlyAuthenticatedUsersPurchases(t *testing.T) {
	store := purchaseHistoryStore{purchasesByUser: map[int][]domain.XsollaPurchaseView{
		7:  {{ID: "purchase_mine", ItemID: 11, ItemName: "Keyboard", XsollaSKU: "golden_ak47_skin_01", Status: "paid"}},
		99: {{ID: "purchase_other", ItemID: 12, ItemName: "Mouse", XsollaSKU: "neon_character_skin_01", Status: "paid"}},
	}}
	purchaseService := service.NewService(service.Config{}, nil, store, webhookUserReader{}, webhookItemReader{})
	handler := NewHandler(purchaseService, "secret")
	request := httptest.NewRequest("GET", "/api/me/purchases", nil)
	request = request.WithContext(context.WithValue(request.Context(), jwt_validator.UserIDKey, 7))
	recorder := httptest.NewRecorder()

	if err := handler.GetPurchases(recorder, request); err != nil {
		t.Fatalf("get purchases error = %v", err)
	}
	var response struct {
		Data []domain.XsollaPurchaseView `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].ID != "purchase_mine" {
		t.Fatalf("get purchases = %+v, want only purchase_mine", response.Data)
	}
}
