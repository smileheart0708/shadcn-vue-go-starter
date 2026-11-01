package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"main/internal/accountpolicies"
	"main/internal/authorization"
	"main/internal/identity"
)

const (
	defaultAccessTTL      = 10 * time.Minute
	defaultRefreshIdleTTL = 7 * 24 * time.Hour
	defaultRefreshAbsTTL  = 30 * 24 * time.Hour
)

var (
	ErrMissingBearerToken     = errors.New("auth: missing bearer token")
	ErrInvalidToken           = errors.New("auth: invalid token")
	ErrInvalidCredentials     = errors.New("auth: invalid credentials")
	ErrAccountDisabled        = errors.New("auth: account disabled")
	ErrInvalidRefreshToken    = errors.New("auth: invalid refresh token")
	ErrRefreshTokenExpired    = errors.New("auth: refresh token expired")
	ErrRefreshTokenRevoked    = errors.New("auth: refresh token revoked")
	ErrTokenReuseDetected     = errors.New("auth: refresh token reuse detected")
	ErrRegistrationDisabled   = errors.New("auth: registration is disabled")
	ErrPasswordTooShort       = errors.New("auth: password too short")
	ErrCurrentPasswordInvalid = errors.New("auth: current password invalid")
	ErrSessionNotFound        = errors.New("auth: session not found")
)

// IsSessionInvalidError reports whether err describes a refresh token the client has
// to throw away, as opposed to an infrastructure failure.
//
// Handlers use it to decide between "clear the cookie and answer 401" and "answer 500".
// Folding every failure into 401 (the previous behaviour) discarded the cookie on
// database errors, so any new session error must be added here deliberately.
func IsSessionInvalidError(err error) bool {
	return errors.Is(err, ErrInvalidRefreshToken) ||
		errors.Is(err, ErrTokenReuseDetected) ||
		errors.Is(err, ErrRefreshTokenRevoked) ||
		errors.Is(err, ErrRefreshTokenExpired) ||
		errors.Is(err, ErrAccountDisabled)
}

type Options struct {
	Secret             []byte
	TTL                time.Duration
	RefreshIdleTTL     time.Duration
	RefreshAbsoluteTTL time.Duration
	RefreshCookieName  string
}

type Service struct {
	secret             []byte
	ttl                time.Duration
	refreshIdleTTL     time.Duration
	refreshAbsoluteTTL time.Duration
	refreshCookieName  string
	sessions           SessionRepository
	identities         *identity.Service
	authorization      *authorization.Service
	policies           *accountpolicies.Service
}

type Principal struct {
	UserID          int64
	SessionID       string
	SecurityVersion int64
}

type Actor struct {
	Principal
	User          identity.User
	Authorization authorization.ViewerAuthorization
}

type Viewer struct {
	User          identity.User
	Authorization authorization.ViewerAuthorization
}

type SessionResult struct {
	Viewer       Viewer
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string
	SessionID    string
}

type RegisterParams struct {
	Username string
	Email    *string
	Password string
}

type refreshClaims struct {
	SessionID       string `json:"sid"`
	SecurityVersion int64  `json:"sv"`
	jwt.RegisteredClaims
}

type Session struct {
	ID               string
	UserID           int64
	RefreshTokenHash string
	CreatedAt        time.Time
	LastUsedAt       time.Time
	LastRotatedAt    *time.Time
	ExpiresAt        time.Time
	IdleExpiresAt    time.Time
	RevokedAt        *time.Time
	RevokeReason     *string
	IP               *string
	UserAgent        *string
}

// SessionRepository is the persistence seam for refresh-token state. Its
// mutation methods own their transaction so token safety is not driver-specific.
type SessionRepository interface {
	CreateSession(ctx context.Context, session Session) error
	GetSession(ctx context.Context, sessionID string) (Session, error)
	RotateSession(ctx context.Context, sessionID string, expectedHash string, nextHash string, lastUsedAt time.Time, idleExpiresAt time.Time) (bool, error)
	RevokeSession(ctx context.Context, sessionID string, reason string, revokedAt time.Time) error
	UpdatePasswordAndRevokeSessions(ctx context.Context, userID int64, passwordHash string, changedAt time.Time) error
}

