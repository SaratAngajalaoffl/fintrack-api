package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExportAccountPayload matches the JSON shape from GET /api/auth/account-data (schemaVersion 1).
func ExportAccountPayload(ctx context.Context, pool *pgxpool.Pool, userID string) ([]byte, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	type userRow struct {
		ID           string `json:"id"`
		Email        string `json:"email"`
		PasswordHash string `json:"passwordHash"`
		IsApproved   bool   `json:"isApproved"`
		CreatedAt    string `json:"createdAt"`
		UpdatedAt    string `json:"updatedAt"`
	}
	var u userRow
	err = tx.QueryRow(ctx,
		`SELECT id, email, password_hash, is_approved, created_at::text, updated_at::text
		 FROM users WHERE id = $1 LIMIT 1`,
		userID,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsApproved, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}

	var prof any
	type profRow struct {
		UserID            string `json:"user_id"`
		Name              string `json:"name"`
		PreferredCurrency string `json:"preferred_currency"`
		CreatedAt         string `json:"created_at"`
		UpdatedAt         string `json:"updated_at"`
	}
	var pr profRow
	err = tx.QueryRow(ctx,
		`SELECT user_id, name, preferred_currency, created_at::text, updated_at::text
		 FROM user_profiles WHERE user_id = $1 LIMIT 1`, userID,
	).Scan(&pr.UserID, &pr.Name, &pr.PreferredCurrency, &pr.CreatedAt, &pr.UpdatedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		prof = nil
	case err != nil:
		return nil, err
	default:
		prof = pr
	}

	bankAccounts, err := collectBankAccounts(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	fundBuckets, err := collectFundBuckets(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	cc, err := collectCreditCards(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	bills, err := collectCreditCardBills(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	cats, err := collectExpenseCategories(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	payload := map[string]any{
		"schemaVersion":      1,
		"exportedAt":         time.Now().UTC().Format(time.RFC3339Nano),
		"user":               u,
		"userProfile":        prof,
		"bankAccounts":       bankAccounts,
		"fundBuckets":        fundBuckets,
		"creditCards":        cc,
		"creditCardBills":    bills,
		"expenseCategories":  cats,
	}
	return json.MarshalIndent(payload, "", "  ")
}

func collectBankAccounts(ctx context.Context, tx pgx.Tx, userID string) ([]map[string]any, error) {
	rows, err := tx.Query(ctx, `SELECT
              ba.id,
              ba.user_id,
              ba.name,
              ba.description,
              ba.account_type::text,
              ba.initial_balance::text,
              ba.balance::text,
              COALESCE(
                array_agg(ec.name ORDER BY ec.name) FILTER (WHERE ec.name IS NOT NULL),
                '{}'::text[]
              ) AS preferred_categories,
              ba.last_debit_at::text,
              ba.last_credit_at::text,
              ba.created_at::text,
              ba.updated_at::text
       FROM bank_accounts ba
       LEFT JOIN bank_account_preferred_categories bapc
         ON bapc.bank_account_id = ba.id
        AND bapc.user_id = ba.user_id
       LEFT JOIN expense_categories ec
         ON ec.id = bapc.expense_category_id
        AND ec.user_id = ba.user_id
       WHERE ba.user_id = $1
       GROUP BY ba.id
       ORDER BY ba.created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var (
			id, uid, name, desc, accType, initBal, bal string
			cats                                       []string
			lastD, lastC, created, updated             *string
		)
		if err := rows.Scan(
			&id, &uid, &name, &desc, &accType, &initBal, &bal, &cats,
			&lastD, &lastC, &created, &updated,
		); err != nil {
			return nil, err
		}
		m := map[string]any{
			"id": id, "user_id": uid, "name": name, "description": desc,
			"account_type": accType, "initial_balance": initBal, "balance": bal,
			"preferred_categories": cats,
			"created_at": created, "updated_at": updated,
		}
		if lastD != nil {
			m["last_debit_at"] = *lastD
		} else {
			m["last_debit_at"] = nil
		}
		if lastC != nil {
			m["last_credit_at"] = *lastC
		} else {
			m["last_credit_at"] = nil
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func collectFundBuckets(ctx context.Context, tx pgx.Tx, userID string) ([]map[string]any, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, user_id, bank_account_id, name, target_amount::text, current_value::text,
              is_locked, priority::text, created_at::text, updated_at::text
       FROM fund_buckets
       WHERE user_id = $1
       ORDER BY created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var (
			id, uid, baid, name, target, current, pri, created, updated string
			locked                                                        bool
		)
		if err := rows.Scan(
			&id, &uid, &baid, &name, &target, &current, &locked, &pri, &created, &updated,
		); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "user_id": uid, "bank_account_id": baid, "name": name,
			"target_amount": target, "current_value": current, "is_locked": locked,
			"priority": pri, "created_at": created, "updated_at": updated,
		})
	}
	return out, rows.Err()
}

func collectCreditCards(ctx context.Context, tx pgx.Tx, userID string) ([]map[string]any, error) {
	rows, err := tx.Query(ctx, `SELECT
         cc.id,
         cc.user_id,
         cc.name,
         cc.description,
         cc.max_balance::text,
         cc.used_balance::text,
         cc.locked_balance::text,
         COALESCE(
           array_agg(ec.name ORDER BY ec.name) FILTER (WHERE ec.name IS NOT NULL),
           '{}'::text[]
         ) AS preferred_categories,
         cc.bill_generation_day,
         cc.bill_due_day,
         cc.created_at::text,
         cc.updated_at::text
       FROM credit_cards cc
       LEFT JOIN credit_card_preferred_categories ccpc
         ON ccpc.credit_card_id = cc.id
        AND ccpc.user_id = cc.user_id
       LEFT JOIN expense_categories ec
         ON ec.id = ccpc.expense_category_id
        AND ec.user_id = cc.user_id
       WHERE cc.user_id = $1
       GROUP BY cc.id
       ORDER BY cc.created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var (
			id, uid, name, desc string
			maxB, used, locked  string
			cats                []string
			genDay, dueDay      int16
			created, updated    string
		)
		if err := rows.Scan(
			&id, &uid, &name, &desc, &maxB, &used, &locked, &cats,
			&genDay, &dueDay, &created, &updated,
		); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "user_id": uid, "name": name, "description": desc,
			"max_balance": maxB, "used_balance": used, "locked_balance": locked,
			"preferred_categories": cats, "bill_generation_day": genDay, "bill_due_day": dueDay,
			"created_at": created, "updated_at": updated,
		})
	}
	return out, rows.Err()
}

func collectCreditCardBills(ctx context.Context, tx pgx.Tx, userID string) ([]map[string]any, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, user_id, credit_card_id, bill_generation_date::text, bill_due_date::text,
              bill_pdf_url, is_bill_paid, bill_payment_date::text,
              created_at::text, updated_at::text
       FROM credit_card_bills
       WHERE user_id = $1
       ORDER BY bill_generation_date ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var (
			id, uid, ccid, gen, due string
			pdf                      *string
			paid                     bool
			payDate                 *string
			created, updated       string
		)
		if err := rows.Scan(&id, &uid, &ccid, &gen, &due, &pdf, &paid, &payDate, &created, &updated); err != nil {
			return nil, err
		}
		m := map[string]any{
			"id": id, "user_id": uid, "credit_card_id": ccid,
			"bill_generation_date": gen, "bill_due_date": due,
			"is_bill_paid": paid, "created_at": created, "updated_at": updated,
		}
		if pdf != nil {
			m["bill_pdf_url"] = *pdf
		} else {
			m["bill_pdf_url"] = nil
		}
		if payDate != nil {
			m["bill_payment_date"] = *payDate
		} else {
			m["bill_payment_date"] = nil
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func collectExpenseCategories(ctx context.Context, tx pgx.Tx, userID string) ([]map[string]any, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, user_id, name, description, icon_url, color,
              created_at::text, updated_at::text
       FROM expense_categories
       WHERE user_id = $1
       ORDER BY created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, uid, name, desc, icon, color, created, updated string
		if err := rows.Scan(&id, &uid, &name, &desc, &icon, &color, &created, &updated); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "user_id": uid, "name": name, "description": desc,
			"icon_url": icon, "color": color, "created_at": created, "updated_at": updated,
		})
	}
	return out, rows.Err()
}
