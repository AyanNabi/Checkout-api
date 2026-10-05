package cart

import (
	models2 "checkout-api/internal/domain"
	"checkout-api/internal/helper"
	"checkout-api/internal/jwt-validator"
	cartservice "checkout-api/internal/services/cart"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type CartService interface {
	AddItem(
		ctx context.Context,
		req models2.AddCartItemRequest,
	) (*models2.Cart, error)
	UpdateItem(
		ctx context.Context,
		req models2.UpdateCartItemRequest,
		itemID int,
	) (*models2.Cart, error)
	RemoveItem(
		ctx context.Context,
		req models2.RemoveCartItemRequest,
		itemID int,
	) (*models2.Cart, error)
	GetCart(
		ctx context.Context,
		userID int,
	) (*models2.Cart, error)
}

type CartHandler struct {
	service CartService
}

func NewCartHandler(s CartService) *CartHandler {
	return &CartHandler{
		service: s,
	}
}

// AddCartItem godoc
// @Summary Add item to cart
// @Description Adds an item with the specified quantity to the authenticated user's cart
// @Tags Cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body domain.LineItemRequest true "Cart item"
// @Success 200 {object} domain.Cart
// @Failure 400 {string} string "Invalid request body or quantity"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Item not found"
// @Failure 409 {string} string "Insufficient stock"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {string} string "Internal server error"
// @Router /user/cart [post]
func (h *CartHandler) AddCartItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	userID, ok := r.Context().Value(jwt_validator.UserIDKey).(int)
	if !ok {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	var req models2.LineItemRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	cart, err := h.service.AddItem(
		r.Context(),
		models2.AddCartItemRequest{
			UserID:   userID,
			ItemID:   req.ItemID,
			Quantity: req.Quantity,
		},
	)

	if err != nil {
		switch {
		case errors.Is(err, cartservice.ErrInvalidQuantity):
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)

		case errors.Is(err, cartservice.ErrItemNotFound):
			http.Error(
				w,
				err.Error(),
				http.StatusNotFound,
			)

		case errors.Is(err, cartservice.ErrInsufficientStock):
			http.Error(
				w,
				err.Error(),
				http.StatusConflict,
			)

		default:
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}

		return
	}

	helper.WriteJSON(w, http.StatusOK, cart)
}

// UpdateCartItem godoc
// @Summary Update cart item quantity
// @Description Updates the quantity of an item in the authenticated user's cart
// @Tags Cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Item ID"
// @Param request body domain.UpdateCartItemRequest true "Updated quantity"
// @Success 200 {object} domain.Cart
// @Failure 400 {string} string "Invalid item ID or request body"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Cart or cart item not found"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {string} string "Internal server error"
// @Router /user/cart/items/{id} [patch]
func (h *CartHandler) UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")

	if len(pathParts) < 5 {
		http.Error(
			w,
			"invalid path",
			http.StatusBadRequest,
		)
		return
	}

	itemID, err := strconv.Atoi(pathParts[5])
	if err != nil {
		http.Error(
			w,
			"invalid item ID",
			http.StatusBadRequest,
		)
		return
	}

	userID, ok := r.Context().Value(jwt_validator.UserIDKey).(int)
	if !ok {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	var req models2.UpdateCartItemRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	cart, err := h.service.UpdateItem(
		r.Context(),
		models2.UpdateCartItemRequest{
			UserID:   userID,
			Quantity: req.Quantity,
		},
		itemID,
	)

	if err != nil {
		switch {
		case errors.Is(err, cartservice.ErrInvalidQuantity):
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)

		case errors.Is(err, cartservice.ErrCartNotFound):
			http.Error(
				w,
				err.Error(),
				http.StatusNotFound,
			)

		case errors.Is(err, cartservice.ErrCartItemNotFound):
			http.Error(
				w,
				err.Error(),
				http.StatusNotFound,
			)

		default:
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}

		return
	}

	helper.WriteJSON(w, http.StatusOK, cart)
}

// RemoveCartItem godoc
// @Summary Remove item from cart
// @Description Removes an item from the authenticated user's cart
// @Tags Cart
// @Produce json
// @Security BearerAuth
// @Param id path int true "Item ID"
// @Success 204 "Item removed successfully"
// @Failure 400 {string} string "Invalid item ID or path"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Cart or cart item not found"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {string} string "Internal server error"
// @Router /user/cart/items/{id} [delete]
func (h *CartHandler) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")

	if len(pathParts) < 5 {
		http.Error(
			w,
			"invalid path",
			http.StatusBadRequest,
		)
		return
	}

	itemID, err := strconv.Atoi(pathParts[5])
	if err != nil {
		http.Error(
			w,
			"invalid item ID",
			http.StatusBadRequest,
		)
		return
	}

	userID, ok := r.Context().Value(jwt_validator.UserIDKey).(int)
	if !ok {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	_, err = h.service.RemoveItem(
		r.Context(),
		models2.RemoveCartItemRequest{
			UserID: userID,
		},
		itemID,
	)

	if err != nil {
		switch {
		case errors.Is(err, cartservice.ErrCartNotFound):
			http.Error(
				w,
				err.Error(),
				http.StatusNotFound,
			)

		case errors.Is(err, cartservice.ErrCartItemNotFound):
			http.Error(
				w,
				err.Error(),
				http.StatusNotFound,
			)

		default:
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetUserCart godoc
// @Summary Get user's cart
// @Description Returns the authenticated user's shopping cart
// @Tags Cart
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.Cart
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Cart not found"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {string} string "Internal server error"
// @Router /user/cart [get]
func (h *CartHandler) GetUserCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	userID, ok := r.Context().Value(jwt_validator.UserIDKey).(int)
	if !ok {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	cart, err := h.service.GetCart(
		r.Context(),
		userID,
	)

	if err != nil {
		switch {
		case errors.Is(err, cartservice.ErrCartNotFound):
			http.Error(
				w,
				err.Error(),
				http.StatusNotFound,
			)

		default:
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}

		return
	}

	helper.WriteJSON(
		w,
		http.StatusOK,
		cart,
	)
}
