package user

import (
	"checkout-api/internal/domain"
	"checkout-api/internal/helper"
	filter "checkout-api/internal/helper/filter"
	pageview "checkout-api/internal/helper/page"
	userservice "checkout-api/internal/services/user"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type UserHandler struct {
	service *userservice.UserService
}

func NewUserHandler(s *userservice.UserService) *UserHandler {
	return &UserHandler{service: s}
}

// CreateUser godoc
// @Summary Create a user
// @Description Create a new user
// @Tags Users
// @Accept json
// @Produce json
// @Param user body domain.User true "User information"
// @Success 201 {object} domain.User
// @Failure 400 {object} helper.AppError
// @Failure 405 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /users [post]
func (h *UserHandler) CreateUser(
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

	var user domain.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return helper.NewAppError(
			http.StatusBadRequest,
			"INVALID_REQUEST_BODY",
			"invalid request body",
			err,
		)
	}

	if err := h.service.CreateUser(r.Context(), &user); err != nil {
		return fmt.Errorf("create user handler: %w", err)
	}

	helper.WriteJSON(w, http.StatusCreated, user)

	return nil
}

// SignUp godoc
// @Summary Sign up
// @Description Register a new user account
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body domain.SignUpRequest true "Sign up information"
// @Success 201 {object} domain.User
// @Failure 400 {object} helper.AppError
// @Failure 409 {object} helper.AppError
// @Failure 405 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /signup [post]
func (h *UserHandler) SignUp(
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

	var req domain.SignUpRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return helper.NewAppError(
			http.StatusBadRequest,
			"INVALID_REQUEST_BODY",
			"invalid request body",
			err,
		)
	}

	user, err := h.service.SignUp(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, userservice.ErrUsernameRequired):
			return helper.NewAppError(
				http.StatusBadRequest,
				"USERNAME_REQUIRED",
				"username is required",
				err,
			)

		case errors.Is(err, userservice.ErrPasswordRequired):
			return helper.NewAppError(
				http.StatusBadRequest,
				"PASSWORD_REQUIRED",
				"password is required",
				err,
			)

		case errors.Is(err, userservice.ErrEmailRequired):
			return helper.NewAppError(
				http.StatusBadRequest,
				"EMAIL_REQUIRED",
				"email is required",
				err,
			)

		case errors.Is(err, userservice.ErrEmailTaken):
			return helper.NewAppError(
				http.StatusConflict,
				"EMAIL_TAKEN",
				"email is already taken",
				err,
			)

		case errors.Is(err, userservice.ErrUsernameTaken):
			return helper.NewAppError(
				http.StatusConflict,
				"Username_TAKEN",
				"username is already taken",
				err,
			)

		default:
			return fmt.Errorf("signup handler: %w", err)
		}
	}

	helper.WriteJSON(w, http.StatusCreated, user)

	return nil
}

// Login godoc
// @Summary Login
// @Description Authenticate a user and return an access token.
// A refresh token is stored in an HttpOnly cookie.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body domain.LoginRequest true "Login credentials"
// @Success 200 {object} map[string]string
// @Failure 400 {object} helper.AppError
// @Failure 401 {object} helper.AppError
// @Failure 405 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /login [post]
func (h *UserHandler) Login(
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

	var req domain.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return helper.NewAppError(
			http.StatusBadRequest,
			"INVALID_REQUEST_BODY",
			"invalid request body",
			err,
		)
	}

	accessToken, refreshToken, err := h.service.Login(
		r.Context(),
		req,
	)

	if err != nil {
		switch {
		case errors.Is(err, userservice.ErrUsernameRequired):
			return helper.NewAppError(
				http.StatusBadRequest,
				"USERNAME_REQUIRED",
				"username is required",
				err,
			)

		case errors.Is(err, userservice.ErrPasswordRequired):
			return helper.NewAppError(
				http.StatusBadRequest,
				"PASSWORD_REQUIRED",
				"password is required",
				err,
			)

		case errors.Is(err, userservice.ErrInvalidCredentials):
			return helper.NewAppError(
				http.StatusUnauthorized,
				"INVALID_CREDENTIALS",
				"invalid credentials",
				err,
			)

		default:
			return fmt.Errorf("login handler: %w", err)
		}
	}

	setRefreshTokenCookie(w, refreshToken)

	helper.WriteJSON(w, http.StatusOK, map[string]string{
		"access_token": accessToken,
	})

	return nil
}

// GetUser godoc
// @Summary Get user by ID
// @Description Get a single user by their ID
// @Tags Users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} domain.User
// @Failure 400 {object} helper.AppError
// @Failure 404 {object} helper.AppError
// @Failure 405 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(
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

	idStr := strings.TrimPrefix(r.URL.Path, "/users/")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return helper.NewAppError(
			http.StatusBadRequest,
			"INVALID_USER_ID",
			"invalid user ID",
			err,
		)
	}

	user, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, userservice.ErrUserNotFound) {
			return helper.NewAppError(
				http.StatusNotFound,
				"USER_NOT_FOUND",
				"user not found",
				err,
			)
		}

		return fmt.Errorf(
			"get user handler: %w",
			err,
		)
	}

	helper.WriteJSON(w, http.StatusOK, user)

	return nil
}

