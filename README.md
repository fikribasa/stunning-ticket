# Ticketing System

A production-ready Go skeleton for a concert/event ticketing system
that handles concurrent seat booking, distributed locking, and race
condition prevention.

Built from the system design article *"Building a Ticketing System:
Concurrency, Locks, and Race Conditions"* by Arvind Kumar.

---

## Features

| Feature | Details |
|---------|---------|
| **Zero double-booking** | Per-seat distributed locks + optimistic version check |
| **Two-phase booking** | Reserve (10 min hold) → Confirm (after payment) |
| **Deadlock prevention** | Seats locked in sorted order when booking multiple seats |
| **Idempotent confirmation** | Supply an `idempotency_key` to safely retry confirmations |
| **Auto-expiry cleanup** | Background goroutine releases abandoned holds every 60 s |
| **JWT authentication** | Register / login → Bearer token; ADMIN role for event management |
| **Swappable components** | `LockManager`, `Logger`, and DB are all interface-backed |
| **Dependency injection** | Full application graph wired with `uber-go/fx` |

---

## Tech Stack

| Layer | Library |
|-------|---------|
| HTTP framework | [Echo v4](https://echo.labstack.com) |
| Database | [SQLite](https://modernc.org/sqlite) (WAL mode, pure Go — no CGO) |
| Dependency injection | [uber-go/fx](https://github.com/uber-go/fx) |
| Auth | [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt) |
| Password hashing | `bcrypt` via `golang.org/x/crypto` |
| Logging | `log/slog` (JSON, interface-backed for easy swap) |
| Locking | In-memory `LockManager` (swap to Redis/Redlock for multi-node) |

---

## Project Structure

```
ticketing-system/
├── cmd/server/main.go              # Entry point
├── internal/
│   ├── domain/models.go            # All domain types
│   ├── repository/                 # Interfaces + SQLite implementations
│   │   ├── interfaces.go
│   │   ├── event_repository.go
│   │   ├── seat_repository.go
│   │   └── booking_reservation_repository.go
│   ├── service/                    # Business logic
│   │   ├── booking_service.go      # Core concurrency logic lives here
│   │   ├── event_service.go
│   │   └── auth_service.go
│   ├── handler/                    # Echo HTTP handlers
│   │   ├── handlers.go
│   │   └── router.go
│   ├── middleware/auth.go          # JWT + role middleware
│   ├── infrastructure/
│   │   ├── db/sqlite.go            # Connection + schema migrations
│   │   └── lock/lock.go            # LockManager interface + in-memory impl
│   └── wire/wire.go                # uber/fx providers and lifecycle hooks
├── pkg/
│   ├── logger/logger.go            # Swappable slog wrapper
│   └── apperrors/errors.go         # Typed application errors
├── PRD.md                          # Full product requirements document
├── Makefile
└── README.md
```

---

## Setup

### Prerequisites

- Go 1.22+
- `make` (optional but recommended)

No CGO, no external database, no Docker required to run locally.

### Run

```bash
git clone https://github.com/fikribasa/ticketing-system
cd ticketing-system

# Download dependencies
go mod tidy

# Run with defaults (port 8080, ./ticketing.db)
make run

# Or with custom config
PORT=9000 DB_PATH=./dev.db JWT_SECRET=my-secret make run
```

### Build binary

```bash
make build
./bin/ticketing-server
```

### Run tests

```bash
make test             # all tests with race detector
make test/unit        # unit tests only
make test/integration # integration tests against :memory: DB
```

---

## API Quick Reference

### Auth

```bash
# Register
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secret123"}'

# Login → get access_token
curl -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secret123"}'
```

### Browse Events

```bash
curl http://localhost:8080/api/events
curl http://localhost:8080/api/events/1
curl http://localhost:8080/api/events/1/seats
```

### Reserve Seats

```bash
curl -X POST http://localhost:8080/api/reservations \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"event_id":1,"seat_numbers":["A1","A2"],"session_id":"sess-abc"}'
# Returns: { reservations[], expires_at }
```

### Confirm Booking

```bash
curl -X POST http://localhost:8080/api/bookings/confirm \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "reservation_ids": [1, 2],
    "payment_id": "pay_stripe_xxx",
    "idempotency_key": "checkout-session-123"
  }'
```

### Cancel Reservation

```bash
curl -X DELETE http://localhost:8080/api/reservations/1 \
  -H 'Authorization: Bearer <token>'
```

### Create Event (Admin)

```bash
curl -X POST http://localhost:8080/api/admin/events \
  -H 'Authorization: Bearer <admin-token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "event_name": "Taylor Swift – Eras Tour",
    "venue_name": "Gelora Bung Karno",
    "total_seats": 50000,
    "seat_config": [
      {"section":"VIP","row_prefix":"V","count":100,"seat_type":"VIP","price":2500000},
      {"section":"Reguler","row_prefix":"A","count":500,"seat_type":"REGULAR","price":750000}
    ]
  }'
```

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `./ticketing.db` | SQLite file path |
| `JWT_SECRET` | `change-me-in-production` | JWT HMAC key — **always override in production** |

---

## Concurrency Design

The system uses **three layers** to prevent double-booking:

1. **Distributed Lock** (`LockManager`) — acquired before reading or writing
   a seat. Key: `seat:<eventID>:<seatNumber>`. TTL: 30 s.
2. **Optimistic Locking** — the `seats.version` column guards against any
   race that slips past an expired lock. The `UPDATE … WHERE version = ?`
   returns 0 rows on mismatch.
3. **Database transactions** — each seat update + reservation insert is
   atomic within SQLite's WAL mode.

Multiple seats are always locked in **sorted order** to prevent deadlocks.

See `PRD.md §6` and `internal/service/booking_service.go` for the full
annotated implementation.

---

## Swapping Components

### Redis distributed lock (multi-node production)

Implement `lock.LockManager` using the Redlock algorithm and register it
in `internal/wire/wire.go`:

```go
// Replace:
fx.Provide(lock.NewInMemoryLockManager),
// With:
fx.Provide(lock.NewRedisLockManager),
```

### Structured logging (OTEL / Grafana Loki)

Implement `logger.Logger` and swap the provider in `wire.go`. All
service call sites use the interface — zero changes required elsewhere.

### PostgreSQL

Replace `internal/infrastructure/db/sqlite.go` with a `pgx`-backed
version. Repository implementations are behind the `repository.*`
interfaces — swap one file per repository.

---

## Infrastructure (Docker Compose)

A minimal `docker-compose.yml` for local development with Redis (when
you're ready to go multi-node):

```yaml
version: "3.9"
services:
  app:
    build: .
    ports: ["8080:8080"]
    environment:
      DB_PATH: /data/ticketing.db
      JWT_SECRET: ${JWT_SECRET}
    volumes:
      - app-data:/data
  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
volumes:
  app-data:
```
