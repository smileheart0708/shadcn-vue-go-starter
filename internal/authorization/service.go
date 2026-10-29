package authorization

import (
	"slices"
	"sort"

	"main/internal/identity"
)

const (
	CapabilitySystemSettingsRead       = "system.settings.read"
	CapabilitySystemSettingsUpdate     = "system.settings.update"
	CapabilityManagementUsersRead      = "management.users.read"
	CapabilityManagementUsersCreate    = "management.users.create"
	CapabilityManagementUsersUpdate    = "management.users.update"
	CapabilityManagementUsersDisable   = "management.users.disable"
	CapabilityManagementUsersEnable    = "management.users.enable"
	CapabilityManagementSystemLogsRead = "management.system_logs.read"
	CapabilityAccountDeleteSelf        = "account.delete_self"
)

type ViewerAuthorization struct {
	Role         string   `json:"role"`
	Capabilities []string `json:"capabilities"`
}

type ViewerOptions struct {
	AllowSelfDelete bool
}

type UserAction string

const (
	UserActionUpdate  UserAction = "update"
	UserActionDisable UserAction = "disable"
	UserActionEnable  UserAction = "enable"
)

type UserTarget struct {
	UserID int64
	Role   string
	Status string
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

var roleCapabilities = map[string][]string{
	identity.RoleOwner: {
		CapabilitySystemSettingsRead,
		CapabilitySystemSettingsUpdate,
		CapabilityManagementUsersRead,
		CapabilityManagementUsersCreate,
		CapabilityManagementUsersUpdate,
		CapabilityManagementUsersDisable,
		CapabilityManagementUsersEnable,
		CapabilityManagementSystemLogsRead,
	},
	identity.RoleUser: {},
}

func (s *Service) ViewerAuthorization(role string, options ViewerOptions) ViewerAuthorization {
	capabilities := s.CapabilitiesForRole(role)
	if options.AllowSelfDelete && !slices.Contains(capabilities, CapabilityAccountDeleteSelf) {
		capabilities = append(capabilities, CapabilityAccountDeleteSelf)
		sort.Strings(capabilities)
	}

	return ViewerAuthorization{
		Role:         role,
		Capabilities: capabilities,
	}
}

func (s *Service) CapabilitiesForRole(role string) []string {
	capabilities := append([]string(nil), roleCapabilities[role]...)
	sort.Strings(capabilities)
	if len(capabilities) == 0 {
		return []string{}
	}
	return capabilities
}

func (s *Service) HasCapability(capabilities []string, capability string) bool {
	return slices.Contains(capabilities, capability)
}

func (s *Service) CanManageUser(actor UserTarget, target UserTarget, action UserAction) bool {
	if actor.UserID == target.UserID {
		return false
	}
	if target.Role == "" {
		return false
	}
	if actor.Role != identity.RoleOwner {
		return false
	}
	if target.Role == identity.RoleOwner {
		return false
	}

	return action == UserActionUpdate || action == UserActionDisable || action == UserActionEnable
}

func (s *Service) ManagedUserActions(actor UserTarget, target UserTarget) []string {
	actions := make([]string, 0, 2)
	if s.CanManageUser(actor, target, UserActionUpdate) {
		actions = append(actions, string(UserActionUpdate))
	}
	switch target.Status {
	case identity.StatusActive:
		if s.CanManageUser(actor, target, UserActionDisable) {
			actions = append(actions, string(UserActionDisable))
		}
	case identity.StatusDisabled:
		if s.CanManageUser(actor, target, UserActionEnable) {
			actions = append(actions, string(UserActionEnable))
		}
	}
	return actions
}
