package repository

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreditCardBillInfo matches web CreditCardBillInfo JSON.
type CreditCardBillInfo struct {
	ID                 string  `json:"id"`
	BillGenerationDate string  `json:"billGenerationDate"`
	BillDueDate        string  `json:"billDueDate"`
	BillPdfUrl         *string `json:"billPdfUrl"`
	IsBillPaid         bool    `json:"isBillPaid"`
	BillPaymentDate    *string `json:"billPaymentDate"`
}

// CreditCardRow matches web CreditCardRow JSON.
type CreditCardRow struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	MaxBalance          float64             `json:"maxBalance"`
	UsedBalance         float64             `json:"usedBalance"`
	LockedBalance       float64             `json:"lockedBalance"`
	PreferredCategories []string            `json:"preferredCategories"`
	BillGenerationDay   int                 `json:"billGenerationDay"`
	BillDueDay          int                 `json:"billDueDay"`
	LatestBill          *CreditCardBillInfo `json:"latestBill"`
}

type CreateCreditCardInput struct {
	UserID              string
	Name                string
	Description         string
	MaxBalance          float64
	UsedBalance         float64
	LockedBalance       float64
	PreferredCategories []string
	BillGenerationDay   int
	BillDueDay          int
}

type UpdateCreditCardInput struct {
	UserID              string
	CardID              string
	Name                *string
	Description         *string
	MaxBalance          *float64
	UsedBalance         *float64
	LockedBalance       *float64
	BillGenerationDay   *int
	BillDueDay          *int
	PreferredCategories *[]string
}

const creditCardSelectCore = `
      SELECT
        cc.id,
        cc.name,
        cc.description,
        cc.max_balance::float8,
        cc.used_balance::float8,
        cc.locked_balance::float8,
        COALESCE(
          array_agg(ec.name ORDER BY ec.name) FILTER (WHERE ec.name IS NOT NULL),
          '{}'
        ) AS preferred_categories,
        cc.bill_generation_day,
        cc.bill_due_day,
        lb.id AS latest_bill_id,
        lb.bill_generation_date::text AS latest_bill_generation_date,
        lb.bill_due_date::text AS latest_bill_due_date,
        lb.bill_pdf_url AS latest_bill_pdf_url,
        lb.is_bill_paid AS latest_bill_paid,
        lb.bill_payment_date::text AS latest_bill_payment_date
      FROM credit_cards cc
      LEFT JOIN credit_card_preferred_categories ccpc
        ON ccpc.credit_card_id = cc.id
       AND ccpc.user_id = cc.user_id
      LEFT JOIN expense_categories ec
        ON ec.id = ccpc.expense_category_id
       AND ec.user_id = cc.user_id
      LEFT JOIN LATERAL (
        SELECT
          ccb.id,
          ccb.bill_generation_date,
          ccb.bill_due_date,
          ccb.bill_pdf_url,
          ccb.is_bill_paid,
          ccb.bill_payment_date
        FROM credit_card_bills ccb
        WHERE ccb.user_id = cc.user_id
          AND ccb.credit_card_id = cc.id
        ORDER BY ccb.bill_generation_date DESC
        LIMIT 1
      ) lb ON TRUE
`

