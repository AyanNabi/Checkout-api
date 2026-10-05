package user

import (
	"checkout-api/internal/domain"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	userrepo "checkout-api/internal/repository/user"
)

type UserStore interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUser(ctx context.Context, id int) (*domain.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateUser(ctx context.Context, user *domain.User) error
	DeleteUser(ctx context.Context, id int) error
	ListUsers(ctx context.Context, limit int, offset int, cursor string) ([]*domain.User, error)
}

type SessionStore interface {
	CreateSession(ctx context.Context, session *domain.Session) error
	GetSession(ctx context.Context, id string) (*domain.Session, error)
	DeleteSession(ctx context.Context, id string) error
}
type UserService struct {
	userStore    UserStore
	sessionStore SessionStore
	jwtSecret    []byte
}

func NewUserService(
	userStore UserStore,
	sessionStore SessionStore,
	jwtSecret []byte,
) *UserService {
	return &UserService{
		userStore:    userStore,
		sessionStore: sessionStore,
		jwtSecret:    jwtSecret,
	}
}

func (s *UserService) CreateUser(ctx context.Context, user *domain.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("create user: hash password: %w", err)
	}
	user.Password = string(hashedPassword)
	if err := s.userStore.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *UserService) SignUp(ctx context.Context, req domain.SignUpRequest) (*domain.User, error) {
	if req.Username == "" {
		return nil, fmt.Errorf("sign up: %w", ErrUsernameRequired)
	}

	if req.Password == "" {
		return nil, fmt.Errorf("sign up: %w", ErrPasswordRequired)
	}

	if req.Email == "" {
		return nil, fmt.Errorf("sign up: %w", ErrEmailRequired)
	}

	existing, err := s.userStore.GetUserByUsername(ctx, req.Username)
	if err != nil && !errors.Is(err, userrepo.ErrUserNotFound) {
		return nil, fmt.Errorf(
			"sign up: check username: %w",
			err,
		)
	}

	if existing != nil {
		return nil, fmt.Errorf("sign up: %w", ErrUsernameTaken)
	}

	existing, err = s.userStore.GetUserByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, userrepo.ErrUserNotFound) {
		return nil, fmt.Errorf(
			"sign up: check email: %w",
			err,
		)
	}

	if existing != nil {
		return nil, fmt.Errorf("sign up: %w", ErrEmailTaken)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"sign up: hash password: %w",
			err,
		)
	}

	user := &domain.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
	}

	if err := s.userStore.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf(
			"sign up: create user: %w",
			err,
		)
	}

	return user, nil
}
func (s *UserService) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	user, err := s.userStore.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return nil, fmt.Errorf("get user by id: %w", ErrUserNotFound)
		}

		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, user *domain.User) error {
	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("update user: hash password: %w", err)
		}
		user.Password = string(hashedPassword)
	}
	if err := s.userStore.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (s *UserService) DeleteUser(ctx context.Context, id int) error {
	if err := s.userStore.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
func (s *UserService) GetAllUsers(ctx context.Context, limit int, offset int, cursor string) ([]*domain.User, error) {

	users, err := s.userStore.ListUsers(
		ctx,
		limit,
		offset,
		cursor,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"get all users: %w",
			err,
		)
	}

	return users, nil
}

func (s *UserService) Login(ctx context.Context, req domain.LoginRequest) (string, string, error) {
	if req.Username == "" {
		return "", "", fmt.Errorf("login: %w", ErrUsernameRequired)
	}
	if req.Password == "" {
		return "", "", fmt.Errorf("login: %w", ErrPasswordRequired)
	}

	user, err := s.userStore.GetUserByUsername(ctx, req.Username)

	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return "", "", fmt.Errorf(
				"login: %w",
				ErrInvalidCredentials,
			)
		}

		return "", "", fmt.Errorf(
			"login: get user: %w",
			err,
		)
	}

	if user == nil {
		return "", "", fmt.Errorf(
			"login: %w",
			ErrInvalidCredentials,
		)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return "", "", fmt.Errorf("login: %w", ErrInvalidCredentials)
	}

	accessClaims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("login: sign access token: %w", err)
	}

	refreshClaims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("login: %w", ErrRefreshTokenGeneration)
	}

	session := &domain.Session{
		ID:        refreshTokenString,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.sessionStore.CreateSession(ctx, session); err != nil {
		return "", "", fmt.Errorf("login: create session: %w", err)
	}

	return accessTokenString, refreshTokenString, nil
}

func (s *UserService) RefreshToken(
	ctx context.Context,
	refreshTokenString string,
) (string, string, error) {

	session, err := s.sessionStore.GetSession(ctx, refreshTokenString)
	if err != nil || session == nil {
		return "", "", fmt.Errorf("refresh token: %w", ErrInvalidRefreshToken)
	}

	if time.Now().After(session.ExpiresAt) {
		if err := s.sessionStore.DeleteSession(ctx, refreshTokenString); err != nil {
			return "", "", fmt.Errorf("refresh token: delete expired session: %w", err)
		}
		return "", "", fmt.Errorf("refresh token: %w", ErrInvalidRefreshToken)
	}

	user, err := s.userStore.GetUser(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return "", "", fmt.Errorf(
				"refresh token: %w",
				ErrUserNotFound,
			)
		}
		return "", "", fmt.Errorf("refresh token: get user: %w", err)
	}
	if user == nil {
		return "", "", fmt.Errorf(
			"refresh token: %w",
			ErrUserNotFound,
		)
	}

	accessClaims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	}

	accessToken := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		accessClaims,
	)

	newAccessToken, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("refresh token: sign new access token: %w", err)
	}

	refreshClaims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	refreshToken := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		refreshClaims,
	)

	newRefreshToken, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("refresh token: %w", ErrRefreshTokenGeneration)
	}

	if err := s.sessionStore.DeleteSession(ctx, refreshTokenString); err != nil {
		return "", "", fmt.Errorf("refresh token: delete old session: %w", err)
	}

	newSession := &domain.Session{
		ID:        newRefreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.sessionStore.CreateSession(ctx, newSession); err != nil {
		return "", "", fmt.Errorf("refresh token: create new session: %w", err)
	}

	return newAccessToken, newRefreshToken, nil
}
