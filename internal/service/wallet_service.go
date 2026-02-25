package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"vaultgo/internal/model"
	"vaultgo/internal/repository"
	_const "vaultgo/pkg/const"
	"vaultgo/pkg/utils"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type WalletService struct {
	walletRepo *repository.WalletRepository
	ledgerRepo *repository.LedgerRepository
	client     *mongo.Client
}

func NewWalletService(
	walletRepo *repository.WalletRepository,
	ledgerRepo *repository.LedgerRepository,
	client *mongo.Client,
) *WalletService {
	return &WalletService{
		walletRepo: walletRepo,
		ledgerRepo: ledgerRepo,
		client:     client,
	}
}

// ValidateAmount checks that the amount is positive and has at most 2 decimal places.
func ValidateAmount(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return _const.ErrInvalidAmount
	}
	if amount.LessThan(_const.WalletMinUnit) {
		return fmt.Errorf("amount %s is less than smallest unit (0.01): %w", amount.String(), _const.ErrInvalidAmount)
	}
	if !amount.Equal(amount.Round(2)) {
		return fmt.Errorf("amount %s has more than 2 decimal places: %w", amount.String(), _const.ErrInvalidAmount)
	}
	return nil
}

func ValidateCurrency(currency string) error {
	if _, ok := _const.ValidISO4217Currencies[currency]; !ok {
		return _const.ErrInvalidCurrency
	}
	return nil
}

// CreateWallet creates a new wallet for a user in a given currency.
func (s *WalletService) CreateWallet(ctx context.Context, ownerID, currency string) (*model.Wallet, error) {
	if strings.TrimSpace(ownerID) == "" {
		return nil, _const.ErrInvalidOwnerID
	}
	if err := ValidateCurrency(currency); err != nil {
		return nil, err
	}

	w := &model.Wallet{
		ID:        bson.NewObjectID(),
		WalletID:  "VAULT-" + utils.GenerateCode(10),
		OwnerID:   ownerID,
		Currency:  currency,
		Balance:   decimal.Zero,
		Status:    _const.WalletStatusActive,
		Version:   1,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.walletRepo.Insert(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

// GetWallets retrieves all wallets.
func (s *WalletService) GetWallets(ctx context.Context) ([]*model.Wallet, error) {
	return s.walletRepo.FindAll(ctx)
}

// GetWalletsByOwner retrieves all wallets for a specific owner.
func (s *WalletService) GetWalletsByOwner(ctx context.Context, ownerID string) ([]*model.Wallet, error) {
	return s.walletRepo.FindByOwnerID(ctx, ownerID)
}

// GetWallet retrieves wallet details.
func (s *WalletService) GetWallet(ctx context.Context, walletID string) (*model.Wallet, error) {
	return s.walletRepo.FindByWalletID(ctx, walletID)
}

// GetLedgerEntries retrieves all ledger entries for a given wallet, ordered by creation time.
func (s *WalletService) GetLedgerEntries(ctx context.Context, walletID string) ([]model.LedgerEntry, error) {
	// Verify wallet exists
	_, err := s.walletRepo.FindByWalletID(ctx, walletID)
	if err != nil {
		return nil, err
	}
	return s.ledgerRepo.FindByWalletID(ctx, walletID)
}

// TopUp adds money to a wallet. Uses MongoDB transaction + locking.
func (s *WalletService) TopUp(ctx context.Context, walletID string, amount decimal.Decimal, idempotencyKey string) (*model.Wallet, error) {
	if err := ValidateAmount(amount); err != nil {
		return nil, err
	}
	amount = amount.Round(2)

	var result *model.Wallet
	err := s.withTransaction(ctx, func(sc context.Context) error {
		// Check idempotency
		if idempotencyKey != "" {
			existing, err := s.ledgerRepo.FindByIdempotencyKey(sc, idempotencyKey)
			if err != nil {
				return err
			}
			if existing != nil {
				// Already processed; fetch current wallet state
				w, err := s.walletRepo.FindByWalletID(sc, walletID)
				if err != nil {
					return err
				}
				result = w
				return _const.ErrDuplicateRequest
			}
		}

		w, err := s.walletRepo.FindByWalletID(sc, walletID)
		if err != nil {
			return err
		}
		if w.Status == _const.WalletStatusSuspended {
			return _const.ErrWalletSuspended
		}

		w.Balance = w.Balance.Add(amount)
		w.Version++
		w.UpdatedAt = time.Now().UTC()

		if err := s.walletRepo.UpdateWithVersion(sc, w); err != nil {
			return err
		}

		entry := &model.LedgerEntry{
			ID:             bson.NewObjectID(),
			WalletID:       w.WalletID,
			Type:           _const.LedgerTypeTopUp,
			Amount:         amount,
			BalanceAfter:   w.Balance,
			Currency:       w.Currency,
			IdempotencyKey: idempotencyKey,
			CreatedAt:      time.Now().UTC(),
		}
		if err := s.ledgerRepo.Insert(sc, entry); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return _const.ErrDuplicateRequest
			}
			return err
		}

		result = w
		return nil
	})

	if errors.Is(err, _const.ErrDuplicateRequest) {
		return result, nil // idempotent: return cached result without error
	}
	return result, err
}

// Pay deducts money from a wallet.
func (s *WalletService) Pay(ctx context.Context, walletID string, amount decimal.Decimal, idempotencyKey string) (*model.Wallet, error) {
	if err := ValidateAmount(amount); err != nil {
		return nil, err
	}
	amount = amount.Round(2)

	var result *model.Wallet
	err := s.withTransaction(ctx, func(sc context.Context) error {
		// Check idempotency
		if idempotencyKey != "" {
			existing, err := s.ledgerRepo.FindByIdempotencyKey(sc, idempotencyKey)
			if err != nil {
				return err
			}
			if existing != nil {
				w, err := s.walletRepo.FindByWalletID(sc, walletID)
				if err != nil {
					return err
				}
				result = w
				return _const.ErrDuplicateRequest
			}
		}

		w, err := s.walletRepo.FindByWalletID(sc, walletID)
		if err != nil {
			return err
		}
		if w.Status == _const.WalletStatusSuspended {
			return _const.ErrWalletSuspended
		}
		if w.Balance.LessThan(amount) {
			return _const.ErrInsufficientFunds
		}

		w.Balance = w.Balance.Sub(amount)
		w.Version++
		w.UpdatedAt = time.Now().UTC()

		if err := s.walletRepo.UpdateWithVersion(sc, w); err != nil {
			return err
		}

		entry := &model.LedgerEntry{
			ID:             bson.NewObjectID(),
			WalletID:       w.WalletID,
			Type:           _const.LedgerTypePayment,
			Amount:         amount.Neg(),
			BalanceAfter:   w.Balance,
			Currency:       w.Currency,
			IdempotencyKey: idempotencyKey,
			CreatedAt:      time.Now().UTC(),
		}
		if err := s.ledgerRepo.Insert(sc, entry); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return _const.ErrDuplicateRequest
			}
			return err
		}

		result = w
		return nil
	})

	if errors.Is(err, _const.ErrDuplicateRequest) {
		return result, nil
	}
	return result, err
}