func NewService(options Options, sessions SessionRepository, identities *identity.Service, authorizationService *authorization.Service, policies *accountpolicies.Service) *Service {
	if options.TTL <= 0 {
		options.TTL = defaultAccessTTL
	}
	if options.RefreshIdleTTL <= 0 {
		options.RefreshIdleTTL = defaultRefreshIdleTTL
	}
	if options.RefreshAbsoluteTTL <= 0 {
		options.RefreshAbsoluteTTL = defaultRefreshAbsTTL
	}

	return &Service{
		secret:             options.Secret,
		ttl:                options.TTL,
		refreshIdleTTL:     options.RefreshIdleTTL,
		refreshAbsoluteTTL: options.RefreshAbsoluteTTL,
		refreshCookieName:  ResolveRefreshCookieName(options),
		sessions:           sessions,
		identities:         identities,
		authorization:      authorizationService,
		policies:           policies,
	}
}

func (s *Service) AuthenticateRequest(ctx context.Context, header string) (Actor, error) {
	token, err := BearerTokenFromHeader(header)
	if err != nil {
		return Actor{}, err
	}

	principal, err := s.ParseAccessToken(token)
	if err != nil {
		return Actor{}, err
	}

	return s.actorForPrincipal(ctx, principal)
}

func (s *Service) BuildViewer(ctx context.Context, userID int64) (Viewer, error) {
	user, err := s.identities.GetUserByID(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}

	viewerAuthorization, err := s.buildViewerAuthorization(ctx, user.Role)
	if err != nil {
		return Viewer{}, err
	}

	return Viewer{
		User:          user,
		Authorization: viewerAuthorization,
	}, nil
}

func (s *Service) Login(ctx context.Context, identifier string, password string, ip string, userAgent string) (SessionResult, error) {
	record, err := s.identities.GetAuthRecordByIdentifier(ctx, identifier)
	if errors.Is(err, identity.ErrUserNotFound) {
		return SessionResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return SessionResult{}, err
	}

	passwordMatches, err := VerifyPassword(password, record.PasswordHash)
	if err != nil {
		return SessionResult{}, fmt.Errorf("auth: verify password: %w", err)
	}
	if !passwordMatches {
		return SessionResult{}, ErrInvalidCredentials
	}
	if record.Status != identity.StatusActive {
		return SessionResult{}, ErrAccountDisabled
	}

	result, err := s.IssueSessionForUser(ctx, record.User, ip, userAgent)
	if err != nil {
		return SessionResult{}, err
	}
	return result, nil
}

func (s *Service) Register(ctx context.Context, params RegisterParams, ip string, userAgent string) (SessionResult, error) {
	policies, err := s.policies.Get(ctx)
	if err != nil {
		return SessionResult{}, err
	}
	if !policies.PublicRegistrationEnabled {
		return SessionResult{}, ErrRegistrationDisabled
	}

	if len(strings.TrimSpace(params.Password)) < MinPasswordLength {
		return SessionResult{}, ErrPasswordTooShort
	}

	passwordHash, err := HashPassword(params.Password)
	if err != nil {
		return SessionResult{}, err
	}

	user, err := s.identities.CreateUser(ctx, identity.CreateUserParams{
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: passwordHash,
		Role:         identity.RoleUser,
	})
	if err != nil {
		return SessionResult{}, err
	}

	result, err := s.IssueSessionForUser(ctx, user, ip, userAgent)
	if err != nil {
		return SessionResult{}, err
	}
	return result, nil
}

// CreateManagedUserParams is the administrative counterpart of registration.
type CreateManagedUserParams struct {
	Username string
	Email    *string
	Password string
}

