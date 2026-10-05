package xsolla

import (
	"checkout-api/internal/domain"
	purchaserepo "checkout-api/internal/repository/xsolla"
	"context"
	"errors"
	"testing"
)

type fakeTokenClient struct {
	request PaymentTokenRequest
	token   string
	err     error
}

func (f *fakeTokenClient) CreatePaymentToken(_ context.Context, request PaymentTokenRequest) (string, error) {
	f.request = request
	return f.token, f.err
}

type fakeUserReader struct{}

func (fakeUserReader) GetUser(context.Context, int) (*domain.User, error) {
	return &domain.User{ID: 7, Username: "test-user", Email: "test@example.com"}, nil
}

type fakeItemReader struct{}

func (fakeItemReader) GetItemByID(context.Context, int) (*domain.Item, error) {
	return &domain.Item{ID: 11, Name: "Virtual item", XsollaSKU: "item-specific-sku"}, nil
}

type fakePurchaseStore struct {
	purchase domain.XsollaPurchase
	payments []purchaserepo.Payment
	credited map[string]bool
	err      error
}

func (f *fakePurchaseStore) CreatePurchase(_ context.Context, purchase domain.XsollaPurchase) error {
	f.purchase = purchase
	return f.err
}

func (f *fakePurchaseStore) ProcessPayment(_ context.Context, payment purchaserepo.Payment) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	if f.credited == nil {
		f.credited = map[string]bool{}
	}
	if f.credited[payment.TransactionID] {
		return false, nil
	}
	f.credited[payment.TransactionID] = true
	f.payments = append(f.payments, payment)
	return true, nil
}

func (f *fakePurchaseStore) UserExists(context.Context, int) (bool, error) {
	return f.err == nil, f.err
}

func (f *fakePurchaseStore) MarkPurchaseFailed(context.Context, string) error {
	return nil
}

func (f *fakePurchaseStore) GetPurchasesByUserID(context.Context, int) ([]domain.XsollaPurchaseView, error) {
	return nil, f.err
}

func newTestService(client TokenClient, store *fakePurchaseStore) *Service {
	return NewService(Config{
		SKU: "virtual-item-sku", Sandbox: true,
		PayStationBaseURL: "https://sandbox-secure.xsolla.com/paystation4/",
		ReturnURL:         "http://checkout.local/",
	}, client, store, fakeUserReader{}, fakeItemReader{})
}

func TestCreatePaymentTokenUsesCatalogSKUAndCorrelation(t *testing.T) {
	client := &fakeTokenClient{token: "token-123"}
	store := &fakePurchaseStore{}
	service := newTestService(client, store)

	response, err := service.CreatePaymentToken(context.Background(), 7, 11, 2)
	if err != nil {
		t.Fatalf("CreatePaymentToken() error = %v", err)
	}
	if response.Token != "token-123" || store.purchase.ID == "" {
		t.Fatalf("unexpected token response or purchase: %+v, %+v", response, store.purchase)
	}
	if client.request.User.Country.Value != "US" || !client.request.Sandbox || client.request.Purchase.Items[0].SKU != "item-specific-sku" {
		t.Fatalf("request did not include required country/catalog SKU: %+v", client.request)
	}
	if store.purchase.SKU != "item-specific-sku" {
		t.Fatalf("purchase SKU = %q, want item-specific-sku", store.purchase.SKU)
	}
	if client.request.CustomParameters["internal_purchase_id"] != store.purchase.ID {
		t.Fatalf("missing internal purchase correlation")
	}
}

func TestCreatePaymentTokenReturnsXsollaError(t *testing.T) {
	client := &fakeTokenClient{err: errors.New("xsolla unavailable")}
	service := newTestService(client, &fakePurchaseStore{})
	if _, err := service.CreatePaymentToken(context.Background(), 7, 11, 1); err == nil {
		t.Fatal("expected Xsolla error")
	}
}