// Transfer moves money between two wallets of the same currency.
func (s *WalletService) Transfer(ctx context.Context, fromWalletID, toWalletID string, amount decimal.Decimal, idempotencyKey string) error {
	if err := ValidateAmount(amount); err != nil {
		return err
	}
	if fromWalletID == toWalletID {
		return _const.ErrSameWallet
	}
	amount = amount.Round(2)

	// Determine deterministic fetch order to prevent deadlocks
	// Always lock the wallet with the smaller ID first
	firstID, secondID := fromWalletID, toWalletID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	err := s.withTransaction(ctx, func(sc context.Context) error {
		// Check idempotency
		if idempotencyKey != "" {
			existing, err := s.ledgerRepo.FindByIdempotencyKey(sc, idempotencyKey+":debit")
			if err != nil {
				return err
			}
			if existing != nil {
				return _const.ErrDuplicateRequest
			}
		}

		// Fetch wallets in deterministic order to prevent deadlocks
		firstW, err := s.walletRepo.FindByWalletID(sc, firstID)
		if err != nil {
			return fmt.Errorf("wallet %s: %w", firstID, err)
		}
		secondW, err := s.walletRepo.FindByWalletID(sc, secondID)
		if err != nil {
			return fmt.Errorf("wallet %s: %w", secondID, err)
		}

		// Map back to from/to based on original IDs
		var fromW, toW *model.Wallet
		if firstID == fromWalletID {
			fromW, toW = firstW, secondW
		} else {
			fromW, toW = secondW, firstW
		}

		if fromW.Currency != toW.Currency {
			return _const.ErrCurrencyMismatch
		}
		if fromW.Status == _const.WalletStatusSuspended {
			return fmt.Errorf("source %w", _const.ErrWalletSuspended)
		}
		if toW.Status == _const.WalletStatusSuspended {
			return fmt.Errorf("destination %w", _const.ErrWalletSuspended)
		}
		if fromW.Balance.LessThan(amount) {
			return _const.ErrInsufficientFunds
		}

		now := time.Now().UTC()

		// Debit source
		fromW.Balance = fromW.Balance.Sub(amount)
		fromW.Version++
		fromW.UpdatedAt = now
		if err := s.walletRepo.UpdateWithVersion(sc, fromW); err != nil {
			return err
		}

		// Credit destination
		toW.Balance = toW.Balance.Add(amount)
		toW.Version++
		toW.UpdatedAt = now
		if err := s.walletRepo.UpdateWithVersion(sc, toW); err != nil {
			return err
		}

		// Build idempotency keys for ledger entries
		debitIdemKey := ""
		creditIdemKey := ""
		if idempotencyKey != "" {
			debitIdemKey = idempotencyKey + ":debit"
			creditIdemKey = idempotencyKey + ":credit"
		}

		// Ledger entries
		debitEntry := &model.LedgerEntry{
			ID:             bson.NewObjectID(),
			WalletID:       fromW.WalletID,
			Type:           _const.LedgerTypeTransferOut,
			Amount:         amount.Neg(),
			BalanceAfter:   fromW.Balance,
			Currency:       fromW.Currency,
			IdempotencyKey: debitIdemKey,
			CreatedAt:      now,
		}
		if err := s.ledgerRepo.Insert(sc, debitEntry); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return _const.ErrDuplicateRequest
			}
			return err
		}

		creditEntry := &model.LedgerEntry{
			ID:             bson.NewObjectID(),
			WalletID:       toW.WalletID,
			Type:           _const.LedgerTypeTransferIn,
			Amount:         amount,
			BalanceAfter:   toW.Balance,
			Currency:       toW.Currency,
			IdempotencyKey: creditIdemKey,
			CreatedAt:      now,
		}
		if err := s.ledgerRepo.Insert(sc, creditEntry); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return _const.ErrDuplicateRequest
			}
			return err
		}

		return nil
	})

	if errors.Is(err, _const.ErrDuplicateRequest) {
		return nil // idempotent: already processed
	}
	return err
}

