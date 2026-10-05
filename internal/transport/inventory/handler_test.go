package inventory

import (
	"checkout-api/internal/domain"
	jwt_validator "checkout-api/internal/jwt-validator"
	inventoryservice "checkout-api/internal/services/inventory"
	"context"
	"net/http/httptest"
	"testing"
)

type fakeInventoryStore struct{}

func (fakeInventoryStore) GetInventory(context.Context, int) ([]*domain.InventoryItem, error) {
	return []*domain.InventoryItem{{ItemID: 11, Name: "Sword", Quantity: 2}}, nil
}

func TestGetInventoryRequiresAuthentication(t *testing.T) {
	handler := NewHandler(inventoryservice.NewService(fakeInventoryStore{}))
	request := httptest.NewRequest("GET", "/api/me/inventory", nil)
	recorder := httptest.NewRecorder()

	if err := handler.GetInventory(recorder, request); err == nil {
		t.Fatal("expected unauthorized error")
	}
}

func TestGetInventoryReturnsOwnedItems(t *testing.T) {
	handler := NewHandler(inventoryservice.NewService(fakeInventoryStore{}))
	request := httptest.NewRequest("GET", "/api/me/inventory", nil)
	request = request.WithContext(context.WithValue(request.Context(), jwt_validator.UserIDKey, 7))
	recorder := httptest.NewRecorder()

	if err := handler.GetInventory(recorder, request); err != nil {
		t.Fatalf("get inventory error = %v", err)
	}
	if recorder.Code != 200 {
		t.Fatalf("inventory status = %d, want 200", recorder.Code)
	}
}
