package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"main/internal/identity"
	"main/internal/setup"
)

func (s *Store) GetState(ctx context.Context) (setup.State, error) {
	db, err := s.requireDB()
	if err != nil {
		return setup.State{}, err
	}

	var state setup.State
	var ownerUserID sql.NullInt64
	var completedAt sql.NullInt64
	var createdAt int64
	var updatedAt int64
	err = db.QueryRowContext(
		ctx,
		`SELECT setup_state, owner_user_id, setup_completed_at, created_at, updated_at
		FROM install_state
		WHERE id = 1`,
	).Scan(&state.SetupState, &ownerUserID, &completedAt, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return setup.State{}, errors.New("setup: install state row missing")
	}
	if err != nil {
		return setup.State{}, fmt.Errorf("setup: get install state: %w", err)
	}

	state.SetupCompleted = state.SetupState == setup.StateCompleted
	if ownerUserID.Valid {
		state.OwnerUserID = &ownerUserID.Int64
	}
	state.SetupCompletedAt = nullableUnixTimePointer(completedAt)
	state.CreatedAt = time.Unix(createdAt, 0).UTC()
	state.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return state, nil
}

func (s *Store) CompleteInitialOwner(ctx context.Context, owner setup.InitialOwner) (identity.User, error) {
	db, err := s.requireDB()
	if err != nil {
		return identity.User{}, err
	}

	var created identity.User
	err = withTx(ctx, db, func(tx *sql.Tx) error {
		var setupState string
		if stateErr := tx.QueryRowContext(ctx, `SELECT setup_state FROM install_state WHERE id = 1`).Scan(&setupState); stateErr != nil {
			return fmt.Errorf("setup: load install state: %w", stateErr)
		}
		if setupState == setup.StateCompleted {
			return setup.ErrAlreadyCompleted
		}

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
			) VALUES (?, NULL, NULL, ?, ?, 1, NULL, ?, ?)`,
			owner.Username,
			identity.StatusActive,
			identity.RoleOwner,
			now.Unix(),
			now.Unix(),
		)
		if insertErr != nil {
			return mapIdentityWriteError(insertErr)
		}
		userID, idErr := result.LastInsertId()
		if idErr != nil {
			return fmt.Errorf("setup: read owner user id: %w", idErr)
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
			owner.PasswordHash,
			now.Unix(),
			now.Unix(),
			now.Unix(),
		); credentialErr != nil {
			return fmt.Errorf("setup: insert owner credential: %w", credentialErr)
		}
		if _, stateErr := tx.ExecContext(
			ctx,
			`UPDATE install_state
			SET setup_state = ?, owner_user_id = ?, setup_completed_at = ?, updated_at = ?
			WHERE id = 1`,
			setup.StateCompleted,
			userID,
			now.Unix(),
			now.Unix(),
		); stateErr != nil {
			return fmt.Errorf("setup: update install state: %w", stateErr)
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
