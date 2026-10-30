package identity

import (
	"slices"
	"strings"
)

// Roles are part of the identity model: authorization policies, object-level
// rules, and storage all read them from here, so there is exactly one source.
const (
	RoleOwner = "owner"
	RoleUser  = "user"
)

// Role carries the ordering used by object-level rules. Rank only answers "who
// outranks whom"; what a role may actually do lives in the authorization role
// matrix, so the two never duplicate each other.
type Role struct {
	Key  string
	Rank int
}

// roleOrder declares every role from the highest rank down. Ranks are spaced out
// so future roles can be inserted without renumbering existing ones.
var roleOrder = []Role{
	{Key: RoleOwner, Rank: 100},
	{Key: RoleUser, Rank: 10},
}

// Roles returns the declared roles ordered from highest to lowest rank.
func Roles() []Role {
	return slices.Clone(roleOrder)
}

// RoleRank reports the rank of role. Unknown roles report false so callers fail
// closed instead of treating an unrecognised role as harmless.
func RoleRank(role string) (int, bool) {
	trimmed := strings.TrimSpace(role)
	for _, candidate := range roleOrder {
		if candidate.Key == trimmed {
			return candidate.Rank, true
		}
	}

	return 0, false
}

func normalizeRole(role string) string {
	if _, known := RoleRank(role); !known {
		return ""
	}

	return strings.TrimSpace(role)
}
