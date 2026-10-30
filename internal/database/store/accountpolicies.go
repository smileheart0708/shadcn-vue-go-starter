package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"main/internal/accountpolicies"
)

func (s *Store) GetPolicies(ctx context.Context) (accountpolicies.Policies, error) {
	row := new(accountPoliciesRow)
	scanErr := s.db.NewSelect().Model(row).Where("id = ?", singletonRowID).Scan(ctx)
	if errors.Is(scanErr, sql.ErrNoRows) {
		return accountpolicies.Policies{}, errors.New("accountpolicies: policies row not found")
	}
	if scanErr != nil {
		return accountpolicies.Policies{}, fmt.Errorf("accountpolicies: get policies: %w", scanErr)
	}
	return row.toDomain(), nil
}

// UpdatePolicies keeps COALESCE semantics: a nil field leaves the stored value
// untouched, so callers can patch a single policy. A missing singleton row needs
// no check here: the update touches nothing and the read below reports it.
func (s *Store) UpdatePolicies(ctx context.Context, input accountpolicies.UpdateInput) (accountpolicies.Policies, error) {
	if _, execErr := s.db.NewUpdate().
		Model((*accountPoliciesRow)(nil)).
		Set("public_registration_enabled = COALESCE(?, public_registration_enabled)", input.PublicRegistrationEnabled).
		Set("self_service_account_deletion_enabled = COALESCE(?, self_service_account_deletion_enabled)", input.SelfServiceAccountDeletionEnabled).
		Where("id = ?", singletonRowID).
		Exec(ctx); execErr != nil {
		return accountpolicies.Policies{}, fmt.Errorf("accountpolicies: update policies: %w", execErr)
	}
	return s.GetPolicies(ctx)
}
