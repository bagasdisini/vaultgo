package test

import (
	"context"
	"sync"
	"testing"
	"time"
	"vaultgo/config"
	"vaultgo/internal/service"
	_const "vaultgo/pkg/const"

	"vaultgo/internal/model"
	"vaultgo/internal/repository"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// getTestDB returns a MongoDB database for integration tests.
// Set MONGO_URI env var to point to a replica set (e.g., mongodb://localhost:27017/?replicaSet=rs0).
// If MONGO_URI is not set, integration tests are skipped.
func getTestDB(t *testing.T) (*mongo.Client, *mongo.Database) {
	t.Helper()

	cf := config.Load()
	if cf.MongoURI == "" {
		t.Skip("MONGO_URI not set, skipping integration test")
	}

	registry := model.NewDecimalRegistry()
	opts := options.Client().ApplyURI(cf.MongoURI).SetRegistry(registry)

	client, err := mongo.Connect(opts)
	if err != nil {
		t.Fatalf("failed to connect to mongo: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("failed to ping mongo: %v", err)
	}

	dbName := "vaultgo_test_" + time.Now().Format("20060102150405")
	db := client.Database(dbName)

	t.Cleanup(func() {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel2()
		_ = db.Drop(ctx2)
		_ = client.Disconnect(ctx2)
	})
	return client, db
}

func setupService(t *testing.T) *service.WalletService {
	t.Helper()
	client, db := getTestDB(t)
	walletRepo := repository.NewWalletRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)

	ctx := context.Background()
	if err := walletRepo.CreateIndexes(ctx); err != nil {
		t.Fatalf("failed to create indexes: %v", err)
	}
	if err := ledgerRepo.CreateIndexes(ctx); err != nil {
		t.Fatalf("failed to create ledger indexes: %v", err)
	}
	return service.NewWalletService(walletRepo, ledgerRepo, client)
}

func TestCreateWallet(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, err := svc.CreateWallet(ctx, "user1", "USD")
	if err != nil {
		t.Fatalf("CreateWallet failed: %v", err)
	}
	if w.OwnerID != "user1" {
		t.Errorf("expected owner_id user1, got %s", w.OwnerID)
	}
	if w.Currency != "USD" {
		t.Errorf("expected currency USD, got %s", w.Currency)
	}
	if !w.Balance.Equal(decimal.Zero) {
		t.Errorf("expected balance 0, got %s", w.Balance)
	}
	if w.Status != _const.WalletStatusActive {
		t.Errorf("expected status ACTIVE, got %s", w.Status)
	}
}

func TestCreateWallet_DuplicateCurrency(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	_, err := svc.CreateWallet(ctx, "user1", "USD")
	if err != nil {
		t.Fatalf("first CreateWallet failed: %v", err)
	}

	_, err = svc.CreateWallet(ctx, "user1", "USD")
	if err == nil {
		t.Fatal("expected error for duplicate wallet, got nil")
	}
}

func TestCreateWallet_MultiCurrency(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	_, err := svc.CreateWallet(ctx, "user1", "USD")
	if err != nil {
		t.Fatalf("CreateWallet USD failed: %v", err)
	}
	_, err = svc.CreateWallet(ctx, "user1", "EUR")
	if err != nil {
		t.Fatalf("CreateWallet EUR failed: %v", err)
	}
}

func TestCreateWallet_InvalidCurrency(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	_, err := svc.CreateWallet(ctx, "user1", "XYZ")
	if err == nil {
		t.Fatal("expected error for invalid currency, got nil")
	}

	_, err = svc.CreateWallet(ctx, "user1", "usd")
	if err == nil {
		t.Fatal("expected error for lowercase currency, got nil")
	}

	_, err = svc.CreateWallet(ctx, "user1", "")
	if err == nil {
		t.Fatal("expected error for empty currency, got nil")
	}
}

func TestCreateWallet_EmptyOwnerID(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	_, err := svc.CreateWallet(ctx, "", "USD")
	if err == nil {
		t.Fatal("expected error for empty owner_id, got nil")
	}

	_, err = svc.CreateWallet(ctx, "   ", "USD")
	if err == nil {
		t.Fatal("expected error for whitespace-only owner_id, got nil")
	}
}

func TestTopUp(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")

	w, err := svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(1000.50), "topup-1")
	if err != nil {
		t.Fatalf("TopUp failed: %v", err)
	}
	if !w.Balance.Equal(decimal.NewFromFloat(1000.50)) {
		t.Errorf("expected balance 1000.50, got %s", w.Balance)
	}
}

