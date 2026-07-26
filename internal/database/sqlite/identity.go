package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"main/internal/identity"
)

func (s *Store) CreateUser(ctx context.Context, params identity.CreateUserParams) (identity.User, error) {
	db, err := s.requireDB()
	if err != nil {
		return identity.User{}, err
	}

	var created identity.User
	err = withTx(ctx, db, func(tx *sql.Tx) error {
		now := time.Now().UTC()
		result, insertErr := tx.ExecContext(
			ctx,
			`INSERT INTO users (
				username,
				email,
				avatar_path,
				status,
				role,
				security_version,
				disabled_at,
				created_at,
				updated_at
			) VALUES (?, ?, NULL, ?, ?, 1, NULL, ?, ?)`,
			params.Username,
			params.Email,
			identity.StatusActive,
			params.Role,
			now.Unix(),
			now.Unix(),
		)
		if insertErr != nil {
			return mapIdentityWriteError(insertErr)
		}

		userID, idErr := result.LastInsertId()
		if idErr != nil {
			return fmt.Errorf("identity: read inserted user id: %w", idErr)
		}
		if _, credentialErr := tx.ExecContext(
			ctx,
			`INSERT INTO credentials (
				user_id,
				password_hash,
				password_changed_at,
				created_at,
				updated_at
			) VALUES (?, ?, ?, ?, ?)`,
			userID,
			params.PasswordHash,
			now.Unix(),
			now.Unix(),
			now.Unix(),
		); credentialErr != nil {
			return fmt.Errorf("identity: insert credential: %w", credentialErr)
		}

		user, getErr := getUser(ctx, tx, userID)
		if getErr != nil {
			return getErr
		}
		created = user
		return nil
	})
	if err != nil {
		return identity.User{}, err
	}
	return created, nil
}

func (s *Store) GetUserByID(ctx context.Context, userID int64) (identity.User, error) {
	db, err := s.requireDB()
	if err != nil {
		return identity.User{}, err
	}
	return getUser(ctx, db, userID)
}

func (s *Store) GetAuthRecordByID(ctx context.Context, userID int64) (identity.AuthRecord, error) {
	db, err := s.requireDB()
	if err != nil {
		return identity.AuthRecord{}, err
	}
	return getAuthRecord(ctx, db, userIDSelectQuery(), userID)
}

func (s *Store) GetAuthRecordByIdentifier(ctx context.Context, identifier string) (identity.AuthRecord, error) {
	db, err := s.requireDB()
	if err != nil {
		return identity.AuthRecord{}, err
	}
	return getAuthRecord(
		ctx,
		db,
		`SELECT
			u.id,
			u.username,
			u.email,
			u.avatar_path,
			u.status,
			u.role,
			u.security_version,
			u.disabled_at,
			u.created_at,
			u.updated_at,
			c.password_hash,
			c.password_changed_at
		FROM users u
		INNER JOIN credentials c ON c.user_id = u.id
		WHERE u.username = ? COLLATE NOCASE OR u.email = ? COLLATE NOCASE
		ORDER BY u.id
		LIMIT 1`,
		identifier,
		strings.ToLower(identifier),
	)
}

func (s *Store) UpdateProfile(ctx context.Context, userID int64, username string, email *string) (identity.User, error) {
	db, err := s.requireDB()
	if err != nil {
		return identity.User{}, err
	}
	result, err := db.ExecContext(ctx, `UPDATE users SET username = ?, email = ?, updated_at = ? WHERE id = ?`, username, email, time.Now().UTC().Unix(), userID)
	if err != nil {
		return identity.User{}, mapIdentityWriteError(err)
	}
	if affectedErr := ensureUserRowsAffected(result); affectedErr != nil {
		return identity.User{}, affectedErr
	}
	return getUser(ctx, db, userID)
}

func (s *Store) UpdateAvatar(ctx context.Context, userID int64, avatarPath *string) (identity.User, error) {
	db, err := s.requireDB()
	if err != nil {
		return identity.User{}, err
	}
	result, err := db.ExecContext(ctx, `UPDATE users SET avatar_path = ?, updated_at = ? WHERE id = ?`, avatarPath, time.Now().UTC().Unix(), userID)
	if err != nil {
		return identity.User{}, fmt.Errorf("identity: update avatar: %w", err)
	}
	if affectedErr := ensureUserRowsAffected(result); affectedErr != nil {
		return identity.User{}, affectedErr
	}
	return getUser(ctx, db, userID)
}

