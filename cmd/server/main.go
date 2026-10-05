package main

import (
	docs "checkout-api/docs"
	balanceRepo "checkout-api/internal/repository/balance"
	"checkout-api/internal/repository/cart"
	"checkout-api/internal/repository/inventory"
	"checkout-api/internal/repository/item"
	"checkout-api/internal/repository/order"
	"checkout-api/internal/repository/session"
	"checkout-api/internal/repository/user"
	xsollaRepo "checkout-api/internal/repository/xsolla"
	balanceService "checkout-api/internal/services/balance"
	cart3 "checkout-api/internal/services/cart"
	inventoryService "checkout-api/internal/services/inventory"
	item3 "checkout-api/internal/services/item"
	order3 "checkout-api/internal/services/order"
	user3 "checkout-api/internal/services/user"
	xsollaService "checkout-api/internal/services/xsolla"
	balanceHandler "checkout-api/internal/transport/balance"
	cart2 "checkout-api/internal/transport/cart"
	inventoryHandler "checkout-api/internal/transport/inventory"
	item2 "checkout-api/internal/transport/item"
	order2 "checkout-api/internal/transport/order"
	user2 "checkout-api/internal/transport/user"
	xsollaHandler "checkout-api/internal/transport/xsolla"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	middleware "checkout-api/internal/jwt-validator"

	httptrace "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Checkout API
// @version 1.0
// @description API for products, users, carts and orders.
// @BasePath /

func main() {

	tracer.Start(
		tracer.WithService("checkout-api"),
	)
	defer tracer.Stop()

	docs.SwaggerInfo.Host = ""
	_ = godotenv.Load("../../.env")

	dbURL := buildDatabaseURL()

	ctx := context.Background()

	conn, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))

	if len(jwtSecret) == 0 {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	itemStore := item.NewPostgresStore(conn)
	inventoryStore := inventory.NewPostgresStore(conn)
	cartStore := cart.NewPostgresStore(conn)
	orderStore := order.NewPostgresStore(conn)
	userStore := user.NewPostgresStore(conn)
	sessionStore := session.NewPostgresStore(conn)
	balanceStore := balanceRepo.NewPostgresStore(conn)
	xsollaStore := xsollaRepo.NewPostgresStore(conn)

	itemService := item3.NewItemService(itemStore)
	cartService := cart3.NewCartService(cartStore, itemStore)
	balanceService := balanceService.NewBalanceService(balanceStore)
	orderService := order3.NewOrderService(orderStore, itemStore, cartStore, balanceService)
	userService := user3.NewUserService(userStore, sessionStore, jwtSecret)
	inventorySvc := inventoryService.NewService(inventoryStore)
	xsollaConfig := xsollaService.Config{
		ProjectID:         os.Getenv("XSOLLA_PROJECT_ID"),
		MerchantID:        os.Getenv("XSOLLA_MERCHANT_ID"),
		APIKey:            os.Getenv("XSOLLA_API_KEY"),
		WebhookSecret:     os.Getenv("XSOLLA_WEBHOOK_SECRET"),
		SKU:               os.Getenv("XSOLLA_ITEM_SKU"),
		Sandbox:           parseBoolEnv("XSOLLA_SANDBOX", true),
		ReturnURL:         os.Getenv("XSOLLA_RETURN_URL"),
		APIBaseURL:        envOrDefault("XSOLLA_API_BASE_URL", "https://store.xsolla.com/api"),
		PayStationBaseURL: envOrDefault("XSOLLA_PAYSTATION_URL", "https://sandbox-secure.xsolla.com/paystation4/"),
	}
	xsollaClient := xsollaService.NewHTTPClient(
		xsollaConfig.APIBaseURL,
		xsollaConfig.ProjectID,
		xsollaConfig.MerchantID,
		xsollaConfig.APIKey,
	)
	xsollaSvc := xsollaService.NewService(xsollaConfig, xsollaClient, xsollaStore, userStore, itemStore, cartStore, orderStore)

	itemHandler := item2.NewItemHandler(itemService)
	cartHandler := cart2.NewCartHandler(cartService)
	orderHandler := order2.NewOrderHandler(orderService, cartService)
	userHandler := user2.NewUserHandler(userService)
	inventoryWebHandler := inventoryHandler.NewHandler(inventorySvc)
	balanceHandler := balanceHandler.NewBalanceHandler(balanceService)
	xsollaWebHandler := xsollaHandler.NewHandler(xsollaSvc, xsollaConfig.WebhookSecret)

	authMiddleware := middleware.JWTAuth(jwtSecret)

	mux := registerRoutes(
		itemHandler,
		cartHandler,
		orderHandler,
		userHandler,
		balanceHandler,
		xsollaWebHandler,
		inventoryWebHandler,
		authMiddleware,
	)

	mux.Handle("/swagger/", httpSwagger.WrapHandler)
	fmt.Println("Server v2 starting on :8080")
	handler := corsMiddleware(mux)
	tracedHandler := httptrace.WrapHandler(
		handler,
		"checkout-api",
		"http.request",
	)

	log.Fatal(http.ListenAndServe(":8080", tracedHandler))
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func parseBoolEnv(name string, fallback bool) bool {
	value, err := strconv.ParseBool(os.Getenv(name))
	if err != nil {
		return fallback
	}
	return value
}
