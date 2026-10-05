package xsolla

import (
	"checkout-api/internal/domain"
	"checkout-api/internal/helper"
	jwt_validator "checkout-api/internal/jwt-validator"
	service "checkout-api/internal/services/xsolla"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type Handler struct {
	service       *service.Service
	webhookSecret string
}

func NewHandler(s *service.Service, webhookSecret string) *Handler {
	return &Handler{service: s, webhookSecret: webhookSecret}
}

func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return helper.NewAppError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	}
	userID, ok := r.Context().Value(jwt_validator.UserIDKey).(int)
	if !ok {
		return helper.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	var request domain.XsollaPaymentRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		return helper.NewAppError(http.StatusBadRequest, "INVALID_REQUEST_BODY", "invalid request body", err)
	}

	// no item_id means "checkout the whole cart" as one order/one Xsolla payment
	if request.ItemID == 0 {
		response, err := h.service.CreateCartPaymentToken(r.Context(), userID)
		if err != nil {
			if errors.Is(err, service.ErrCartEmpty) {
				return helper.NewAppError(http.StatusBadRequest, "CART_EMPTY", "cart is empty", err)
			}
			return fmt.Errorf("create xsolla cart payment: %w", err)
		}
		helper.WriteJSON(w, http.StatusOK, response)
		return nil
	}

	if request.Quantity <= 0 {
		return helper.NewAppError(http.StatusBadRequest, "INVALID_REQUEST_BODY", "item_id and quantity must be greater than zero", nil)
	}
	response, err := h.service.CreatePaymentToken(r.Context(), userID, request.ItemID, request.Quantity)
	if err != nil {
		return fmt.Errorf("create xsolla payment: %w", err)
	}
	helper.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *Handler) GetPurchases(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return helper.NewAppError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	}
	userID, ok := r.Context().Value(jwt_validator.UserIDKey).(int)
	if !ok {
		return helper.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	purchases, err := h.service.GetPurchaseHistory(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("get purchases handler: %w", err)
	}
	helper.WriteJSON(w, http.StatusOK, map[string]any{"data": purchases})
	return nil
}

func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return helper.NewAppError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	}
	rawBody, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		return helper.NewAppError(http.StatusBadRequest, "INVALID_WEBHOOK", "unable to read webhook body", err)
	}
	if !verifySignature(rawBody, h.webhookSecret, r.Header.Get("Authorization")) {
		return helper.NewAppError(http.StatusBadRequest, "INVALID_SIGNATURE", "invalid signature", nil)
	}

	log.Printf("Xsolla webhook payload: %s", string(rawBody))
	var payload map[string]any
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return helper.NewAppError(http.StatusBadRequest, "INVALID_WEBHOOK", "invalid webhook body", err)
	}
	if err := h.service.HandleWebhook(r.Context(), payload); err != nil {
		if errors.Is(err, service.ErrInvalidUser) {
			return helper.NewAppError(http.StatusBadRequest, "INVALID_USER", "invalid user", err)
		}
		return fmt.Errorf("process xsolla webhook: %w", err)
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func verifySignature(body []byte, secret, authorization string) bool {
	if secret == "" || !strings.HasPrefix(authorization, "Signature ") {
		return false
	}
	received := strings.TrimSpace(strings.TrimPrefix(authorization, "Signature "))
	hash := sha1.Sum(append(body, []byte(secret)...))
	computed := hex.EncodeToString(hash[:])
	return len(received) == len(computed) && subtle.ConstantTimeCompare([]byte(strings.ToLower(received)), []byte(computed)) == 1
}
