package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	if IsUniqueViolation(errors.New("other")) {
		t.Fatal("expected false")
	}
	err := &pgconn.PgError{Code: "23505"}
	if !IsUniqueViolation(err) {
		t.Fatal("expected true for unique violation")
	}
}
