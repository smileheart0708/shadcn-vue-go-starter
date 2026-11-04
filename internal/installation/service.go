// Package installation coordinates the one-time database choice and owner
// creation. It owns the candidate database lifecycle until setup publishes the
// complete application runtime.
package installation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"main/internal/database"
	"main/internal/database/dialect"
	"main/internal/setup"
)

type Phase string

const (
	PhaseDatabaseRequired     Phase = "database_required"
	PhaseFinalizationRequired Phase = "finalization_required"
	PhaseCompleted            Phase = "completed"
)

var (
	ErrDatabaseInputRequired     = errors.New("installation: database selection is required")
	ErrDatabaseAlreadyConfigured = errors.New("installation: database is already configured")
	ErrDatabaseConfigInvalid     = errors.New("installation: database configuration is invalid")
	ErrDatabaseConnection        = errors.New("installation: database connection failed")
	ErrDatabaseNotEmpty          = errors.New("installation: database is not an empty application database")
	ErrDatabaseMigration         = errors.New("installation: database migration failed")
)

type State struct {
	Phase          Phase
	DatabaseDriver string
	OwnerUserID    *int64
	CompletedAt    *time.Time
}

type PostgreSQLConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string
}

type DatabaseInput struct {
	Driver     string
	PostgreSQL *PostgreSQLConfig
}

type CompleteInput struct {
	Username string
	Password string
	Database *DatabaseInput
}

type SelectionRepository interface {
	Load() (database.Selection, bool, error)
	Save(database.Selection) error
}

type Options struct {
	DataDir        string
	SelectionStore SelectionRepository
	Publish        func(*database.Runtime)
}

type Service struct {
	mu sync.Mutex

	dataDir        string
	selectionStore SelectionRepository
	publish        func(*database.Runtime)

	runtime     *database.Runtime
	setup       *setup.Service
	selection   database.Selection
	phase       Phase
	ownerUserID *int64
	completedAt *time.Time
}

func New(ctx context.Context, options Options) (*Service, error) {
	if strings.TrimSpace(options.DataDir) == "" {
		return nil, errors.New("installation: data directory is required")
	}
	if options.SelectionStore == nil {
		return nil, errors.New("installation: selection store is required")
	}
	service := &Service{
		dataDir:        options.DataDir,
		selectionStore: options.SelectionStore,
		publish:        options.Publish,
	}
	selection, configured, err := options.SelectionStore.Load()
	if err != nil {
		return nil, fmt.Errorf("installation: load database selection: %w", err)
	}
	if !configured {
		selection, configured, err = database.DetectDefaultSQLite(ctx, options.DataDir)
		if err != nil {
			return nil, fmt.Errorf("installation: detect default SQLite database: %w", err)
		}
	}
	if !configured {
		service.phase = PhaseDatabaseRequired
		return service, nil
	}

	runtime, err := database.Open(ctx, database.Config(selection))
	if err != nil {
		return nil, fmt.Errorf("installation: open selected database: %w", err)
	}
	service.activate(runtime, selection, setup.NewService(runtime.Store))

	state, err := service.setup.GetState(ctx)
	if err != nil {
		service.closeRuntime()
		return nil, fmt.Errorf("installation: read setup state: %w", err)
	}
	service.ownerUserID = state.OwnerUserID
	service.completedAt = state.SetupCompletedAt
	if state.SetupCompleted {
		service.phase = PhaseCompleted
	} else {
		service.phase = PhaseFinalizationRequired
	}
	return service, nil
}

func (s *Service) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.publicState()
}

func (s *Service) Runtime() *database.Runtime {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.runtime
}

func (s *Service) Complete(ctx context.Context, input CompleteInput) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.phase == PhaseCompleted {
		return State{}, setup.ErrAlreadyCompleted
	}
	username, password, err := setup.ValidateInitialOwner(input.Username, input.Password)
	if err != nil {
		return State{}, err
	}
	if configErr := s.configureDatabase(ctx, input.Database); configErr != nil {
		return State{}, configErr
	}

	owner, completeErr := s.setup.Complete(ctx, setup.CompleteSetupInput{
		Username: username,
		Password: password,
	})
	if completeErr != nil {
		return State{}, completeErr
	}

	completedAt := time.Now().UTC()
	s.ownerUserID = &owner.ID
	s.completedAt = &completedAt
	s.phase = PhaseCompleted
	// Publish before the follow-up read: once the owner transaction committed, a
	// transient read error must not leave the process in setup-only mode.
	if s.publish != nil {
		s.publish(s.runtime)
	}
	if state, stateErr := s.setup.GetState(ctx); stateErr == nil {
		s.ownerUserID = state.OwnerUserID
		s.completedAt = state.SetupCompletedAt
	}
	return s.publicState(), nil
}