func TestPaymentWebhookIsIdempotent(t *testing.T) {
	store := &fakePurchaseStore{}
	service := newTestService(&fakeTokenClient{}, store)
	payload := map[string]any{
		"notification_type": "payment",
		"transaction_id":    "TX123",
		"user":              map[string]any{"id": float64(7)},
		"custom_parameters": map[string]any{
			"internal_purchase_id": "purchase-1", "user_id": "7", "item_id": "11", "sku": "virtual-item-sku", "quantity": "1",
		},
	}
	if err := service.HandleWebhook(context.Background(), payload); err != nil {
		t.Fatalf("first payment webhook error = %v", err)
	}
	if err := service.HandleWebhook(context.Background(), payload); err != nil {
		t.Fatalf("duplicate payment webhook error = %v", err)
	}
	if len(store.payments) != 1 {
		t.Fatalf("payment count = %d, want 1", len(store.payments))
	}
}

func TestPaymentWebhookDifferentTransactionsAreProcessed(t *testing.T) {
	store := &fakePurchaseStore{}
	service := newTestService(&fakeTokenClient{}, store)
	for _, transactionID := range []string{"TX123", "TX124"} {
		payload := map[string]any{
			"notification_type": "payment", "transaction_id": transactionID,
			"custom_parameters": map[string]any{"internal_purchase_id": transactionID, "user_id": "7", "item_id": "11", "sku": "virtual-item-sku", "quantity": "1"},
		}
		if err := service.HandleWebhook(context.Background(), payload); err != nil {
			t.Fatalf("payment webhook error = %v", err)
		}
	}
	if len(store.payments) != 2 {
		t.Fatalf("payment count = %d, want 2", len(store.payments))
	}
}

func TestUserValidationWebhook(t *testing.T) {
	service := newTestService(&fakeTokenClient{}, &fakePurchaseStore{})
	if err := service.HandleWebhook(context.Background(), map[string]any{
		"notification_type": "user_validation", "user": map[string]any{"id": float64(7)},
	}); err != nil {
		t.Fatalf("user validation error = %v", err)
	}
}

func TestWebhookMalformedAndStoreError(t *testing.T) {
	service := newTestService(&fakeTokenClient{}, &fakePurchaseStore{err: errors.New("transaction failed")})
	if err := service.HandleWebhook(context.Background(), map[string]any{"notification_type": "payment"}); err == nil {
		t.Fatal("expected malformed payment error")
	}
	if err := service.HandleWebhook(context.Background(), map[string]any{
		"notification_type": "payment", "transaction_id": "TX123",
		"custom_parameters": map[string]any{"internal_purchase_id": "purchase-1", "user_id": "7", "item_id": "11", "sku": "virtual-item-sku", "quantity": "1"},
	}); err == nil {
		t.Fatal("expected payment store error")
	}
}

type multiItemReader struct {
	items map[int]*domain.Item
}

func (r multiItemReader) GetItemByID(_ context.Context, id int) (*domain.Item, error) {
	return r.items[id], nil
}

type fakeCartReader struct {
	cart    *domain.Cart
	cleared bool
}

func (f *fakeCartReader) GetUserCart(context.Context, int) (*domain.Cart, error) {
	return f.cart, nil
}

func (f *fakeCartReader) DeleteUserCart(context.Context, int) error {
	f.cleared = true
	return nil
}

type fakeOrderCreator struct {
	order *domain.Order
}

func (f *fakeOrderCreator) CreateOrder(_ context.Context, userID int, items []domain.LineItem, total int, status string) (*domain.Order, error) {
	f.order = &domain.Order{ID: 42, UserID: userID, Items: items, Total: total, Status: status}
	return f.order, nil
}

