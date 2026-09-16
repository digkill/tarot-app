package storage

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests exercise SQL that unit tests cannot: ON CONFLICT semantics, the
// widened admin search, and the new decks column. They are skipped unless a
// scratch database is provided, because they write to it.
//
//	docker run -d --name tarot-test -e POSTGRES_PASSWORD=test -e POSTGRES_DB=tarot \
//	  -p 55432:5432 postgres:16-alpine
//	goose -dir migrations postgres "$TEST_DATABASE_URL" up
//	TEST_DATABASE_URL="postgres://postgres:test@localhost:55432/tarot?sslmode=disable" \
//	  go test ./internal/storage/...
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping database-backed tests")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return pool
}

// makeUser inserts a throwaway user and removes it afterwards, which cascades
// to everything keyed on it.
func makeUser(t *testing.T, pool *pgxpool.Pool, email string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, consent_version)
		VALUES ($1, 'x', '1.0') RETURNING id`, email).Scan(&id)
	if err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

func TestAppleSubRepoBindIsFirstBinderWins(t *testing.T) {
	pool := testPool(t)
	repo := NewAppleSubRepo(pool)
	ctx := context.Background()

	owner := makeUser(t, pool, "apple-owner@example.test")
	attacker := makeUser(t, pool, "apple-attacker@example.test")
	origID := "2000000900000001"
	expires := time.Now().Add(366 * 24 * time.Hour).UTC().Truncate(time.Second)

	first, err := repo.Bind(ctx, AppleSubscription{
		OriginalTransactionID: origID,
		UserID:                owner,
		ProductID:             "premium_yearly",
		AppleProductID:        "org.mediarise.tarot.premium.yearly",
		Environment:           "Sandbox",
		ExpiresAt:             &expires,
		LastTransactionID:     "2000000900000002",
	})
	if err != nil {
		t.Fatalf("first bind: %v", err)
	}
	if first.UserID != owner {
		t.Fatalf("owner = %s, want %s", first.UserID, owner)
	}

	// A second account claiming the same Apple subscription must not take it;
	// Bind returns the existing owner so the caller can refuse the request.
	second, err := repo.Bind(ctx, AppleSubscription{
		OriginalTransactionID: origID,
		UserID:                attacker,
		ProductID:             "premium_yearly",
		AppleProductID:        "org.mediarise.tarot.premium.yearly",
	})
	if err != nil {
		t.Fatalf("second bind: %v", err)
	}
	if second.UserID != owner {
		t.Fatalf("ownership was stolen: got %s, want %s", second.UserID, owner)
	}
	// Blank fields must not wipe what we already knew.
	if second.LastTransactionID != "2000000900000002" {
		t.Fatalf("lastTransactionId was cleared: %q", second.LastTransactionID)
	}
	if second.ExpiresAt == nil || !second.ExpiresAt.Equal(expires) {
		t.Fatalf("expiresAt was cleared: %v", second.ExpiresAt)
	}

	// A renewal pushes the expiry forward.
	later := expires.Add(366 * 24 * time.Hour)
	updated, err := repo.Bind(ctx, AppleSubscription{
		OriginalTransactionID: origID,
		UserID:                owner,
		ExpiresAt:             &later,
		LastTransactionID:     "2000000900000003",
	})
	if err != nil {
		t.Fatalf("renewal bind: %v", err)
	}
	if updated.ExpiresAt == nil || !updated.ExpiresAt.Equal(later) {
		t.Fatalf("expiresAt = %v, want %v", updated.ExpiresAt, later)
	}

	got, err := repo.GetByOriginalTransactionID(ctx, origID)
	if err != nil || got.UserID != owner {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := repo.GetByOriginalTransactionID(ctx, "2000000900000999"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing subscription err = %v, want ErrNotFound", err)
	}
	if _, err := repo.GetByOriginalTransactionID(ctx, "  "); !errors.Is(err, ErrNotFound) {
		t.Fatalf("blank id err = %v, want ErrNotFound", err)
	}

	subs, err := repo.ListByUser(ctx, owner)
	if err != nil || len(subs) != 1 {
		t.Fatalf("ListByUser = %d subs, err %v", len(subs), err)
	}
}

// Apple retries a notification up to 5 times, and a replay must not grant or
// revoke twice.
func TestAppleSubRepoMarkNotificationDeduplicates(t *testing.T) {
	pool := testPool(t)
	repo := NewAppleSubRepo(pool)
	ctx := context.Background()

	uuid := "9ad8c4b1-0000-4000-8000-900000000001"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM apple_notifications WHERE notification_uuid = $1`, uuid)
	})

	fresh, err := repo.MarkNotification(ctx, uuid, "DID_RENEW", "", "2000000900000001")
	if err != nil || !fresh {
		t.Fatalf("first delivery: fresh=%v err=%v", fresh, err)
	}
	fresh, err = repo.MarkNotification(ctx, uuid, "DID_RENEW", "", "2000000900000001")
	if err != nil || fresh {
		t.Fatalf("retry must not be fresh: fresh=%v err=%v", fresh, err)
	}

	// No uuid to key on: treat it as fresh rather than dropping the event.
	fresh, err = repo.MarkNotification(ctx, "  ", "DID_RENEW", "", "")
	if err != nil || !fresh {
		t.Fatalf("blank uuid: fresh=%v err=%v", fresh, err)
	}
}

