package model

import (
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// LedgerEntry is an append-only record of a balance change.
type LedgerEntry struct {
	ID             bson.ObjectID   `bson:"_id"              json:"entry_id"`
	WalletID       string          `bson:"wallet_id"        json:"wallet_id"`
	Type           string          `bson:"type"             json:"type"`
	Amount         decimal.Decimal `bson:"amount"           json:"amount"`
	BalanceAfter   decimal.Decimal `bson:"balance_after"    json:"balance_after"`
	Currency       string          `bson:"currency"         json:"currency"`
	IdempotencyKey string          `bson:"idempotency_key"  json:"idempotency_key,omitempty"`
	CreatedAt      time.Time       `bson:"created_at"       json:"created_at"`
}