// SuspendWallet marks a wallet as suspended.
func (s *WalletService) SuspendWallet(ctx context.Context, walletID string) (*model.Wallet, error) {
	var result *model.Wallet
	err := s.withTransaction(ctx, func(sc context.Context) error {
		w, err := s.walletRepo.FindByWalletID(sc, walletID)
		if err != nil {
			return err
		}

		if w.Status == _const.WalletStatusSuspended {
			return _const.ErrWalletSuspended
		}

		w.Status = _const.WalletStatusSuspended
		w.Version++
		w.UpdatedAt = time.Now().UTC()

		if err := s.walletRepo.UpdateWithVersion(sc, w); err != nil {
			return err
		}
		result = w
		return nil
	})
	return result, err
}

// UnsuspendWallet marks a wallet as active.
func (s *WalletService) UnsuspendWallet(ctx context.Context, walletID string) (*model.Wallet, error) {
	var result *model.Wallet
	err := s.withTransaction(ctx, func(sc context.Context) error {
		w, err := s.walletRepo.FindByWalletID(sc, walletID)
		if err != nil {
			return err
		}

		if w.Status == _const.WalletStatusActive {
			return _const.ErrWalletAlreadyActive
		}

		w.Status = _const.WalletStatusActive
		w.Version++
		w.UpdatedAt = time.Now().UTC()

		if err := s.walletRepo.UpdateWithVersion(sc, w); err != nil {
			return err
		}
		result = w
		return nil
	})
	return result, err
}

// withTransaction runs fn inside a MongoDB transaction with automatic retry on transient errors.
func (s *WalletService) withTransaction(ctx context.Context, fn func(sc context.Context) error) error {
	session, err := s.client.StartSession()
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
		if err := fn(sc); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}