func TestTopUp_Idempotent(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")

	_, err := svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(100.00), "idem-topup-1")
	if err != nil {
		t.Fatalf("TopUp failed: %v", err)
	}

	// Repeat with same idempotency key — should not add again
	w2, err := svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(100.00), "idem-topup-1")
	if err != nil {
		t.Fatalf("TopUp idempotent failed: %v", err)
	}
	if !w2.Balance.Equal(decimal.NewFromFloat(100.00)) {
		t.Errorf("idempotent topup should still be 100.00, got %s", w2.Balance)
	}
}

func TestTopUp_SuspendedWallet(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")
	svc.SuspendWallet(ctx, w.WalletID)

	_, err := svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(100.00), "topup-susp")
	if err == nil {
		t.Fatal("expected error for topup on suspended wallet, got nil")
	}
}

func TestTopUp_WalletNotFound(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	_, err := svc.TopUp(ctx, "VAULT-NONEXISTENT", decimal.NewFromFloat(100.00), "topup-nf")
	if err == nil {
		t.Fatal("expected error for non-existent wallet, got nil")
	}
}

func TestTopUp_WithoutIdempotencyKey(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")

	// Multiple top-ups without idempotency key should all succeed
	w, err := svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(100.00), "")
	if err != nil {
		t.Fatalf("TopUp 1 failed: %v", err)
	}

	w, err = svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(100.00), "")
	if err != nil {
		t.Fatalf("TopUp 2 failed: %v", err)
	}

	if !w.Balance.Equal(decimal.NewFromFloat(200.00)) {
		t.Errorf("expected balance 200.00 after two top-ups, got %s", w.Balance)
	}
}

func TestPay(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")
	w, _ = svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(500.00), "topup-pay")

	w, err := svc.Pay(ctx, w.WalletID, decimal.NewFromFloat(200.10), "pay-1")
	if err != nil {
		t.Fatalf("Pay failed: %v", err)
	}
	expected := decimal.NewFromFloat(299.90)
	if !w.Balance.Equal(expected) {
		t.Errorf("expected balance %s, got %s", expected, w.Balance)
	}
}

func TestPay_InsufficientFunds(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")
	w, _ = svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(10.00), "topup-insuf")

	_, err := svc.Pay(ctx, w.WalletID, decimal.NewFromFloat(20.00), "pay-insuf")
	if err == nil {
		t.Fatal("expected insufficient funds error, got nil")
	}
}

func TestPay_ZeroAmount(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")

	_, err := svc.Pay(ctx, w.WalletID, decimal.Zero, "pay-zero")
	if err == nil {
		t.Fatal("expected error for zero amount, got nil")
	}
}

func TestPay_NegativeAmount(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")

	_, err := svc.Pay(ctx, w.WalletID, decimal.NewFromFloat(-10.00), "pay-neg")
	if err == nil {
		t.Fatal("expected error for negative amount, got nil")
	}
}

func TestPay_Idempotent(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")
	svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(500.00), "topup-pay-idem")

	_, err := svc.Pay(ctx, w.WalletID, decimal.NewFromFloat(100.00), "idem-pay-1")
	if err != nil {
		t.Fatalf("Pay failed: %v", err)
	}

	// Repeat with same idempotency key
	w2, err := svc.Pay(ctx, w.WalletID, decimal.NewFromFloat(100.00), "idem-pay-1")
	if err != nil {
		t.Fatalf("Pay idempotent returned error: %v", err)
	}

	if !w2.Balance.Equal(decimal.NewFromFloat(400.00)) {
		t.Errorf("idempotent pay should keep balance at 400.00, got %s", w2.Balance)
	}
}

