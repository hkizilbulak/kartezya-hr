package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation_SQLSTATE23505(t *testing.T) {
	err := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
	if !IsUniqueViolation(err) {
		t.Fatal("expected unique violation for SQLSTATE 23505")
	}
}

func TestIsUniqueViolation_Wrapped(t *testing.T) {
	inner := &pgconn.PgError{Code: "23505"}
	err := errors.Join(errors.New("create history failed"), inner)
	if !IsUniqueViolation(err) {
		t.Fatal("expected unique violation for wrapped SQLSTATE 23505")
	}
}

func TestIsUniqueViolation_OtherCodes(t *testing.T) {
	err := &pgconn.PgError{Code: "23503"}
	if IsUniqueViolation(err) {
		t.Fatal("foreign_key_violation must not map as unique violation")
	}
	if IsUniqueViolation(errors.New("duplicate key value violates unique constraint")) {
		t.Fatal("must not detect unique violation from message string alone")
	}
	if IsUniqueViolation(nil) {
		t.Fatal("nil must not be unique violation")
	}
}
