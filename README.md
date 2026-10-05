# Checkout API

Training project — a simplified checkout API built with Go.

Students build this service incrementally across lectures, starting from a basic HTTP server and evolving it into a production-ready system with persistence, authentication, observability, and more.

## Prerequisites

- Go 1.21+
- curl or Postman (for testing endpoints)
- A code editor (VS Code, GoLand, etc.)

## Quick Start

```bash
go run cmd/server/main.go
```

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
## License

Internal — Xsolla School use only.
