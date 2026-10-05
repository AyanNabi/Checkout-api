package inventory

import (
	"checkout-api/internal/domain"
	"checkout-api/internal/helper"
	jwt_validator "checkout-api/internal/jwt-validator"
	inventoryservice "checkout-api/internal/services/inventory"
	"fmt"
	"net/http"
)

type Handler struct {
	service *inventoryservice.Service
}

func NewHandler(service *inventoryservice.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetInventory(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return helper.NewAppError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	}
	userID, ok := r.Context().Value(jwt_validator.UserIDKey).(int)
	if !ok {
		return helper.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	items, err := h.service.GetInventory(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("get inventory handler: %w", err)
	}
	helper.WriteJSON(w, http.StatusOK, domain.InventoryResponse{Items: items})
	return nil
}