func TestPay_WalletNotFound(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	_, err := svc.Pay(ctx, "VAULT-NONEXISTENT", decimal.NewFromFloat(100.00), "pay-nf")
	if err == nil {
		t.Fatal("expected error for non-existent wallet, got nil")
	}
}

func TestTransfer(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	w2, _ := svc.CreateWallet(ctx, "user2", "USD")

	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(1000.00), "topup-xfer")

	err := svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(300.40), "xfer-1")
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}

	// Check balances
	w1After, _ := svc.GetWallet(ctx, w1.WalletID)
	w2After, _ := svc.GetWallet(ctx, w2.WalletID)

	expectedW1 := decimal.NewFromFloat(699.60)
	expectedW2 := decimal.NewFromFloat(300.40)

	if !w1After.Balance.Equal(expectedW1) {
		t.Errorf("w1 expected %s, got %s", expectedW1, w1After.Balance)
	}
	if !w2After.Balance.Equal(expectedW2) {
		t.Errorf("w2 expected %s, got %s", expectedW2, w2After.Balance)
	}
}

func TestTransfer_CurrencyMismatch(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	w2, _ := svc.CreateWallet(ctx, "user2", "EUR")

	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(100.00), "topup-mismatch")

	err := svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(50.00), "xfer-mismatch")
	if err == nil {
		t.Fatal("expected currency mismatch error, got nil")
	}
}

func TestTransfer_DestinationNotFound(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(100.00), "topup-noexist")

	err := svc.Transfer(ctx, w1.WalletID, "VAULT-NONEXISTENT", decimal.NewFromFloat(50.00), "xfer-noexist")
	if err == nil {
		t.Fatal("expected error for missing destination wallet, got nil")
	}
}

func TestTransfer_Idempotent(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	w2, _ := svc.CreateWallet(ctx, "user2", "USD")
	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(1000.00), "topup-xfer-idem")

	err := svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(100.00), "xfer-idem-1")
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}

	// Repeat with same idempotency key — should not transfer again
	err = svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(100.00), "xfer-idem-1")
	if err != nil {
		t.Fatalf("Idempotent transfer returned error: %v", err)
	}

	w1After, _ := svc.GetWallet(ctx, w1.WalletID)
	w2After, _ := svc.GetWallet(ctx, w2.WalletID)

	if !w1After.Balance.Equal(decimal.NewFromFloat(900.00)) {
		t.Errorf("w1 expected 900.00, got %s", w1After.Balance)
	}
	if !w2After.Balance.Equal(decimal.NewFromFloat(100.00)) {
		t.Errorf("w2 expected 100.00, got %s", w2After.Balance)
	}
}

func TestTransfer_EmptyIdempotencyKey(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	w2, _ := svc.CreateWallet(ctx, "user2", "USD")
	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(500.00), "topup-empty-idem")

	// Multiple transfers without idempotency key should all succeed
	err := svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(100.00), "")
	if err != nil {
		t.Fatalf("Transfer 1 failed: %v", err)
	}

	err = svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(100.00), "")
	if err != nil {
		t.Fatalf("Transfer 2 failed: %v", err)
	}

	w1After, _ := svc.GetWallet(ctx, w1.WalletID)
	w2After, _ := svc.GetWallet(ctx, w2.WalletID)

	if !w1After.Balance.Equal(decimal.NewFromFloat(300.00)) {
		t.Errorf("w1 expected 300.00, got %s", w1After.Balance)
	}
	if !w2After.Balance.Equal(decimal.NewFromFloat(200.00)) {
		t.Errorf("w2 expected 200.00, got %s", w2After.Balance)
	}
}

func TestTransfer_InsufficientFunds(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	w2, _ := svc.CreateWallet(ctx, "user2", "USD")
	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(50.00), "topup-xfer-insuf")

	err := svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(100.00), "xfer-insuf")
	if err == nil {
		t.Fatal("expected insufficient funds error, got nil")
	}
}