func (s *Store) UpdateManagedUser(ctx context.Context, userID int64, params identity.UpdateManagedUserParams) (identity.User, error) {
	db, err := s.requireDB()
	if err != nil {
		return identity.User{}, err
	}

	var updated identity.User
	err = withTx(ctx, db, func(tx *sql.Tx) error {
		result, updateErr := tx.ExecContext(ctx, `UPDATE users SET username = ?, email = ?, updated_at = ? WHERE id = ?`, params.Username, params.Email, time.Now().UTC().Unix(), userID)
		if updateErr != nil {
			return mapIdentityWriteError(updateErr)
		}
		if affectedErr := ensureUserRowsAffected(result); affectedErr != nil {
			return affectedErr
		}
		user, getErr := getUser(ctx, tx, userID)
		if getErr != nil {
			return getErr
		}
		updated = user
		return nil
	})
	if err != nil {
		return identity.User{}, err
	}
	return updated, nil
}

func (s *Store) SetUserStatus(ctx context.Context, userID int64, status string) (identity.User, error) {
	db, err := s.requireDB()
	if err != nil {
		return identity.User{}, err
	}

	var updated identity.User
	err = withTx(ctx, db, func(tx *sql.Tx) error {
		now := time.Now().UTC()
		var disabledAt any
		if status == identity.StatusDisabled {
			disabledAt = now.Unix()
		}
		result, updateErr := tx.ExecContext(
			ctx,
			`UPDATE users
			SET status = ?, disabled_at = ?, security_version = security_version + 1, updated_at = ?
			WHERE id = ?`,
			status,
			disabledAt,
			now.Unix(),
			userID,
		)
		if updateErr != nil {
			return fmt.Errorf("identity: update status: %w", updateErr)
		}
		if affectedErr := ensureUserRowsAffected(result); affectedErr != nil {
			return affectedErr
		}
		if revokeErr := revokeUserSessionsTx(ctx, tx, userID, "status_changed", now); revokeErr != nil {
			return revokeErr
		}
		user, getErr := getUser(ctx, tx, userID)
		if getErr != nil {
			return getErr
		}
		updated = user
		return nil
	})
	if err != nil {
		return identity.User{}, err
	}
	return updated, nil
}

func (s *Store) DeleteUser(ctx context.Context, userID int64) error {
	db, err := s.requireDB()
	if err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("identity: delete user: %w", err)
	}
	return ensureUserRowsAffected(result)
}

