package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"main/internal/auth"
)

func (s *Store) CreateSession(ctx context.Context, session auth.Session) error {
	db, err := s.requireDB()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(
		ctx,
		`INSERT INTO auth_sessions (
			id,
			user_id,
			refresh_token_hash,
			created_at,
			last_used_at,
			last_rotated_at,
			expires_at,
			idle_expires_at,
			revoked_at,
			revoke_reason,
			ip,
			user_agent
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID,
		session.UserID,
		session.RefreshTokenHash,
		session.CreatedAt.Unix(),
		session.LastUsedAt.Unix(),
		nullableUnix(session.LastRotatedAt),
		session.ExpiresAt.Unix(),
		session.IdleExpiresAt.Unix(),
		nullableUnix(session.RevokedAt),
		session.RevokeReason,
		session.IP,
		session.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("auth: insert refresh session: %w", err)
	}
	return nil
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (auth.Session, error) {
	db, err := s.requireDB()
	if err != nil {
		return auth.Session{}, err
	}

	var session auth.Session
	var createdAt int64
	var lastUsedAt int64
	var lastRotatedAt sql.NullInt64
	var expiresAt int64
	var idleExpiresAt int64
	var revokedAt sql.NullInt64
	var revokeReason sql.NullString
	err = db.QueryRowContext(
		ctx,
		`SELECT
			id,
			user_id,
			refresh_token_hash,
			created_at,
			last_used_at,
			last_rotated_at,
			expires_at,
			idle_expires_at,
			revoked_at,
			revoke_reason
		FROM auth_sessions
		WHERE id = ?`,
		sessionID,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&createdAt,
		&lastUsedAt,
		&lastRotatedAt,
		&expiresAt,
		&idleExpiresAt,
		&revokedAt,
		&revokeReason,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	if err != nil {
		return auth.Session{}, fmt.Errorf("auth: get refresh session: %w", err)
	}

	session.CreatedAt = time.Unix(createdAt, 0).UTC()
	session.LastUsedAt = time.Unix(lastUsedAt, 0).UTC()
	session.LastRotatedAt = nullableUnixTimePointer(lastRotatedAt)
	session.ExpiresAt = time.Unix(expiresAt, 0).UTC()
	session.IdleExpiresAt = time.Unix(idleExpiresAt, 0).UTC()
	session.RevokedAt = nullableUnixTimePointer(revokedAt)
	if revokeReason.Valid {
		session.RevokeReason = &revokeReason.String
	}
	return session, nil
}

func (s *Store) RotateSession(ctx context.Context, sessionID string, expectedHash string, nextHash string, lastUsedAt time.Time, idleExpiresAt time.Time) (bool, error) {
	db, err := s.requireDB()
	if err != nil {
		return false, err
	}
	result, err := db.ExecContext(
		ctx,
		`UPDATE auth_sessions
		SET refresh_token_hash = ?, last_used_at = ?, last_rotated_at = ?, idle_expires_at = ?
		WHERE id = ? AND refresh_token_hash = ? AND revoked_at IS NULL`,
		nextHash,
		lastUsedAt.Unix(),
		lastUsedAt.Unix(),
		idleExpiresAt.Unix(),
		sessionID,
		expectedHash,
	)
	if err != nil {
		return false, fmt.Errorf("auth: rotate refresh session: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("auth: rotate refresh session rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}

func (s *Store) RevokeSession(ctx context.Context, sessionID string, reason string, revokedAt time.Time) error {
	db, err := s.requireDB()
	if err != nil {
		return err
	}
	if _, execErr := db.ExecContext(
		ctx,
		`UPDATE auth_sessions
		SET revoked_at = COALESCE(revoked_at, ?), revoke_reason = COALESCE(revoke_reason, ?)
		WHERE id = ?`,
		revokedAt.Unix(),
		strings.TrimSpace(reason),
		sessionID,
	); execErr != nil {
		return fmt.Errorf("auth: revoke session: %w", execErr)
	}
	return nil
}

func (s *Store) UpdatePasswordAndRevokeSessions(ctx context.Context, userID int64, passwordHash string, changedAt time.Time) error {
	db, err := s.requireDB()
	if err != nil {
		return err
	}
	return withTx(ctx, db, func(tx *sql.Tx) error {
		if _, credentialErr := tx.ExecContext(
			ctx,
			`UPDATE credentials SET password_hash = ?, password_changed_at = ?, updated_at = ? WHERE user_id = ?`,
			passwordHash,
			changedAt.Unix(),
			changedAt.Unix(),
			userID,
		); credentialErr != nil {
			return fmt.Errorf("auth: update password credential: %w", credentialErr)
		}
		if _, versionErr := tx.ExecContext(
			ctx,
			`UPDATE users SET security_version = security_version + 1, updated_at = ? WHERE id = ?`,
			changedAt.Unix(),
			userID,
		); versionErr != nil {
			return fmt.Errorf("auth: bump security version: %w", versionErr)
		}
		if _, revokeErr := tx.ExecContext(
			ctx,
			`UPDATE auth_sessions
			SET revoked_at = COALESCE(revoked_at, ?), revoke_reason = COALESCE(revoke_reason, 'password_changed')
			WHERE user_id = ? AND revoked_at IS NULL`,
			changedAt.Unix(),
			userID,
		); revokeErr != nil {
			return fmt.Errorf("auth: revoke sessions after password change: %w", revokeErr)
		}
		return nil
	})
}

func nullableUnix(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Unix()
}