func scanCreditCardRows(rows pgx.Rows) ([]CreditCardRow, error) {
	defer rows.Close()
	var out []CreditCardRow
	for rows.Next() {
		var (
			id, name, desc     string
			maxB, usedB, lockB float64
			pref               []string
			genDay, dueDay     int32
			lbID, lbGen, lbDue sql.NullString
			lbPdf              sql.NullString
			lbPaid             sql.NullBool
			lbPay              sql.NullString
		)
		if err := rows.Scan(
			&id, &name, &desc, &maxB, &usedB, &lockB, &pref, &genDay, &dueDay,
			&lbID, &lbGen, &lbDue, &lbPdf, &lbPaid, &lbPay,
		); err != nil {
			return nil, err
		}
		if pref == nil {
			pref = []string{}
		}
		r := CreditCardRow{
			ID: id, Name: name, Description: desc,
			MaxBalance: maxB, UsedBalance: usedB, LockedBalance: lockB,
			PreferredCategories: pref,
			BillGenerationDay:   int(genDay),
			BillDueDay:          int(dueDay),
			LatestBill:          nil,
		}
		if lbID.Valid && lbGen.Valid && lbDue.Valid && lbID.String != "" && lbGen.String != "" && lbDue.String != "" {
			var pdf *string
			if lbPdf.Valid {
				s := lbPdf.String
				pdf = &s
			}
			paid := false
			if lbPaid.Valid {
				paid = lbPaid.Bool
			}
			var pay *string
			if lbPay.Valid {
				s := lbPay.String
				pay = &s
			}
			r.LatestBill = &CreditCardBillInfo{
				ID:                 lbID.String,
				BillGenerationDate: lbGen.String,
				BillDueDate:        lbDue.String,
				BillPdfUrl:         pdf,
				IsBillPaid:         paid,
				BillPaymentDate:    pay,
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListCreditCards returns cards for a user (newest first).
func ListCreditCards(ctx context.Context, pool *pgxpool.Pool, userID string) ([]CreditCardRow, error) {
	q := creditCardSelectCore + `
      WHERE cc.user_id = $1
      GROUP BY
        cc.id,
        lb.id,
        lb.bill_generation_date,
        lb.bill_due_date,
        lb.bill_pdf_url,
        lb.is_bill_paid,
        lb.bill_payment_date
      ORDER BY cc.created_at DESC
    `
	rows, err := pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	return scanCreditCardRows(rows)
}

// GetCreditCardByID returns one card or nil.
func GetCreditCardByID(ctx context.Context, pool *pgxpool.Pool, userID, cardID string) (*CreditCardRow, error) {
	q := creditCardSelectCore + `
      WHERE cc.user_id = $1 AND cc.id = $2
      GROUP BY
        cc.id,
        lb.id,
        lb.bill_generation_date,
        lb.bill_due_date,
        lb.bill_pdf_url,
        lb.is_bill_paid,
        lb.bill_payment_date
      LIMIT 1
    `
	rows, err := pool.Query(ctx, q, userID, cardID)
	if err != nil {
		return nil, err
	}
	list, err := scanCreditCardRows(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return &list[0], nil
}

func syncCreditCardPreferredCategoryMappings(
	ctx context.Context,
	tx pgx.Tx,
	userID, cardID string,
	preferredCategories []string,
) error {
	categoryIDs, err := resolvePreferredCategoryIDs(ctx, tx, userID, preferredCategories)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM credit_card_preferred_categories WHERE user_id = $1 AND credit_card_id = $2`,
		userID, cardID,
	); err != nil {
		return err
	}
	if len(categoryIDs) == 0 {
		return nil
	}
	_, err = tx.Exec(ctx, `
      INSERT INTO credit_card_preferred_categories (
        credit_card_id,
        expense_category_id,
        user_id
      )
      SELECT $1, ids.category_id, $2
      FROM unnest($3::uuid[]) AS ids(category_id)
      ON CONFLICT (credit_card_id, expense_category_id) DO NOTHING
    `, cardID, userID, categoryIDs)
	return err
}

// CreateCreditCard inserts a card and preferred category links.
func CreateCreditCard(ctx context.Context, pool *pgxpool.Pool, in CreateCreditCardInput) (*CreditCardRow, error) {
	pc := normalizePreferredCategoryNames(in.PreferredCategories)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var newID string
	err = tx.QueryRow(ctx, `
      INSERT INTO credit_cards (
        user_id, name, description, max_balance, used_balance, locked_balance,
        bill_generation_day, bill_due_day
      )
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
      RETURNING id
    `, in.UserID, in.Name, in.Description, in.MaxBalance, in.UsedBalance, in.LockedBalance,
		in.BillGenerationDay, in.BillDueDay).Scan(&newID)
	if err != nil {
		return nil, err
	}
	if err := syncCreditCardPreferredCategoryMappings(ctx, tx, in.UserID, newID, pc); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return GetCreditCardByID(ctx, pool, in.UserID, newID)
}

// UpdateCreditCard patches scalar fields and optionally replaces preferred categories.
func UpdateCreditCard(ctx context.Context, pool *pgxpool.Pool, in UpdateCreditCardInput) (*CreditCardRow, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	cmd, err := tx.Exec(ctx, `
      UPDATE credit_cards
      SET
        name = COALESCE($3, name),
        description = COALESCE($4, description),
        max_balance = COALESCE($5, max_balance),
        used_balance = COALESCE($6, used_balance),
        locked_balance = COALESCE($7, locked_balance),
        bill_generation_day = COALESCE($8, bill_generation_day),
        bill_due_day = COALESCE($9, bill_due_day),
        updated_at = NOW()
      WHERE user_id = $1 AND id = $2
    `, in.UserID, in.CardID,
		in.Name, in.Description, in.MaxBalance, in.UsedBalance, in.LockedBalance,
		in.BillGenerationDay, in.BillDueDay)
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, nil
	}
	if in.PreferredCategories != nil {
		pc := normalizePreferredCategoryNames(*in.PreferredCategories)
		if err := syncCreditCardPreferredCategoryMappings(ctx, tx, in.UserID, in.CardID, pc); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return GetCreditCardByID(ctx, pool, in.UserID, in.CardID)
}

// DeleteCreditCard removes a card owned by the user.
func DeleteCreditCard(ctx context.Context, pool *pgxpool.Pool, userID, cardID string) (bool, error) {
	cmd, err := pool.Exec(ctx,
		`DELETE FROM credit_cards WHERE user_id = $1 AND id = $2`,
		userID, cardID,
	)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}
