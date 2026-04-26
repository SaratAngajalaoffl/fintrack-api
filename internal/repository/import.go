package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type importPayload struct {
	SchemaVersion     int                        `json:"schemaVersion"`
	UserProfile       *importUserProfileRow      `json:"userProfile"`
	BankAccounts      []importBankAccountRow     `json:"bankAccounts"`
	FundBuckets       []importFundBucketRow      `json:"fundBuckets"`
	CreditCards       []importCreditCardRow      `json:"creditCards"`
	CreditCardBills   []importCreditCardBillRow  `json:"creditCardBills"`
	ExpenseCategories []importExpenseCategoryRow `json:"expenseCategories"`
}

type importUserProfileRow struct {
	Name              string `json:"name"`
	PreferredCurrency string `json:"preferred_currency"`
}

type importBankAccountRow struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	AccountType         string   `json:"account_type"`
	InitialBalance      string   `json:"initial_balance"`
	Balance             string   `json:"balance"`
	PreferredCategories []string `json:"preferred_categories"`
	LastDebitAt         *string  `json:"last_debit_at"`
	LastCreditAt        *string  `json:"last_credit_at"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
}

type importFundBucketRow struct {
	ID            string `json:"id"`
	BankAccountID string `json:"bank_account_id"`
	Name          string `json:"name"`
	TargetAmount  string `json:"target_amount"`
	CurrentValue  string `json:"current_value"`
	IsLocked      bool   `json:"is_locked"`
	Priority      string `json:"priority"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type importCreditCardRow struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	MaxBalance          string   `json:"max_balance"`
	UsedBalance         string   `json:"used_balance"`
	LockedBalance       string   `json:"locked_balance"`
	PreferredCategories []string `json:"preferred_categories"`
	BillGenerationDay   int16    `json:"bill_generation_day"`
	BillDueDay          int16    `json:"bill_due_day"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
}

type importCreditCardBillRow struct {
	ID                 string  `json:"id"`
	CreditCardID       string  `json:"credit_card_id"`
	BillGenerationDate string  `json:"bill_generation_date"`
	BillDueDate        string  `json:"bill_due_date"`
	BillPDFURL         *string `json:"bill_pdf_url"`
	IsBillPaid         bool    `json:"is_bill_paid"`
	BillPaymentDate    *string `json:"bill_payment_date"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

