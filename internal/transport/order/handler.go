package order

import (
	"checkout-api/internal/domain"
	"checkout-api/internal/helper"
	filter "checkout-api/internal/helper/filter"
	pageview "checkout-api/internal/helper/page"
	jwt_validator "checkout-api/internal/jwt-validator"
	cartservice "checkout-api/internal/services/cart"
	orderservice "checkout-api/internal/services/order"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type OrderHandler struct {
	orderService *orderservice.OrderService
	cartService  *cartservice.CartService
}

func NewOrderHandler(
	os *orderservice.OrderService,
	cs *cartservice.CartService,
) *OrderHandler {
	return &OrderHandler{
		orderService: os,
		cartService:  cs,
	}
}

// GetOrderByID godoc
// @Summary Get order by ID
// @Description Get a single order by its ID
// @Tags Orders
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} domain.Order
// @Failure 400 {object} helper.AppError
// @Failure 401 {object} helper.AppError
// @Failure 404 {object} helper.AppError
// @Failure 405 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /orders/{id} [get]
func (h *OrderHandler) GetOrderByID(
	w http.ResponseWriter,
	r *http.Request,
) error {

	if r.Method != http.MethodGet {
		return helper.NewAppError(
			http.StatusMethodNotAllowed,
			"METHOD_NOT_ALLOWED",
			"method not allowed",
			nil,
		)
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/orders/")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return helper.NewAppError(
			http.StatusBadRequest,
			"INVALID_ORDER_ID",
			"invalid order ID",
			err,
		)
	}

	order, err := h.orderService.GetOrderByID(
		r.Context(),
		id,
	)

	if err != nil {
		if errors.Is(err, orderservice.ErrOrderNotFound) {
			return helper.NewAppError(
				http.StatusNotFound,
				"ORDER_NOT_FOUND",
				"order not found",
				err,
			)
		}

		return fmt.Errorf(
			"get order by id handler: %w",
			err,
		)
	}

	helper.WriteJSON(
		w,
		http.StatusOK,
		order,
	)

	return nil
}

// GetOrdersByUserID godoc
// @Summary Get orders for current user
// @Description Get orders for the authenticated user with filter
// @Tags Orders
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Number of orders to return" default(10)
// @Param offset query int false "Number of orders to skip" default(0)
// @Param cursor query int false "ID of the last order from previous page"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} helper.AppError
// @Failure 401 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /user/orders [get]
func (h *OrderHandler) GetOrdersByUserID(
	w http.ResponseWriter,
	r *http.Request,
) error {

	if r.Method != http.MethodGet {
		return helper.NewAppError(
			http.StatusMethodNotAllowed,
			"METHOD_NOT_ALLOWED",
			"method not allowed",
			nil,
		)
	}

	userID, ok := r.Context().Value(
		jwt_validator.UserIDKey,
	).(int)

	if !ok {
		return helper.NewAppError(
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"unauthorized",
			nil,
		)
	}

	request, err := filter.DecodeRequest(r)
	if err != nil {
		return err
	}

	orders, err := h.orderService.GetOrdersByUserID(
		r.Context(),
		userID,
		request.Pagination.Limit,
		request.Pagination.Offset,
		request.Pagination.Cursor,
	)

	if err != nil {
		return fmt.Errorf(
			"get orders by user id handler: %w",
			err,
		)
	}

	page := pageview.CreatePage(
		orders,
		request.Pagination.Limit,
		func(order *domain.Order) int {
			return order.ID
		},
	)

	helper.WriteJSON(
		w,
		http.StatusOK,
		page,
	)

	return nil
}

// UpdateOrderStatus godoc
// @Summary Update order status
// @Description Update the status of an order
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Param request body domain.UpdateOrderStatusRequest true "Order status"
// @Success 204
// @Failure 400 {object} helper.AppError
// @Failure 401 {object} helper.AppError
// @Failure 404 {object} helper.AppError
// @Failure 405 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /orders/{id}/status [patch]
func (h *OrderHandler) UpdateOrderStatus(
	w http.ResponseWriter,
	r *http.Request,
) error {

	if r.Method != http.MethodPatch {
		return helper.NewAppError(
			http.StatusMethodNotAllowed,
			"METHOD_NOT_ALLOWED",
			"method not allowed",
			nil,
		)
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/orders/")
	idStr = strings.TrimSuffix(idStr, "/status")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return helper.NewAppError(
			http.StatusBadRequest,
			"INVALID_ORDER_ID",
			"invalid order ID",
			err,
		)
	}

	var req domain.UpdateOrderStatusRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return helper.NewAppError(
			http.StatusBadRequest,
			"INVALID_REQUEST_BODY",
			"invalid request body",
			err,
		)
	}

	err = h.orderService.UpdateOrderStatus(
		r.Context(),
		id,
		req.Status,
	)

	if err != nil {
		switch {
		case errors.Is(err, orderservice.ErrOrderNotFound):
			return helper.NewAppError(
				http.StatusNotFound,
				"ORDER_NOT_FOUND",
				"order not found",
				err,
			)

		case errors.Is(err, orderservice.ErrInvalidOrderStatus):
			return helper.NewAppError(
				http.StatusBadRequest,
				"INVALID_ORDER_STATUS",
				"invalid order status",
				err,
			)

		default:
			return fmt.Errorf(
				"update order status handler: %w",
				err,
			)
		}
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}

