package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InvalidPreferredCategoriesError is returned when a preferred category name does not exist for the user.
type InvalidPreferredCategoriesError struct {
	Missing []string
}

func (e *InvalidPreferredCategoriesError) Error() string {
	return fmt.Sprintf(
		"Invalid preferred categories: %s. Create them in Expense Categories first.",
		strings.Join(e.Missing, ", "),
	)
}

// BankAccountRow matches web BankAccountRow JSON (camelCase from handler).
type BankAccountRow struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	AccountType         string   `json:"accountType"`
	Balance             float64  `json:"balance"`
	CreditsThisMonth    float64  `json:"creditsThisMonth"`
	DebitsThisMonth     float64  `json:"debitsThisMonth"`
	BucketNames         []string `json:"bucketNames"`
	PreferredCategories []string `json:"preferredCategories"`
}

type CreateBankAccountInput struct {
	UserID              string
	Name                string
	Description         string
	AccountType         string
	InitialBalance      float64
	LastDebitAt         *string
	LastCreditAt        *string
	PreferredCategories []string
}

type UpdateBankAccountInput struct {
	UserID       string
	AccountID    string
	Name         *string
	Description  *string
	AccountType  *string
	Balance      *float64
	LastDebitAt  *string
	LastCreditAt *string
	PreferredCat *[]string
}

const bankAccountSelect = `
      SELECT
        ba.id,
        ba.name,
        ba.description,
        ba.account_type::text,
        ba.balance::float8,
        COALESCE(
          ARRAY_AGG(DISTINCT fb.name ORDER BY fb.name) FILTER (
            WHERE fb.name IS NOT NULL
          ),
          '{}'
        ) AS bucket_names,
        COALESCE(
          ARRAY_AGG(DISTINCT ec.name ORDER BY ec.name) FILTER (
            WHERE ec.name IS NOT NULL
          ),
          '{}'
        ) AS preferred_categories
      FROM bank_accounts ba
      LEFT JOIN bank_account_preferred_categories bapc
        ON bapc.bank_account_id = ba.id
       AND bapc.user_id = ba.user_id
      LEFT JOIN fund_buckets fb
        ON fb.bank_account_id = ba.id
       AND fb.user_id = ba.user_id
      LEFT JOIN expense_categories ec
        ON ec.id = bapc.expense_category_id
       AND ec.user_id = ba.user_id
`

func mapBankAccountRow(
	id, name, description, accountType string,
	balance float64,
	bucketNames []string,
	preferredCategories []string,
) BankAccountRow {
	if bucketNames == nil {
		bucketNames = []string{}
	}
	if preferredCategories == nil {
		preferredCategories = []string{}
	}
	return BankAccountRow{
		ID:                  id,
		Name:                name,
		Description:         description,
		AccountType:         accountType,
		Balance:             balance,
		CreditsThisMonth:    0,
		DebitsThisMonth:     0,
		BucketNames:         bucketNames,
		PreferredCategories: preferredCategories,
	}
}

// ListBankAccounts returns accounts for a user (newest first).
func ListBankAccounts(ctx context.Context, pool *pgxpool.Pool, userID string) ([]BankAccountRow, error) {
	rows, err := pool.Query(ctx, bankAccountSelect+`
      WHERE ba.user_id = $1
      GROUP BY ba.id
      ORDER BY ba.created_at DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BankAccountRow
	for rows.Next() {
		var (
			id, name, description, accountType string
			balance                            float64
			bucketNames                        []string
			preferredCategories                  []string
		)
		if err := rows.Scan(
			&id, &name, &description, &accountType, &balance,
			&bucketNames, &preferredCategories,
		); err != nil {
			return nil, err
		}
		out = append(out, mapBankAccountRow(id, name, description, accountType, balance, bucketNames, preferredCategories))
	}
	return out, rows.Err()
}

// GetBankAccountByID returns one account owned by the user.
func GetBankAccountByID(ctx context.Context, pool *pgxpool.Pool, userID, accountID string) (*BankAccountRow, error) {
	row := pool.QueryRow(ctx, bankAccountSelect+`
      WHERE ba.user_id = $1 AND ba.id = $2
      GROUP BY ba.id
      LIMIT 1
    `, userID, accountID)
	var (
		id, name, description, accountType string
		balance                            float64
		bucketNames                        []string
		preferredCategories                  []string
	)
	err := row.Scan(
		&id, &name, &description, &accountType, &balance,
		&bucketNames, &preferredCategories,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r := mapBankAccountRow(id, name, description, accountType, balance, bucketNames, preferredCategories)
	return &r, nil
}

func normalizePreferredCategoryNames(names []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func resolvePreferredCategoryIDs(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	preferredCategories []string,
) ([]string, error) {
	if len(preferredCategories) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx,
		`SELECT id, name FROM expense_categories WHERE user_id = $1 AND name = ANY($2::text[])`,
		userID, preferredCategories,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	foundByName := make(map[string]string)
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		foundByName[name] = id
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var missing []string
	for _, name := range preferredCategories {
		if _, ok := foundByName[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, &InvalidPreferredCategoriesError{Missing: missing}
	}
	ids := make([]string, 0, len(preferredCategories))
	for _, name := range preferredCategories {
		ids = append(ids, foundByName[name])
	}
	return ids, nil
}

func syncPreferredCategoryMappings(
	ctx context.Context,
	tx pgx.Tx,
	userID, accountID string,
	preferredCategories []string,
) error {
	categoryIDs, err := resolvePreferredCategoryIDs(ctx, tx, userID, preferredCategories)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM bank_account_preferred_categories WHERE user_id = $1 AND bank_account_id = $2`,
		userID, accountID,
	); err != nil {
		return err
	}
	if len(categoryIDs) == 0 {
		return nil
	}
	_, err = tx.Exec(ctx, `
      INSERT INTO bank_account_preferred_categories (
        bank_account_id,
        expense_category_id,
        user_id
      )
      SELECT $1, ids.category_id, $2
      FROM unnest($3::uuid[]) AS ids(category_id)
      ON CONFLICT (bank_account_id, expense_category_id) DO NOTHING
    `, accountID, userID, categoryIDs)
	return err
}