type importExpenseCategoryRow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IconURL     string `json:"icon_url"`
	Color       string `json:"color"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ImportAccountPayload restores account data from GET /api/auth/account-data for one user.
func ImportAccountPayload(ctx context.Context, pool *pgxpool.Pool, userID string, raw []byte) error {
	var payload importPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if payload.SchemaVersion != 1 {
		return errors.New("unsupported schemaVersion")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := clearUserDataForImport(ctx, tx, userID); err != nil {
		return err
	}
	if err := insertExpenseCategoriesFromImport(ctx, tx, userID, payload.ExpenseCategories); err != nil {
		return err
	}
	if err := upsertProfileFromImport(ctx, tx, userID, payload.UserProfile); err != nil {
		return err
	}
	if err := insertBankAccountsFromImport(ctx, tx, userID, payload.BankAccounts); err != nil {
		return err
	}
	if err := insertCreditCardsFromImport(ctx, tx, userID, payload.CreditCards); err != nil {
		return err
	}
	if err := insertFundBucketsFromImport(ctx, tx, userID, payload.FundBuckets); err != nil {
		return err
	}
	if err := insertCreditCardBillsFromImport(ctx, tx, userID, payload.CreditCardBills); err != nil {
		return err
	}
	if err := insertBankAccountPreferredCategoriesFromImport(ctx, tx, userID, payload.BankAccounts); err != nil {
		return err
	}
	if err := insertCreditCardPreferredCategoriesFromImport(ctx, tx, userID, payload.CreditCards); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func clearUserDataForImport(ctx context.Context, tx pgx.Tx, userID string) error {
	statements := []string{
		`DELETE FROM bank_account_preferred_categories WHERE user_id = $1`,
		`DELETE FROM credit_card_preferred_categories WHERE user_id = $1`,
		`DELETE FROM credit_card_bills WHERE user_id = $1`,
		`DELETE FROM fund_buckets WHERE user_id = $1`,
		`DELETE FROM bank_accounts WHERE user_id = $1`,
		`DELETE FROM credit_cards WHERE user_id = $1`,
		`DELETE FROM expense_categories WHERE user_id = $1`,
		`DELETE FROM user_profiles WHERE user_id = $1`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement, userID); err != nil {
			return err
		}
	}
	return nil
}

func upsertProfileFromImport(ctx context.Context, tx pgx.Tx, userID string, profile *importUserProfileRow) error {
	if profile == nil {
		return nil
	}
	_, err := tx.Exec(ctx, `INSERT INTO user_profiles (user_id, name, preferred_currency)
		VALUES ($1, $2, $3)`, userID, profile.Name, profile.PreferredCurrency)
	return err
}

func insertExpenseCategoriesFromImport(ctx context.Context, tx pgx.Tx, userID string, rows []importExpenseCategoryRow) error {
	for _, row := range rows {
		_, err := tx.Exec(ctx, `INSERT INTO expense_categories
			(id, user_id, name, description, icon_url, color, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7::timestamptz, $8::timestamptz)`,
			row.ID, userID, row.Name, row.Description, row.IconURL, row.Color, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertBankAccountsFromImport(ctx context.Context, tx pgx.Tx, userID string, rows []importBankAccountRow) error {
	for _, row := range rows {
		_, err := tx.Exec(ctx, `INSERT INTO bank_accounts
			(id, user_id, name, description, account_type, initial_balance, balance, last_debit_at, last_credit_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5::bank_account_type, $6::numeric, $7::numeric, $8::timestamptz, $9::timestamptz, $10::timestamptz, $11::timestamptz)`,
			row.ID, userID, row.Name, row.Description, row.AccountType, row.InitialBalance, row.Balance,
			row.LastDebitAt, row.LastCreditAt, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertFundBucketsFromImport(ctx context.Context, tx pgx.Tx, userID string, rows []importFundBucketRow) error {
	for _, row := range rows {
		_, err := tx.Exec(ctx, `INSERT INTO fund_buckets
			(id, user_id, bank_account_id, name, target_amount, current_value, is_locked, priority, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5::numeric, $6::numeric, $7, $8::fund_bucket_priority, $9::timestamptz, $10::timestamptz)`,
			row.ID, userID, row.BankAccountID, row.Name, row.TargetAmount, row.CurrentValue, row.IsLocked, row.Priority, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertCreditCardsFromImport(ctx context.Context, tx pgx.Tx, userID string, rows []importCreditCardRow) error {
	for _, row := range rows {
		_, err := tx.Exec(ctx, `INSERT INTO credit_cards
			(id, user_id, name, description, max_balance, used_balance, locked_balance, bill_generation_day, bill_due_day, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5::numeric, $6::numeric, $7::numeric, $8, $9, $10::timestamptz, $11::timestamptz)`,
			row.ID, userID, row.Name, row.Description, row.MaxBalance, row.UsedBalance, row.LockedBalance,
			row.BillGenerationDay, row.BillDueDay, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertCreditCardBillsFromImport(ctx context.Context, tx pgx.Tx, userID string, rows []importCreditCardBillRow) error {
	for _, row := range rows {
		_, err := tx.Exec(ctx, `INSERT INTO credit_card_bills
			(id, user_id, credit_card_id, bill_generation_date, bill_due_date, bill_pdf_url, is_bill_paid, bill_payment_date, created_at, updated_at)
			VALUES ($1, $2, $3, $4::date, $5::date, $6, $7, $8::date, $9::timestamptz, $10::timestamptz)`,
			row.ID, userID, row.CreditCardID, row.BillGenerationDate, row.BillDueDate,
			row.BillPDFURL, row.IsBillPaid, row.BillPaymentDate, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertBankAccountPreferredCategoriesFromImport(ctx context.Context, tx pgx.Tx, userID string, rows []importBankAccountRow) error {
	for _, row := range rows {
		for _, categoryName := range row.PreferredCategories {
			_, err := tx.Exec(ctx, `INSERT INTO bank_account_preferred_categories (user_id, bank_account_id, expense_category_id)
				SELECT $1, $2, ec.id
				FROM expense_categories ec
				WHERE ec.user_id = $1 AND ec.name = $3
				LIMIT 1`, userID, row.ID, categoryName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func insertCreditCardPreferredCategoriesFromImport(ctx context.Context, tx pgx.Tx, userID string, rows []importCreditCardRow) error {
	for _, row := range rows {
		for _, categoryName := range row.PreferredCategories {
			_, err := tx.Exec(ctx, `INSERT INTO credit_card_preferred_categories (user_id, credit_card_id, expense_category_id)
				SELECT $1, $2, ec.id
				FROM expense_categories ec
				WHERE ec.user_id = $1 AND ec.name = $3
				LIMIT 1`, userID, row.ID, categoryName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
