# VaultGo — Multi-Currency E-Wallet Backend

A simplified multi-currency e-wallet ledger backend built with **Go**, **Gin**, and **MongoDB**.

## Quick Start

### Prerequisites

- **Docker** and **Docker Compose** (for MongoDB replica set)
- **Go 1.25.7+**

### Run with Docker Compose

```bash
docker-compose up --build
```

This starts:
- **MongoDB 7** as a single-node replica set on port `27017`
- **VaultGo** API server on port `8080`

### Run Locally (with external MongoDB)

Start a MongoDB replica set, then:

```bash
export PORT="8080"
export MONGO_URI="mongodb://localhost:27017/?replicaSet=rs0"
export DB_NAME="vaultgo"

go run main.go
```

---

## Swagger Documentation

When the server is running, visit:

```
http://localhost:8080/
```

And choose docs to open the Swagger UI with interactive API documentation.

---

## API Reference

All responses follow a consistent envelope format:

```json
{
  "status": "success",
  "message": "description",
  "data": { ... }
}
```

Error responses:

```json
{
  "status": "error",
  "message": "description of the error"
}
```

---

### Health Check

```
GET /health
```

**Response (200):**
```json
{
  "status": "success",
  "message": "VaultGo server is healthy"
}
```

---

### Create Wallet

```
POST /wallets
Content-Type: application/json

{
  "owner_id": "user1",
  "currency": "USD"
}
```

**Response (201):**
```json
{
  "status": "success",
  "message": "wallet created successfully",
  "data": {
    "wallet_id": "VAULT-A1B2C3D4E5",
    "owner_id": "user1",
    "currency": "USD",
    "balance": "0.00",
    "status": "ACTIVE",
    "created_at": "2026-02-25T10:00:00Z",
    "updated_at": "2026-02-25T10:00:00Z"
  }
}
```

---

### Top-Up

```
POST /wallets/{wallet_id}/topup
Content-Type: application/json
Idempotency-Key: topup-001

{
  "amount": "1000.50"
}
```

**Response (200):**
```json
{
  "status": "success",
  "message": "wallet topped up successfully",
  "data": {
    "wallet_id": "VAULT-A1B2C3D4E5",
    "owner_id": "user1",
    "currency": "USD",
    "balance": "1000.50",
    "status": "ACTIVE",
    "created_at": "2026-02-25T10:00:00Z",
    "updated_at": "2026-02-25T10:01:00Z"
  }
}
```

---

### Payment

```
POST /wallets/{wallet_id}/pay
Content-Type: application/json
Idempotency-Key: pay-001

{
  "amount": "200.10"
}
```

**Response (200):**
```json
{
  "status": "success",
  "message": "payment successfully",
  "data": {
    "wallet_id": "VAULT-A1B2C3D4E5",
    "owner_id": "user1",
    "currency": "USD",
    "balance": "800.40",
    "status": "ACTIVE",
    "created_at": "2026-02-25T10:00:00Z",
    "updated_at": "2026-02-25T10:02:00Z"
  }
}
```

---

### Transfer

```
POST /wallets/transfer
Content-Type: application/json
Idempotency-Key: xfer-001

{
  "from_wallet_id": "VAULT-A1B2C3D4E5",
  "to_wallet_id": "VAULT-F6G7H8I9J0",
  "amount": "300.40"
}
```

**Response (200):**
```json
{
  "status": "success",
  "message": "transfer successfully"
}
```

---

### Suspend Wallet

```
POST /wallets/{wallet_id}/suspend
```

**Response (200):**
```json
{
  "status": "success",
  "message": "wallet suspended successfully",
  "data": {
    "wallet_id": "VAULT-A1B2C3D4E5",
    "owner_id": "user1",
    "currency": "USD",
    "balance": "500.00",
    "status": "SUSPENDED",
    "created_at": "2026-02-25T10:00:00Z",
    "updated_at": "2026-02-25T10:05:00Z"
  }
}
```

---

### Unsuspend Wallet

```
POST /wallets/{wallet_id}/unsuspend
```

**Response (200):** Updated wallet with status `ACTIVE`.

---

### Query Wallet

```
GET /wallets/{wallet_id}
```

**Response (200):**
```json
{
  "status": "success",
  "message": "wallet retrieved successfully",
  "data": {
    "wallet_id": "VAULT-A1B2C3D4E5",
    "owner_id": "user1",
    "currency": "USD",
    "balance": "500.00",
    "status": "ACTIVE",
    "created_at": "2026-02-25T10:00:00Z",
    "updated_at": "2026-02-25T10:01:00Z"
  }
}
```

---

### Get All Wallets

```
GET /wallets
```

