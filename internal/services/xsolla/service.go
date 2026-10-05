package xsolla

import (
	"checkout-api/internal/domain"
	purchaserepo "checkout-api/internal/repository/xsolla"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
)

type TokenClient interface {
	CreatePaymentToken(ctx context.Context, request PaymentTokenRequest) (string, error)
}

type PurchaseStore interface {
	CreatePurchase(ctx context.Context, purchase domain.XsollaPurchase) error
	ProcessPayment(ctx context.Context, payment purchaserepo.Payment) (bool, error)
	UserExists(ctx context.Context, userID int) (bool, error)
	MarkPurchaseFailed(ctx context.Context, purchaseID string) error
	GetPurchasesByUserID(ctx context.Context, userID int) ([]domain.XsollaPurchaseView, error)
}

type UserReader interface {
	GetUser(ctx context.Context, id int) (*domain.User, error)
}

type ItemReader interface {
	GetItemByID(ctx context.Context, id int) (*domain.Item, error)
}

type CartReader interface {
	GetUserCart(ctx context.Context, userID int) (*domain.Cart, error)
	DeleteUserCart(ctx context.Context, userID int) error
}

type OrderCreator interface {
	CreateOrder(ctx context.Context, userID int, items []domain.LineItem, total int, status string) (*domain.Order, error)
}

type PaymentTokenRequest struct {
	Sandbox bool `json:"sandbox"`
	User    struct {
		ID struct {
			Value string `json:"value"`
		} `json:"id"`
		Name struct {
			Value string `json:"value"`
		} `json:"name"`
		Email struct {
			Value string `json:"value"`
		} `json:"email"`
		Country struct {
			Value string `json:"value"`
		} `json:"country"`
	} `json:"user"`
	Purchase struct {
		Items []struct {
			SKU      string `json:"sku"`
			Quantity int    `json:"quantity"`
		} `json:"items"`
	} `json:"purchase"`
	Settings struct {
		ReturnURL      string `json:"return_url,omitempty"`
		RedirectPolicy struct {
			RedirectConditions string `json:"redirect_conditions,omitempty"`
		} `json:"redirect_policy,omitempty"`
	} `json:"settings,omitempty"`
	CustomParameters map[string]string `json:"custom_parameters"`
}

type Config struct {
	ProjectID         string
	MerchantID        string
	APIKey            string
	WebhookSecret     string
	SKU               string
	Sandbox           bool
	ReturnURL         string
	APIBaseURL        string
	PayStationBaseURL string
}

type Service struct {
	config    Config
	client    TokenClient
	purchases PurchaseStore
	users     UserReader
	items     ItemReader
	carts     CartReader
	orders    OrderCreator
}

var ErrInvalidUser = errors.New("invalid user")
var ErrCartEmpty = errors.New("cart is empty")

func NewService(config Config, client TokenClient, purchases PurchaseStore, users UserReader, items ItemReader, dependencies ...any) *Service {
	service := &Service{config: config, client: client, purchases: purchases, users: users, items: items}
	for _, dependency := range dependencies {
		switch value := dependency.(type) {
		case CartReader:
			service.carts = value
		case OrderCreator:
			service.orders = value
		}
	}
	return service
}

func (s *Service) CreatePaymentToken(ctx context.Context, userID int, legacy ...int) (domain.XsollaPaymentResponse, error) {
	itemID, quantity := 0, 0
	if len(legacy) >= 2 {
		itemID, quantity = legacy[0], legacy[1]
	}
	if len(legacy) > 0 && quantity <= 0 {
		return domain.XsollaPaymentResponse{}, errors.New("quantity must be greater than zero")
	}
	user, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return domain.XsollaPaymentResponse{}, fmt.Errorf("load xsolla user: %w", err)
	}
	item, err := s.items.GetItemByID(ctx, itemID)
	if err != nil {
		return domain.XsollaPaymentResponse{}, fmt.Errorf("load xsolla item: %w", err)
	}
	if item == nil {
		return domain.XsollaPaymentResponse{}, errors.New("item not found")
	}
	sku := item.XsollaSKU
	if sku == "" {
		sku = s.config.SKU
	}
	if strings.TrimSpace(sku) == "" {
		return domain.XsollaPaymentResponse{}, errors.New("item has no xsolla SKU")
	}
	purchaseID, err := newPurchaseID()
	if err != nil {
		return domain.XsollaPaymentResponse{}, err
	}
	if err := s.purchases.CreatePurchase(ctx, domain.XsollaPurchase{
		ID: purchaseID, UserID: userID, ItemID: itemID, SKU: sku, Quantity: quantity, Status: "pending",
	}); err != nil {
		return domain.XsollaPaymentResponse{}, err
	}

	request := PaymentTokenRequest{Sandbox: s.config.Sandbox, CustomParameters: map[string]string{
		"internal_purchase_id": purchaseID,
		"user_id":              strconv.Itoa(userID),
		"item_id":              strconv.Itoa(itemID),
		"sku":                  sku,
		"quantity":             strconv.Itoa(quantity),
	}}
	request.User.ID.Value = strconv.Itoa(userID)
	request.User.Name.Value = user.Username
	request.User.Email.Value = user.Email
	request.User.Country.Value = "US"
	request.Purchase.Items = []struct {
		SKU      string `json:"sku"`
		Quantity int    `json:"quantity"`
	}{{SKU: sku, Quantity: quantity}}
	request.Settings.ReturnURL = s.config.ReturnURL
	request.Settings.RedirectPolicy.RedirectConditions = "successful_or_canceled"
	log.Printf(
		"Xsolla token: user_id=%s sku=%s quantity=%d sandbox=%v",
		request.User.ID.Value,
		request.Purchase.Items[0].SKU,
		request.Purchase.Items[0].Quantity,
		request.Sandbox,
	)
	token, err := s.client.CreatePaymentToken(ctx, request)
	if err != nil {
		if markErr := s.purchases.MarkPurchaseFailed(ctx, purchaseID); markErr != nil {
			return domain.XsollaPaymentResponse{}, fmt.Errorf("create xsolla token: %w (mark failed: %v)", err, markErr)
		}
		return domain.XsollaPaymentResponse{}, fmt.Errorf("create xsolla token: %w", err)
	}
	baseURL := s.config.PayStationBaseURL
	return domain.XsollaPaymentResponse{
		PurchaseID:    purchaseID,
		Token:         token,
		PayStationURL: strings.TrimRight(baseURL, "/") + "/?token=" + url.QueryEscape(token),
	}, nil
}