// CreateManagedUser creates an account on behalf of an administrator.
//
// Username is rejected before the password is hashed: hashing is deliberately expensive,
// and an empty username must keep answering CodeUsernameRequired (the registration handler
// checks it first too) instead of turning into a password error when both fields are bad.
// Accounts created this way are always plain users — promoting someone is a separate
// decision with its own authorization rule, never a side effect of creating the account.
func (s *Service) CreateManagedUser(ctx context.Context, params CreateManagedUserParams) (identity.User, error) {
	if strings.TrimSpace(params.Username) == "" {
		return identity.User{}, identity.ErrUsernameRequired
	}
	if len(strings.TrimSpace(params.Password)) < MinPasswordLength {
		return identity.User{}, ErrPasswordTooShort
	}

	passwordHash, err := HashPassword(params.Password)
	if err != nil {
		return identity.User{}, err
	}

	return s.identities.CreateUser(ctx, identity.CreateUserParams{
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: passwordHash,
		Role:         identity.RoleUser,
	})
}

func (s *Service) Refresh(ctx context.Context, rawToken string) (SessionResult, error) {
	sessionID, _, err := ParseRefreshToken(rawToken)
	if err != nil {
		return SessionResult{}, ErrInvalidRefreshToken
	}

	session, err := s.getSessionByID(ctx, sessionID)
	if errors.Is(err, ErrSessionNotFound) {
		return SessionResult{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return SessionResult{}, err
	}

	expectedHash := hashRefreshToken(rawToken)
	if subtle.ConstantTimeCompare([]byte(session.RefreshTokenHash), []byte(expectedHash)) != 1 {
		if revokeErr := s.revokeSessionByID(ctx, session.ID, "token_reuse_detected"); revokeErr != nil {
			return SessionResult{}, revokeErr
		}
		return SessionResult{}, ErrTokenReuseDetected
	}

	now := time.Now().UTC()
	switch {
	case session.RevokedAt != nil:
		return SessionResult{}, ErrRefreshTokenRevoked
	case now.After(session.ExpiresAt):
		if revokeErr := s.revokeSessionByID(ctx, session.ID, "refresh_expired"); revokeErr != nil {
			return SessionResult{}, revokeErr
		}
		return SessionResult{}, ErrRefreshTokenExpired
	case now.After(session.IdleExpiresAt):
		if revokeErr := s.revokeSessionByID(ctx, session.ID, "refresh_idle_expired"); revokeErr != nil {
			return SessionResult{}, revokeErr
		}
		return SessionResult{}, ErrRefreshTokenExpired
	}

	user, err := s.identities.GetUserByID(ctx, session.UserID)
	if errors.Is(err, identity.ErrUserNotFound) {
		if revokeErr := s.revokeSessionByID(ctx, session.ID, "user_not_found"); revokeErr != nil {
			return SessionResult{}, revokeErr
		}
		return SessionResult{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return SessionResult{}, err
	}
	if user.Status != identity.StatusActive {
		if revokeErr := s.revokeSessionByID(ctx, session.ID, "account_disabled"); revokeErr != nil {
			return SessionResult{}, revokeErr
		}
		return SessionResult{}, ErrAccountDisabled
	}

	nextSecret, err := randomTokenComponent(32)
	if err != nil {
		return SessionResult{}, err
	}
	nextRefreshToken := composeRefreshToken(session.ID, nextSecret)
	nextTokenHash := hashRefreshToken(nextRefreshToken)
	nextIdleExpiresAt := now.Add(s.refreshIdleTTL)

	rotated, err := s.rotateSession(ctx, session.ID, expectedHash, nextTokenHash, now, nextIdleExpiresAt)
	if err != nil {
		return SessionResult{}, err
	}
	if !rotated {
		if revokeErr := s.revokeSessionByID(ctx, session.ID, "token_reuse_detected"); revokeErr != nil {
			return SessionResult{}, revokeErr
		}
		return SessionResult{}, ErrTokenReuseDetected
	}

	accessToken, expiresAt, err := s.issueAccessToken(user, session.ID)
	if err != nil {
		return SessionResult{}, s.revokeSessionAfterAccessTokenIssueFailure(ctx, session.ID, err)
	}

	viewerAuthorization, err := s.buildViewerAuthorization(ctx, user.Role)
	if err != nil {
		return SessionResult{}, err
	}
	return SessionResult{
		Viewer: Viewer{
			User:          user,
			Authorization: viewerAuthorization,
		},
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
		RefreshToken: nextRefreshToken,
		SessionID:    session.ID,
	}, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	sessionID, _, err := ParseRefreshToken(rawToken)
	if err != nil {
		return err
	}

	_, err = s.getSessionByID(ctx, sessionID)
	if err != nil && !errors.Is(err, ErrSessionNotFound) {
		return err
	}

	if revokeErr := s.revokeSessionByID(ctx, sessionID, "logout"); revokeErr != nil {
		return revokeErr
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, actor Actor, currentPassword string, newPassword string) error {
	if len(strings.TrimSpace(newPassword)) < MinPasswordLength {
		return ErrPasswordTooShort
	}

	record, err := s.identities.GetAuthRecordByID(ctx, actor.User.ID)
	if err != nil {
		return err
	}

	passwordMatches, err := VerifyPassword(currentPassword, record.PasswordHash)
	if err != nil {
		return fmt.Errorf("auth: verify current password: %w", err)
	}
	if !passwordMatches {
		return ErrCurrentPasswordInvalid
	}

	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	sessions, err := s.requireSessions()
	if err != nil {
		return err
	}
	if updateErr := sessions.UpdatePasswordAndRevokeSessions(ctx, actor.User.ID, passwordHash, time.Now().UTC()); updateErr != nil {
		return fmt.Errorf("auth: change password: %w", updateErr)
	}
	return nil
}

func (s *Service) IssueSessionForUser(ctx context.Context, user identity.User, ip string, userAgent string) (SessionResult, error) {
	if user.Status != identity.StatusActive {
		return SessionResult{}, ErrAccountDisabled
	}

	now := time.Now().UTC()
	sessionID, err := randomTokenComponent(16)
	if err != nil {
		return SessionResult{}, err
	}
	sessionSecret, err := randomTokenComponent(32)
	if err != nil {
		return SessionResult{}, err
	}

	refreshToken := composeRefreshToken(sessionID, sessionSecret)
	sessions, err := s.requireSessions()
	if err != nil {
		return SessionResult{}, err
	}
	if createErr := sessions.CreateSession(ctx, Session{
		ID:               sessionID,
		UserID:           user.ID,
		RefreshTokenHash: hashRefreshToken(refreshToken),
		CreatedAt:        now,
		LastUsedAt:       now,
		ExpiresAt:        now.Add(s.refreshAbsoluteTTL),
		IdleExpiresAt:    now.Add(s.refreshIdleTTL),
		IP:               stringPointerOrNil(ip),
		UserAgent:        stringPointerOrNil(userAgent),
	}); createErr != nil {
		return SessionResult{}, fmt.Errorf("auth: create refresh session: %w", createErr)
	}

	accessToken, expiresAt, err := s.issueAccessToken(user, sessionID)
	if err != nil {
		return SessionResult{}, s.revokeSessionAfterAccessTokenIssueFailure(ctx, sessionID, err)
	}

	viewerAuthorization, err := s.buildViewerAuthorization(ctx, user.Role)
	if err != nil {
		return SessionResult{}, err
	}

	return SessionResult{
		Viewer: Viewer{
			User:          user,
			Authorization: viewerAuthorization,
		},
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
		RefreshToken: refreshToken,
		SessionID:    sessionID,
	}, nil
}

func (s *Service) ParseAccessToken(token string) (Principal, error) {
	parsed, err := jwt.ParseWithClaims(token, &refreshClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return Principal{}, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*refreshClaims)
	if !ok {
		return Principal{}, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 || claims.SecurityVersion <= 0 || strings.TrimSpace(claims.SessionID) == "" {
		return Principal{}, ErrInvalidToken
	}

	return Principal{
		UserID:          userID,
		SessionID:       claims.SessionID,
		SecurityVersion: claims.SecurityVersion,
	}, nil
}

func (s *Service) actorForPrincipal(ctx context.Context, principal Principal) (Actor, error) {
	user, err := s.identities.GetUserByID(ctx, principal.UserID)
	if err != nil {
		return Actor{}, ErrInvalidToken
	}
	if user.SecurityVersion != principal.SecurityVersion || user.Status != identity.StatusActive {
		return Actor{}, ErrInvalidToken
	}

	session, err := s.getSessionByID(ctx, principal.SessionID)
	if err != nil {
		return Actor{}, ErrInvalidToken
	}

	now := time.Now().UTC()
	switch {
	case session.UserID != principal.UserID:
		return Actor{}, ErrInvalidToken
	case session.RevokedAt != nil:
		return Actor{}, ErrInvalidToken
	case now.After(session.ExpiresAt):
		return Actor{}, ErrInvalidToken
	case now.After(session.IdleExpiresAt):
		return Actor{}, ErrInvalidToken
	}

	viewerAuthorization, err := s.buildViewerAuthorization(ctx, user.Role)
	if err != nil {
		return Actor{}, err
	}

	return Actor{
		Principal:     principal,
		User:          user,
		Authorization: viewerAuthorization,
	}, nil
}

func (s *Service) buildViewerAuthorization(ctx context.Context, role string) (authorization.ViewerAuthorization, error) {
	policies, err := s.policies.Get(ctx)
	if err != nil {
		return authorization.ViewerAuthorization{}, err
	}

	// An owner must not delete themselves: that is how an installation loses its last
	// administrator. Rank decides who counts as top-level, so a role inserted above
	// owner inherits the restriction without touching this line.
	allowSelfDelete := policies.SelfServiceAccountDeletionEnabled && !identity.IsHighestRole(role)

	return s.authorization.ViewerAuthorization(role, authorization.ViewerOptions{
		AllowSelfDelete: allowSelfDelete,
	}), nil
}

func (s *Service) issueAccessToken(user identity.User, sessionID string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.ttl)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims{
		SessionID:       sessionID,
		SecurityVersion: user.SecurityVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	encoded, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign access token: %w", err)
	}
	return encoded, expiresAt, nil
}

func (s *Service) getSessionByID(ctx context.Context, sessionID string) (Session, error) {
	sessions, err := s.requireSessions()
	if err != nil {
		return Session{}, err
	}
	return sessions.GetSession(ctx, sessionID)
}

func (s *Service) rotateSession(ctx context.Context, sessionID string, expectedHash string, nextHash string, now time.Time, idleExpiresAt time.Time) (bool, error) {
	sessions, err := s.requireSessions()
	if err != nil {
		return false, err
	}
	rotated, err := sessions.RotateSession(ctx, sessionID, expectedHash, nextHash, now, idleExpiresAt)
	if err != nil {
		return false, fmt.Errorf("auth: rotate refresh session: %w", err)
	}
	return rotated, nil
}

func (s *Service) revokeSessionByID(ctx context.Context, sessionID string, reason string) error {
	sessions, err := s.requireSessions()
	if err != nil {
		return err
	}
	if revokeErr := sessions.RevokeSession(ctx, sessionID, strings.TrimSpace(reason), time.Now().UTC()); revokeErr != nil {
		return fmt.Errorf("auth: revoke session: %w", revokeErr)
	}
	return nil
}

func (s *Service) requireSessions() (SessionRepository, error) {
	if s == nil || s.sessions == nil {
		return nil, errors.New("auth: nil service")
	}
	return s.sessions, nil
}

func (s *Service) revokeSessionAfterAccessTokenIssueFailure(ctx context.Context, sessionID string, issueErr error) error {
	if revokeErr := s.revokeSessionByID(ctx, sessionID, "access_token_issue_failed"); revokeErr != nil {
		return errors.Join(issueErr, fmt.Errorf("auth: revoke session after access token issue failure: %w", revokeErr))
	}
	return issueErr
}

func stringPointerOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return new(value)
}