func TestTransfer_SameWallet(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")
	svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(100.00), "topup-same")

	err := svc.Transfer(ctx, w.WalletID, w.WalletID, decimal.NewFromFloat(50.00), "xfer-same")
	if err == nil {
		t.Fatal("expected error for same wallet transfer, got nil")
	}
}

func TestTransfer_DestinationSuspended(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	w2, _ := svc.CreateWallet(ctx, "user2", "USD")
	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(200.00), "topup-dest-susp")
	svc.SuspendWallet(ctx, w2.WalletID)

	err := svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(50.00), "xfer-dest-susp")
	if err == nil {
		t.Fatal("expected error for transfer to suspended wallet, got nil")
	}
}

func TestTransfer_ZeroAmount(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	w2, _ := svc.CreateWallet(ctx, "user2", "USD")
	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(100.00), "topup-xfer-zero")

	err := svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.Zero, "xfer-zero")
	if err == nil {
		t.Fatal("expected error for zero amount transfer, got nil")
	}
}

func TestTransfer_NegativeAmount(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	w2, _ := svc.CreateWallet(ctx, "user2", "USD")
	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(100.00), "topup-xfer-neg")

	err := svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(-50.00), "xfer-neg")
	if err == nil {
		t.Fatal("expected error for negative amount transfer, got nil")
	}
}

func TestConcurrentOppositeTransfers(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	w2, _ := svc.CreateWallet(ctx, "user2", "USD")
	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(1000.00), "topup-conc-xfer-1")
	svc.TopUp(ctx, w2.WalletID, decimal.NewFromFloat(1000.00), "topup-conc-xfer-2")

	// Concurrent opposite transfers should not deadlock
	var wg sync.WaitGroup
	errors1 := make([]error, 10)
	errors2 := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			errors1[idx] = svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(10.00), "")
		}(i)
		go func(idx int) {
			defer wg.Done()
			errors2[idx] = svc.Transfer(ctx, w2.WalletID, w1.WalletID, decimal.NewFromFloat(10.00), "")
		}(i)
	}
	wg.Wait()

	// Verify no negative balances
	w1Final, _ := svc.GetWallet(ctx, w1.WalletID)
	w2Final, _ := svc.GetWallet(ctx, w2.WalletID)

	if w1Final.Balance.LessThan(decimal.Zero) {
		t.Fatalf("w1 balance went negative: %s", w1Final.Balance)
	}
	if w2Final.Balance.LessThan(decimal.Zero) {
		t.Fatalf("w2 balance went negative: %s", w2Final.Balance)
	}

	// Total money in the system should be conserved (2000.00)
	totalBalance := w1Final.Balance.Add(w2Final.Balance)
	if !totalBalance.Equal(decimal.NewFromFloat(2000.00)) {
		t.Errorf("money not conserved: expected total 2000.00, got %s", totalBalance)
	}

	t.Logf("concurrent opposite transfers: w1=%s, w2=%s, total=%s",
		w1Final.Balance, w2Final.Balance, totalBalance)
}

func TestSuspendWallet(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")
	svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(100.00), "topup-suspend")

	w, err := svc.SuspendWallet(ctx, w.WalletID)
	if err != nil {
		t.Fatalf("SuspendWallet failed: %v", err)
	}
	if w.Status != _const.WalletStatusSuspended {
		t.Errorf("expected SUSPENDED, got %s", w.Status)
	}

	// TopUp should fail
	_, err = svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(50.00), "topup-after-suspend")
	if err == nil {
		t.Fatal("expected error for topup on suspended wallet, got nil")
	}

	// Pay should fail
	_, err = svc.Pay(ctx, w.WalletID, decimal.NewFromFloat(10.00), "pay-after-suspend")
	if err == nil {
		t.Fatal("expected error for pay on suspended wallet, got nil")
	}
}

func TestSuspendWallet_TransferBlocked(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	w2, _ := svc.CreateWallet(ctx, "user2", "USD")
	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(200.00), "topup-xfer-susp")

	svc.SuspendWallet(ctx, w1.WalletID)

	err := svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(50.00), "xfer-susp")
	if err == nil {
		t.Fatal("expected error for transfer from suspended wallet, got nil")
	}
}

