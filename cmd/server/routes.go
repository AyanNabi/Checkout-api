package main

import (
	"checkout-api/internal/helper/middleware"
	balance "checkout-api/internal/transport/balance"
	"checkout-api/internal/transport/cart"
	"checkout-api/internal/transport/inventory"
	"checkout-api/internal/transport/item"
	"checkout-api/internal/transport/order"
	userpackage "checkout-api/internal/transport/user"
	xsolla "checkout-api/internal/transport/xsolla"
	"fmt"
	"net/http"
	"os"
)

func buildDatabaseURL() string {
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user,
		password,
		host,
		port,
		dbName,
	)
}

func registerRoutes(
	itemHandler *item.ItemHandler,
	cartHandler *cart.CartHandler,
	orderHandler *order.OrderHandler,
	userHandler *userpackage.UserHandler,
	balanceHandler *balance.BalanceHandler,
	xsollaHandler *xsolla.Handler,
	inventoryHandler *inventory.Handler,
	authMiddleware func(http.Handler) http.Handler,
) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle(
		"/api/signup",
		middleware.ErrorMiddleware(userHandler.SignUp),
	)

	mux.Handle(
		"/api/login",
		middleware.ErrorMiddleware(userHandler.Login),
	)

	mux.Handle(
		"/api/token",
		middleware.ErrorMiddleware(userHandler.RefreshToken),
	)

	mux.Handle(
		"/api/logout",
		middleware.ErrorMiddleware(userHandler.Logout),
	)

	mux.Handle(
		"/api/items",
		middleware.ErrorMiddleware(itemHandler.GetItems),
	)

	mux.Handle(
		"/api/items/",
		middleware.ErrorMiddleware(itemHandler.GetItemByID),
	)
	mux.Handle(
		"GET /api/me/balance",
		authMiddleware(
			middleware.ErrorMiddleware(balanceHandler.GetBalance),
		),
	)

	mux.Handle(
		"POST /api/me/balance/top-up",
		authMiddleware(
			middleware.ErrorMiddleware(balanceHandler.TopUp),
		),
	)

	mux.Handle(
		"GET /api/me/transactions",
		authMiddleware(
			middleware.ErrorMiddleware(balanceHandler.GetTransactions),
		),
	)

	mux.Handle(
		"POST /api/xsolla/purchase",
		authMiddleware(middleware.ErrorMiddleware(xsollaHandler.CreatePayment)),
	)

	mux.Handle(
		"GET /api/me/purchases",
		authMiddleware(middleware.ErrorMiddleware(xsollaHandler.GetPurchases)),
	)

	mux.Handle(
		"POST /api/webhook",
		middleware.ErrorMiddleware(xsollaHandler.Webhook),
	)

	mux.Handle(
		"GET /api/me/inventory",
		authMiddleware(middleware.ErrorMiddleware(inventoryHandler.GetInventory)),
	)

	mux.Handle(
		"GET /api/user/cart",
		authMiddleware(
			http.HandlerFunc(cartHandler.GetUserCart),
		),
	)

	mux.Handle(
		"POST /api/user/cart",
		authMiddleware(
			http.HandlerFunc(cartHandler.AddCartItem),
		),
	)

	mux.Handle(
		"PATCH /api/user/cart/items/",
		authMiddleware(
			http.HandlerFunc(cartHandler.UpdateCartItem),
		),
	)

	mux.Handle(
		"DELETE /api/user/cart/items/",
		authMiddleware(
			http.HandlerFunc(cartHandler.RemoveCartItem),
		),
	)

	mux.Handle(
		"GET /api/user/orders",
		authMiddleware(
			middleware.ErrorMiddleware(
				orderHandler.GetOrdersByUserID,
			),
		),
	)

	mux.Handle(
		"POST /api/user/orders",
		authMiddleware(
			middleware.ErrorMiddleware(
				orderHandler.CreateOrderFromCart,
			),
		),
	)

	mux.Handle(
		"GET /users",
		middleware.ErrorMiddleware(
			userHandler.GetAllUsers,
		),
	)
	mux.Handle(
		"POST /users",
		middleware.ErrorMiddleware(userHandler.CreateUser),
	)

	mux.Handle(
		"GET /users/",
		authMiddleware(
			middleware.ErrorMiddleware(userHandler.GetUser),
		),
	)

	mux.Handle(
		"PUT /users/",
		authMiddleware(
			middleware.ErrorMiddleware(userHandler.UpdateUser),
		),
	)

	mux.Handle(
		"DELETE /users/",
		authMiddleware(
			middleware.ErrorMiddleware(userHandler.DeleteUser),
		),
	)

	mux.Handle(
		"GET /orders/",
		middleware.ErrorMiddleware(orderHandler.GetOrderByID),
	)

	mux.Handle(
		"PATCH /orders/",
		middleware.ErrorMiddleware(orderHandler.UpdateOrderStatus),
	)

	mux.HandleFunc("GET /live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	return mux
}
