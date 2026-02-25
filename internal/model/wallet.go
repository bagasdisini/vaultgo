package model

import (
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Wallet represents a user's wallet for a specific currency.
type Wallet struct {
	ID        bson.ObjectID   `bson:"_id"         json:"_id"`
	WalletID  string          `bson:"wallet_id"   json:"wallet_id"`
	OwnerID   string          `bson:"owner_id"    json:"owner_id"`
	Currency  string          `bson:"currency"    json:"currency"`
	Balance   decimal.Decimal `bson:"balance"     json:"balance"`
	Status    string          `bson:"status"      json:"status"`
	Version   int64           `bson:"version"     json:"-"`
	CreatedAt time.Time       `bson:"created_at"  json:"created_at"`
	UpdatedAt time.Time       `bson:"updated_at"  json:"updated_at"`
}