func TestSuspendWallet_AlreadySuspended(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")

	_, err := svc.SuspendWallet(ctx, w.WalletID)
	if err != nil {
		t.Fatalf("SuspendWallet failed: %v", err)
	}

	_, err = svc.SuspendWallet(ctx, w.WalletID)
	if err == nil {
		t.Fatal("expected error for suspending already suspended wallet, got nil")
	}
}

func TestLargeBalance(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")

	large := decimal.NewFromFloat(1000000000.00)
	w, err := svc.TopUp(ctx, w.WalletID, large, "topup-large")
	if err != nil {
		t.Fatalf("TopUp large amount failed: %v", err)
	}
	if !w.Balance.Equal(large) {
		t.Errorf("expected %s, got %s", large, w.Balance)
	}

	// Top up again
	w, err = svc.TopUp(ctx, w.WalletID, large, "topup-large-2")
	if err != nil {
		t.Fatalf("TopUp large amount second failed: %v", err)
	}
	expected := large.Add(large)
	if !w.Balance.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, w.Balance)
	}
}

func TestDecimalPrecision_Reject3Decimals(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")

	_, err := svc.TopUp(ctx, w.WalletID, decimal.RequireFromString("12.345"), "topup-3dec")
	if err == nil {
		t.Fatal("expected error for 3 decimal places, got nil")
	}
}

func TestConcurrentPayments(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")
	svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(100.00), "topup-concurrent")

	// Try 20 concurrent payments of 10.00 each — only 10 should succeed
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := "pay-concurrent-" + time.Now().Format("150405.000000000") + "-" + string(rune('A'+idx))
			_, err := svc.Pay(ctx, w.WalletID, decimal.NewFromFloat(10.00), key)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	wFinal, _ := svc.GetWallet(ctx, w.WalletID)
	if wFinal.Balance.LessThan(decimal.Zero) {
		t.Fatalf("balance went negative: %s", wFinal.Balance)
	}
	t.Logf("concurrent payments: %d succeeded, final balance: %s", successCount, wFinal.Balance)
}

func TestUnsuspendWallet(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")
	svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(100.00), "topup-unsuspend")

	// Suspend first
	w, err := svc.SuspendWallet(ctx, w.WalletID)
	if err != nil {
		t.Fatalf("SuspendWallet failed: %v", err)
	}
	if w.Status != _const.WalletStatusSuspended {
		t.Errorf("expected SUSPENDED, got %s", w.Status)
	}

	// Unsuspend
	w, err = svc.UnsuspendWallet(ctx, w.WalletID)
	if err != nil {
		t.Fatalf("UnsuspendWallet failed: %v", err)
	}
	if w.Status != _const.WalletStatusActive {
		t.Errorf("expected ACTIVE, got %s", w.Status)
	}

	// Should be able to top up again
	w, err = svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(50.00), "topup-after-unsuspend")
	if err != nil {
		t.Fatalf("TopUp after unsuspend failed: %v", err)
	}
	if !w.Balance.Equal(decimal.NewFromFloat(150.00)) {
		t.Errorf("expected balance 150.00, got %s", w.Balance)
	}
}

func TestUnsuspendWallet_AlreadyActive(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")

	_, err := svc.UnsuspendWallet(ctx, w.WalletID)
	if err == nil {
		t.Fatal("expected error for unsuspending already active wallet, got nil")
	}
}

func TestGetWallet(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	created, _ := svc.CreateWallet(ctx, "user1", "USD")

	w, err := svc.GetWallet(ctx, created.WalletID)
	if err != nil {
		t.Fatalf("GetWallet failed: %v", err)
	}
	if w.WalletID != created.WalletID {
		t.Errorf("expected wallet_id %s, got %s", created.WalletID, w.WalletID)
	}
	if w.OwnerID != "user1" {
		t.Errorf("expected owner_id user1, got %s", w.OwnerID)
	}
}

func TestGetWallet_NotFound(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	_, err := svc.GetWallet(ctx, "VAULT-NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for non-existent wallet, got nil")
	}
}

