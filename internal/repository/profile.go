package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProfileRow struct {
	Name               string
	PreferredCurrency string
}

func GetProfile(ctx context.Context, pool *pgxpool.Pool, userID string) (*ProfileRow, error) {
	const q = `SELECT name, preferred_currency FROM user_profiles WHERE user_id = $1 LIMIT 1`
	var p ProfileRow
	err := pool.QueryRow(ctx, q, userID).Scan(&p.Name, &p.PreferredCurrency)
	if errors.Is(err, pgx.ErrNoRows) {
		return &ProfileRow{Name: "User", PreferredCurrency: "USD"}, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpsertProfile matches the PATCH /api/auth/me upsert SQL.
func UpsertProfile(ctx context.Context, pool *pgxpool.Pool, userID, normalizedEmail, insertName, insertCurrency string, updateName, updateCurrency *string) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO user_profiles (user_id, name, preferred_currency)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id)
		 DO UPDATE SET
		   name = COALESCE($4, user_profiles.name),
		   preferred_currency = COALESCE($5, user_profiles.preferred_currency),
		   updated_at = NOW()`,
		userID, insertName, insertCurrency, updateName, updateCurrency,
	)
	return err
}

func UpdatePassword(ctx context.Context, pool *pgxpool.Pool, userID, hash string) error {
	_, err := pool.Exec(ctx, `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`, hash, userID)
	return err
}

func DeleteUser(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	_, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	return err
}
