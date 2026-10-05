# Checkout API

Xsolla School training project — a simplified checkout API built with Go.

Students build this service incrementally across lectures, starting from a basic HTTP server and evolving it into a production-ready system with persistence, authentication, observability, and more.

## Prerequisites

- Go 1.21+
- curl or Postman (for testing endpoints)
- A code editor (VS Code, GoLand, etc.)

## Quick Start

```bash
go run cmd/server/main.go
```

The server starts on http://localhost:8080.

## API Endpoints

### GET /items

Returns all available items.

```bash
curl http://localhost:8080/items
```

# Xsolla Pay Station Sandbox

Set the variables in `.env` (or the Helm values/Secret) before using the Buy button:

- `XSOLLA_PROJECT_ID`: Xsolla project ID.
- `XSOLLA_MERCHANT_ID`: Xsolla merchant ID used as the Basic Auth username.
- `XSOLLA_API_KEY`: Xsolla API key used as the Basic Auth password.
- `XSOLLA_WEBHOOK_SECRET`: Xsolla webhook secret key.
- `XSOLLA_ITEM_SKU`: SKU that exists in the Xsolla Store catalog.
- `XSOLLA_SANDBOX=true` and `XSOLLA_PAYSTATION_URL=https://sandbox-secure.xsolla.com/paystation4/`.

The backend creates tokens through `POST /v3/project/{project_id}/admin/payment/token` and the browser is redirected to the returned sandbox Pay Station URL. Each item can store its Store SKU in `items.xsolla_sku`; `XSOLLA_ITEM_SKU` is only a compatibility fallback. Configure the Xsolla webhook URL as `https://<public-host>/api/webhook`; local testing requires a public HTTPS tunnel. The webhook signature is `SHA1(raw_body + XSOLLA_WEBHOOK_SECRET)` in the `Authorization: Signature <hash>` header.

After a successful payment webhook, the purchased quantity is added to `GET /api/me/inventory`. The wallet balance remains separate from virtual-item ownership. The same transaction ID can be delivered repeatedly; the database constraint and transaction lock ensure that inventory is granted once.

Run migrations through the existing migration workflow before using payments. Populate `items.xsolla_sku` with the exact SKU from the Xsolla Store catalog. Set the webhook secret in the Publisher Account and copy the credentials into deployment secrets, never into frontend code or committed files.

For a manual sandbox test: log in, click Buy on a catalog item, complete the Pay Station checkout with an Xsolla sandbox card and expiry `12/40`, then verify the balance/transactions endpoint. Re-send the same signed payment webhook and verify the balance changes only once.

### POST /orders

Creates an order with mock payment processing.

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "items": [{"item_id": 1, "quantity": 2}]}'
```

## Running Tests

```bash
go test -v ./internal/transport/
```

## Branch Guide

Each lecture has two branches:

| Branch | Purpose |
|--------|---------|
| `week-XX/lecture-XX` | **Starter** — scaffold with TODOs and pre-written tests. Fork from here at the start of class. |
| `week-XX/lecture-XX-final` | **Final** — completed code matching the lecture. Compare your work against this. |

### Available Branches

- `week-01/lecture-01` — Intro to HTTP and JSON APIs (starter)
- `week-01/lecture-01-final` — Intro to HTTP and JSON APIs (completed)

## Project Structure

```
checkout-api/
├── cmd/server/
│   └── main.go              # HTTP server entry point
├── internal/
│   ├── models/              # Domain models
│   ├── services/            # Business logic
│   ├── storage/             # Data storage (Postgres)
│   ├── transport/           # HTTP route handlers
│   └── jwt-validator/       # JWT middleware
├── migrations/              # Database migrations
└── README.md
```# Auth

```bash
curl -X POST http://localhost:8080/signup \
-H "Content-Type: application/json" \
-d '{"username":"ayan","email":"ayan@gmail.com","password":"password123"}'

curl -X POST http://localhost:8080/login \
-H "Content-Type: application/json" \
-d '{"username":"ayan","password":"password123"}'

curl -X POST http://localhost:8080/token \
-H "Content-Type: application/json" \
-d '{"refresh_token":"REFRESH_TOKEN"}'
```

# Items

```bash
curl http://localhost:8080/items

curl http://localhost:8080/items/1
```

# Users (JWT)

```bash
curl -H "Authorization: Bearer TOKEN" \
http://localhost:8080/users

curl -H "Authorization: Bearer TOKEN" \
http://localhost:8080/users/1

curl -X POST http://localhost:8080/users \
-H "Authorization: Bearer TOKEN" \
-H "Content-Type: application/json" \
-d '{"username":"test","email":"test@gmail.com","password":"123456"}'

curl -X PUT http://localhost:8080/users/1 \
-H "Authorization: Bearer TOKEN" \
-H "Content-Type: application/json" \
-d '{"username":"updated"}'

curl -X DELETE http://localhost:8080/users/1 \
-H "Authorization: Bearer TOKEN"
```

# Cart (JWT)

```bash
curl -H "Authorization: Bearer TOKEN" \
http://localhost:8080/user/cart

curl -X POST http://localhost:8080/user/cart \
-H "Authorization: Bearer TOKEN" \
-H "Content-Type: application/json" \
-d '{"item_id":1,"quantity":2}'

curl -X PATCH http://localhost:8080/user/cart/items/1 \
-H "Authorization: Bearer TOKEN" \
-H "Content-Type: application/json" \
-d '{"quantity":5}'

curl -X DELETE http://localhost:8080/user/cart/items/1 \
-H "Authorization: Bearer TOKEN"
```

# Orders (JWT)

```bash
curl -X POST http://localhost:8080/user/orders \
-H "Authorization: Bearer TOKEN" \
-H "Content-Type: application/json" \
-d '{}'

curl -H "Authorization: Bearer TOKEN" \
http://localhost:8080/user/orders
```

# Order Management

```bash
curl http://localhost:8080/orders/1

curl -X PATCH http://localhost:8080/orders/1/status \
-H "Content-Type: application/json" \
-d '{"status":"completed"}'
```

```bash
Task 3: 
Add a compound index on (user_id, status) on orders. Use EXPLAIN ANALYZE to prove it helped.

EXPLAIN ANALYZE
SELECT *
FROM orders
WHERE user_id = 1
AND status = 'completed';

--
CREATE INDEX idx_orders_user_status
ON orders(user_id, status);

EXPLAIN ANALYZE
SELECT *
FROM orders
WHERE user_id = 1
AND status = 'completed'
```
### Create a migration

```bash
migrate create -ext sql -dir db/migrations -seq create_items_table
```

### Run migrations

```bash
go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest

migrate -database "$DATABASE_URL" -path db/migrations up
```

### Roll back one migration

```bash
migrate -database "$DATABASE_URL" -path db/migrations down 1
```

### Check PostgreSQL

```bash
psql -U postgres -d xsolla_ecommerce
```

### Seed Data to PostgreSQL

```bash
psql xsolla_ecommerce < db/seeds/items.sql
```

## License

Internal — Xsolla School use only.
