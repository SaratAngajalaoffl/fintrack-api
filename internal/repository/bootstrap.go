package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// fintrackUserInstallLock serializes initial setup vs concurrent first-user flows.
const fintrackUserInstallLock int64 = 0x46696e747261636b // "Fintrack" ASCII

// ErrBootstrapUnavailable is returned when at least one user already exists.
var ErrBootstrapUnavailable = errors.New("initial setup is no longer available")

// BootstrapAdminWithProfile creates the first user as an approved administrator
// and their profile. It must only succeed when the users table is empty.
func BootstrapAdminWithProfile(ctx context.Context, pool *pgxpool.Pool, email, passwordHash, name, preferredCurrency string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, fintrackUserInstallLock); err != nil {
		return err
	}

	var n int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrBootstrapUnavailable
	}

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, is_approved, is_admin)
		 VALUES ($1, $2, TRUE, TRUE)
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