func (s *Service) GetPurchaseHistory(ctx context.Context, userID int) ([]domain.XsollaPurchaseView, error) {
	purchases, err := s.purchases.GetPurchasesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get purchase history: %w", err)
	}
	return purchases, nil
}

// CreateCartPaymentToken creates ONE order from the user's cart and ONE Xsolla payment for it.
func (s *Service) CreateCartPaymentToken(ctx context.Context, userID int) (domain.XsollaPaymentResponse, error) {
	if s.carts == nil || s.orders == nil {
		return domain.XsollaPaymentResponse{}, errors.New("cart checkout is not configured")
	}
	cart, err := s.carts.GetUserCart(ctx, userID)
	if err != nil {
		return domain.XsollaPaymentResponse{}, fmt.Errorf("load xsolla cart: %w", err)
	}
	if cart == nil || len(cart.Items) == 0 {
		return domain.XsollaPaymentResponse{}, ErrCartEmpty
	}
	user, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return domain.XsollaPaymentResponse{}, fmt.Errorf("load xsolla user: %w", err)
	}

	lineItems := make([]domain.LineItem, 0, len(cart.Items))
	purchaseItems := make([]struct {
		SKU      string `json:"sku"`
		Quantity int    `json:"quantity"`
	}, 0, len(cart.Items))
	total := 0
	for _, cartItem := range cart.Items {
		if cartItem.Quantity <= 0 {
			return domain.XsollaPaymentResponse{}, fmt.Errorf("cart item %d: quantity must be greater than zero", cartItem.ItemID)
		}
		item, err := s.items.GetItemByID(ctx, cartItem.ItemID)
		if err != nil {
			return domain.XsollaPaymentResponse{}, fmt.Errorf("load xsolla cart item %d: %w", cartItem.ItemID, err)
		}
		if item == nil {
			return domain.XsollaPaymentResponse{}, fmt.Errorf("cart item %d not found", cartItem.ItemID)
		}
		if strings.TrimSpace(item.XsollaSKU) == "" {
			return domain.XsollaPaymentResponse{}, fmt.Errorf("cart item %d has no xsolla SKU", cartItem.ItemID)
		}
		total += item.Price * cartItem.Quantity
		lineItems = append(lineItems, domain.LineItem{ItemID: item.ID, Name: item.Name, Quantity: cartItem.Quantity, Price: item.Price})
		purchaseItems = append(purchaseItems, struct {
			SKU      string `json:"sku"`
			Quantity int    `json:"quantity"`
		}{SKU: item.XsollaSKU, Quantity: cartItem.Quantity})
	}

	order, err := s.orders.CreateOrder(ctx, userID, lineItems, total, "pending")
	if err != nil {
		return domain.XsollaPaymentResponse{}, fmt.Errorf("create xsolla order: %w", err)
	}

	purchaseID, err := newPurchaseID()
	if err != nil {
		return domain.XsollaPaymentResponse{}, err
	}
	if err := s.purchases.CreatePurchase(ctx, domain.XsollaPurchase{
		ID: purchaseID, OrderID: order.ID, UserID: userID, Status: "pending",
	}); err != nil {
		return domain.XsollaPaymentResponse{}, err
	}

	// the cart is a snapshot source; the order already owns its own order_items, so it's safe to clear now
	if err := s.carts.DeleteUserCart(ctx, userID); err != nil {
		return domain.XsollaPaymentResponse{}, fmt.Errorf("clear xsolla cart: %w", err)
	}

	request := PaymentTokenRequest{Sandbox: s.config.Sandbox, CustomParameters: map[string]string{
		"internal_purchase_id": purchaseID,
		"order_id":             strconv.Itoa(order.ID),
		"user_id":              strconv.Itoa(userID),
	}}
	request.User.ID.Value = strconv.Itoa(userID)
	request.User.Name.Value = user.Username
	request.User.Email.Value = user.Email
	request.User.Country.Value = "US"
	request.Purchase.Items = purchaseItems
	request.Settings.ReturnURL = s.config.ReturnURL
	request.Settings.RedirectPolicy.RedirectConditions = "successful_or_canceled"

	token, err := s.client.CreatePaymentToken(ctx, request)
	if err != nil {
		if markErr := s.purchases.MarkPurchaseFailed(ctx, purchaseID); markErr != nil {
			return domain.XsollaPaymentResponse{}, fmt.Errorf("create xsolla token: %w (mark failed: %v)", err, markErr)
		}
		return domain.XsollaPaymentResponse{}, fmt.Errorf("create xsolla token: %w", err)
	}
	baseURL := s.config.PayStationBaseURL
	return domain.XsollaPaymentResponse{
		PurchaseID:    purchaseID,
		OrderID:       order.ID,
		Total:         total,
		Token:         token,
		PayStationURL: strings.TrimRight(baseURL, "/") + "/?token=" + url.QueryEscape(token),
	}, nil
}

