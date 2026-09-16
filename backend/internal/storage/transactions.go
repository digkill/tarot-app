package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotRefundable = errors.New("transaction is not refundable")

type Transaction struct {
	ID                 string
	UserID             string
	UserEmail          string
	ProductID          string
	Kind               string
	Status             string
	Provider           string
	ProviderInvoiceID  *string
	ProviderPurchaseID *string
	AmountKop          int
	Currency           string
	Sandbox            bool
	Notes              string
	PaidAt             time.Time
	RefundedAt         *time.Time
	RefundedBy         *string
	RefundReason       string
	CreatedAt          time.Time
}

type CreateTransactionParams struct {
	UserID             string
	ProductID          string
	Kind               string
	Status             string
	Provider           string
	ProviderInvoiceID  *string
	ProviderPurchaseID *string
	AmountKop          int
	Currency           string
	Sandbox            bool
	Notes              string
}

type TransactionRepo struct {
	pool *pgxpool.Pool
}

func NewTransactionRepo(pool *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{pool: pool}
}

const txSelect = `
	t.id, t.user_id, u.email, t.product_id, t.kind, t.status, t.provider,
	t.provider_invoice_id, t.provider_purchase_id, t.amount_kop, t.currency,
	t.sandbox, t.notes, t.paid_at, t.refunded_at, t.refunded_by, t.refund_reason, t.created_at`

