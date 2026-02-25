package dto

import (
	"errors"
	"net/http"
	"vaultgo/internal/model"
	_const "vaultgo/pkg/const"
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type WalletResponse struct {
	WalletID  string `json:"wallet_id"`
	OwnerID   string `json:"owner_id"`
	Currency  string `json:"currency"`
	Balance   string `json:"balance"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type LedgerEntryResponse struct {
	EntryID        string `json:"entry_id"`
	WalletID       string `json:"wallet_id"`
	Type           string `json:"type"`
	Amount         string `json:"amount"`
	BalanceAfter   string `json:"balance_after"`
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	CreatedAt      string `json:"created_at"`
}

func ToWalletResponse(w *model.Wallet) WalletResponse {
	return WalletResponse{
		WalletID:  w.WalletID,
		OwnerID:   w.OwnerID,
		Currency:  w.Currency,
		Balance:   w.Balance.StringFixed(2),
		Status:    w.Status,
		CreatedAt: w.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: w.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func ToLedgerEntryResponse(e *model.LedgerEntry) LedgerEntryResponse {
	return LedgerEntryResponse{
		EntryID:        e.ID.Hex(),
		WalletID:       e.WalletID,
		Type:           e.Type,
		Amount:         e.Amount.StringFixed(2),
		BalanceAfter:   e.BalanceAfter.StringFixed(2),
		Currency:       e.Currency,
		IdempotencyKey: e.IdempotencyKey,
		CreatedAt:      e.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func MapErrorToStatus(err error) int {
	switch {
	case errors.Is(err, _const.ErrWalletNotFound):
		return http.StatusNotFound
	case errors.Is(err, _const.ErrDuplicateWallet):
		return http.StatusConflict
	case errors.Is(err, _const.ErrVersionConflict):
		return http.StatusConflict
	case errors.Is(err, _const.ErrInvalidAmount):
		return http.StatusBadRequest
	case errors.Is(err, _const.ErrInvalidAmountFormat):
		return http.StatusBadRequest
	case errors.Is(err, _const.ErrInvalidCurrency):
		return http.StatusBadRequest
	case errors.Is(err, _const.ErrInvalidOwnerID):
		return http.StatusBadRequest
	case errors.Is(err, _const.ErrInsufficientFunds):
		return http.StatusUnprocessableEntity
	case errors.Is(err, _const.ErrWalletSuspended):
		return http.StatusForbidden
	case errors.Is(err, _const.ErrWalletAlreadyActive):
		return http.StatusConflict
	case errors.Is(err, _const.ErrCurrencyMismatch):
		return http.StatusBadRequest
	case errors.Is(err, _const.ErrSameWallet):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
