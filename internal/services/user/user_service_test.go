package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"checkout-api/internal/domain"
	userrepo "checkout-api/internal/repository/user"

	"golang.org/x/crypto/bcrypt"
)

type fakeUserStore struct {
	createUserFunc        func(context.Context, *domain.User) error
	getUserFunc           func(context.Context, int) (*domain.User, error)
	getUserByUsernameFunc func(context.Context, string) (*domain.User, error)
	getUserByEmailFunc    func(context.Context, string) (*domain.User, error)
	updateUserFunc        func(context.Context, *domain.User) error
	deleteUserFunc        func(context.Context, int) error
	listUsersFunc         func(context.Context, int, int, string) ([]*domain.User, error)
}

func (f *fakeUserStore) CreateUser(
	ctx context.Context,
	user *domain.User,
) error {
	if f.createUserFunc != nil {
		return f.createUserFunc(ctx, user)
	}

	return nil
}

func (f *fakeUserStore) GetUser(
	ctx context.Context,
	id int,
) (*domain.User, error) {
	if f.getUserFunc != nil {
		return f.getUserFunc(ctx, id)
	}

	return nil, nil
}

func (f *fakeUserStore) GetUserByUsername(
	ctx context.Context,
	username string,
) (*domain.User, error) {
	if f.getUserByUsernameFunc != nil {
		return f.getUserByUsernameFunc(ctx, username)
	}

	return nil, userrepo.ErrUserNotFound
}

func (f *fakeUserStore) GetUserByEmail(
	ctx context.Context,
	email string,
) (*domain.User, error) {
	if f.getUserByEmailFunc != nil {
		return f.getUserByEmailFunc(ctx, email)
	}

	return nil, userrepo.ErrUserNotFound
}

func (f *fakeUserStore) UpdateUser(
	ctx context.Context,
	user *domain.User,
) error {
	if f.updateUserFunc != nil {
		return f.updateUserFunc(ctx, user)
	}

	return nil
}

func (f *fakeUserStore) DeleteUser(
	ctx context.Context,
	id int,
) error {
	if f.deleteUserFunc != nil {
		return f.deleteUserFunc(ctx, id)
	}

	return nil
}

func (f *fakeUserStore) ListUsers(
	ctx context.Context,
	limit int,
	offset int,
	cursor string,
) ([]*domain.User, error) {
	if f.listUsersFunc != nil {
		return f.listUsersFunc(ctx, limit, offset, cursor)
	}

	return nil, nil
}

type fakeSessionStore struct {
	createSessionFunc func(context.Context, *domain.Session) error
	getSessionFunc    func(context.Context, string) (*domain.Session, error)
	deleteSessionFunc func(context.Context, string) error
}

func (f *fakeSessionStore) CreateSession(
	ctx context.Context,
	session *domain.Session,
) error {
	if f.createSessionFunc != nil {
		return f.createSessionFunc(ctx, session)
	}

	return nil
}

func (f *fakeSessionStore) GetSession(
	ctx context.Context,
	id string,
) (*domain.Session, error) {
	if f.getSessionFunc != nil {
		return f.getSessionFunc(ctx, id)
	}

	return nil, nil
}

func (f *fakeSessionStore) DeleteSession(
	ctx context.Context,
	id string,
) error {
	if f.deleteSessionFunc != nil {
		return f.deleteSessionFunc(ctx, id)
	}

	return nil
}

func newTestUserService(
	userStore *fakeUserStore,
	sessionStore *fakeSessionStore,
) *UserService {
	return NewUserService(
		userStore,
		sessionStore,
		[]byte("test-secret"),
	)
}

func validTestUser() *domain.User {
	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte("password123"),
		bcrypt.DefaultCost,
	)

	if err != nil {
		panic(err)
	}

	return &domain.User{
		ID:        1,
		Username:  "ayan",
		Email:     "ayan@example.com",
		Password:  string(passwordHash),
		CreatedAt: time.Now(),
	}
}