func NullIfBlank(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func scanTx(row rowScanner) (*Transaction, error) {
	var tx Transaction
	err := row.Scan(
		&tx.ID,
		&tx.UserID,
		&tx.UserEmail,
		&tx.ProductID,
		&tx.Kind,
		&tx.Status,
		&tx.Provider,
		&tx.ProviderInvoiceID,
		&tx.ProviderPurchaseID,
		&tx.AmountKop,
		&tx.Currency,
		&tx.Sandbox,
		&tx.Notes,
		&tx.PaidAt,
		&tx.RefundedAt,
		&tx.RefundedBy,
		&tx.RefundReason,
		&tx.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *TransactionRepo) Create(ctx context.Context, p CreateTransactionParams) (*Transaction, error) {
	if p.Status == "" {
		p.Status = "paid"
	}
	if p.Currency == "" {
		p.Currency = "RUB"
	}
	const q = `
		INSERT INTO transactions (
			user_id, product_id, kind, status, provider,
			provider_invoice_id, provider_purchase_id, amount_kop, currency,
			sandbox, notes
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id`

	var id string
	err := r.pool.QueryRow(ctx, q,
		p.UserID, p.ProductID, p.Kind, p.Status, p.Provider,
		p.ProviderInvoiceID, p.ProviderPurchaseID, p.AmountKop, p.Currency,
		p.Sandbox, p.Notes,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert transaction: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *TransactionRepo) SetProviderRefs(ctx context.Context, id, invoiceID, purchaseID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE transactions
		SET provider_invoice_id = NULLIF($2, ''),
		    provider_purchase_id = NULLIF($3, '')
		WHERE id = $1`, id, invoiceID, purchaseID)
	if err != nil {
		return fmt.Errorf("set transaction provider refs: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TransactionRepo) SetStatus(ctx context.Context, id, status string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE transactions SET status = $2, paid_at = CASE WHEN $2 = 'paid' THEN NOW() ELSE paid_at END WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("set transaction status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkPaidFromPending moves a pending transaction to paid and reports whether
// this call made the transition, so concurrent webhook and polling
// reconciliation grant an entitlement exactly once.
func (r *TransactionRepo) MarkPaidFromPending(ctx context.Context, id, purchaseID string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE transactions
		SET status = 'paid',
		    paid_at = NOW(),
		    provider_purchase_id = COALESCE(NULLIF($2, ''), provider_purchase_id)
		WHERE id = $1 AND status = 'pending'`, id, purchaseID)
	if err != nil {
		return false, fmt.Errorf("mark transaction paid: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *TransactionRepo) GetByID(ctx context.Context, id string) (*Transaction, error) {
	q := `SELECT ` + txSelect + ` FROM transactions t JOIN users u ON u.id = t.user_id WHERE t.id = $1`
	tx, err := scanTx(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get transaction: %w", err)
	}
	return tx, nil
}

func (r *TransactionRepo) GetByProviderInvoice(ctx context.Context, provider, invoiceID string) (*Transaction, error) {
	if strings.TrimSpace(invoiceID) == "" {
		return nil, ErrNotFound
	}
	q := `SELECT ` + txSelect + `
		FROM transactions t JOIN users u ON u.id = t.user_id
		WHERE t.provider = $1 AND t.provider_invoice_id = $2`
	tx, err := scanTx(r.pool.QueryRow(ctx, q, provider, invoiceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get transaction by invoice: %w", err)
	}
	return tx, nil
}

type TxListFilter struct {
	Query  string
	Status string
	UserID string
	Limit  int
	Offset int
}

func (r *TransactionRepo) List(ctx context.Context, f TxListFilter) ([]Transaction, int, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	// provider_purchase_id is searched too: for Apple it holds the
	// originalTransactionId, which is the only handle that groups every renewal
	// of one subscription.
	const countQ = `
		SELECT COUNT(*)
		FROM transactions t
		JOIN users u ON u.id = t.user_id
		WHERE ($1 = '' OR u.email ILIKE '%' || $1 || '%'
		        OR COALESCE(t.provider_invoice_id,'') ILIKE '%' || $1 || '%'
		        OR COALESCE(t.provider_purchase_id,'') ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR t.status = $2)
		  AND ($3 = '' OR t.user_id::text = $3)`
	var total int
	if err := r.pool.QueryRow(ctx, countQ, f.Query, f.Status, f.UserID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count transactions: %w", err)
	}

	q := `SELECT ` + txSelect + `
		FROM transactions t
		JOIN users u ON u.id = t.user_id
		WHERE ($1 = '' OR u.email ILIKE '%' || $1 || '%'
		        OR COALESCE(t.provider_invoice_id,'') ILIKE '%' || $1 || '%'
		        OR COALESCE(t.provider_purchase_id,'') ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR t.status = $2)
		  AND ($3 = '' OR t.user_id::text = $3)
		ORDER BY t.created_at DESC
		LIMIT $4 OFFSET $5`
	rows, err := r.pool.Query(ctx, q, f.Query, f.Status, f.UserID, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	out := make([]Transaction, 0)
	for rows.Next() {
		tx, err := scanTx(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *tx)
	}
	return out, total, rows.Err()
}

func (r *TransactionRepo) Refund(ctx context.Context, id, adminEmail, reason string) (*Transaction, bool, error) {
	dbtx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin refund: %w", err)
	}
	defer func() { _ = dbtx.Rollback(ctx) }()

	const lockQ = `SELECT user_id, status, product_id FROM transactions WHERE id = $1 FOR UPDATE`
	var userID, status, productID string
	err = dbtx.QueryRow(ctx, lockQ, id).Scan(&userID, &status, &productID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrNotFound
		}
		return nil, false, fmt.Errorf("lock transaction: %w", err)
	}
	if status != "paid" {
		return nil, false, ErrNotRefundable
	}

	tag, err := dbtx.Exec(ctx, `
		UPDATE transactions
		SET status = 'refunded',
		    refunded_at = NOW(),
		    refunded_by = $2,
		    refund_reason = $3
		WHERE id = $1 AND status = 'paid'`, id, adminEmail, reason)
	if err != nil {
		return nil, false, fmt.Errorf("update refund: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, false, ErrNotRefundable
	}

	var revokedPremium bool
	if strings.HasPrefix(productID, "premium_") {
		var keepPremium bool
		if err = dbtx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM transactions
				WHERE user_id = $1 AND status = 'paid' AND id <> $2::uuid
				  AND product_id LIKE 'premium_%'
			)`, userID, id).Scan(&keepPremium); err != nil {
			return nil, false, fmt.Errorf("check remaining paid: %w", err)
		}
		if !keepPremium {
			if _, err = dbtx.Exec(ctx, `
				UPDATE users
				SET has_premium = FALSE,
				    premium_product_id = NULL,
				    premium_source = NULL,
				    premium_expires_at = NULL
				WHERE id = $1`, userID); err != nil {
				return nil, false, fmt.Errorf("revoke premium: %w", err)
			}
			revokedPremium = true
		}
	}

	if err = dbtx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit refund: %w", err)
	}

	tx, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, revokedPremium, err
	}
	return tx, revokedPremium, nil
}
