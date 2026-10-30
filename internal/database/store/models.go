package store

import (
	"time"

	"github.com/uptrace/bun"
	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/identity"
	"main/internal/setup"
)

// Storage contract shared by every supported dialect: timestamps are Unix
// seconds in UTC, pointers mean NULL, booleans use the dialect's native boolean.
//
// Row models exist so the domain packages stay free of bun tags and column
// layout; the mapping below is the only place where storage and domain meet.

// userFields maps the users table and is embedded in every row that reads it, so
// a column is declared once. Literals have to name the embedded field explicitly
// because go.mod pins Go 1.26, which does not accept promoted keys.
type userFields struct {
	ID              int64   `bun:"id,pk,autoincrement"`
	Username        string  `bun:"username,notnull"`
	Email           *string `bun:"email"`
	AvatarPath      *string `bun:"avatar_path"`
	Status          string  `bun:"status,notnull"`
	Role            string  `bun:"role,notnull"`
	SecurityVersion int64   `bun:"security_version,notnull"`
	DisabledAt      *int64  `bun:"disabled_at"`
	CreatedAt       int64   `bun:"created_at,notnull"`
	UpdatedAt       int64   `bun:"updated_at,notnull"`
}

type userRow struct {
	bun.BaseModel `bun:"table:users,alias:u"`
	userFields

	// Read from a correlated subquery only, never written.
	LastActiveAt *int64 `bun:"last_active_at,scanonly"`
}

// authRecordRow flattens users and credentials for the authentication path.
type authRecordRow struct {
	bun.BaseModel `bun:"table:users,alias:u"`
	userFields

	PasswordHash      string `bun:"password_hash"`
	PasswordChangedAt *int64 `bun:"password_changed_at"`
}

type credentialRow struct {
	bun.BaseModel `bun:"table:credentials"`

	UserID            int64  `bun:"user_id,pk"`
	PasswordHash      string `bun:"password_hash,notnull"`
	PasswordChangedAt *int64 `bun:"password_changed_at"`
	CreatedAt         int64  `bun:"created_at,notnull"`
	UpdatedAt         int64  `bun:"updated_at,notnull"`
}

type sessionRow struct {
	bun.BaseModel `bun:"table:auth_sessions"`

	ID               string  `bun:"id,pk"`
	UserID           int64   `bun:"user_id,notnull"`
	RefreshTokenHash string  `bun:"refresh_token_hash,notnull"`
	CreatedAt        int64   `bun:"created_at,notnull"`
	LastUsedAt       int64   `bun:"last_used_at,notnull"`
	LastRotatedAt    *int64  `bun:"last_rotated_at"`
	ExpiresAt        int64   `bun:"expires_at,notnull"`
	IdleExpiresAt    int64   `bun:"idle_expires_at,notnull"`
	RevokedAt        *int64  `bun:"revoked_at"`
	RevokeReason     *string `bun:"revoke_reason"`
	IP               *string `bun:"ip"`
	UserAgent        *string `bun:"user_agent"`
}

type installStateRow struct {
	bun.BaseModel `bun:"table:install_state"`

	ID               int64  `bun:"id,pk"`
	SetupCompleted   bool   `bun:"setup_completed,notnull"`
	OwnerUserID      *int64 `bun:"owner_user_id"`
	SetupCompletedAt *int64 `bun:"setup_completed_at"`
	CreatedAt        int64  `bun:"created_at,notnull"`
	UpdatedAt        int64  `bun:"updated_at,notnull"`
}

type accountPoliciesRow struct {
	bun.BaseModel `bun:"table:account_policies"`

	ID                                int64 `bun:"id,pk"`
	PublicRegistrationEnabled         bool  `bun:"public_registration_enabled,notnull"`
	SelfServiceAccountDeletionEnabled bool  `bun:"self_service_account_deletion_enabled,notnull"`
}

func unixSeconds(value time.Time) int64 {
	return value.UTC().Unix()
}

func unixTimePtr(seconds *int64) *time.Time {
	if seconds == nil {
		return nil
	}
	converted := time.Unix(*seconds, 0).UTC()
	return &converted
}

func unixTime(seconds int64) time.Time {
	return time.Unix(seconds, 0).UTC()
}

func (f userFields) toDomain() identity.User {
	return identity.User{
		ID:              f.ID,
		Username:        f.Username,
		Email:           f.Email,
		AvatarPath:      f.AvatarPath,
		Status:          f.Status,
		Role:            f.Role,
		SecurityVersion: f.SecurityVersion,
		DisabledAt:      unixTimePtr(f.DisabledAt),
		CreatedAt:       unixTime(f.CreatedAt),
		UpdatedAt:       unixTime(f.UpdatedAt),
	}
}

func (r userRow) toDomain() identity.User {
	user := r.userFields.toDomain()
	user.LastActiveAt = unixTimePtr(r.LastActiveAt)
	return user
}

func (r authRecordRow) toDomain() identity.AuthRecord {
	return identity.AuthRecord{
		User:              r.userFields.toDomain(),
		PasswordHash:      r.PasswordHash,
		PasswordChangedAt: unixTimePtr(r.PasswordChangedAt),
	}
}

func sessionFromDomain(session auth.Session) *sessionRow {
	return &sessionRow{
		ID:               session.ID,
		UserID:           session.UserID,
		RefreshTokenHash: session.RefreshTokenHash,
		CreatedAt:        unixSeconds(session.CreatedAt),
		LastUsedAt:       unixSeconds(session.LastUsedAt),
		LastRotatedAt:    unixTimeToPtr(session.LastRotatedAt),
		ExpiresAt:        unixSeconds(session.ExpiresAt),
		IdleExpiresAt:    unixSeconds(session.IdleExpiresAt),
		RevokedAt:        unixTimeToPtr(session.RevokedAt),
		RevokeReason:     session.RevokeReason,
		IP:               session.IP,
		UserAgent:        session.UserAgent,
	}
}

func (r sessionRow) toDomain() auth.Session {
	return auth.Session{
		ID:               r.ID,
		UserID:           r.UserID,
		RefreshTokenHash: r.RefreshTokenHash,
		CreatedAt:        unixTime(r.CreatedAt),
		LastUsedAt:       unixTime(r.LastUsedAt),
		LastRotatedAt:    unixTimePtr(r.LastRotatedAt),
		ExpiresAt:        unixTime(r.ExpiresAt),
		IdleExpiresAt:    unixTime(r.IdleExpiresAt),
		RevokedAt:        unixTimePtr(r.RevokedAt),
		RevokeReason:     r.RevokeReason,
		IP:               r.IP,
		UserAgent:        r.UserAgent,
	}
}

func (r installStateRow) toDomain() setup.State {
	return setup.State{
		SetupCompleted:   r.SetupCompleted,
		OwnerUserID:      r.OwnerUserID,
		SetupCompletedAt: unixTimePtr(r.SetupCompletedAt),
		CreatedAt:        unixTime(r.CreatedAt),
		UpdatedAt:        unixTime(r.UpdatedAt),
	}
}

func (r accountPoliciesRow) toDomain() accountpolicies.Policies {
	return accountpolicies.Policies{
		PublicRegistrationEnabled:         r.PublicRegistrationEnabled,
		SelfServiceAccountDeletionEnabled: r.SelfServiceAccountDeletionEnabled,
	}
}

func unixTimeToPtr(value *time.Time) *int64 {
	if value == nil {
		return nil
	}
	converted := unixSeconds(*value)
	return &converted
}
