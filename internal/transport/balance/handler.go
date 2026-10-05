package balance

import (
	"checkout-api/internal/domain"
	"checkout-api/internal/helper"
	jwt_validator "checkout-api/internal/jwt-validator"
	balanceservice "checkout-api/internal/services/balance"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type BalanceHandler struct {
	service *balanceservice.BalanceService
}

func NewBalanceHandler(service *balanceservice.BalanceService) *BalanceHandler {
	return &BalanceHandler{service: service}
}

// GetBalance godoc
// @Summary Get current user balance
// @Description Returns the authenticated user's wallet balance.
// @Tags Balance
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.BalanceResponse
// @Failure 401 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /api/me/balance [get]
func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return helper.NewAppError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	}

	userID, ok := r.Context().Value(jwt_validator.UserIDKey).(int)
	if !ok {
		return helper.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}

	balance, err := h.service.GetBalance(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("get balance handler: %w", err)
	}

	helper.WriteJSON(w, http.StatusOK, domain.BalanceResponse{Balance: balance})
	return nil
}

// TopUp godoc
// @Summary Mock top-up for the current user
// @Description Adds a mock balance amount to the authenticated user's wallet. This is a mock payment flow and does not represent a real payment provider transaction.
// @Tags Balance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body domain.TopUpRequest true "Top-up request"
// @Success 200 {object} domain.BalanceResponse
// @Failure 400 {object} helper.AppError
// @Failure 401 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /api/me/balance/top-up [post]
func (h *BalanceHandler) TopUp(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return helper.NewAppError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	}

	userID, ok := r.Context().Value(jwt_validator.UserIDKey).(int)
	if !ok {
		return helper.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}

	var req domain.TopUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return helper.NewAppError(http.StatusBadRequest, "INVALID_REQUEST_BODY", "invalid request body", err)
	}

	balance, err := h.service.TopUp(r.Context(), userID, req.Amount)
	if err != nil {
		switch {
		case errors.Is(err, balanceservice.ErrInvalidAmount):
			return helper.NewAppError(http.StatusBadRequest, "INVALID_AMOUNT", "amount must be greater than zero and within supported range", err)
		default:
			return fmt.Errorf("top up handler: %w", err)
		}
	}

	helper.WriteJSON(w, http.StatusOK, domain.BalanceResponse{Balance: balance})
	return nil
}

// GetTransactions godoc
// @Summary Get balance transaction history
// @Description Returns the authenticated user's balance transaction history.
// @Tags Balance
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.TransactionsResponse
// @Failure 401 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /api/me/transactions [get]
func (h *BalanceHandler) GetTransactions(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return helper.NewAppError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	}

	userID, ok := r.Context().Value(jwt_validator.UserIDKey).(int)
	if !ok {
		return helper.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}

	transactions, err := h.service.GetTransactions(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("get transactions handler: %w", err)
	}

	helper.WriteJSON(w, http.StatusOK, domain.TransactionsResponse{Transactions: transactions})
	return nil
}
