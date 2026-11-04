package postgresql

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestUniqueViolationMapsKnownIndexNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		constraint string
		want       string
	}{
		{constraint: "users_username_unique_idx", want: "users.username"},
		{constraint: "users_email_unique_idx", want: "users.email"},
		{constraint: "users_owner_unique_idx", want: "users.role"},
	}
	for _, test := range tests {
		got, ok := New().UniqueViolation(&pgconn.PgError{
			Code:           codeUniqueViolation,
			ConstraintName: test.constraint,
		})
		if !ok || got != test.want {
			t.Fatalf("UniqueViolation(%q) = (%q, %t), want (%q, true)", test.constraint, got, ok, test.want)
		}
	}

	if got, ok := New().UniqueViolation(&pgconn.PgError{
		Code:           codeUniqueViolation,
		ConstraintName: "auth_sessions_pkey",
	}); !ok || got != "" {
		t.Fatalf("unknown unique violation = (%q, %t), want (\"\", true)", got, ok)
	}
	if _, ok := New().UniqueViolation(errors.New("other")); ok {
		t.Fatal("non-PostgreSQL error reported as a unique violation")
	}
}

func TestCaseInsensitivePredicates(t *testing.T) {
	t.Parallel()

	if got := New().CaseInsensitiveEqual("u.username"); got != "LOWER(u.username) = LOWER(?)" {
		t.Fatalf("CaseInsensitiveEqual() = %q", got)
	}
	if got := New().CaseInsensitiveLike("u.email"); got != "LOWER(u.email) LIKE LOWER(?)" {
		t.Fatalf("CaseInsensitiveLike() = %q", got)
	}
}

func TestOpenRejectsEmptyDSN(t *testing.T) {
	t.Parallel()

	if _, err := New().Open(context.Background(), "   "); err == nil {
		t.Fatal("Open() error = nil, want an empty DSN error")
	}
}

func TestOpenDoesNotLeakCredentialsOnInvalidDSN(t *testing.T) {
	t.Parallel()

	_, err := New().Open(context.Background(), "postgres://user:supersecret@%zz/app")
	if err == nil {
		t.Fatal("Open() error = nil, want an invalid DSN error")
	}
	if strings.Contains(err.Error(), "supersecret") {
		t.Fatalf("Open() error leaks the password: %v", err)
	}
}
