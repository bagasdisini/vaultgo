package dto

type CreateWalletRequest struct {
	OwnerID  string `json:"owner_id"  binding:"required"`
	Currency string `json:"currency"  binding:"required"`
}

type AmountRequest struct {
	Amount string `json:"amount" binding:"required"`
}

type TransferRequest struct {
	FromWalletID string `json:"from_wallet_id" binding:"required"`
	ToWalletID   string `json:"to_wallet_id"   binding:"required"`
	Amount       string `json:"amount"         binding:"required"`
}