func (h *OrderHandler) CreateOrderFromCart(
	w http.ResponseWriter,
	r *http.Request,
) error {

	if r.Method != http.MethodPost {
		return helper.NewAppError(
			http.StatusMethodNotAllowed,
			"METHOD_NOT_ALLOWED",
			"method not allowed",
			nil,
		)
	}

	userID, ok := r.Context().Value(
		jwt_validator.UserIDKey,
	).(int)

	if !ok {
		return helper.NewAppError(
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"unauthorized",
			nil,
		)
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")

	if idempotencyKey == "" {
		return helper.NewAppError(
			http.StatusBadRequest,
			"MISSING_IDEMPOTENCY_KEY",
			"Idempotency-Key header is required",
			nil,
		)
	}

	if record, ok := h.orderService.GetIdempotentResponse(idempotencyKey); ok {
		w.Header().Set(
			"Content-Type",
			"application/json",
		)

		w.WriteHeader(record.StatusCode)

		_, _ = w.Write(record.Response)

		return nil
	}

	cart, err := h.cartService.GetCart(
		r.Context(),
		userID,
	)

	if err != nil {
		if errors.Is(err, cartservice.ErrCartNotFound) {
			return helper.NewAppError(
				http.StatusNotFound,
				"CART_NOT_FOUND",
				"cart not found",
				err,
			)
		}

		return fmt.Errorf(
			"get cart for order: %w",
			err,
		)
	}

	if cart == nil || len(cart.Items) == 0 {
		return helper.NewAppError(
			http.StatusBadRequest,
			"CART_EMPTY",
			"cart is empty",
			nil,
		)
	}

	order, paymentResult, err :=
		h.orderService.CreateOrderFromCart(
			r.Context(),
			domain.CreateOrderFromCartRequest{
				UserID: userID,
			},
			cart,
		)

	if err != nil {
		switch {
		case errors.Is(err, orderservice.ErrCartEmpty):
			return helper.NewAppError(
				http.StatusBadRequest,
				"CART_EMPTY",
				"cart is empty",
				err,
			)

		case errors.Is(err, orderservice.ErrItemNotFound):
			return helper.NewAppError(
				http.StatusNotFound,
				"ITEM_NOT_FOUND",
				"item not found",
				err,
			)

		case errors.Is(err, orderservice.ErrInsufficientBalance):
			return helper.NewAppError(
				http.StatusPaymentRequired,
				"INSUFFICIENT_BALANCE",
				"insufficient balance",
				err,
			)

		case errors.Is(err, orderservice.ErrInsufficientStock):
			return helper.NewAppError(
				http.StatusConflict,
				"INSUFFICIENT_STOCK",
				"insufficient stock",
				err,
			)

		default:
			return fmt.Errorf(
				"create order from cart: %w",
				err,
			)
		}
	}

	responseData := map[string]any{
		"order":   order,
		"payment": paymentResult,
	}

	responseBody, err := json.Marshal(responseData)

	if err != nil {
		return fmt.Errorf(
			"marshal order response: %w",
			err,
		)
	}

	statusCode := http.StatusCreated

	if !paymentResult.Success {
		statusCode = http.StatusPaymentRequired
	}

	h.orderService.CacheResponse(
		idempotencyKey,
		responseBody,
		statusCode,
	)

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(statusCode)

	_, _ = w.Write(responseBody)

	return nil
}

func (h *OrderHandler) CreateOrder(
	w http.ResponseWriter,
	r *http.Request,
) error {

	if r.Method != http.MethodPost {
		return helper.NewAppError(
			http.StatusMethodNotAllowed,
			"METHOD_NOT_ALLOWED",
			"method not allowed",
			nil,
		)
	}

	var req domain.CreateOrderRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return helper.NewAppError(
			http.StatusBadRequest,
			"INVALID_REQUEST_BODY",
			"invalid request body",
			err,
		)
	}

	order, paymentResult, err :=
		h.orderService.CreateOrder(
			r.Context(),
			req,
		)

	if err != nil {
		switch {
		case errors.Is(err, orderservice.ErrItemNotFound):
			return helper.NewAppError(
				http.StatusNotFound,
				"ITEM_NOT_FOUND",
				"item not found",
				err,
			)

		case errors.Is(err, orderservice.ErrInsufficientBalance):
			return helper.NewAppError(
				http.StatusPaymentRequired,
				"INSUFFICIENT_BALANCE",
				"insufficient balance",
				err,
			)

		case errors.Is(err, orderservice.ErrInsufficientStock):
			return helper.NewAppError(
				http.StatusConflict,
				"INSUFFICIENT_STOCK",
				"insufficient stock",
				err,
			)

		default:
			return fmt.Errorf(
				"create order handler: %w",
				err,
			)
		}
	}

	statusCode := http.StatusCreated

	if !paymentResult.Success {
		statusCode = http.StatusPaymentRequired
	}

	helper.WriteJSON(
		w,
		statusCode,
		map[string]any{
			"order":   order,
			"payment": paymentResult,
		},
	)

	return nil
}
