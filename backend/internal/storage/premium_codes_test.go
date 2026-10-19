package storage

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNormalizePremiumCode(t *testing.T) {
	for _, raw := range []string{"taro73", " TARO73\n", "TARO73"} {
		got, ok := NormalizePremiumCode(raw)
		if !ok || got != "TARO73" {
			t.Fatalf("normalize %q = %q, %v", raw, got, ok)
		}
	}
	for _, raw := range []string{"", "TARO7", "TARO733", "ТАРО73", "TAR 73", "TARO7!"} {
		if _, ok := NormalizePremiumCode(raw); ok {
			t.Fatalf("accepted %q", raw)
		}
	}
}

func testPremiumCode(t *testing.T, pool *pgxpool.Pool, code string) {
	t.Helper()
	hash := PremiumCodeHash(code)
	_, err := pool.Exec(context.Background(), `INSERT INTO premium_codes(code_hash) VALUES ($1)`, hash)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM premium_codes WHERE code_hash=$1`, hash) })
}

func TestPremiumCodeConcurrentClaimAndRetry(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()
	a := makeUser(t, pool, "code-a@example.test")
	b := makeUser(t, pool, "code-b@example.test")
	testPremiumCode(t, pool, "TEST73")
	var wg sync.WaitGroup
	errs := make([]error, 2)
	ids := []string{a, b}
	start := make(chan struct{})
	for i := range ids {
		wg.Add(1)
		go func(i int) { defer wg.Done(); <-start; _, errs[i] = repo.RedeemPremiumCode(ctx, ids[i], "TEST73") }(i)
	}
	close(start)
	wg.Wait()
	winner := -1
	for i, err := range errs {
		if err == nil {
			if winner != -1 {
				t.Fatal("two winners")
			}
			winner = i
		} else if !errors.Is(err, ErrPremiumCodeUnavailable) {
			t.Fatal(err)
		}
	}
	if winner == -1 {
		t.Fatal("no winner")
	}
	u, err := repo.RedeemPremiumCode(ctx, ids[winner], "test73")
	if err != nil || !u.HasPremium || u.PremiumExpiresAt != nil || u.PremiumSource != "promo" || u.PremiumProductID != "premium_lifetime" {
		t.Fatalf("retry = %+v, %v", u, err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE provider='promo' AND provider_invoice_id=$1`, PremiumCodeHash("TEST73")).Scan(&count); err != nil || count != 1 {
		t.Fatalf("transactions=%d, %v", count, err)
	}
	// A later subscription refresh must never shorten this permanent grant.
	expires := time.Now().Add(time.Hour)
	if err = repo.GrantPremium(ctx, u.ID, PremiumGrant{ProductID: "premium_monthly", Source: "apple", ExpiresAt: &expires}); err != nil {
		t.Fatal(err)
	}
	u, err = repo.GetByID(ctx, u.ID)
	if err != nil || u.PremiumExpiresAt != nil || u.PremiumSource != "promo" {
		t.Fatalf("lifetime overwritten: %+v, %v", u, err)
	}
	// A reader holding the old expired subscription must not revoke the grant.
	past := time.Now().Add(-time.Hour)
	stale := *u
	stale.PremiumExpiresAt = &past
	stale.PremiumSource = "apple"
	fresh, err := repo.applyExpiry(ctx, &stale)
	if err != nil || !fresh.HasPremium || fresh.PremiumExpiresAt != nil {
		t.Fatalf("stale expiry revoked lifetime: %+v %v", fresh, err)
	}

	if err = repo.DeleteByID(ctx, ids[winner]); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.RedeemPremiumCode(ctx, ids[1-winner], "TEST73"); !errors.Is(err, ErrPremiumCodeUnavailable) {
		t.Fatalf("code reused after deletion: %v", err)
	}
}

func TestPremiumCodeRollbackAndAlreadyLifetime(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()
	id := makeUser(t, pool, "code-rollback@example.test")
	testPremiumCode(t, pool, "TEST12")
	// Force transaction-record insertion to fail after the code and user update.
	_, err := pool.Exec(ctx, `INSERT INTO transactions (user_id, product_id, provider, provider_invoice_id) VALUES ($1,'premium_lifetime','promo',$2)`, id, PremiumCodeHash("TEST12"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.RedeemPremiumCode(ctx, id, "TEST12"); err == nil {
		t.Fatal("expected transaction failure")
	}
	u, err := repo.GetByID(ctx, id)
	if err != nil || u.HasPremium {
		t.Fatalf("partial grant: %+v %v", u, err)
	}
	var unused bool
	if err = pool.QueryRow(ctx, `SELECT redeemed_at IS NULL FROM premium_codes WHERE code_hash=$1`, PremiumCodeHash("TEST12")).Scan(&unused); err != nil || !unused {
		t.Fatalf("code burned on failure: %v", err)
	}
	_, err = pool.Exec(ctx, `DELETE FROM transactions WHERE user_id=$1`, id)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.GrantPremium(ctx, id, PremiumGrant{ProductID: "premium_lifetime", Source: "admin"}); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.RedeemPremiumCode(ctx, id, "TEST12"); !errors.Is(err, ErrAlreadyLifetime) {
		t.Fatalf("already lifetime: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT redeemed_at IS NULL FROM premium_codes WHERE code_hash=$1`, PremiumCodeHash("TEST12")).Scan(&unused); err != nil || !unused {
		t.Fatal("already-lifetime account burned code")
	}
}