func newCartTestService(client TokenClient, store *fakePurchaseStore, cartReader *fakeCartReader, orderCreator *fakeOrderCreator) *Service {
	items := multiItemReader{items: map[int]*domain.Item{
		2:  {ID: 2, Name: "Neon Character Skin", Price: 1600, XsollaSKU: "neon_character_skin_01"},
		1:  {ID: 1, Name: "Golden AK-47 Skin", Price: 3000, XsollaSKU: "golden_ak47_skin_01"},
		10: {ID: 10, Name: "Dragon Sword", Price: 1200, XsollaSKU: "dragon_sword_01"},
	}}
	return NewService(Config{
		Sandbox: true, PayStationBaseURL: "https://sandbox-secure.xsolla.com/paystation4/",
		ReturnURL: "http://checkout.local/",
	}, client, store, fakeUserReader{}, items, cartReader, orderCreator)
}

func TestCreateCartPaymentTokenCreatesOneOrderAndOnePayment(t *testing.T) {
	client := &fakeTokenClient{token: "cart-token"}
	store := &fakePurchaseStore{}
	cartReader := &fakeCartReader{cart: &domain.Cart{UserID: 7, Items: []domain.LineItem{
		{ItemID: 2, Quantity: 3},
		{ItemID: 1, Quantity: 1},
		{ItemID: 10, Quantity: 2},
	}}}
	orderCreator := &fakeOrderCreator{}
	service := newCartTestService(client, store, cartReader, orderCreator)

	response, err := service.CreateCartPaymentToken(context.Background(), 7)
	if err != nil {
		t.Fatalf("CreateCartPaymentToken() error = %v", err)
	}
	if orderCreator.order == nil || len(orderCreator.order.Items) != 3 {
		t.Fatalf("expected one order with 3 line items, got %+v", orderCreator.order)
	}
	wantTotal := 1600*3 + 3000*1 + 1200*2
	if orderCreator.order.Total != wantTotal || response.Total != wantTotal {
		t.Fatalf("total = %d/%d, want %d", orderCreator.order.Total, response.Total, wantTotal)
	}
	if len(client.request.Purchase.Items) != 3 {
		t.Fatalf("expected one Xsolla token request with 3 purchase items, got %+v", client.request.Purchase.Items)
	}
	if client.request.CustomParameters["order_id"] != "42" {
		t.Fatalf("expected order_id correlation in custom_parameters, got %+v", client.request.CustomParameters)
	}
	if response.OrderID != 42 || response.Token != "cart-token" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if !cartReader.cleared {
		t.Fatal("expected cart to be cleared after order snapshot")
	}
}

func TestCreateCartPaymentTokenRejectsEmptyCart(t *testing.T) {
	service := newCartTestService(&fakeTokenClient{}, &fakePurchaseStore{}, &fakeCartReader{cart: &domain.Cart{}}, &fakeOrderCreator{})
	if _, err := service.CreateCartPaymentToken(context.Background(), 7); !errors.Is(err, ErrCartEmpty) {
		t.Fatalf("error = %v, want ErrCartEmpty", err)
	}
}

func TestOrderPaymentWebhookGrantsAllItemsAndIsIdempotent(t *testing.T) {
	store := &fakePurchaseStore{}
	service := newCartTestService(&fakeTokenClient{}, store, &fakeCartReader{}, &fakeOrderCreator{})
	payload := map[string]any{
		"notification_type": "payment",
		"transaction_id":    "ORDER-TX-1",
		"user":              map[string]any{"id": float64(7)},
		"custom_parameters": map[string]any{
			"internal_purchase_id": "purchase-order-42", "order_id": "42", "user_id": "7",
		},
	}
	if err := service.HandleWebhook(context.Background(), payload); err != nil {
		t.Fatalf("first order payment webhook error = %v", err)
	}
	if err := service.HandleWebhook(context.Background(), payload); err != nil {
		t.Fatalf("duplicate order payment webhook error = %v", err)
	}
	if len(store.payments) != 1 {
		t.Fatalf("payment count = %d, want 1 (idempotent)", len(store.payments))
	}
	if store.payments[0].OrderID != 42 {
		t.Fatalf("payment order id = %d, want 42", store.payments[0].OrderID)
	}
}
