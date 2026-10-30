package setup

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"main/internal/auth"
	"main/internal/identity"
)

var ErrAlreadyCompleted = errors.New("setup: install already completed")

type State struct {
	SetupCompleted   bool       `json:"setupCompleted"`
	OwnerUserID      *int64     `json:"ownerUserId"`
	SetupCompletedAt *time.Time `json:"setupCompletedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type CompleteSetupInput struct {
	Username string
	Password string
}

type InitialOwner struct {
	Username     string
	PasswordHash string
}

// Repository owns the transaction that creates the initial owner and marks
// installation complete. The service must never coordinate those writes itself.
type Repository interface {
	GetState(ctx context.Context) (State, error)
	CompleteInitialOwner(ctx context.Context, owner InitialOwner) (identity.User, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetState(ctx context.Context) (State, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return State{}, err
	}
	return repository.GetState(ctx)
}

func (s *Service) Complete(ctx context.Context, input CompleteSetupInput) (identity.User, error) {
	repository, err := s.requireRepository()
	if err != nil {
		return identity.User{}, err
	}

	username := strings.TrimSpace(input.Username)
	if username == "" {
		return identity.User{}, fmt.Errorf("setup: username is required")
	}
	password := strings.TrimSpace(input.Password)
	if len(password) < auth.MinPasswordLength {
		return identity.User{}, auth.ErrPasswordTooShort
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return identity.User{}, fmt.Errorf("setup: hash owner password: %w", err)
	}
	return repository.CompleteInitialOwner(ctx, InitialOwner{
		Username:     username,
		PasswordHash: passwordHash,
	})
}

func (s *Service) requireRepository() (Repository, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("setup: nil service")
	}
	return s.repository, nil
}
