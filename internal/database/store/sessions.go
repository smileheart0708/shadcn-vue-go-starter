package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"main/internal/auth"
)

func (s *Store) CreateSession(ctx context.Context, session auth.Session) error {
	if _, insertErr := s.db.NewInsert().Model(sessionFromDomain(session)).Exec(ctx); insertErr != nil {
		return fmt.Errorf("auth: insert refresh session: %w", insertErr)
	}
	return nil
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (auth.Session, error) {
	row := new(sessionRow)
	scanErr := s.db.NewSelect().Model(row).Where("id = ?", sessionID).Scan(ctx)
	if errors.Is(scanErr, sql.ErrNoRows) {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	if scanErr != nil {
		return auth.Session{}, fmt.Errorf("auth: get refresh session: %w", scanErr)
	}
	return row.toDomain(), nil
}

// RotateSession only succeeds while the stored hash still matches expectedHash
// and the session is not revoked, which keeps refresh-token reuse detectable.
func (s *Store) RotateSession(ctx context.Context, sessionID string, expectedHash string, nextHash string, lastUsedAt time.Time, idleExpiresAt time.Time) (bool, error) {
	result, updateErr := s.db.NewUpdate().
		Model((*sessionRow)(nil)).
		Set("refresh_token_hash = ?", nextHash).
		Set("last_used_at = ?", unixSeconds(lastUsedAt)).
		Set("last_rotated_at = ?", unixSeconds(lastUsedAt)).
		Set("idle_expires_at = ?", unixSeconds(idleExpiresAt)).
		Where("id = ?", sessionID).
		Where("refresh_token_hash = ?", expectedHash).
		Where("revoked_at IS NULL").
		Exec(ctx)
	if updateErr != nil {
		return false, fmt.Errorf("auth: rotate refresh session: %w", updateErr)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("auth: rotate refresh session rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}

func (s *Store) RevokeSession(ctx context.Context, sessionID string, reason string, revokedAt time.Time) error {
	if _, execErr := s.db.NewUpdate().
		Model((*sessionRow)(nil)).
		Set("revoked_at = COALESCE(revoked_at, ?)", unixSeconds(revokedAt)).
		Set("revoke_reason = COALESCE(revoke_reason, ?)", strings.TrimSpace(reason)).
		Where("id = ?", sessionID).
		Exec(ctx); execErr != nil {
		return fmt.Errorf("auth: revoke session: %w", execErr)
	}
	return nil
}

func (s *Store) UpdatePasswordAndRevokeSessions(ctx context.Context, userID int64, passwordHash string, changedAt time.Time) error {
	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, credentialErr := tx.NewUpdate().
			Model((*credentialRow)(nil)).
			Set("password_hash = ?", passwordHash).
			Set("password_changed_at = ?", unixSeconds(changedAt)).
			Set("updated_at = ?", unixSeconds(changedAt)).
			Where("user_id = ?", userID).
			Exec(ctx); credentialErr != nil {
			return fmt.Errorf("auth: update password credential: %w", credentialErr)
		}
		if _, versionErr := tx.NewUpdate().
			Model((*userRow)(nil)).
			Set("security_version = security_version + 1").
			Set("updated_at = ?", unixSeconds(changedAt)).
			Where("id = ?", userID).
			Exec(ctx); versionErr != nil {
			return fmt.Errorf("auth: bump security version: %w", versionErr)
		}
		if revokeErr := revokeActiveSessions(ctx, tx, userID, "password_changed", changedAt); revokeErr != nil {
			return fmt.Errorf("auth: revoke sessions after password change: %w", revokeErr)
		}
		return nil
	})
}