func TestGetWallets(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	svc.CreateWallet(ctx, "user1", "USD")
	svc.CreateWallet(ctx, "user2", "EUR")

	wallets, err := svc.GetWallets(ctx)
	if err != nil {
		t.Fatalf("GetWallets failed: %v", err)
	}
	if len(wallets) != 2 {
		t.Errorf("expected 2 wallets, got %d", len(wallets))
	}
}

func TestGetWalletsByOwner(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	svc.CreateWallet(ctx, "user1", "USD")
	svc.CreateWallet(ctx, "user1", "EUR")
	svc.CreateWallet(ctx, "user2", "USD")

	wallets, err := svc.GetWalletsByOwner(ctx, "user1")
	if err != nil {
		t.Fatalf("GetWalletsByOwner failed: %v", err)
	}
	if len(wallets) != 2 {
		t.Errorf("expected 2 wallets for user1, got %d", len(wallets))
	}

	wallets, err = svc.GetWalletsByOwner(ctx, "user2")
	if err != nil {
		t.Fatalf("GetWalletsByOwner failed: %v", err)
	}
	if len(wallets) != 1 {
		t.Errorf("expected 1 wallet for user2, got %d", len(wallets))
	}

	wallets, err = svc.GetWalletsByOwner(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetWalletsByOwner failed: %v", err)
	}
	if len(wallets) != 0 {
		t.Errorf("expected 0 wallets for nonexistent user, got %d", len(wallets))
	}
}

func TestGetLedgerEntries(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")

	// Top up
	svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(500.00), "topup-ledger")
	// Pay
	svc.Pay(ctx, w.WalletID, decimal.NewFromFloat(100.00), "pay-ledger")
	// Another top up
	svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(50.00), "topup-ledger-2")

	entries, err := svc.GetLedgerEntries(ctx, w.WalletID)
	if err != nil {
		t.Fatalf("GetLedgerEntries failed: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 ledger entries, got %d", len(entries))
	}

	// Verify entry types and order
	if entries[0].Type != _const.LedgerTypeTopUp {
		t.Errorf("entry 0: expected type %s, got %s", _const.LedgerTypeTopUp, entries[0].Type)
	}
	if !entries[0].Amount.Equal(decimal.NewFromFloat(500.00)) {
		t.Errorf("entry 0: expected amount 500.00, got %s", entries[0].Amount)
	}
	if !entries[0].BalanceAfter.Equal(decimal.NewFromFloat(500.00)) {
		t.Errorf("entry 0: expected balance_after 500.00, got %s", entries[0].BalanceAfter)
	}

	if entries[1].Type != _const.LedgerTypePayment {
		t.Errorf("entry 1: expected type %s, got %s", _const.LedgerTypePayment, entries[1].Type)
	}
	if !entries[1].Amount.Equal(decimal.NewFromFloat(-100.00)) {
		t.Errorf("entry 1: expected amount -100.00, got %s", entries[1].Amount)
	}
	if !entries[1].BalanceAfter.Equal(decimal.NewFromFloat(400.00)) {
		t.Errorf("entry 1: expected balance_after 400.00, got %s", entries[1].BalanceAfter)
	}

	if entries[2].Type != _const.LedgerTypeTopUp {
		t.Errorf("entry 2: expected type %s, got %s", _const.LedgerTypeTopUp, entries[2].Type)
	}
	if !entries[2].Amount.Equal(decimal.NewFromFloat(50.00)) {
		t.Errorf("entry 2: expected amount 50.00, got %s", entries[2].Amount)
	}

	// Verify currency is set on all entries
	for i, e := range entries {
		if e.Currency != "USD" {
			t.Errorf("entry %d: expected currency USD, got %s", i, e.Currency)
		}
		if e.WalletID != w.WalletID {
			t.Errorf("entry %d: expected wallet_id %s, got %s", i, w.WalletID, e.WalletID)
		}
	}
}

func TestGetLedgerEntries_WalletNotFound(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	_, err := svc.GetLedgerEntries(ctx, "VAULT-NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for non-existent wallet, got nil")
	}
}

