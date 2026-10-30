// Package store implements the persistence ports declared by the domain
// packages on top of bun.
//
// It is dialect neutral: everything that differs between databases is resolved
// by internal/database/dialect, and the domain packages never see bun or the
// column layout.
package store

import (
	"github.com/uptrace/bun"
	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/identity"
	"main/internal/setup"
)

// singletonRowID is the primary key of the single-row configuration tables
// (install_state, account_policies).
const singletonRowID = 1

// dialectRules is the part of a dialect the store needs: decoding driver errors
// and rendering the predicates that are not portable SQL. Keeping it narrow
// means the store cannot open connections or run migrations.
type dialectRules interface {
	UniqueViolation(err error) (constraint string, ok bool)
	CaseInsensitiveEqual(column string) string
	CaseInsensitiveLike(column string) string
}

// Store owns the bun handle plus the dialect rules used by the repositories.
type Store struct {
	db      *bun.DB
	dialect dialectRules
}

// New binds the store to an opened bun database and the dialect rules it needs.
func New(db *bun.DB, driver dialectRules) *Store {
	return &Store{db: db, dialect: driver}
}

var (
	_ identity.Repository        = (*Store)(nil)
	_ auth.SessionRepository     = (*Store)(nil)
	_ setup.Repository           = (*Store)(nil)
	_ accountpolicies.Repository = (*Store)(nil)
)