func TestUserService_CreateUser(t *testing.T) {
	storeErr := errors.New("database error")

	tests := []struct {
		name    string
		user    *domain.User
		setup   func(*fakeUserStore)
		wantErr error
	}{
		{
			name: "success",
			user: &domain.User{
				Username: "ayan",
				Email:    "ayan@example.com",
				Password: "password123",
			},
		},
		{
			name: "store error",
			user: &domain.User{
				Username: "ayan",
				Email:    "ayan@example.com",
				Password: "password123",
			},
			setup: func(store *fakeUserStore) {
				store.createUserFunc = func(
					ctx context.Context,
					user *domain.User,
				) error {
					return storeErr
				}
			},
			wantErr: storeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeUserStore{}

			if tt.setup != nil {
				tt.setup(store)
			}

			service := newTestUserService(
				store,
				&fakeSessionStore{},
			)

			originalPassword := tt.user.Password

			err := service.CreateUser(
				context.Background(),
				tt.user,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.user.Password == originalPassword {
				t.Fatal("password was not hashed")
			}

			if tt.user.Password == "password123" {
				t.Fatal("password must not be stored as plain text")
			}
		})
	}
}

func TestUserService_SignUp(t *testing.T) {
	storeErr := errors.New("database error")

	tests := []struct {
		name    string
		req     domain.SignUpRequest
		setup   func(*fakeUserStore)
		wantErr error
	}{
		{
			name: "success",
			req: domain.SignUpRequest{
				Username: "ayan",
				Email:    "ayan@example.com",
				Password: "password123",
			},
		},
		{
			name: "username required",
			req: domain.SignUpRequest{
				Email:    "ayan@example.com",
				Password: "password123",
			},
			wantErr: ErrUsernameRequired,
		},
		{
			name: "password required",
			req: domain.SignUpRequest{
				Username: "ayan",
				Email:    "ayan@example.com",
			},
			wantErr: ErrPasswordRequired,
		},
		{
			name: "email required",
			req: domain.SignUpRequest{
				Username: "ayan",
				Password: "password123",
			},
			wantErr: ErrEmailRequired,
		},
		{
			name: "username taken",
			req: domain.SignUpRequest{
				Username: "ayan",
				Email:    "ayan@example.com",
				Password: "password123",
			},
			setup: func(store *fakeUserStore) {
				store.getUserByUsernameFunc = func(
					ctx context.Context,
					username string,
				) (*domain.User, error) {
					return validTestUser(), nil
				}
			},
			wantErr: ErrUsernameTaken,
		},
		{
			name: "email taken",
			req: domain.SignUpRequest{
				Username: "ayan",
				Email:    "ayan@example.com",
				Password: "password123",
			},
			setup: func(store *fakeUserStore) {
				store.getUserByUsernameFunc = func(
					ctx context.Context,
					username string,
				) (*domain.User, error) {
					return nil, userrepo.ErrUserNotFound
				}

				store.getUserByEmailFunc = func(
					ctx context.Context,
					email string,
				) (*domain.User, error) {
					return validTestUser(), nil
				}
			},
			wantErr: ErrEmailTaken,
		},
		{
			name: "username lookup error",
			req: domain.SignUpRequest{
				Username: "ayan",
				Email:    "ayan@example.com",
				Password: "password123",
			},
			setup: func(store *fakeUserStore) {
				store.getUserByUsernameFunc = func(
					ctx context.Context,
					username string,
				) (*domain.User, error) {
					return nil, storeErr
				}
			},
			wantErr: storeErr,
		},
		{
			name: "email lookup error",
			req: domain.SignUpRequest{
				Username: "ayan",
				Email:    "ayan@example.com",
				Password: "password123",
			},
			setup: func(store *fakeUserStore) {
				store.getUserByUsernameFunc = func(
					ctx context.Context,
					username string,
				) (*domain.User, error) {
					return nil, userrepo.ErrUserNotFound
				}

				store.getUserByEmailFunc = func(
					ctx context.Context,
					email string,
				) (*domain.User, error) {
					return nil, storeErr
				}
			},
			wantErr: storeErr,
		},
		{
			name: "create user error",
			req: domain.SignUpRequest{
				Username: "ayan",
				Email:    "ayan@example.com",
				Password: "password123",
			},
			setup: func(store *fakeUserStore) {
				store.createUserFunc = func(
					ctx context.Context,
					user *domain.User,
				) error {
					return storeErr
				}
			},
			wantErr: storeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeUserStore{}

			if tt.setup != nil {
				tt.setup(store)
			}

			service := newTestUserService(
				store,
				&fakeSessionStore{},
			)

			user, err := service.SignUp(
				context.Background(),
				tt.req,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				if user != nil {
					t.Fatal("expected nil user")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if user == nil {
				t.Fatal("expected user")
			}

			if user.Username != tt.req.Username {
				t.Fatalf(
					"username = %q, want %q",
					user.Username,
					tt.req.Username,
				)
			}

			if user.Email != tt.req.Email {
				t.Fatalf(
					"email = %q, want %q",
					user.Email,
					tt.req.Email,
				)
			}

			if user.Password == tt.req.Password {
				t.Fatal("password was not hashed")
			}
		})
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	storeErr := errors.New("database error")

	tests := []struct {
		name    string
		setup   func(*fakeUserStore)
		wantErr error
	}{
		{
			name: "success",
			setup: func(store *fakeUserStore) {
				store.getUserFunc = func(
					ctx context.Context,
					id int,
				) (*domain.User, error) {
					return validTestUser(), nil
				}
			},
		},
		{
			name: "user not found",
			setup: func(store *fakeUserStore) {
				store.getUserFunc = func(
					ctx context.Context,
					id int,
				) (*domain.User, error) {
					return nil, userrepo.ErrUserNotFound
				}
			},
			wantErr: ErrUserNotFound,
		},
		{
			name: "store error",
			setup: func(store *fakeUserStore) {
				store.getUserFunc = func(
					ctx context.Context,
					id int,
				) (*domain.User, error) {
					return nil, storeErr
				}
			},
			wantErr: storeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeUserStore{}

			if tt.setup != nil {
				tt.setup(store)
			}

			service := newTestUserService(
				store,
				&fakeSessionStore{},
			)

			user, err := service.GetUserByID(
				context.Background(),
				1,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if user == nil {
				t.Fatal("expected user")
			}
		})
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	storeErr := errors.New("database error")

	tests := []struct {
		name      string
		user      *domain.User
		setup     func(*fakeUserStore)
		wantErr   error
		checkPass bool
	}{
		{
			name: "success with password",
			user: &domain.User{
				ID:       1,
				Username: "ayan",
				Email:    "ayan@example.com",
				Password: "password123",
			},
			checkPass: true,
		},
		{
			name: "success without password",
			user: &domain.User{
				ID:       1,
				Username: "ayan",
				Email:    "ayan@example.com",
			},
		},
		{
			name: "store error",
			user: &domain.User{
				ID:       1,
				Username: "ayan",
				Email:    "ayan@example.com",
			},
			setup: func(store *fakeUserStore) {
				store.updateUserFunc = func(
					ctx context.Context,
					user *domain.User,
				) error {
					return storeErr
				}
			},
			wantErr: storeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeUserStore{}

			if tt.setup != nil {
				tt.setup(store)
			}

			service := newTestUserService(
				store,
				&fakeSessionStore{},
			)

			originalPassword := tt.user.Password

			err := service.UpdateUser(
				context.Background(),
				tt.user,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkPass &&
				tt.user.Password == originalPassword {
				t.Fatal("password was not hashed")
			}
		})
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	storeErr := errors.New("database error")

	tests := []struct {
		name    string
		setup   func(*fakeUserStore)
		wantErr error
	}{
		{
			name: "success",
		},
		{
			name: "store error",
			setup: func(store *fakeUserStore) {
				store.deleteUserFunc = func(
					ctx context.Context,
					id int,
				) error {
					return storeErr
				}
			},
			wantErr: storeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeUserStore{}

			if tt.setup != nil {
				tt.setup(store)
			}

			service := newTestUserService(
				store,
				&fakeSessionStore{},
			)

			err := service.DeleteUser(
				context.Background(),
				1,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUserService_GetAllUsers(t *testing.T) {
	storeErr := errors.New("database error")

	tests := []struct {
		name    string
		setup   func(*fakeUserStore)
		wantErr error
	}{
		{
			name: "success",
			setup: func(store *fakeUserStore) {
				store.listUsersFunc = func(
					ctx context.Context,
					limit int,
					offset int,
					cursor string,
				) ([]*domain.User, error) {
					return []*domain.User{
						validTestUser(),
					}, nil
				}
			},
		},
		{
			name: "store error",
			setup: func(store *fakeUserStore) {
				store.listUsersFunc = func(
					ctx context.Context,
					limit int,
					offset int,
					cursor string,
				) ([]*domain.User, error) {
					return nil, storeErr
				}
			},
			wantErr: storeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeUserStore{}

			if tt.setup != nil {
				tt.setup(store)
			}

			service := newTestUserService(
				store,
				&fakeSessionStore{},
			)

			users, err := service.GetAllUsers(
				context.Background(),
				10,
				0,
				"",
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(users) != 1 {
				t.Fatalf(
					"users length = %d, want 1",
					len(users),
				)
			}
		})
	}
}

func TestUserService_Login(t *testing.T) {
	dbErr := errors.New("database error")
	sessionErr := errors.New("session database error")

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte("password123"),
		bcrypt.DefaultCost,
	)

	if err != nil {
		t.Fatalf(
			"failed to generate password hash: %v",
			err,
		)
	}

	validUser := &domain.User{
		ID:        1,
		Username:  "ayan",
		Email:     "ayan@example.com",
		Password:  string(passwordHash),
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name    string
		req     domain.LoginRequest
		setup   func(*fakeUserStore, *fakeSessionStore)
		wantErr error
	}{
		{
			name: "success",
			req: domain.LoginRequest{
				Username: "ayan",
				Password: "password123",
			},
			setup: func(
				userStore *fakeUserStore,
				sessionStore *fakeSessionStore,
			) {
				userStore.getUserByUsernameFunc = func(
					ctx context.Context,
					username string,
				) (*domain.User, error) {
					return validUser, nil
				}

				sessionStore.createSessionFunc = func(
					ctx context.Context,
					session *domain.Session,
				) error {
					return nil
				}
			},
		},
		{
			name: "username required",
			req: domain.LoginRequest{
				Password: "password123",
			},
			wantErr: ErrUsernameRequired,
		},
		{
			name: "password required",
			req: domain.LoginRequest{
				Username: "ayan",
			},
			wantErr: ErrPasswordRequired,
		},
		{
			name: "user not found",
			req: domain.LoginRequest{
				Username: "ayan",
				Password: "password123",
			},
			setup: func(
				userStore *fakeUserStore,
				sessionStore *fakeSessionStore,
			) {
				userStore.getUserByUsernameFunc = func(
					ctx context.Context,
					username string,
				) (*domain.User, error) {
					return nil, userrepo.ErrUserNotFound
				}
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "wrong password",
			req: domain.LoginRequest{
				Username: "ayan",
				Password: "wrong-password",
			},
			setup: func(
				userStore *fakeUserStore,
				sessionStore *fakeSessionStore,
			) {
				userStore.getUserByUsernameFunc = func(
					ctx context.Context,
					username string,
				) (*domain.User, error) {
					return validUser, nil
				}
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "store error",
			req: domain.LoginRequest{
				Username: "ayan",
				Password: "password123",
			},
			setup: func(
				userStore *fakeUserStore,
				sessionStore *fakeSessionStore,
			) {
				userStore.getUserByUsernameFunc = func(
					ctx context.Context,
					username string,
				) (*domain.User, error) {
					return nil, dbErr
				}
			},
			wantErr: dbErr,
		},
		{
			name: "session creation error",
			req: domain.LoginRequest{
				Username: "ayan",
				Password: "password123",
			},
			setup: func(
				userStore *fakeUserStore,
				sessionStore *fakeSessionStore,
			) {
				userStore.getUserByUsernameFunc = func(
					ctx context.Context,
					username string,
				) (*domain.User, error) {
					return validUser, nil
				}

				sessionStore.createSessionFunc = func(
					ctx context.Context,
					session *domain.Session,
				) error {
					return sessionErr
				}
			},
			wantErr: sessionErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userStore := &fakeUserStore{}
			sessionStore := &fakeSessionStore{}

			if tt.setup != nil {
				tt.setup(
					userStore,
					sessionStore,
				)
			}

			service := newTestUserService(
				userStore,
				sessionStore,
			)

			accessToken, refreshToken, err := service.Login(
				context.Background(),
				tt.req,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"unexpected error: %v",
					err,
				)
			}

			if accessToken == "" {
				t.Fatal("expected access token")
			}

			if refreshToken == "" {
				t.Fatal("expected refresh token")
			}
		})
	}
}

func TestUserService_RefreshToken(t *testing.T) {
	sessionErr := errors.New("session database error")

	tests := []struct {
		name    string
		token   string
		setup   func(*fakeUserStore, *fakeSessionStore)
		wantErr error
	}{
		{
			name:  "invalid token",
			token: "invalid-token",
			setup: func(
				userStore *fakeUserStore,
				sessionStore *fakeSessionStore,
			) {
				sessionStore.getSessionFunc = func(
					ctx context.Context,
					id string,
				) (*domain.Session, error) {
					return nil, sessionErr
				}
			},
			wantErr: ErrInvalidRefreshToken,
		},
		{
			name:  "expired session",
			token: "expired-token",
			setup: func(
				userStore *fakeUserStore,
				sessionStore *fakeSessionStore,
			) {
				sessionStore.getSessionFunc = func(
					ctx context.Context,
					id string,
				) (*domain.Session, error) {
					return &domain.Session{
						ID:        id,
						UserID:    1,
						ExpiresAt: time.Now().Add(-time.Hour),
					}, nil
				}

				sessionStore.deleteSessionFunc = func(
					ctx context.Context,
					id string,
				) error {
					return nil
				}
			},
			wantErr: ErrInvalidRefreshToken,
		},
		{
			name:  "user not found",
			token: "refresh-token",
			setup: func(
				userStore *fakeUserStore,
				sessionStore *fakeSessionStore,
			) {
				sessionStore.getSessionFunc = func(
					ctx context.Context,
					id string,
				) (*domain.Session, error) {
					return &domain.Session{
						ID:        id,
						UserID:    1,
						ExpiresAt: time.Now().Add(time.Hour),
					}, nil
				}

				userStore.getUserFunc = func(
					ctx context.Context,
					id int,
				) (*domain.User, error) {
					return nil, userrepo.ErrUserNotFound
				}
			},
			wantErr: ErrUserNotFound,
		},
		{
			name:  "success",
			token: "refresh-token",
			setup: func(
				userStore *fakeUserStore,
				sessionStore *fakeSessionStore,
			) {
				sessionStore.getSessionFunc = func(
					ctx context.Context,
					id string,
				) (*domain.Session, error) {
					return &domain.Session{
						ID:        id,
						UserID:    1,
						ExpiresAt: time.Now().Add(time.Hour),
					}, nil
				}

				userStore.getUserFunc = func(
					ctx context.Context,
					id int,
				) (*domain.User, error) {
					return validTestUser(), nil
				}

				sessionStore.deleteSessionFunc = func(
					ctx context.Context,
					id string,
				) error {
					return nil
				}

				sessionStore.createSessionFunc = func(
					ctx context.Context,
					session *domain.Session,
				) error {
					return nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userStore := &fakeUserStore{}
			sessionStore := &fakeSessionStore{}

			if tt.setup != nil {
				tt.setup(
					userStore,
					sessionStore,
				)
			}

			service := newTestUserService(
				userStore,
				sessionStore,
			)

			accessToken, refreshToken, err := service.RefreshToken(
				context.Background(),
				tt.token,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"unexpected error: %v",
					err,
				)
			}

			if accessToken == "" {
				t.Fatal("expected access token")
			}

			if refreshToken == "" {
				t.Fatal("expected refresh token")
			}
		})
	}
}
