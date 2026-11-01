package identity

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Statuses are part of the identity model: authorization policies and storage
// layers both read them from here, so there is exactly one source. Roles live in
// role.go together with their rank.
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

// statusOrder is the authoritative list of every status the API carries, in the order
// the generated frontend enum follows.
//
// It is read by normalizeStatus as well as by the contract generator, so one entry makes a
// status acceptable to the service and visible to the frontend at once. Go cannot reflect
// constants, so the const block above and this list still have to be edited together — a
// constant that never reaches this list fails closed (the service rejects the value)
// instead of being accepted and then dropped downstream.
var statusOrder = []string{
	StatusActive,
	StatusDisabled,
}

// Statuses returns every declared status in declaration order.
//
// Exported for the frontend contract generator; no production code calls it.
func Statuses() []string {
	return slices.Clone(statusOrder)
}

var (
	ErrUserNotFound       = errors.New("identity: user not found")
	ErrUsernameTaken      = errors.New("identity: username already exists")
	ErrEmailTaken         = errors.New("identity: email already exists")
	ErrOwnerAlreadyExists = errors.New("identity: owner already exists")

	// ErrUsernameRequired is the only validation error the transport layer maps; it has
	// to stay exported so handlers answer CodeUsernameRequired instead of an opaque 500.
	ErrUsernameRequired = errors.New("identity: username is required")

	// Internal invariants: callers always supply a known role and a non-empty hash, so
	// these never reach the transport layer and stay unexported.
	errRoleRequired         = errors.New("identity: role is required")
	errPasswordHashRequired = errors.New("identity: password hash is required")
)

type User struct {
	ID              int64
	Username        string
	Email           *string
	AvatarPath      *string
	Status          string
	Role            string
	SecurityVersion int64
	DisabledAt      *time.Time
	LastActiveAt    *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type AuthRecord struct {
	User
	PasswordHash      string
	PasswordChangedAt *time.Time
}

type CreateUserParams struct {
	Username     string
	Email        *string
	PasswordHash string
	Role         string
}

type UpdateManagedUserParams struct {
	Username string
	Email    *string
}

type ListUsersParams struct {
	Query    string
	Status   string
	Page     int
	PageSize int
}

type ListUsersResult struct {
	Items    []User
	Page     int
	PageSize int
	Total    int
}

// Repository is the persistence seam for identity data. Implementations own
// SQL dialects, constraint decoding, and transactions required by each method.
type Repository interface {
	CreateUser(ctx context.Context, params CreateUserParams) (User, error)
	GetUserByID(ctx context.Context, userID int64) (User, error)
	GetAuthRecordByID(ctx context.Context, userID int64) (AuthRecord, error)
	GetAuthRecordByIdentifier(ctx context.Context, identifier string) (AuthRecord, error)
	UpdateProfile(ctx context.Context, userID int64, username string, email *string) (User, error)
	UpdateAvatar(ctx context.Context, userID int64, avatarPath *string) (User, error)
	SetUserStatus(ctx context.Context, userID int64, status string) (User, error)
	DeleteUser(ctx context.Context, userID int64) error
	ListUsers(ctx context.Context, params ListUsersParams) (ListUsersResult, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) CreateUser(ctx context.Context, params CreateUserParams) (User, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return User{}, err
	}

	params.Username = normalizeUsername(params.Username)
	if params.Username == "" {
		return User{}, ErrUsernameRequired
	}
	params.Role = normalizeRole(params.Role)
	if params.Role == "" {
		return User{}, errRoleRequired
	}
	params.PasswordHash = strings.TrimSpace(params.PasswordHash)
	if params.PasswordHash == "" {
		return User{}, errPasswordHashRequired
	}
	params.Email = normalizeEmailPointer(params.Email)

	return repository.CreateUser(ctx, params)
}

func (s *Service) GetUserByID(ctx context.Context, userID int64) (User, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return User{}, err
	}
	return repository.GetUserByID(ctx, userID)
}

func (s *Service) GetAuthRecordByID(ctx context.Context, userID int64) (AuthRecord, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return AuthRecord{}, err
	}
	return repository.GetAuthRecordByID(ctx, userID)
}

func (s *Service) GetAuthRecordByIdentifier(ctx context.Context, identifier string) (AuthRecord, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return AuthRecord{}, err
	}

	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return AuthRecord{}, ErrUserNotFound
	}
	return repository.GetAuthRecordByIdentifier(ctx, identifier)
}

func (s *Service) UpdateProfile(ctx context.Context, userID int64, username string, email *string) (User, error) {
	normalizedUsername := normalizeUsername(username)
	if normalizedUsername == "" {
		return User{}, ErrUsernameRequired
	}

	repository, err := s.requireRepository()
	if err != nil {
		return User{}, err
	}
	return repository.UpdateProfile(ctx, userID, normalizedUsername, normalizeEmailPointer(email))
}

func (s *Service) UpdateAvatar(ctx context.Context, userID int64, avatarPath *string) (User, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return User{}, err
	}
	return repository.UpdateAvatar(ctx, userID, avatarPath)
}

func (s *Service) UpdateManagedUser(ctx context.Context, userID int64, params UpdateManagedUserParams) (User, error) {
	// Shares UpdateProfile because both paths write the same two columns, and that method
	// normalizes them, rejects a blank username and reports ErrUserNotFound without a
	// separate lookup.
	return s.UpdateProfile(ctx, userID, params.Username, params.Email)
}

func (s *Service) SetUserStatus(ctx context.Context, userID int64, status string) (User, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return User{}, err
	}

	current, err := repository.GetUserByID(ctx, userID)
	if err != nil {
		return User{}, err
	}

	status = normalizeStatus(status)
	if status == "" {
		return User{}, fmt.Errorf("identity: invalid status")
	}
	if current.Status == status {
		return current, nil
	}
	return repository.SetUserStatus(ctx, userID, status)
}

func (s *Service) DeleteUser(ctx context.Context, userID int64) error {
	repository, err := s.requireRepository()
	if err != nil {
		return err
	}
	if _, lookupErr := repository.GetUserByID(ctx, userID); lookupErr != nil {
		return lookupErr
	}
	return repository.DeleteUser(ctx, userID)
}

func (s *Service) ListUsers(ctx context.Context, params ListUsersParams) (ListUsersResult, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return ListUsersResult{}, err
	}
	return repository.ListUsers(ctx, normalizeListParams(params))
}

func (s *Service) requireRepository() (Repository, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("identity: nil service")
	}
	return s.repository, nil
}

func normalizeListParams(params ListUsersParams) ListUsersParams {
	params.Query = strings.TrimSpace(params.Query)
	params.Status = normalizeStatus(params.Status)
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	return params
}

// normalizeStatus returns the canonical spelling of status, or "" when statusOrder does
// not declare it.
//
// It scans the catalog instead of repeating a switch: the accepted set and the generated
// frontend enum then have exactly one source, which is the same shape as RoleRank reading
// roleOrder. "" is what the repository reads as "no filter", so an unrecognised filter is
// dropped rather than matched.
func normalizeStatus(status string) string {
	trimmed := strings.TrimSpace(status)
	for _, candidate := range statusOrder {
		if candidate == trimmed {
			return candidate
		}
	}

	return ""
}

func normalizeUsername(value string) string {
	return strings.TrimSpace(value)
}

func normalizeEmailPointer(email *string) *string {
	if email == nil {
		return nil
	}

	normalized := strings.ToLower(strings.TrimSpace(*email))
	if normalized == "" {
		return nil
	}
	return &normalized
}
