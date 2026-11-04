package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"main/internal/auth"
)

func (s *Store) GetJWTSecret(ctx context.Context) (string, error) {
	row := new(authSecretRow)
	scanErr := s.db.NewSelect().Model(row).Where("id = ?", singletonRowID).Scan(ctx)
	if errors.Is(scanErr, sql.ErrNoRows) {
		return "", auth.ErrJWTSecretNotFound
	}
	if scanErr != nil {
		return "", fmt.Errorf("auth: get JWT secret: %w", scanErr)
	}
	return row.Secret, nil
}

// EnsureJWTSecret treats a primary-key conflict as another process having
// seeded the row first and returns that process's secret, so concurrent
// starts agree on one signing key instead of one of them failing.
func (s *Store) EnsureJWTSecret(ctx context.Context, candidate string) (string, error) {
	row := &authSecretRow{ID: singletonRowID, Secret: candidate}
	if _, err := s.db.NewInsert().Model(row).Exec(ctx); err != nil {
		if _, isConflict := s.dialect.UniqueViolation(err); !isConflict {
			return "", fmt.Errorf("auth: save JWT secret: %w", err)
		}
		stored, getErr := s.GetJWTSecret(ctx)
		if getErr != nil {
			return "", errors.Join(fmt.Errorf("auth: save JWT secret: %w", err), getErr)
		}
		return stored, nil
	}
	return candidate, nil
}