func (s *Service) configureDatabase(ctx context.Context, input *DatabaseInput) error {
	if s.phase != PhaseDatabaseRequired {
		if input == nil {
			return nil
		}
		selection, err := s.selectionFromInput(*input)
		if err != nil {
			return err
		}
		// A client retrying after a failed owner transaction still carries the
		// selection it chose earlier; accept it instead of locking setup out.
		if selection != s.selection {
			return ErrDatabaseAlreadyConfigured
		}
		return nil
	}
	if input == nil {
		return ErrDatabaseInputRequired
	}

	selection, err := s.selectionFromInput(*input)
	if err != nil {
		return err
	}
	runtime, openErr := database.OpenForSetup(ctx, database.Config(selection))
	if openErr != nil {
		return mapSetupOpenError(openErr)
	}
	// SQLite is derived from DataDir and carries no credentials, so it needs no
	// pointer file; only an external database has to be remembered across starts.
	if selection.Driver != database.DriverSQLite {
		if saveErr := s.selectionStore.Save(selection); saveErr != nil {
			if closeErr := runtime.Close(); closeErr != nil {
				return errors.Join(fmt.Errorf("installation: save database selection: %w", saveErr), closeErr)
			}
			return fmt.Errorf("installation: save database selection: %w", saveErr)
		}
	}
	s.activate(runtime, selection, setup.NewService(runtime.Store))
	return nil
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeRuntimeLocked()
}

func (s *Service) activate(runtime *database.Runtime, selection database.Selection, setupService *setup.Service) {
	s.runtime = runtime
	s.selection = selection
	s.setup = setupService
	s.phase = PhaseFinalizationRequired
}

func (s *Service) publicState() State {
	state := State{
		Phase:       s.phase,
		OwnerUserID: s.ownerUserID,
		CompletedAt: s.completedAt,
	}
	if s.phase != PhaseDatabaseRequired {
		state.DatabaseDriver = s.selection.Driver
	}
	return state
}

func (s *Service) closeRuntime() {
	if err := s.closeRuntimeLocked(); err != nil {
		slog.Error("failed to close installation database", "error", err)
	}
}

func (s *Service) closeRuntimeLocked() error {
	if s.runtime == nil {
		return nil
	}
	runtime := s.runtime
	s.runtime = nil
	s.setup = nil
	return runtime.Close()
}

func (s *Service) selectionFromInput(input DatabaseInput) (database.Selection, error) {
	driver := strings.ToLower(strings.TrimSpace(input.Driver))
	switch driver {
	case database.DriverSQLite:
		return database.DefaultSQLiteSelection(s.dataDir), nil
	case database.DriverPostgreSQL:
		if input.PostgreSQL == nil {
			return database.Selection{}, ErrDatabaseConfigInvalid
		}
		dsn, err := postgreSQLDSN(*input.PostgreSQL)
		if err != nil {
			return database.Selection{}, err
		}
		return database.Selection{Driver: database.DriverPostgreSQL, DSN: dsn}, nil
	default:
		return database.Selection{}, fmt.Errorf("%w: unsupported driver %q", ErrDatabaseConfigInvalid, driver)
	}
}

func postgreSQLDSN(config PostgreSQLConfig) (string, error) {
	host := strings.TrimSpace(config.Host)
	databaseName := strings.TrimSpace(config.Database)
	username := strings.TrimSpace(config.Username)
	password := config.Password
	sslMode := strings.ToLower(strings.TrimSpace(config.SSLMode))

	if host == "" || databaseName == "" || username == "" || password == "" {
		return "", ErrDatabaseConfigInvalid
	}
	if config.Port < 1 || config.Port > 65535 {
		return "", ErrDatabaseConfigInvalid
	}
	switch sslMode {
	case "disable", "require", "verify-full":
	default:
		return "", ErrDatabaseConfigInvalid
	}

	connection := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(username, password),
		Host:   net.JoinHostPort(host, strconv.Itoa(config.Port)),
		Path:   "/" + databaseName,
	}
	query := connection.Query()
	query.Set("sslmode", sslMode)
	query.Set("connect_timeout", "5")
	connection.RawQuery = query.Encode()
	return connection.String(), nil
}

func mapSetupOpenError(err error) error {
	switch {
	case errors.Is(err, dialect.ErrSetupTargetCompleted):
		return setup.ErrAlreadyCompleted
	case errors.Is(err, dialect.ErrSetupTargetNotEmpty):
		return fmt.Errorf("%w: %w", ErrDatabaseNotEmpty, err)
	case errors.Is(err, database.ErrMigrate):
		return fmt.Errorf("%w: %w", ErrDatabaseMigration, err)
	default:
		return fmt.Errorf("%w: %w", ErrDatabaseConnection, err)
	}
}
