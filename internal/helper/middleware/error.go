package middleware

import (
	"checkout-api/internal/helper"
	"errors"
	"log"
	"net/http"
)

type AppHandler func(http.ResponseWriter, *http.Request) error

func ErrorMiddleware(next AppHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		err := next(w, r)
		if err == nil {
			return
		}

		encodeErrorHTTP(w, err)
	})
}

func encodeErrorHTTP(w http.ResponseWriter, err error) {

	var appErr *helper.AppError

	if errors.As(err, &appErr) {
		helper.WriteJSON(w, appErr.Status, map[string]any{
			"error": map[string]string{
				"code":    appErr.Code,
				"message": appErr.Message,
			},
		})
		return
	}

	log.Printf("internal server error: %v", err)

	helper.WriteJSON(w, http.StatusInternalServerError, map[string]any{
		"error": map[string]string{
			"code":    "INTERNAL_SERVER_ERROR",
			"message": "internal server error",
		},
	})
}
