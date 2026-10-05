package item

import (
	"checkout-api/internal/domain"
	"checkout-api/internal/helper"
	filter "checkout-api/internal/helper/filter"
	pageview "checkout-api/internal/helper/page"
	itemservice "checkout-api/internal/services/item"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type ItemHandler struct {
	service *itemservice.ItemService
}

func NewItemHandler(s *itemservice.ItemService) *ItemHandler {
	return &ItemHandler{service: s}
}

// GetItems godoc
// @Summary Get all items
// @Description Get items with pagination and price filtering
// @Tags Items
// @Produce json
// @Param limit query int false "Number of items per page"
// @Param cursor query string false "Cursor for cursor-based pagination"
// @Param min query int false "Minimum price"
// @Param max query int false "Maximum price"
// @Success 200 {object} domain.ItemPage
// @Failure 400 {object} helper.AppError
// @Failure 405 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /items [get]
func (h *ItemHandler) GetItems(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return helper.NewAppError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	}

	request, err := filter.DecodeRequest(r)
	if err != nil {
		return err
	}

	items, err := h.service.GetAllItems(r.Context(), request)
	if err != nil {
		return fmt.Errorf("get all items handler: %w", err)
	}

	page := pageview.CreatePage(
		items,
		request.Pagination.Limit,
		func(item *domain.Item) int {
			return item.ID
		},
	)
	if err != nil {
		return fmt.Errorf("create page: %w", err)
	}
	helper.WriteJSON(w, http.StatusOK, page)
	return nil
}

// GetItemByID godoc
// @Summary Get item by ID
// @Description Get a single item by its ID
// @Tags Items
// @Produce json
// @Param id path int true "Item ID"
// @Success 200 {object} domain.Item
// @Failure 400 {string} string "Invalid item ID"
// @Failure 404 {string} string "Item not found"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {string} string "Internal server error"
// @Router /items/{id} [get]
func (h *ItemHandler) GetItemByID(w http.ResponseWriter, r *http.Request) error {

	if r.Method != http.MethodGet {
		return helper.NewAppError(
			http.StatusMethodNotAllowed,
			"METHOD_NOT_ALLOWED",
			"method not allowed",
			nil,
		)
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/items/")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return helper.NewAppError(
			http.StatusBadRequest,
			"INVALID_ITEM_ID",
			"invalid item ID",
			err,
		)
	}

	item, err := h.service.GetItemByID(r.Context(), id)

	if err != nil {
		if errors.Is(err, itemservice.ErrItemNotFound) {
			return helper.NewAppError(
				http.StatusNotFound,
				"ITEM_NOT_FOUND",
				"item not found",
				err,
			)
		}

		return fmt.Errorf("get item by id: %w", err)
	}

	helper.WriteJSON(w, http.StatusOK, item)

	return nil
}

// GetItems godoc
// @Summary Get all items
// @Description Get all available items
// @Tags Items
// @Produce json
// @Success 200 {array} domain.Item
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {string} string "Internal server error"
// @Router /items [get]
