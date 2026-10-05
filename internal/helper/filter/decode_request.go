package helper

import (
	newapperror "checkout-api/internal/helper"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type Pagination struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Cursor string `json:"cursor"`
}

type ItemFilter struct {
	Min      int    `json:"min"`
	Max      int    `json:"max"`
	Category string `json:"category"`
}

type Request struct {
	Pagination *Pagination `json:"pagination"`
	Filter     *ItemFilter `json:"filter"`
}

func DecodeRequest(r *http.Request) (Request, error) {
	request := Request{
		Pagination: &Pagination{Limit: 10},
		Filter:     &ItemFilter{},
	}

	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return Request{}, newapperror.NewAppError(http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", err)
		}
	}

	query := r.URL.Query()
	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			return Request{}, newapperror.NewAppError(http.StatusBadRequest, "INVALID_LIMIT", "limit must be a positive integer", err)
		}
		request.Pagination.Limit = limit
	}
	if value := query.Get("offset"); value != "" {
		offset, err := strconv.Atoi(value)
		if err != nil {
			return Request{}, newapperror.NewAppError(http.StatusBadRequest, "INVALID_OFFSET", "offset must be a non-negative integer", err)
		}
		request.Pagination.Offset = offset
	}
	if value := query.Get("cursor"); value != "" {
		request.Pagination.Cursor = value
	}
	if value := query.Get("min"); value != "" {
		min, err := strconv.Atoi(value)
		if err != nil {
			return Request{}, newapperror.NewAppError(http.StatusBadRequest, "INVALID_MIN", "min price must be a non-negative integer", err)
		}
		request.Filter.Min = min
	}
	if value := query.Get("max"); value != "" {
		max, err := strconv.Atoi(value)
		if err != nil {
			return Request{}, newapperror.NewAppError(http.StatusBadRequest, "INVALID_MAX", "max price must be a non-negative integer", err)
		}
		request.Filter.Max = max
	}
	if value := query.Get("category"); value != "" {
		request.Filter.Category = value
	}

	request.Filter.Category = strings.TrimSpace(request.Filter.Category)

	if err := request.ValidateRequest(); err != nil {
		return Request{}, err
	}

	return request, nil
}