**Response (200):**
```json
{
  "status": "success",
  "message": "wallet retrieved successfully",
  "data": [ ... ]
}
```

---

### Get Wallets by Owner

```
GET /wallets/owner/{owner_id}
```

**Response (200):**
```json
{
  "status": "success",
  "message": "wallet retrieved successfully",
  "data": [ ... ]
}
```

---

## Error Responses

| HTTP Code | Meaning |
|-----------|---------|
| 400 | Bad request (invalid amount, currency, currency mismatch, same wallet transfer) |
| 403 | Wallet is suspended |
| 404 | Wallet not found |
| 409 | Conflict (duplicate wallet, version conflict, wallet already active) |
| 422 | Insufficient funds |
| 500 | Internal server error |

---

## Sample Usage Flow

```bash
# 1. Create wallets
curl -s -X POST http://localhost:8080/wallets \
  -H "Content-Type: application/json" \
  -d '{"owner_id":"user1","currency":"USD"}'

curl -s -X POST http://localhost:8080/wallets \
  -H "Content-Type: application/json" \
  -d '{"owner_id":"user1","currency":"EUR"}'

curl -s -X POST http://localhost:8080/wallets \
  -H "Content-Type: application/json" \
  -d '{"owner_id":"user2","currency":"USD"}'

# 2. Top-ups (replace {wallet_id} with actual wallet_id from step 1)
curl -s -X POST http://localhost:8080/wallets/{user1-usd-wallet-id}/topup \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: topup-1" \
  -d '{"amount":"1000.50"}'

curl -s -X POST http://localhost:8080/wallets/{user1-eur-wallet-id}/topup \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: topup-2" \
  -d '{"amount":"500.25"}'

curl -s -X POST http://localhost:8080/wallets/{user2-usd-wallet-id}/topup \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: topup-3" \
  -d '{"amount":"200.75"}'

# 3. Payments
curl -s -X POST http://localhost:8080/wallets/{user1-usd-wallet-id}/pay \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: pay-1" \
  -d '{"amount":"200.10"}'

curl -s -X POST http://localhost:8080/wallets/{user1-eur-wallet-id}/pay \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: pay-2" \
  -d '{"amount":"100.50"}'

# 4. Transfer (same currency only)
curl -s -X POST http://localhost:8080/wallets/transfer \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: xfer-1" \
  -d '{"from_wallet_id":"{user1-usd-wallet-id}","to_wallet_id":"{user2-usd-wallet-id}","amount":"300.40"}'

# This should FAIL — user2 has no EUR wallet:
curl -s -X POST http://localhost:8080/wallets/transfer \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: xfer-2" \
  -d '{"from_wallet_id":"{user1-eur-wallet-id}","to_wallet_id":"{user2-eur-wallet-id}","amount":"100.00"}'

# 5. Query
curl -s http://localhost:8080/wallets/{user1-usd-wallet-id}
curl -s http://localhost:8080/wallets/{user1-eur-wallet-id}
curl -s http://localhost:8080/wallets/{user2-usd-wallet-id}

# 6. Suspend
curl -s -X POST http://localhost:8080/wallets/{user1-usd-wallet-id}/suspend

# 7. Unsuspend
curl -s -X POST http://localhost:8080/wallets/{user1-usd-wallet-id}/unsuspend
```

---

## Limitations

This project is a **simplified e-wallet ledger service** intended for demonstration purposes. The following limitations apply:

- **No User/Owner Management** — There is no user registration, profile, or owner entity. The `owner_id` field is a free-form string, meaning anyone can create a wallet with any `owner_id` value without verification.
- **No Authentication & Authorization** — The API is completely open. There is no login, JWT, API key, OAuth, or any form of identity verification. Any client can perform any operation on any wallet.
- **No Query Parameters, Filtering, or Pagination** — List endpoints (`GET /wallets`, `GET /wallets/owner/{owner_id}`, `GET /wallets/{id}/ledger`) return all matching records without support for filtering, searching, sorting, or pagination via query parameters.
- **No Soft Delete** — Wallets can only be suspended/unsuspended. There is no mechanism to close or delete a wallet permanently.
- **No Withdrawal Endpoint** — There is no dedicated withdrawal flow; funds can only leave a wallet via payments or transfers.
- **No Currency Exchange** — Transfers are restricted to wallets of the same currency. There is no foreign exchange or currency conversion support.

---

## Testing

### Unit Tests (no MongoDB required)

```bash
go test ./test -run "TestValidate" -v
```

### Integration Tests (requires MongoDB replica set)

```bash
export MONGO_URI="mongodb://localhost:27017/?replicaSet=rs0"
go test ./test -v
```

### All Tests

```bash
go test ./... -v
```