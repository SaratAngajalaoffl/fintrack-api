package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Common error messages aligned with former Next.js handlers.

var (
	ErrCannotAllocateUnlocked = errors.New("Cannot allocate funds to an unlocked fund bucket")
	ErrInsufficientAllocation = errors.New("Insufficient available balance in bank account for this allocation")
	ErrBankAccountNotFound    = errors.New("Bank account not found")
)

// FundBucketRow matches web FundBucketRow JSON.
type FundBucketRow struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	TargetAmount    float64 `json:"targetAmount"`
	BankAccountID   string  `json:"bankAccountId"`
	BankAccountName string  `json:"bankAccountName"`
	CurrentValue    float64 `json:"currentValue"`
	IsLocked        bool    `json:"isLocked"`
	Priority        string  `json:"priority"`
}

type CreateFundBucketInput struct {
	UserID        string
	Name          string
	TargetAmount  float64
	BankAccountID string
	Priority      string
}

// ListFundBuckets returns buckets for a user (newest first).
func ListFundBuckets(ctx context.Context, pool *pgxpool.Pool, userID string) ([]FundBucketRow, error) {
	rows, err := pool.Query(ctx, `
      SELECT
        fb.id,
        fb.name,
        fb.target_amount::float8,
        fb.bank_account_id,
        ba.name AS bank_account_name,
        fb.current_value::float8,
        fb.is_locked,
        fb.priority::text
      FROM fund_buckets fb
      INNER JOIN bank_accounts ba
        ON ba.id = fb.bank_account_id
       AND ba.user_id = fb.user_id
      WHERE fb.user_id = $1
      ORDER BY fb.created_at DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FundBucketRow
	for rows.Next() {
		var r FundBucketRow
		if err := rows.Scan(&r.ID, &r.Name, &r.TargetAmount, &r.BankAccountID, &r.BankAccountName, &r.CurrentValue, &r.IsLocked, &r.Priority); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreateFundBucket inserts a fund bucket if the bank account belongs to the user.
func CreateFundBucket(ctx context.Context, pool *pgxpool.Pool, in CreateFundBucketInput) (*FundBucketRow, error) {
	var one int
	err := pool.QueryRow(ctx,
		`SELECT 1 FROM bank_accounts WHERE user_id = $1 AND id = $2 LIMIT 1`,
		in.UserID, in.BankAccountID,
	).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBankAccountNotFound
		}
		return nil, err
	}

	row := pool.QueryRow(ctx, `
      INSERT INTO fund_buckets (
        user_id, name, target_amount, bank_account_id, current_value, is_locked, priority
      )
      VALUES ($1,$2,$3,$4,0,TRUE,$5::fund_bucket_priority)
      RETURNING
        id,
        name,
        target_amount::float8,
        bank_account_id,
        (
          SELECT name FROM bank_accounts WHERE id = fund_buckets.bank_account_id
        ),
        current_value::float8,
        is_locked,
        priority::text
    `, in.UserID, in.Name, in.TargetAmount, in.BankAccountID, in.Priority)

	var r FundBucketRow
	if err := row.Scan(&r.ID, &r.Name, &r.TargetAmount, &r.BankAccountID, &r.BankAccountName, &r.CurrentValue, &r.IsLocked, &r.Priority); err != nil {
		return nil, err
	}
	return &r, nil
}

// AllocateFundsToBucket adds locked funds if rules allow (same as TS).
func AllocateFundsToBucket(ctx context.Context, pool *pgxpool.Pool, userID, bucketID string, amount float64) (*FundBucketRow, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var bID, bankID string
	var currentLocked float64
	var isLocked bool
	err = tx.QueryRow(ctx, `
      SELECT id, bank_account_id, current_value::float8, is_locked
      FROM fund_buckets
      WHERE id = $1 AND user_id = $2
      LIMIT 1
      FOR UPDATE
    `, bucketID, userID).Scan(&bID, &bankID, &currentLocked, &isLocked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if !isLocked {
		return nil, ErrCannotAllocateUnlocked
	}

	var balance float64
	err = tx.QueryRow(ctx, `
      SELECT balance::float8
      FROM bank_accounts
      WHERE id = $1 AND user_id = $2
      LIMIT 1
      FOR UPDATE
    `, bankID, userID).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBankAccountNotFound
		}
		return nil, err
	}

	var totalLocked float64
	err = tx.QueryRow(ctx, `
      SELECT COALESCE(SUM(current_value), 0)::float8
      FROM fund_buckets
      WHERE user_id = $1
        AND bank_account_id = $2
        AND is_locked = TRUE
    `, userID, bankID).Scan(&totalLocked)
	if err != nil {
		return nil, err
	}

	available := balance - totalLocked
	if amount > available {
		return nil, ErrInsufficientAllocation
	}

	if _, err := tx.Exec(ctx, `
      UPDATE fund_buckets
      SET current_value = current_value + $3, updated_at = NOW()
      WHERE id = $1 AND user_id = $2
    `, bucketID, userID, amount); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, `
      SELECT
        fb.id,
        fb.name,
        fb.target_amount::float8,
        fb.bank_account_id,
        ba.name AS bank_account_name,
        fb.current_value::float8,
        fb.is_locked,
        fb.priority::text
      FROM fund_buckets fb
      INNER JOIN bank_accounts ba
        ON ba.id = fb.bank_account_id AND ba.user_id = fb.user_id
      WHERE fb.id = $1 AND fb.user_id = $2
      LIMIT 1
    `, bucketID, userID)

	var r FundBucketRow
	if err := row.Scan(&r.ID, &r.Name, &r.TargetAmount, &r.BankAccountID, &r.BankAccountName, &r.CurrentValue, &r.IsLocked, &r.Priority); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &r, nil
}

// UnlockFundBucket unlocks a bucket when current_value >= target_amount (same WHERE as TS).
func UnlockFundBucket(ctx context.Context, pool *pgxpool.Pool, userID, bucketID string) (*FundBucketRow, error) {
	row := pool.QueryRow(ctx, `
      UPDATE fund_buckets fb
      SET is_locked = FALSE, updated_at = NOW()
      FROM bank_accounts ba
      WHERE fb.id = $1
        AND fb.user_id = $2
        AND ba.id = fb.bank_account_id
        AND ba.user_id = fb.user_id
        AND fb.is_locked = TRUE
        AND fb.current_value >= fb.target_amount
      RETURNING
        fb.id,
        fb.name,
        fb.target_amount::float8,
        fb.bank_account_id,
        ba.name AS bank_account_name,
        fb.current_value::float8,
        fb.is_locked,
        fb.priority::text
    `, bucketID, userID)

	var r FundBucketRow
	err := row.Scan(&r.ID, &r.Name, &r.TargetAmount, &r.BankAccountID, &r.BankAccountName, &r.CurrentValue, &r.IsLocked, &r.Priority)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// SetFundBucketPriority updates priority for a bucket owned by the user.
func SetFundBucketPriority(ctx context.Context, pool *pgxpool.Pool, userID, bucketID, priority string) (*FundBucketRow, error) {
	row := pool.QueryRow(ctx, `
      UPDATE fund_buckets fb
      SET priority = $3::fund_bucket_priority, updated_at = NOW()
      FROM bank_accounts ba
      WHERE fb.id = $1
        AND fb.user_id = $2
        AND ba.id = fb.bank_account_id
        AND ba.user_id = fb.user_id
      RETURNING
        fb.id,
        fb.name,
        fb.target_amount::float8,
        fb.bank_account_id,
        ba.name AS bank_account_name,
        fb.current_value::float8,
        fb.is_locked,
        fb.priority::text
    `, bucketID, userID, priority)

	var r FundBucketRow
	err := row.Scan(&r.ID, &r.Name, &r.TargetAmount, &r.BankAccountID, &r.BankAccountName, &r.CurrentValue, &r.IsLocked, &r.Priority)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}
