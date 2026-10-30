package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"main/internal/identity"
)

// userColumns is passed through ColumnExpr, which replaces bun's default column
// list instead of extending it, so the row columns have to be spelled out here.
const userColumns = `u.*,
	(SELECT MAX(s.last_used_at) FROM auth_sessions AS s WHERE s.user_id = u.id) AS last_active_at`

const authRecordQuery = `
SELECT u.*, c.password_hash, c.password_changed_at
FROM users AS u
INNER JOIN credentials AS c ON c.user_id = u.id`

func (s *Store) CreateUser(ctx context.Context, params identity.CreateUserParams) (identity.User, error) {
	now := time.Now().UTC()
	var created identity.User
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		user := &userRow{userFields: userFields{
			Username:        params.Username,
			Email:           params.Email,
			Status:          identity.StatusActive,
			Role:            params.Role,
			SecurityVersion: 1,
			CreatedAt:       unixSeconds(now),
			UpdatedAt:       unixSeconds(now),
		}}
		// Returning is explicit: bun only fills an autoincrement primary key
		// automatically for identity columns, while SQLite and PostgreSQL both
		// support RETURNING.
		if _, insertErr := tx.NewInsert().Model(user).Returning("id").Exec(ctx); insertErr != nil {
			return s.identityWriteError(insertErr)
		}
		credential := &credentialRow{
			UserID:            user.ID,
			PasswordHash:      params.PasswordHash,
			PasswordChangedAt: unixTimeToPtr(&now),
			CreatedAt:         unixSeconds(now),
			UpdatedAt:         unixSeconds(now),
		}
		if _, credentialErr := tx.NewInsert().Model(credential).Exec(ctx); credentialErr != nil {
			return fmt.Errorf("identity: insert credential: %w", credentialErr)
		}
		loaded, getErr := s.getUser(ctx, tx, user.ID)
		if getErr != nil {
			return getErr
		}
		created = loaded
		return nil
	})
	if err != nil {
		return identity.User{}, err
	}
	return created, nil
}

func (s *Store) GetUserByID(ctx context.Context, userID int64) (identity.User, error) {
	return s.getUser(ctx, s.db, userID)
}

func (s *Store) GetAuthRecordByID(ctx context.Context, userID int64) (identity.AuthRecord, error) {
	return s.getAuthRecord(ctx, s.db.NewRaw(authRecordQuery+"\nWHERE u.id = ?", userID))
}

func (s *Store) GetAuthRecordByIdentifier(ctx context.Context, identifier string) (identity.AuthRecord, error) {
	query := authRecordQuery +
		"\nWHERE " + s.dialect.CaseInsensitiveEqual("u.username") +
		" OR " + s.dialect.CaseInsensitiveEqual("u.email") +
		"\nORDER BY u.id\nLIMIT 1"
	return s.getAuthRecord(ctx, s.db.NewRaw(query, identifier, identifier))
}

func (s *Store) UpdateProfile(ctx context.Context, userID int64, username string, email *string) (identity.User, error) {
	result, err := s.db.NewUpdate().
		Model((*userRow)(nil)).
		Set("username = ?", username).
		Set("email = ?", email).
		Set("updated_at = ?", time.Now().UTC().Unix()).
		Where("id = ?", userID).
		Exec(ctx)
	if err != nil {
		return identity.User{}, s.identityWriteError(err)
	}
	if affectedErr := ensureUserUpdated(result); affectedErr != nil {
		return identity.User{}, affectedErr
	}
	return s.getUser(ctx, s.db, userID)
}

func (s *Store) UpdateAvatar(ctx context.Context, userID int64, avatarPath *string) (identity.User, error) {
	result, err := s.db.NewUpdate().
		Model((*userRow)(nil)).
		Set("avatar_path = ?", avatarPath).
		Set("updated_at = ?", time.Now().UTC().Unix()).
		Where("id = ?", userID).
		Exec(ctx)
	if err != nil {
		return identity.User{}, fmt.Errorf("identity: update avatar: %w", err)
	}
	if affectedErr := ensureUserUpdated(result); affectedErr != nil {
		return identity.User{}, affectedErr
	}
	return s.getUser(ctx, s.db, userID)
}

