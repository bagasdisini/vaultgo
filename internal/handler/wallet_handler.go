package handler

import (
	"net/http"
	"strings"
	"vaultgo/internal/dto"
	"vaultgo/internal/service"
	_const "vaultgo/pkg/const"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type WalletHandler struct {
	svc *service.WalletService
}

func NewWalletHandler(svc *service.WalletService) *WalletHandler {
	return &WalletHandler{
		svc: svc,
	}
}

// RegisterRoutes sets up the Gin router group.
func (h *WalletHandler) RegisterRoutes(r *gin.Engine) {
	wallets := r.Group("/wallets")

	{
		wallets.GET("", h.GetAllWallets)
		wallets.GET("/:id", h.GetWallet)
		wallets.GET("/:id/ledger", h.GetLedger)
		wallets.GET("/owner/:owner_id", h.GetWalletsByOwner)

		wallets.POST("", h.CreateWallet)
		wallets.POST("/transfer", h.Transfer)
		wallets.POST("/:id/topup", h.TopUp)
		wallets.POST("/:id/pay", h.Pay)
		wallets.POST("/:id/suspend", h.SuspendWallet)
		wallets.POST("/:id/unsuspend", h.UnsuspendWallet)
	}
}

// GetAllWallets
// @Tags Wallet
// @Summary Get all wallets
// @Description Retrieve a list of all wallets in the system
// @ID get-all-wallets
// @Router /wallets [get]
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{data=[]dto.WalletResponse}
func (h *WalletHandler) GetAllWallets(c *gin.Context) {
	wallets, err := h.svc.GetWallets(c.Request.Context())
	if err != nil {
		c.JSON(dto.MapErrorToStatus(err), dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	walletResponses := make([]dto.WalletResponse, 0, len(wallets))
	for _, w := range wallets {
		walletResponses = append(walletResponses, dto.ToWalletResponse(w))
	}

	c.JSON(http.StatusOK, dto.Response{
		Status:  "success",
		Message: _const.SuccessGetWallet,
		Data:    walletResponses,
	})
}

// GetWallet
// @Tags Wallet
// @Summary Get wallet details
// @Description Retrieve wallet information by Wallet ID
// @ID get-wallet
// @Param id path string true "Wallet ID"
// @Router /wallets/{id} [get]
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{data=dto.WalletResponse}
func (h *WalletHandler) GetWallet(c *gin.Context) {
	w, err := h.svc.GetWallet(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(dto.MapErrorToStatus(err), dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Status:  "success",
		Message: _const.SuccessGetWallet,
		Data:    dto.ToWalletResponse(w),
	})
}

// GetWalletsByOwner
// @Tags Wallet
// @Summary Get wallets by owner
// @Description Retrieve all wallets associated with a specific owner ID
// @ID get-wallets-by-owner
// @Param owner_id path string true "Owner ID"
// @Router /wallets/owner/{owner_id} [get]
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{data=[]dto.WalletResponse}
func (h *WalletHandler) GetWalletsByOwner(c *gin.Context) {
	ownerID := c.Param("owner_id")

	wallets, err := h.svc.GetWalletsByOwner(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(dto.MapErrorToStatus(err), dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	walletResponses := make([]dto.WalletResponse, 0, len(wallets))
	for _, w := range wallets {
		walletResponses = append(walletResponses, dto.ToWalletResponse(w))
	}

	c.JSON(http.StatusOK, dto.Response{
		Status:  "success",
		Message: _const.SuccessGetWallet,
		Data:    walletResponses,
	})
}

// GetLedger
// @Tags Ledger
// @Summary Get ledger entries for a wallet
// @Description Retrieve all ledger entries for a given wallet ID, ordered by creation time
// @ID get-ledger
// @Param id path string true "Wallet ID"
// @Router /wallets/{id}/ledger [get]
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{data=[]dto.LedgerEntryResponse}
func (h *WalletHandler) GetLedger(c *gin.Context) {
	entries, err := h.svc.GetLedgerEntries(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(dto.MapErrorToStatus(err), dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	ledgerResponses := make([]dto.LedgerEntryResponse, 0, len(entries))
	for i := range entries {
		ledgerResponses = append(ledgerResponses, dto.ToLedgerEntryResponse(&entries[i]))
	}

	c.JSON(http.StatusOK, dto.Response{
		Status:  "success",
		Message: _const.SuccessGetLedger,
		Data:    ledgerResponses,
	})
}

// Transfer
// @Tags Wallet
// @Summary Transfer funds between wallets
// @Description Transfer a specified amount from one wallet to another
// @ID transfer-wallet
// @Param body body dto.TransferRequest true "Transfer details"
// @Param Idempotency-Key header string false "Idempotency Key for ensuring idempotent requests"
// @Router /wallets/transfer [post]
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response
func (h *WalletHandler) Transfer(c *gin.Context) {
	var req dto.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Status:  "error",
			Message: "invalid amount format",
		})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	if err := h.svc.Transfer(c.Request.Context(), req.FromWalletID, req.ToWalletID, amount, idempotencyKey); err != nil {
		c.JSON(dto.MapErrorToStatus(err), dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Status:  "success",
		Message: _const.SuccessTransfer,
	})
}

// CreateWallet
// @Tags Wallet
// @Summary Create a new wallet
// @Description Create a wallet for a specific owner and currency
// @ID create-wallet
// @Param body body dto.CreateWalletRequest true "Wallet creation details"
// @Router /wallets [post]
// @Accept json
// @Produce json
// @Success 201 {object} dto.Response{data=dto.WalletResponse}
func (h *WalletHandler) CreateWallet(c *gin.Context) {
	var req dto.CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	w, err := h.svc.CreateWallet(c.Request.Context(), req.OwnerID, strings.ToUpper(req.Currency))
	if err != nil {
		c.JSON(dto.MapErrorToStatus(err), dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.Response{
		Status:  "success",
		Message: _const.SuccessWalletCreated,
		Data:    dto.ToWalletResponse(w),
	})
}

// TopUp
// @Tags Wallet
// @Summary Top up wallet balance
// @Description Add funds to a wallet by ID
// @ID topup-wallet
// @Param id path string true "Wallet ID"
// @Param body body dto.AmountRequest true "Amount to top up"
// @Param Idempotency-Key header string false "Idempotency Key for ensuring idempotent requests"
// @Router /wallets/{id}/topup [post]
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{data=dto.WalletResponse}
func (h *WalletHandler) TopUp(c *gin.Context) {
	var req dto.AmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Status:  "error",
			Message: _const.ErrInvalidAmountFormat.Error(),
		})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	w, err := h.svc.TopUp(c.Request.Context(), c.Param("id"), amount, idempotencyKey)
	if err != nil {
		c.JSON(dto.MapErrorToStatus(err), dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Status:  "success",
		Message: _const.SuccessTopUpWallet,
		Data:    dto.ToWalletResponse(w),
	})
}

// Pay
// @Tags Wallet
// @Summary Pay from wallet
// @Description Deduct funds from a wallet by ID
// @ID pay-wallet
// @Param id path string true "Wallet ID"
// @Param body body dto.AmountRequest true "Amount to pay"
// @Param Idempotency-Key header string false "Idempotency Key for ensuring idempotent requests"
// @Router /wallets/{id}/pay [post]
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{data=dto.WalletResponse}
func (h *WalletHandler) Pay(c *gin.Context) {
	var req dto.AmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Status:  "error",
			Message: _const.ErrInvalidAmountFormat.Error(),
		})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	w, err := h.svc.Pay(c.Request.Context(), c.Param("id"), amount, idempotencyKey)
	if err != nil {
		c.JSON(dto.MapErrorToStatus(err), dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Status:  "success",
		Message: _const.SuccessPayment,
		Data:    dto.ToWalletResponse(w),
	})
}

// SuspendWallet
// @Tags Wallet
// @Summary Suspend a wallet
// @Description Suspend a wallet by ID, preventing any transactions until reactivated
// @ID suspend-wallet
// @Param id path string true "Wallet ID"
// @Router /wallets/{id}/suspend [post]
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{data=dto.WalletResponse}
func (h *WalletHandler) SuspendWallet(c *gin.Context) {
	w, err := h.svc.SuspendWallet(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(dto.MapErrorToStatus(err), dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Status:  "success",
		Message: _const.SuccessWalletSuspended,
		Data:    dto.ToWalletResponse(w),
	})
}

// UnsuspendWallet
// @Tags Wallet
// @Summary Unsuspend a wallet
// @Description Reactivate a suspended wallet by ID, allowing transactions to resume
// @ID unsuspend-wallet
// @Param id path string true "Wallet ID"
// @Router /wallets/{id}/unsuspend [post]
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{data=dto.WalletResponse}
func (h *WalletHandler) UnsuspendWallet(c *gin.Context) {
	w, err := h.svc.UnsuspendWallet(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(dto.MapErrorToStatus(err), dto.Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Status:  "success",
		Message: _const.SuccessWalletUnsuspended,
		Data:    dto.ToWalletResponse(w),
	})
}
