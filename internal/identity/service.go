package identity

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Roles and statuses are part of the identity model: authorization policies and
// storage layers both read them from here, so there is exactly one source.
const (
	RoleOwner = "owner"
	RoleUser  = "user"

	StatusActive   = "active"
	StatusDisabled = "disabled"
)

var (
	ErrUserNotFound       = errors.New("identity: user not found")
	ErrUsernameTaken      = errors.New("identity: username already exists")
	ErrEmailTaken         = errors.New("identity: email already exists")
	ErrOwnerAlreadyExists = errors.New("identity: owner already exists")
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
	Username         string
	Email            *string
	PasswordHash     string
	Role             string
	AssignedByUserID *int64
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
	UpdateManagedUser(ctx context.Context, userID int64, params UpdateManagedUserParams) (User, error)
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
		return User{}, fmt.Errorf("identity: username is required")
	}
	params.Role = normalizeRole(params.Role)
	if params.Role == "" {
		return User{}, fmt.Errorf("identity: role is required")
	}
	params.PasswordHash = strings.TrimSpace(params.PasswordHash)
	if params.PasswordHash == "" {
		return User{}, fmt.Errorf("identity: password hash is required")
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
	repository, err := s.requireRepository()
	if err != nil {
		return User{}, err
	}
	return repository.UpdateProfile(ctx, userID, normalizeUsername(username), normalizeEmailPointer(email))
}

func (s *Service) UpdateAvatar(ctx context.Context, userID int64, avatarPath *string) (User, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return User{}, err
	}
	return repository.UpdateAvatar(ctx, userID, avatarPath)
}

func (s *Service) UpdateManagedUser(ctx context.Context, userID int64, params UpdateManagedUserParams) (User, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return User{}, err
	}
	if _, lookupErr := repository.GetUserByID(ctx, userID); lookupErr != nil {
		return User{}, lookupErr
	}

	params.Username = normalizeUsername(params.Username)
	params.Email = normalizeEmailPointer(params.Email)
	return repository.UpdateManagedUser(ctx, userID, params)
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

func normalizeStatus(status string) string {
	switch strings.TrimSpace(status) {
	case StatusActive:
		return StatusActive
	case StatusDisabled:
		return StatusDisabled
	default:
		return ""
	}
}

func normalizeRole(role string) string {
	switch strings.TrimSpace(role) {
	case RoleOwner:
		return RoleOwner
	case RoleUser:
		return RoleUser
	default:
		return ""
	}
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

func AvatarDir(dataDir string) string {
	return filepath.Join(dataDir, "avatars")
}
