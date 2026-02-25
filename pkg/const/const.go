package _const

import (
	"errors"

	"github.com/shopspring/decimal"
)

const (
	LedgerTypeTopUp       = "topup"
	LedgerTypePayment     = "payment"
	LedgerTypeTransferIn  = "transfer_in"
	LedgerTypeTransferOut = "transfer_out"
)

const (
	WalletStatusActive    = "ACTIVE"
	WalletStatusSuspended = "SUSPENDED"
)

var (
	WalletMinUnit = decimal.NewFromFloat(0.01)
)

// ValidISO4217Currencies is the set of active ISO 4217 currency codes.
var ValidISO4217Currencies = map[string]struct{}{
	"AED": {}, "AFN": {}, "ALL": {}, "AMD": {}, "ANG": {}, "AOA": {}, "ARS": {},
	"AUD": {}, "AWG": {}, "AZN": {}, "BAM": {}, "BBD": {}, "BDT": {}, "BGN": {},
	"BHD": {}, "BIF": {}, "BMD": {}, "BND": {}, "BOB": {}, "BOV": {}, "BRL": {},
	"BSD": {}, "BTN": {}, "BWP": {}, "BYN": {}, "BZD": {}, "CAD": {}, "CDF": {},
	"CHE": {}, "CHF": {}, "CHW": {}, "CLF": {}, "CLP": {}, "CNY": {}, "COP": {},
	"COU": {}, "CRC": {}, "CUP": {}, "CVE": {}, "CZK": {}, "DJF": {}, "DKK": {},
	"DOP": {}, "DZD": {}, "EGP": {}, "ERN": {}, "ETB": {}, "EUR": {}, "FJD": {},
	"FKP": {}, "GBP": {}, "GEL": {}, "GHS": {}, "GIP": {}, "GMD": {}, "GNF": {},
	"GTQ": {}, "GYD": {}, "HKD": {}, "HNL": {}, "HTG": {}, "HUF": {}, "IDR": {},
	"ILS": {}, "INR": {}, "IQD": {}, "IRR": {}, "ISK": {}, "JMD": {}, "JOD": {},
	"JPY": {}, "KES": {}, "KGS": {}, "KHR": {}, "KMF": {}, "KPW": {}, "KRW": {},
	"KWD": {}, "KYD": {}, "KZT": {}, "LAK": {}, "LBP": {}, "LKR": {}, "LRD": {},
	"LSL": {}, "LYD": {}, "MAD": {}, "MDL": {}, "MGA": {}, "MKD": {}, "MMK": {},
	"MNT": {}, "MOP": {}, "MRU": {}, "MUR": {}, "MVR": {}, "MWK": {}, "MXN": {},
	"MXV": {}, "MYR": {}, "MZN": {}, "NAD": {}, "NGN": {}, "NIO": {}, "NOK": {},
	"NPR": {}, "NZD": {}, "OMR": {}, "PAB": {}, "PEN": {}, "PGK": {}, "PHP": {},
	"PKR": {}, "PLN": {}, "PYG": {}, "QAR": {}, "RON": {}, "RSD": {}, "RUB": {},
	"RWF": {}, "SAR": {}, "SBD": {}, "SCR": {}, "SDG": {}, "SEK": {}, "SGD": {},
	"SHP": {}, "SLE": {}, "SLL": {}, "SOS": {}, "SRD": {}, "SSP": {}, "STN": {},
	"SVC": {}, "SYP": {}, "SZL": {}, "THB": {}, "TJS": {}, "TMT": {}, "TND": {},
	"TOP": {}, "TRY": {}, "TTD": {}, "TWD": {}, "TZS": {}, "UAH": {}, "UGX": {},
	"USD": {}, "USN": {}, "UYI": {}, "UYU": {}, "UYW": {}, "UZS": {}, "VED": {},
	"VES": {}, "VND": {}, "VUV": {}, "WST": {}, "XAF": {}, "XAG": {}, "XAU": {},
	"XBA": {}, "XBB": {}, "XBC": {}, "XBD": {}, "XCD": {}, "XDR": {}, "XOF": {},
	"XPD": {}, "XPF": {}, "XPT": {}, "XSU": {}, "XTS": {}, "XUA": {}, "XXX": {},
	"YER": {}, "ZAR": {}, "ZMW": {}, "ZWG": {},
}

var (
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrWalletSuspended     = errors.New("wallet is suspended")
	ErrWalletAlreadyActive = errors.New("wallet is already active")
	ErrSameWallet          = errors.New("cannot transfer to the same wallet")
	ErrDuplicateWallet     = errors.New("wallet already exists for this owner and currency")

	ErrInvalidAmount       = errors.New("amount must be positive and have at most 2 decimal places")
	ErrInvalidAmountFormat = errors.New("invalid amount format")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrCurrencyMismatch    = errors.New("transfer currencies do not match")
	ErrInvalidCurrency     = errors.New("invalid currency code")
	ErrInvalidOwnerID      = errors.New("owner_id must not be empty")

	ErrDuplicateRequest = errors.New("duplicate request (idempotency key already used)")
	ErrVersionConflict  = errors.New("version conflict: wallet was modified concurrently")

	SuccessWalletCreated     = "wallet created successfully"
	SuccessGetWallet         = "wallet retrieved successfully"
	SuccessTopUpWallet       = "wallet topped up successfully"
	SuccessWalletSuspended   = "wallet suspended successfully"
	SuccessWalletUnsuspended = "wallet unsuspended successfully"
	SuccessPayment           = "payment successfully"
	SuccessTransfer          = "transfer successfully"
	SuccessGetLedger         = "ledger entries retrieved successfully"
)
