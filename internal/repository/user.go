package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRow struct {
	ID           string
	Email        string
	PasswordHash string
	IsApproved   bool
	IsAdmin      bool
}

func FindUserByEmail(ctx context.Context, pool *pgxpool.Pool, emailNorm string) (*UserRow, error) {
	const q = `SELECT id, email, password_hash, is_approved, is_admin
FROM users WHERE lower(email) = $1 LIMIT 1`
	var u UserRow
	err := pool.QueryRow(ctx, q, emailNorm).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsApproved, &u.IsAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func FindUserByID(ctx context.Context, pool *pgxpool.Pool, id string) (*UserRow, error) {
	const q = `SELECT id, email, password_hash, is_approved, is_admin FROM users WHERE id = $1 LIMIT 1`
	var u UserRow
	err := pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsApproved, &u.IsAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func CountUsers(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
