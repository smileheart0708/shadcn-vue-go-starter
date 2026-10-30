package authorization

import "slices"

// ViewerAuthorization is the domain view of what a role may do right now.
//
// It deliberately carries no JSON tags: transport shapes belong to httpapi, so
// renaming a response field never reaches into the permission model.
type ViewerAuthorization struct {
	Role         string
	Capabilities []Capability
}

// ViewerOptions carries request-time policy inputs that are not part of the
// static role matrix.
type ViewerOptions struct {
	AllowSelfDelete bool
}

// Service is the only way to turn a role into capabilities. It is stateless and
// side-effect free so permission changes take effect on the next request without
// cache invalidation.
type Service struct{}

func NewService() *Service {
	return &Service{}
}

// ViewerAuthorization resolves the capabilities granted to role plus any
// capability enabled by request-time policy.
func (s *Service) ViewerAuthorization(role string, options ViewerOptions) ViewerAuthorization {
	capabilities := s.CapabilitiesForRole(role)
	if options.AllowSelfDelete && !slices.Contains(capabilities, CapabilityAccountDeleteSelf) {
		capabilities = append(capabilities, CapabilityAccountDeleteSelf)
		slices.Sort(capabilities)
	}

	return ViewerAuthorization{
		Role:         role,
		Capabilities: capabilities,
	}
}

// CapabilitiesForRole returns a sorted copy of the role matrix entry. Unknown
// roles yield no capabilities, which keeps the model fail-closed.
func (s *Service) CapabilitiesForRole(role string) []Capability {
	capabilities := slices.Clone(roleCapabilities[role])
	slices.Sort(capabilities)
	return capabilities
}

func (s *Service) HasCapability(capabilities []Capability, capability Capability) bool {
	return slices.Contains(capabilities, capability)
}
