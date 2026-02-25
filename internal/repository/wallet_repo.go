package repository

import (
	"context"
	"errors"
	"vaultgo/internal/model"
	_const "vaultgo/pkg/const"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type WalletRepository struct {
	col *mongo.Collection
}

func NewWalletRepository(db *mongo.Database) *WalletRepository {
	return &WalletRepository{
		col: db.Collection("wallets"),
	}
}

// CreateIndexes sets up unique indexes for the wallets collection.
func (r *WalletRepository) CreateIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{
			Key:   "owner_id",
			Value: 1,
		}, {
			Key:   "currency",
			Value: 1,
		}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// Insert creates a new wallet document.
func (r *WalletRepository) Insert(ctx context.Context, w *model.Wallet) error {
	_, err := r.col.InsertOne(ctx, w)

	if mongo.IsDuplicateKeyError(err) {
		return _const.ErrDuplicateWallet
	}
	return err
}

// FindAll retrieves all wallets.
func (r *WalletRepository) FindAll(ctx context.Context) ([]*model.Wallet, error) {
	var wallets []*model.Wallet

	cursor, err := r.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var w model.Wallet
		if err := cursor.Decode(&w); err != nil {
			return nil, err
		}
		wallets = append(wallets, &w)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return wallets, nil
}

// FindByOwnerID retrieves all wallets for a given owner ID.
func (r *WalletRepository) FindByOwnerID(ctx context.Context, ownerID string) ([]*model.Wallet, error) {
	var wallets []*model.Wallet

	cursor, err := r.col.Find(ctx, bson.M{
		"owner_id": ownerID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var w model.Wallet
		if err := cursor.Decode(&w); err != nil {
			return nil, err
		}
		wallets = append(wallets, &w)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return wallets, nil
}

// FindByWalletID retrieves a wallet by its WalletID.
func (r *WalletRepository) FindByWalletID(ctx context.Context, id string) (*model.Wallet, error) {
	var w model.Wallet

	err := r.col.FindOne(ctx, bson.M{
		"wallet_id": id,
	}).Decode(&w)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, _const.ErrWalletNotFound
	}

	if err != nil {
		return nil, err
	}
	return &w, nil
}

// UpdateWithVersion performs a locking update on a wallet.
// It matches both _id and version, then sets the new fields and increments version.
func (r *WalletRepository) UpdateWithVersion(ctx context.Context, w *model.Wallet) error {
	filter := bson.M{
		"_id":     w.ID,
		"version": w.Version - 1, // the old version before increment
	}
	update := bson.M{
		"$set": bson.M{
			"balance":    w.Balance,
			"status":     w.Status,
			"version":    w.Version,
			"updated_at": w.UpdatedAt,
		},
	}

	res := r.col.FindOneAndUpdate(ctx, filter, update)
	if res.Err() != nil {
		if errors.Is(res.Err(), mongo.ErrNoDocuments) {
			return _const.ErrVersionConflict
		}
		return res.Err()
	}
	return nil
}
