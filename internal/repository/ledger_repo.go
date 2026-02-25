package repository

import (
	"context"
	"errors"

	"vaultgo/internal/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type LedgerRepository struct {
	col *mongo.Collection
}

func NewLedgerRepository(db *mongo.Database) *LedgerRepository {
	return &LedgerRepository{
		col: db.Collection("ledger_entries"),
	}
}

// CreateIndexes sets up indexes for the ledger_entries collection.
func (r *LedgerRepository) CreateIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{
					Key:   "wallet_id",
					Value: 1,
				}, {
					Key:   "created_at",
					Value: 1,
				},
			},
		},
		{
			Keys: bson.D{
				{
					Key:   "idempotency_key",
					Value: 1,
				},
			},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{
				"idempotency_key": bson.M{"$gt": ""},
			}),
		},
	})
	return err
}

// Insert appends a new ledger entry.
func (r *LedgerRepository) Insert(ctx context.Context, entry *model.LedgerEntry) error {
	_, err := r.col.InsertOne(ctx, entry)
	return err
}

// FindByWalletID returns all ledger entries for a wallet, ordered by created_at ascending.
func (r *LedgerRepository) FindByWalletID(ctx context.Context, walletID string) ([]model.LedgerEntry, error) {
	cursor, err := r.col.Find(ctx, bson.M{
		"wallet_id": walletID,
	}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, err
	}

	var entries []model.LedgerEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// FindByIdempotencyKey checks if a ledger entry already exists for a given idempotency key.
func (r *LedgerRepository) FindByIdempotencyKey(ctx context.Context, key string) (*model.LedgerEntry, error) {
	var entry model.LedgerEntry
	err := r.col.FindOne(ctx, bson.M{
		"idempotency_key": key,
	}).Decode(&entry)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}
