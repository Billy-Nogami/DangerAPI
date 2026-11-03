package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"threat-service/internal/repository"
)

func TestMapDatabaseError(t *testing.T) {
	t.Parallel()

	err := mapDatabaseError(&pgconn.PgError{Code: "23505"})
	if !errors.Is(err, repository.ErrAlreadyExists) {
		t.Fatalf("mapDatabaseError() = %v, want %v", err, repository.ErrAlreadyExists)
	}
}

func TestCategoriesForStorage(t *testing.T) {
	t.Parallel()

	if categories := categoriesForStorage(nil); categories == nil || len(categories) != 0 {
		t.Fatalf("categoriesForStorage(nil) = %#v, want empty non-nil slice", categories)
	}
}
