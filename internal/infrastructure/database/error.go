package database

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jarviisha/darkvoid/pkg/errors"
)

// MapDBError maps database errors to application errors
// This is a generic mapper that handles common database errors
// Domain-specific error handling should be done in the service layer
func MapDBError(err error) error {
	if err == nil {
		return nil
	}

	// No rows found - let the caller decide what domain error to return
	if err == pgx.ErrNoRows {
		return errors.ErrNotFound
	}

	// PostgreSQL error
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return mapPgError(pgErr)
	}

	// Unknown error -> internal error
	return errors.NewInternalError(err)
}

// mapPgError maps PostgreSQL error codes to application errors
func mapPgError(pgErr *pgconn.PgError) error {
	switch pgErr.Code {
	case "23505": // unique_violation
		// Note: We intentionally return a generic conflict error here
		// Services should check for uniqueness BEFORE attempting insert/update
		// This error handler is a fallback for race conditions
		return errors.NewConflictError("resource already exists").
			WithDetail("constraint", pgErr.ConstraintName)

	case "23503": // foreign_key_violation
		return errors.NewConflictError("foreign key violation").
			WithDetail("constraint", pgErr.ConstraintName)

	case "23502": // not_null_violation
		return errors.NewValidationError(pgErr.ColumnName, "cannot be null")

	case "23514": // check_violation
		return errors.NewBadRequestError("check constraint violation").
			WithDetail("constraint", pgErr.ConstraintName)

	case "40001": // serialization_failure
		return errors.NewConflictError("transaction conflict, please retry")

	case "40P01": // deadlock_detected
		return errors.NewConflictError("deadlock detected, please retry")

	default:
		// Unknown PostgreSQL error
		return errors.NewInternalError(pgErr).
			WithDetail("pg_code", pgErr.Code).
			WithDetail("pg_message", pgErr.Message)
	}
}

// IsNotFound checks if the error is a "not found" error
func IsNotFound(err error) bool {
	return err == pgx.ErrNoRows || errors.Is(err, errors.ErrNotFound)
}

// IsConflict checks if the error is a conflict error
func IsConflict(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" || pgErr.Code == "40001" || pgErr.Code == "40P01"
	}
	return errors.Is(err, errors.ErrConflict)
}