func TestGetLedgerEntries_TransferCreatesEntries(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	w2, _ := svc.CreateWallet(ctx, "user2", "USD")
	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(500.00), "topup-xfer-ledger")

	svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(200.00), "xfer-ledger")

	// Check source wallet ledger
	entries1, err := svc.GetLedgerEntries(ctx, w1.WalletID)
	if err != nil {
		t.Fatalf("GetLedgerEntries for w1 failed: %v", err)
	}
	if len(entries1) != 2 { // topup + transfer_out
		t.Fatalf("expected 2 entries for w1, got %d", len(entries1))
	}
	if entries1[1].Type != _const.LedgerTypeTransferOut {
		t.Errorf("w1 entry 1: expected type %s, got %s", _const.LedgerTypeTransferOut, entries1[1].Type)
	}

	// Check destination wallet ledger
	entries2, err := svc.GetLedgerEntries(ctx, w2.WalletID)
	if err != nil {
		t.Fatalf("GetLedgerEntries for w2 failed: %v", err)
	}
	if len(entries2) != 1 { // transfer_in
		t.Fatalf("expected 1 entry for w2, got %d", len(entries2))
	}
	if entries2[0].Type != _const.LedgerTypeTransferIn {
		t.Errorf("w2 entry 0: expected type %s, got %s", _const.LedgerTypeTransferIn, entries2[0].Type)
	}
}

func TestLedgerConsistency(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx, "user1", "USD")

	// Perform several operations
	svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(1000.00), "topup-consistency-1")
	svc.Pay(ctx, w.WalletID, decimal.NewFromFloat(250.00), "pay-consistency-1")
	svc.TopUp(ctx, w.WalletID, decimal.NewFromFloat(100.50), "topup-consistency-2")
	svc.Pay(ctx, w.WalletID, decimal.NewFromFloat(50.25), "pay-consistency-2")

	// Get current wallet balance
	wFinal, err := svc.GetWallet(ctx, w.WalletID)
	if err != nil {
		t.Fatalf("GetWallet failed: %v", err)
	}

	// Get all ledger entries and sum them
	entries, err := svc.GetLedgerEntries(ctx, w.WalletID)
	if err != nil {
		t.Fatalf("GetLedgerEntries failed: %v", err)
	}

	ledgerSum := decimal.Zero
	for _, e := range entries {
		ledgerSum = ledgerSum.Add(e.Amount)
	}

	// Wallet balance must equal sum of ledger entries
	if !wFinal.Balance.Equal(ledgerSum) {
		t.Errorf("ledger consistency check failed: wallet balance = %s, ledger sum = %s",
			wFinal.Balance, ledgerSum)
	}

	expectedBalance := decimal.NewFromFloat(800.25) // 1000 - 250 + 100.50 - 50.25
	if !wFinal.Balance.Equal(expectedBalance) {
		t.Errorf("expected balance %s, got %s", expectedBalance, wFinal.Balance)
	}
}

func TestLedgerConsistency_WithTransfer(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	w1, _ := svc.CreateWallet(ctx, "user1", "USD")
	w2, _ := svc.CreateWallet(ctx, "user2", "USD")

	svc.TopUp(ctx, w1.WalletID, decimal.NewFromFloat(1000.00), "topup-lc-xfer")
	svc.TopUp(ctx, w2.WalletID, decimal.NewFromFloat(500.00), "topup-lc-xfer-2")
	svc.Transfer(ctx, w1.WalletID, w2.WalletID, decimal.NewFromFloat(300.00), "xfer-lc")
	svc.Pay(ctx, w1.WalletID, decimal.NewFromFloat(100.00), "pay-lc")

	// Verify consistency for both wallets
	for _, walletID := range []string{w1.WalletID, w2.WalletID} {
		w, _ := svc.GetWallet(ctx, walletID)
		entries, _ := svc.GetLedgerEntries(ctx, walletID)

		ledgerSum := decimal.Zero
		for _, e := range entries {
			ledgerSum = ledgerSum.Add(e.Amount)
		}

		if !w.Balance.Equal(ledgerSum) {
			t.Errorf("wallet %s: balance %s != ledger sum %s", walletID, w.Balance, ledgerSum)
		}
	}
}
