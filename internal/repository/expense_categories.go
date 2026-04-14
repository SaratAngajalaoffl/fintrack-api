package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExpenseCategoryRow matches web ExpenseCategoryRow JSON.
type ExpenseCategoryRow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IconURL     string `json:"iconUrl"`
	Color       string `json:"color"`
}

type CreateExpenseCategoryInput struct {
	UserID      string
	Name        string
	Description string
	IconURL     string
	Color       string
}

type UpdateExpenseCategoryInput struct {
	UserID      string
	CategoryID  string
	Name        *string
	Description *string
	IconURL     *string
	Color       *string
}

// ListExpenseCategories returns categories for a user (newest first).
func ListExpenseCategories(ctx context.Context, pool *pgxpool.Pool, userID string) ([]ExpenseCategoryRow, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, name, description, icon_url, color
		 FROM expense_categories
		 WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExpenseCategoryRow
	for rows.Next() {
		var r ExpenseCategoryRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.IconURL, &r.Color); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetExpenseCategoryByID returns one category for the user or nil if missing.
func GetExpenseCategoryByID(ctx context.Context, pool *pgxpool.Pool, userID, categoryID string) (*ExpenseCategoryRow, error) {
	row := pool.QueryRow(ctx,
		`SELECT id, name, description, icon_url, color
		 FROM expense_categories
		 WHERE user_id = $1 AND id = $2
		 LIMIT 1`,
		userID, categoryID,
	)
	var r ExpenseCategoryRow
	err := row.Scan(&r.ID, &r.Name, &r.Description, &r.IconURL, &r.Color)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateExpenseCategory inserts a row and returns the full record.
func CreateExpenseCategory(ctx context.Context, pool *pgxpool.Pool, in CreateExpenseCategoryInput) (*ExpenseCategoryRow, error) {
	var newID string
	err := pool.QueryRow(ctx,
		`INSERT INTO expense_categories (user_id, name, description, icon_url, color)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id`,
		in.UserID, in.Name, in.Description, in.IconURL, in.Color,
	).Scan(&newID)
	if err != nil {
		return nil, err
	}
	return GetExpenseCategoryByID(ctx, pool, in.UserID, newID)
}

// UpdateExpenseCategory patches fields; returns nil if no row matched.
func UpdateExpenseCategory(ctx context.Context, pool *pgxpool.Pool, in UpdateExpenseCategoryInput) (*ExpenseCategoryRow, error) {
	cmd, err := pool.Exec(ctx,
		`UPDATE expense_categories
		 SET
		   name = COALESCE($3, name),
		   description = COALESCE($4, description),
		   icon_url = COALESCE($5, icon_url),
		   color = COALESCE($6, color),
		   updated_at = NOW()
		 WHERE user_id = $1 AND id = $2`,
		in.UserID, in.CategoryID,
		in.Name, in.Description, in.IconURL, in.Color,
	)
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, nil
	}
	return GetExpenseCategoryByID(ctx, pool, in.UserID, in.CategoryID)
}

// DeleteExpenseCategory deletes a row if owned by the user.
func DeleteExpenseCategory(ctx context.Context, pool *pgxpool.Pool, userID, categoryID string) (bool, error) {
	cmd, err := pool.Exec(ctx,
		`DELETE FROM expense_categories WHERE user_id = $1 AND id = $2`,
		userID, categoryID,
	)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}
