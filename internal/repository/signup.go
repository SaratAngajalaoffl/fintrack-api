package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateUserWithProfile inserts users + user_profiles in one transaction (matches signup route).
func CreateUserWithProfile(ctx context.Context, pool *pgxpool.Pool, email, passwordHash, name, preferredCurrency string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, is_approved, is_admin)
		 VALUES ($1, $2, FALSE, FALSE)
		 RETURNING id`,
		email, passwordHash,
	).Scan(&id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO user_profiles (user_id, name, preferred_currency)
		 VALUES ($1, $2, $3)`,
		id, name, preferredCurrency,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