// UpdateUser godoc
// @Summary Update user
// @Description Update an existing user's information
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body domain.User true "Updated user information"
// @Success 200 {object} domain.User
// @Failure 400 {object} helper.AppError
// @Failure 404 {object} helper.AppError
// @Failure 405 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(
	w http.ResponseWriter,
	r *http.Request,
) error {

	if r.Method != http.MethodPut {
		return helper.NewAppError(
			http.StatusMethodNotAllowed,
			"METHOD_NOT_ALLOWED",
			"method not allowed",
			nil,
		)
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/users/")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return helper.NewAppError(
			http.StatusBadRequest,
			"INVALID_USER_ID",
			"invalid user ID",
			err,
		)
	}

	var user domain.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return helper.NewAppError(
			http.StatusBadRequest,
			"INVALID_REQUEST_BODY",
			"invalid request body",
			err,
		)
	}

	user.ID = id

	if err := h.service.UpdateUser(
		r.Context(),
		&user,
	); err != nil {

		if errors.Is(err, userservice.ErrUserNotFound) {
			return helper.NewAppError(
				http.StatusNotFound,
				"USER_NOT_FOUND",
				"user not found",
				err,
			)
		}

		return fmt.Errorf(
			"update user handler: %w",
			err,
		)
	}

	helper.WriteJSON(w, http.StatusOK, user)

	return nil
}

// DeleteUser godoc
// @Summary Delete user
// @Description Delete a user by their ID
// @Tags Users
// @Produce json
// @Param id path int true "User ID"
// @Success 204
// @Failure 400 {object} helper.AppError
// @Failure 404 {object} helper.AppError
// @Failure 405 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(
	w http.ResponseWriter,
	r *http.Request,
) error {

	if r.Method != http.MethodDelete {
		return helper.NewAppError(
			http.StatusMethodNotAllowed,
			"METHOD_NOT_ALLOWED",
			"method not allowed",
			nil,
		)
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/users/")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return helper.NewAppError(
			http.StatusBadRequest,
			"INVALID_USER_ID",
			"invalid user ID",
			err,
		)
	}

	if err := h.service.DeleteUser(
		r.Context(),
		id,
	); err != nil {

		if errors.Is(err, userservice.ErrUserNotFound) {
			return helper.NewAppError(
				http.StatusNotFound,
				"USER_NOT_FOUND",
				"user not found",
				err,
			)
		}

		return fmt.Errorf(
			"delete user handler: %w",
			err,
		)
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}

// GetAllUsers godoc
// @Summary Get all users
// @Description Get all users with offset-based or cursor-based filter
// @Tags Users
// @Produce json
// @Param limit query int false "Number of users to return" default(10)
// @Param offset query int false "Number of users to skip" default(0)
// @Param cursor query int false "ID of the last user from the previous page"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} helper.AppError
// @Failure 405 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /users [get]
func (h *UserHandler) GetAllUsers(
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

	request, err := filter.DecodeRequest(r)
	if err != nil {
		return fmt.Errorf(
			"parse filter: %w",
			err,
		)
	}

	users, err := h.service.GetAllUsers(
		r.Context(),
		request.Pagination.Limit,
		request.Pagination.Offset,
		request.Pagination.Cursor,
	)

	if err != nil {
		return fmt.Errorf(
			"get all users handler: %w",
			err,
		)
	}

	page := pageview.CreatePage(
		users,
		request.Pagination.Limit,
		func(user *domain.User) int {
			return user.ID
		},
	)

	helper.WriteJSON(
		w,
		http.StatusOK,
		page,
	)

	return nil
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Generate a new access token using the refresh token stored in an HttpOnly cookie
// @Tags Authentication
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} helper.AppError
// @Failure 405 {object} helper.AppError
// @Failure 500 {object} helper.AppError
// @Router /token [post]
func (h *UserHandler) RefreshToken(
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

	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return helper.NewAppError(
			http.StatusUnauthorized,
			"REFRESH_TOKEN_REQUIRED",
			"refresh token cookie required",
			err,
		)
	}

	accessToken, newRefreshToken, err :=
		h.service.RefreshToken(
			r.Context(),
			cookie.Value,
		)

	if err != nil {
		deleteRefreshTokenCookie(w)

		switch {
		case errors.Is(err, userservice.ErrInvalidRefreshToken):
			return helper.NewAppError(
				http.StatusUnauthorized,
				"INVALID_REFRESH_TOKEN",
				"invalid refresh token",
				err,
			)

		case errors.Is(err, userservice.ErrUserNotFound):
			return helper.NewAppError(
				http.StatusUnauthorized,
				"USER_NOT_FOUND",
				"user not found",
				err,
			)

		default:
			return fmt.Errorf(
				"refresh token handler: %w",
				err,
			)
		}
	}

	setRefreshTokenCookie(w, newRefreshToken)

	helper.WriteJSON(w, http.StatusOK, map[string]string{
		"access_token": accessToken,
	})

	return nil
}

// Logout godoc
// @Summary Logout
// @Description Log out the current user by deleting the refresh token cookie
// @Tags Authentication
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 405 {object} helper.AppError
// @Router /logout [post]
func (h *UserHandler) Logout(
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

	deleteRefreshTokenCookie(w)

	helper.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "successfully logged out",
	})

	return nil
}

func setRefreshTokenCookie(
	w http.ResponseWriter,
	token string,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

func deleteRefreshTokenCookie(
	w http.ResponseWriter,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