func (s *Store) UpdateManagedUser(ctx context.Context, userID int64, params identity.UpdateManagedUserParams) (identity.User, error) {
	var updated identity.User
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, updateErr := tx.NewUpdate().
			Model((*userRow)(nil)).
			Set("username = ?", params.Username).
			Set("email = ?", params.Email).
			Set("updated_at = ?", time.Now().UTC().Unix()).
			Where("id = ?", userID).
			Exec(ctx)
		if updateErr != nil {
			return s.identityWriteError(updateErr)
		}
		if affectedErr := ensureUserUpdated(result); affectedErr != nil {
			return affectedErr
		}
		loaded, getErr := s.getUser(ctx, tx, userID)
		if getErr != nil {
			return getErr
		}
		updated = loaded
		return nil
	})
	if err != nil {
		return identity.User{}, err
	}
	return updated, nil
}

func (s *Store) SetUserStatus(ctx context.Context, userID int64, status string) (identity.User, error) {
	var updated identity.User
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		now := time.Now().UTC()
		var disabledAt *int64
		if status == identity.StatusDisabled {
			disabledAt = unixTimeToPtr(&now)
		}
		result, updateErr := tx.NewUpdate().
			Model((*userRow)(nil)).
			Set("status = ?", status).
			Set("disabled_at = ?", disabledAt).
			Set("security_version = security_version + 1").
			Set("updated_at = ?", unixSeconds(now)).
			Where("id = ?", userID).
			Exec(ctx)
		if updateErr != nil {
			return fmt.Errorf("identity: update status: %w", updateErr)
		}
		if affectedErr := ensureUserUpdated(result); affectedErr != nil {
			return affectedErr
		}
		if revokeErr := revokeActiveSessions(ctx, tx, userID, "status_changed", now); revokeErr != nil {
			return revokeErr
		}
		loaded, getErr := s.getUser(ctx, tx, userID)
		if getErr != nil {
			return getErr
		}
		updated = loaded
		return nil
	})
	if err != nil {
		return identity.User{}, err
	}
	return updated, nil
}

func (s *Store) DeleteUser(ctx context.Context, userID int64) error {
	result, err := s.db.NewDelete().
		Model((*userRow)(nil)).
		Where("id = ?", userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("identity: delete user: %w", err)
	}
	return ensureUserUpdated(result)
}

func (s *Store) ListUsers(ctx context.Context, params identity.ListUsersParams) (identity.ListUsersResult, error) {
	rows := make([]userRow, 0, params.PageSize)
	query := s.db.NewSelect().
		Model(&rows).
		ColumnExpr(userColumns).
		OrderExpr("u.created_at DESC, u.id DESC").
		Limit(int64(params.PageSize)).
		Offset(int64((params.Page - 1) * params.PageSize))

	if params.Query != "" {
		pattern := "%" + params.Query + "%"
		query = query.Where(
			"("+s.dialect.CaseInsensitiveLike("u.username")+" OR "+s.dialect.CaseInsensitiveLike("u.email")+")",
			pattern,
			pattern,
		)
	}
	if params.Status != "" {
		query = query.Where("u.status = ?", params.Status)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return identity.ListUsersResult{}, fmt.Errorf("identity: list users: %w", err)
	}

	items := make([]identity.User, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}
	return identity.ListUsersResult{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    int(total),
	}, nil
}

func (s *Store) getUser(ctx context.Context, idb bun.IDB, userID int64) (identity.User, error) {
	row := new(userRow)
	err := idb.NewSelect().
		Model(row).
		ColumnExpr(userColumns).
		Where("u.id = ?", userID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.User{}, identity.ErrUserNotFound
	}
	if err != nil {
		return identity.User{}, fmt.Errorf("identity: get user: %w", err)
	}
	return row.toDomain(), nil
}

func (s *Store) getAuthRecord(ctx context.Context, query *bun.RawQuery) (identity.AuthRecord, error) {
	row := new(authRecordRow)
	err := query.Scan(ctx, row)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.AuthRecord{}, identity.ErrUserNotFound
	}
	if err != nil {
		return identity.AuthRecord{}, fmt.Errorf("identity: get auth record: %w", err)
	}
	return row.toDomain(), nil
}

// revokeActiveSessions revokes every session of userID that is still active,
// keeping the first revocation reason and timestamp. It runs inside the caller's
// transaction, so the state change and the revocation commit together.
func revokeActiveSessions(ctx context.Context, idb bun.IDB, userID int64, reason string, now time.Time) error {
	_, err := idb.NewUpdate().
		Model((*sessionRow)(nil)).
		Set("revoked_at = COALESCE(revoked_at, ?)", unixSeconds(now)).
		Set("revoke_reason = COALESCE(revoke_reason, ?)", reason).
		Where("user_id = ?", userID).
		Where("revoked_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("identity: revoke sessions: %w", err)
	}
	return nil
}

func ensureUserUpdated(result sql.Result) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("identity: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return identity.ErrUserNotFound
	}
	return nil
}