func (s *Service) HandleWebhook(ctx context.Context, payload map[string]any) error {
	notificationType, _ := payload["notification_type"].(string)
	if notificationType == "user_validation" {
		userID, err := nestedInt(payload, "user", "id")
		if err != nil {
			return fmt.Errorf("user validation: %w", err)
		}
		exists, err := s.purchases.UserExists(ctx, userID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrInvalidUser
		}
		return nil
	}
	if notificationType != "payment" && notificationType != "order_paid" {
		return fmt.Errorf("unsupported notification type: %s", notificationType)
	}

	transactionID, err := webhookTransactionID(payload)
	if err != nil {
		return err
	}
	custom, _ := payload["custom_parameters"].(map[string]any)
	purchaseID, _ := custom["internal_purchase_id"].(string)
	if purchaseID == "" {
		return errors.New("internal_purchase_id is required")
	}
	userID, err := valueInt(custom["user_id"])
	if err != nil {
		userID, err = nestedInt(payload, "user", "id")
		if err != nil {
			return err
		}
	}
	orderID, orderErr := valueInt(custom["order_id"])
	if orderErr == nil && orderID > 0 {
		_, err = s.purchases.ProcessPayment(ctx, purchaserepo.Payment{
			PurchaseID: purchaseID, TransactionID: transactionID, UserID: userID, OrderID: orderID,
		})
		return err
	}
	itemID, err := valueInt(custom["item_id"])
	if err != nil {
		return err
	}
	quantity, err := valueInt(custom["quantity"])
	if err != nil || quantity <= 0 {
		quantity = 1
	}
	sku, _ := custom["sku"].(string)
	_, err = s.purchases.ProcessPayment(ctx, purchaserepo.Payment{
		PurchaseID: purchaseID, TransactionID: transactionID, UserID: userID,
		ItemID: itemID, SKU: sku, Quantity: quantity,
	})
	return err
}

func webhookTransactionID(payload map[string]any) (string, error) {
	if value, ok := payload["transaction_id"].(string); ok && value != "" {
		return value, nil
	}

	value, err := nestedValue(payload, "billing", "transaction", "id")
	if err != nil {
		return "", errors.New("transaction ID is required")
	}

	return fmt.Sprint(value), nil
}
func newPurchaseID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate purchase ID: %w", err)
	}
	return "purchase_" + hex.EncodeToString(bytes), nil
}

func nestedInt(payload map[string]any, keys ...string) (int, error) {
	var current any = payload
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return 0, errors.New("missing webhook field: " + key)
		}
		current = object[key]
	}
	return valueInt(current)
}

func valueInt(value any) (int, error) {
	switch typed := value.(type) {
	case float64:
		return int(typed), nil
	case string:
		return strconv.Atoi(typed)
	case json.Number:
		return strconv.Atoi(string(typed))
	default:
		return 0, errors.New("invalid integer webhook field")
	}
}

func firstString(payload map[string]any, top []string, nested []string) (string, error) {
	if value, ok := payload[top[0]].(string); ok && value != "" {
		return value, nil
	}
	value, err := nestedValue(payload, nested...)
	if err != nil {
		return "", errors.New("transaction ID is required")
	}
	result, ok := value.(string)
	if !ok || result == "" {
		return "", errors.New("transaction ID is required")
	}
	return result, nil
}

func nestedValue(payload map[string]any, keys ...string) (any, error) {
	var current any = payload
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, errors.New("missing webhook field")
		}
		current = object[key]
	}
	return current, nil
}
