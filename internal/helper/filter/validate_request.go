package helper

import (
	newapperror "checkout-api/internal/helper"
	"net/http"
)

func (r Request) ValidateRequest() error {
	if r.Pagination.Limit <= 0 {
		return newapperror.NewAppError(http.StatusBadRequest, "INVALID_LIMIT", "limit must be a positive integer", nil)
	}

	//if r.Pagination.Cursor < 0 {
	//	return newapperror.NewAppError(http.StatusBadRequest, "INVALID_CURSOR", "cursor must be a non-negative integer", nil)
	//}

	if r.Filter.Min < 0 {
		return newapperror.NewAppError(http.StatusBadRequest, "INVALID_MIN", "min price must be a non-negative integer", nil)
	}

	if r.Filter.Max < 0 {
		return newapperror.NewAppError(http.StatusBadRequest, "INVALID_MAX", "max price must be a non-negative integer", nil)
	}

	if r.Filter.Min > 0 && r.Filter.Max > 0 && r.Filter.Min > r.Filter.Max {
		return newapperror.NewAppError(http.StatusBadRequest, "INVALID_PRICE_RANGE", "min price cannot be greater than max price", nil)
	}

	return nil
}