func TestDeckAppleProductIDRoundTrips(t *testing.T) {
	pool := testPool(t)
	repo := NewDeckRepo(pool)
	ctx := context.Background()

	sku := "org.mediarise.tarot.deck.integrationtest"
	created, err := repo.Create(ctx, UpsertDeckParams{
		Slug:            "apple-integration-test",
		TitleI18n:       I18nMap{"ru": "Тест"},
		DescriptionI18n: I18nMap{"ru": "Тест"},
		PriceKop:        49900,
		AppleProductID:  NullIfBlank(sku),
		IsPublished:     true,
	})
	if err != nil {
		t.Fatalf("create deck: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM decks WHERE id = $1`, created.ID)
	})
	if created.AppleProductID == nil || *created.AppleProductID != sku {
		t.Fatalf("apple sku not stored: %v", created.AppleProductID)
	}

	found, err := repo.GetByAppleProductID(ctx, sku)
	if err != nil || found.ID != created.ID {
		t.Fatalf("lookup by apple sku: %+v %v", found, err)
	}

	// Unlike GetByProductID there is no slug-derived fallback, so an unknown
	// SKU must miss rather than match some other deck.
	if _, err := repo.GetByAppleProductID(ctx, "org.mediarise.tarot.deck.nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown sku err = %v, want ErrNotFound", err)
	}
	if _, err := repo.GetByAppleProductID(ctx, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("blank sku err = %v, want ErrNotFound", err)
	}

	// Update must preserve the column.
	updated, err := repo.Update(ctx, created.ID, UpsertDeckParams{
		Slug:            created.Slug,
		TitleI18n:       I18nMap{"ru": "Тест 2"},
		DescriptionI18n: I18nMap{"ru": "Тест 2"},
		PriceKop:        59900,
		AppleProductID:  NullIfBlank(sku),
		IsPublished:     true,
	})
	if err != nil {
		t.Fatalf("update deck: %v", err)
	}
	if updated.AppleProductID == nil || *updated.AppleProductID != sku {
		t.Fatalf("apple sku lost on update: %v", updated.AppleProductID)
	}
}

// An Apple originalTransactionId lives in provider_purchase_id and groups every
// renewal, so the admin search has to cover that column.
func TestTransactionListSearchesPurchaseID(t *testing.T) {
	pool := testPool(t)
	repo := NewTransactionRepo(pool)
	ctx := context.Background()

	userID := makeUser(t, pool, "apple-search@example.test")
	originalID := "2000000900000500"

	for _, appleTxID := range []string{"2000000900000501", "2000000900000502"} {
		if _, err := repo.Create(ctx, CreateTransactionParams{
			UserID:             userID,
			ProductID:          "premium_yearly",
			Kind:               "subscription",
			Status:             "paid",
			Provider:           "apple",
			ProviderInvoiceID:  NullIfBlank(appleTxID),
			ProviderPurchaseID: NullIfBlank(originalID),
			AmountKop:          5999,
			Currency:           "USD",
			Sandbox:            true,
		}); err != nil {
			t.Fatalf("create transaction %s: %v", appleTxID, err)
		}
	}

	found, total, err := repo.List(ctx, TxListFilter{Query: originalID, Limit: 50})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 || len(found) != 2 {
		t.Fatalf("searching the originalTransactionId found %d rows (total %d), want 2", len(found), total)
	}

	// Searching a single renewal's own id still works.
	found, _, err = repo.List(ctx, TxListFilter{Query: "2000000900000501", Limit: 50})
	if err != nil || len(found) != 1 {
		t.Fatalf("invoice search found %d rows, err %v", len(found), err)
	}
}

// The provider CHECK constraint must accept 'apple', or every write fails.
func TestTransactionsAcceptAppleProvider(t *testing.T) {
	pool := testPool(t)
	repo := NewTransactionRepo(pool)
	ctx := context.Background()

	userID := makeUser(t, pool, "apple-provider@example.test")
	tx, err := repo.Create(ctx, CreateTransactionParams{
		UserID:            userID,
		ProductID:         "premium_monthly",
		Kind:              "subscription",
		Status:            "paid",
		Provider:          "apple",
		ProviderInvoiceID: NullIfBlank("2000000900000600"),
		AmountKop:         799,
		Currency:          "USD",
	})
	if err != nil {
		t.Fatalf("apple provider rejected: %v", err)
	}
	if tx.Provider != "apple" || tx.Currency != "USD" {
		t.Fatalf("stored: %+v", tx)
	}

	// The UNIQUE (provider, provider_invoice_id) index is what makes Apple's
	// transactionId an exactly-once key across /verify and notifications.
	if _, err := repo.Create(ctx, CreateTransactionParams{
		UserID:            userID,
		ProductID:         "premium_monthly",
		Kind:              "subscription",
		Status:            "paid",
		Provider:          "apple",
		ProviderInvoiceID: NullIfBlank("2000000900000600"),
		AmountKop:         799,
		Currency:          "USD",
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate apple transactionId err = %v, want ErrConflict", err)
	}
}