func (s *Store) ListUsers(ctx context.Context, params identity.ListUsersParams) (identity.ListUsersResult, error) {
	db, err := s.requireDB()
	if err != nil {
		return identity.ListUsersResult{}, err
	}

	clauses := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if params.Query != "" {
		pattern := "%" + params.Query + "%"
		clauses = append(clauses, "(u.username LIKE ? COLLATE NOCASE OR u.email LIKE ? COLLATE NOCASE)")
		args = append(args, pattern, pattern)
	}
	if params.Status != "" {
		clauses = append(clauses, "u.status = ?")
		args = append(args, params.Status)
	}

	whereSQL := ""
	if len(clauses) > 0 {
		whereSQL = " WHERE " + strings.Join(clauses, " AND ")
	}
	var total int
	if countErr := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users u"+whereSQL, args...).Scan(&total); countErr != nil {
		return identity.ListUsersResult{}, fmt.Errorf("identity: count users: %w", countErr)
	}
	queryArgs := append(append([]any{}, args...), params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := db.QueryContext(ctx, `SELECT
		u.id,
		u.username,
		u.email,
		u.avatar_path,
		u.status,
		u.role,
		u.security_version,
		u.disabled_at,
		(SELECT MAX(s.last_used_at) FROM auth_sessions s WHERE s.user_id = u.id) AS last_active_at,
		u.created_at,
		u.updated_at
	FROM users u`+whereSQL+`
	ORDER BY u.created_at DESC, u.id DESC
	LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return identity.ListUsersResult{}, fmt.Errorf("identity: list users: %w", err)
	}

	items := make([]identity.User, 0, params.PageSize)
	for rows.Next() {
		item, scanErr := scanUser(rows)
		if scanErr != nil {
			return identity.ListUsersResult{}, errors.Join(scanErr, closeUserRows(rows))
		}
		items = append(items, item)
	}
	if iterateErr := errors.Join(rows.Err(), closeUserRows(rows)); iterateErr != nil {
		return identity.ListUsersResult{}, fmt.Errorf("identity: iterate users: %w", iterateErr)
	}
	return identity.ListUsersResult{Items: items, Page: params.Page, PageSize: params.PageSize, Total: total}, nil
}

func getUser(ctx context.Context, db dbTX, userID int64) (identity.User, error) {
	return scanUser(db.QueryRowContext(ctx, `SELECT
		u.id,
		u.username,
		u.email,
		u.avatar_path,
		u.status,
		u.role,
		u.security_version,
		u.disabled_at,
		(SELECT MAX(s.last_used_at) FROM auth_sessions s WHERE s.user_id = u.id) AS last_active_at,
		u.created_at,
		u.updated_at
	FROM users u
	WHERE u.id = ?`, userID))
}

func getAuthRecord(ctx context.Context, db dbTX, query string, args ...any) (identity.AuthRecord, error) {
	var record identity.AuthRecord
	var email sql.NullString
	var avatarPath sql.NullString
	var disabledAt sql.NullInt64
	var passwordChangedAt sql.NullInt64
	var createdAt int64
	var updatedAt int64
	err := db.QueryRowContext(ctx, query, args...).Scan(
		&record.ID,
		&record.Username,
		&email,
		&avatarPath,
		&record.Status,
		&record.Role,
		&record.SecurityVersion,
		&disabledAt,
		&createdAt,
		&updatedAt,
		&record.PasswordHash,
		&passwordChangedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.AuthRecord{}, identity.ErrUserNotFound
	}
	if err != nil {
		return identity.AuthRecord{}, fmt.Errorf("identity: get auth record: %w", err)
	}
	record.Email = nullableStringPointer(email)
	record.AvatarPath = nullableStringPointer(avatarPath)
	record.DisabledAt = nullableUnixTimePointer(disabledAt)
	record.PasswordChangedAt = nullableUnixTimePointer(passwordChangedAt)
	record.CreatedAt = time.Unix(createdAt, 0).UTC()
	record.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return record, nil
}

func userIDSelectQuery() string {
	return `SELECT
		u.id,
		u.username,
		u.email,
		u.avatar_path,
		u.status,
		u.role,
		u.security_version,
		u.disabled_at,
		u.created_at,
		u.updated_at,
		c.password_hash,
		c.password_changed_at
	FROM users u
	INNER JOIN credentials c ON c.user_id = u.id
	WHERE u.id = ?`
}

func scanUser(scanner interface{ Scan(dest ...any) error }) (identity.User, error) {
	var user identity.User
	var email sql.NullString
	var avatarPath sql.NullString
	var disabledAt sql.NullInt64
	var lastActiveAt sql.NullInt64
	var createdAt int64
	var updatedAt int64
	err := scanner.Scan(
		&user.ID,
		&user.Username,
		&email,
		&avatarPath,
		&user.Status,
		&user.Role,
		&user.SecurityVersion,
		&disabledAt,
		&lastActiveAt,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.User{}, identity.ErrUserNotFound
	}
	if err != nil {
		return identity.User{}, fmt.Errorf("identity: scan user: %w", err)
	}
	user.Email = nullableStringPointer(email)
	user.AvatarPath = nullableStringPointer(avatarPath)
	user.DisabledAt = nullableUnixTimePointer(disabledAt)
	user.LastActiveAt = nullableUnixTimePointer(lastActiveAt)
	user.CreatedAt = time.Unix(createdAt, 0).UTC()
	user.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return user, nil
}

func revokeUserSessionsTx(ctx context.Context, tx *sql.Tx, userID int64, reason string, now time.Time) error {
	if _, err := tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at = COALESCE(revoked_at, ?), revoke_reason = COALESCE(revoke_reason, ?) WHERE user_id = ? AND revoked_at IS NULL`, now.Unix(), strings.TrimSpace(reason), userID); err != nil {
		return fmt.Errorf("identity: revoke sessions: %w", err)
	}
	return nil
}

func nullableStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullableUnixTimePointer(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	result := time.Unix(value.Int64, 0).UTC()
	return &result
}

func ensureUserRowsAffected(result sql.Result) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("identity: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return identity.ErrUserNotFound
	}
	return nil
}

func closeUserRows(rows *sql.Rows) error {
	if err := rows.Close(); err != nil {
		return fmt.Errorf("identity: close user rows: %w", err)
	}
	return nil
}

func mapIdentityWriteError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unique constraint failed: users.username"):
		return identity.ErrUsernameTaken
	case strings.Contains(message, "unique constraint failed: users.email"):
		return identity.ErrEmailTaken
	case strings.Contains(message, "unique constraint failed: users.role"):
		return identity.ErrOwnerAlreadyExists
	default:
		return fmt.Errorf("identity: write failed: %w", err)
	}
}