// CreateBankAccount inserts an account and preferred categories.
func CreateBankAccount(ctx context.Context, pool *pgxpool.Pool, in CreateBankAccountInput) (*BankAccountRow, error) {
	pc := normalizePreferredCategoryNames(in.PreferredCategories)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var newID string
	err = tx.QueryRow(ctx, `
      INSERT INTO bank_accounts (
        user_id, name, description, account_type, initial_balance, balance, last_debit_at, last_credit_at
      )
      VALUES ($1,$2,$3,$4::bank_account_type,$5,$5,$6::timestamptz,$7::timestamptz)
      RETURNING id
	`, in.UserID, in.Name, in.Description, in.AccountType, in.InitialBalance, nullIfEmptyTime(in.LastDebitAt), nullIfEmptyTime(in.LastCreditAt)).Scan(&newID)
	if err != nil {
		return nil, err
	}

	if err := syncPreferredCategoryMappings(ctx, tx, in.UserID, newID, pc); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return GetBankAccountByID(ctx, pool, in.UserID, newID)
}

// nullIfEmptyTime converts optional RFC3339-style strings; nil or empty -> nil for SQL NULL.
func nullIfEmptyTime(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

// UpdateBankAccount patches fields; preferred categories use pointer-to-slice (nil = omit).
func UpdateBankAccount(ctx context.Context, pool *pgxpool.Pool, in UpdateBankAccountInput) (*BankAccountRow, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	scalar := in.Name != nil || in.Description != nil || in.AccountType != nil ||
		in.Balance != nil || in.LastDebitAt != nil || in.LastCreditAt != nil
	if scalar {
		cmd, err := tx.Exec(ctx, `
        UPDATE bank_accounts
        SET
          name = COALESCE($3, name),
          description = COALESCE($4, description),
          account_type = COALESCE($5::bank_account_type, account_type),
          balance = COALESCE($6, balance),
          last_debit_at = COALESCE($7::timestamptz, last_debit_at),
          last_credit_at = COALESCE($8::timestamptz, last_credit_at),
          updated_at = NOW()
        WHERE user_id = $1 AND id = $2
      `, in.UserID, in.AccountID,
			in.Name, in.Description, in.AccountType, in.Balance,
			nullIfEmptyTime(in.LastDebitAt), nullIfEmptyTime(in.LastCreditAt))
		if err != nil {
			return nil, err
		}
		if cmd.RowsAffected() == 0 {
			_ = tx.Rollback(ctx)
			return nil, nil
		}
	}

	if in.PreferredCat != nil {
		pc := normalizePreferredCategoryNames(*in.PreferredCat)
		if err := syncPreferredCategoryMappings(ctx, tx, in.UserID, in.AccountID, pc); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return GetBankAccountByID(ctx, pool, in.UserID, in.AccountID)
}

// DeleteBankAccount removes a row if owned by the user.
func DeleteBankAccount(ctx context.Context, pool *pgxpool.Pool, userID, accountID string) (bool, error) {
	cmd, err := pool.Exec(ctx,
		`DELETE FROM bank_accounts WHERE user_id = $1 AND id = $2`,
		userID, accountID,
	)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}
