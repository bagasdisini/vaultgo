package test

import (
	"testing"
	"vaultgo/internal/service"

	"github.com/shopspring/decimal"
)

func TestValidateAmount(t *testing.T) {
	tests := []struct {
		name    string
		amount  string
		wantErr bool
	}{
		{"valid 100.00", "100.00", false},
		{"valid 0.01", "0.01", false},
		{"valid 1000000000.00", "1000000000.00", false},
		{"valid 12.50", "12.50", false},
		{"valid integer", "100", false},
		{"valid one decimal", "10.5", false},

		{"zero", "0.00", true},
		{"negative", "-10.00", true},
		{"too many decimals 12.345", "12.345", true},
		{"too many decimals 0.001", "0.001", true},
		{"very small below min", "0.001", true},
		{"negative amount", "-5.00", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := decimal.NewFromString(tt.amount)
			if err != nil {
				t.Fatalf("invalid test amount: %s", tt.amount)
			}
			err = service.ValidateAmount(amount)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAmount(%s) error = %v, wantErr %v", tt.amount, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		wantErr  bool
	}{
		{"valid USD", "USD", false},
		{"valid EUR", "EUR", false},
		{"valid IDR", "IDR", false},
		{"lowercase", "usd", true},

		{"too short", "US", true},
		{"too long", "USDD", true},
		{"empty", "", true},
		{"numbers", "123", true},
		{"mixed", "U1D", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateCurrency(tt.currency)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCurrency(%s) error = %v, wantErr %v", tt.currency, err, tt.wantErr)
			}
		})
	}
}
