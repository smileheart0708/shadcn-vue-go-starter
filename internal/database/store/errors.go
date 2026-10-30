package store

import (
	"fmt"
	"strings"

	"main/internal/identity"
)

// identityWriteError maps a failed user write to the domain error the caller can
// branch on. The dialect reports the violated columns as <table>.<column>; what
// each column means for identity stays here.
func (s *Store) identityWriteError(err error) error {
	if err == nil {
		return nil
	}
	columns, isUniqueViolation := s.dialect.UniqueViolation(err)
	if !isUniqueViolation {
		return fmt.Errorf("identity: write failed: %w", err)
	}
	switch {
	case strings.Contains(columns, "username"):
		return identity.ErrUsernameTaken
	case strings.Contains(columns, "email"):
		return identity.ErrEmailTaken
	case strings.Contains(columns, "role"):
		return identity.ErrOwnerAlreadyExists
	default:
		return fmt.Errorf("identity: write failed: %w", err)
	}
}
