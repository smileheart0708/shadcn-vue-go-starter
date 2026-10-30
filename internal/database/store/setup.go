package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"main/internal/identity"
	"main/internal/setup"
)

func (s *Store) GetState(ctx context.Context) (setup.State, error) {
	row := new(installStateRow)
	scanErr := s.db.NewSelect().Model(row).Where("id = ?", singletonRowID).Scan(ctx)
	if errors.Is(scanErr, sql.ErrNoRows) {
		return setup.State{}, errors.New("setup: install state row missing")
	}
	if scanErr != nil {
		return setup.State{}, fmt.Errorf("setup: get install state: %w", scanErr)
	}
	return row.toDomain(), nil
}

// CompleteInitialOwner owns the transaction that creates the owner, stores the
// credential and marks the installation complete.
func (s *Store) CompleteInitialOwner(ctx context.Context, owner setup.InitialOwner) (identity.User, error) {
	var created identity.User
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		state := new(installStateRow)
		if stateErr := tx.NewSelect().
			Model(state).
			Column("setup_completed").
			Where("id = ?", singletonRowID).
			Scan(ctx); stateErr != nil {
			return fmt.Errorf("setup: load install state: %w", stateErr)
		}
		if state.SetupCompleted {
			return setup.ErrAlreadyCompleted
		}

		now := time.Now().UTC()
		user := &userRow{userFields: userFields{
			Username:        owner.Username,
			Status:          identity.StatusActive,
			Role:            identity.RoleOwner,
			SecurityVersion: 1,
			CreatedAt:       unixSeconds(now),
			UpdatedAt:       unixSeconds(now),
		}}
		if _, insertErr := tx.NewInsert().Model(user).Returning("id").Exec(ctx); insertErr != nil {
			return s.identityWriteError(insertErr)
		}
		credential := &credentialRow{
			UserID:            user.ID,
			PasswordHash:      owner.PasswordHash,
			PasswordChangedAt: unixTimeToPtr(&now),
			CreatedAt:         unixSeconds(now),
			UpdatedAt:         unixSeconds(now),
		}
		if _, credentialErr := tx.NewInsert().Model(credential).Exec(ctx); credentialErr != nil {
			return fmt.Errorf("setup: insert owner credential: %w", credentialErr)
		}
		if _, stateErr := tx.NewUpdate().
			Model((*installStateRow)(nil)).
			Set("setup_completed = ?", true).
			Set("owner_user_id = ?", user.ID).
			Set("setup_completed_at = ?", unixSeconds(now)).
			Set("updated_at = ?", unixSeconds(now)).
			Where("id = ?", singletonRowID).
			Exec(ctx); stateErr != nil {
			return fmt.Errorf("setup: update install state: %w", stateErr)
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
