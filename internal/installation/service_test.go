package installation

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"main/internal/database"
	"main/internal/database/dialect"
	"main/internal/identity"
	"main/internal/setup"
)

func TestCompleteSelectsSQLiteAndPublishesRuntime(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	selectionStore := database.NewSelectionStore(dataDir)
	var published atomic.Int32
	service, err := New(context.Background(), Options{
		DataDir:        dataDir,
		SelectionStore: selectionStore,
		Publish: func(*database.Runtime) {
			published.Add(1)
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := service.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	if state := service.State(); state.Phase != PhaseDatabaseRequired {
		t.Fatalf("initial phase = %q, want %q", state.Phase, PhaseDatabaseRequired)
	}

	state, err := service.Complete(context.Background(), CompleteInput{
		Username: "owner",
		Password: "owner1234",
		Database: &DatabaseInput{Driver: database.DriverSQLite},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if state.Phase != PhaseCompleted || state.DatabaseDriver != database.DriverSQLite || state.OwnerUserID == nil {
		t.Fatalf("Complete() state = %+v, want a completed SQLite installation", state)
	}
	if published.Load() != 1 {
		t.Fatalf("published runtimes = %d, want 1", published.Load())
	}

	if _, configured, loadErr := selectionStore.Load(); loadErr != nil || configured {
		t.Fatalf("selection configured = %t, error = %v; SQLite must not need a selection file", configured, loadErr)
	}

	if _, secondErr := service.Complete(context.Background(), CompleteInput{
		Username: "other",
		Password: "other1234",
	}); !errors.Is(secondErr, setup.ErrAlreadyCompleted) {
		t.Fatalf("second Complete() error = %v, want ErrAlreadyCompleted", secondErr)
	}
	if published.Load() != 1 {
		t.Fatalf("published runtimes after duplicate = %d, want 1", published.Load())
	}
}

func TestCompleteRequiresDatabaseOnFirstRun(t *testing.T) {
	t.Parallel()

	service, err := New(context.Background(), Options{
		DataDir:        t.TempDir(),
		SelectionStore: database.NewSelectionStore(t.TempDir()),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := service.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	_, err = service.Complete(context.Background(), CompleteInput{
		Username: "owner",
		Password: "owner1234",
	})
	if !errors.Is(err, ErrDatabaseInputRequired) {
		t.Fatalf("Complete() error = %v, want ErrDatabaseInputRequired", err)
	}
}

func TestCompleteRejectsInvalidAdminBeforePersistingSelection(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	selectionStore := database.NewSelectionStore(dataDir)
	service, err := New(context.Background(), Options{
		DataDir:        dataDir,
		SelectionStore: selectionStore,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := service.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	_, err = service.Complete(context.Background(), CompleteInput{
		Username: "   ",
		Password: "short",
		Database: &DatabaseInput{Driver: database.DriverSQLite},
	})
	if !errors.Is(err, identity.ErrUsernameRequired) {
		t.Fatalf("Complete() error = %v, want ErrUsernameRequired", err)
	}
	if _, configured, loadErr := selectionStore.Load(); loadErr != nil || configured {
		t.Fatalf("selection configured = %t, error = %v; want no selection", configured, loadErr)
	}
}

func TestNewAdoptsIncompleteDefaultSQLite(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	selection := database.DefaultSQLiteSelection(dataDir)
	runtime, err := database.OpenForSetup(context.Background(), database.Config{
		Driver: selection.Driver,
		DSN:    selection.DSN,
	})
	if err != nil {
		t.Fatalf("OpenForSetup() error = %v", err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatalf("Close() migrated runtime error = %v", err)
	}

	service, err := New(context.Background(), Options{
		DataDir:        dataDir,
		SelectionStore: database.NewSelectionStore(dataDir),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := service.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})
	if state := service.State(); state.Phase != PhaseFinalizationRequired {
		t.Fatalf("initial phase = %q, want %q", state.Phase, PhaseFinalizationRequired)
	}

	state, err := service.Complete(context.Background(), CompleteInput{
		Username: "owner",
		Password: "owner1234",
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if state.Phase != PhaseCompleted || state.DatabaseDriver != database.DriverSQLite {
		t.Fatalf("Complete() state = %+v, want completed", state)
	}
}

func TestCompleteAcceptsRepeatedSelectionWhileFinalizing(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	selection := database.DefaultSQLiteSelection(dataDir)
	runtime, err := database.OpenForSetup(context.Background(), database.Config{
		Driver: selection.Driver,
		DSN:    selection.DSN,
	})
	if err != nil {
		t.Fatalf("OpenForSetup() error = %v", err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatalf("Close() migrated runtime error = %v", err)
	}

	service, err := New(context.Background(), Options{
		DataDir:        dataDir,
		SelectionStore: database.NewSelectionStore(dataDir),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := service.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})
	if state := service.State(); state.Phase != PhaseFinalizationRequired {
		t.Fatalf("initial phase = %q, want %q", state.Phase, PhaseFinalizationRequired)
	}

	_, err = service.Complete(context.Background(), CompleteInput{
		Username: "owner",
		Password: "owner1234",
		Database: &DatabaseInput{
			Driver: database.DriverPostgreSQL,
			PostgreSQL: &PostgreSQLConfig{
				Host: "localhost", Port: 5432, Database: "app", Username: "app", Password: "secret", SSLMode: "require",
			},
		},
	})
	if !errors.Is(err, ErrDatabaseAlreadyConfigured) {
		t.Fatalf("Complete() with a different database error = %v, want ErrDatabaseAlreadyConfigured", err)
	}

	state, err := service.Complete(context.Background(), CompleteInput{
		Username: "owner",
		Password: "owner1234",
		Database: &DatabaseInput{Driver: database.DriverSQLite},
	})
	if err != nil {
		t.Fatalf("Complete() with the repeated selection error = %v", err)
	}
	if state.Phase != PhaseCompleted {
		t.Fatalf("Complete() phase = %q, want %q", state.Phase, PhaseCompleted)
	}
}

func TestNewReopensCompletedInstallation(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	selectionStore := database.NewSelectionStore(dataDir)
	service, err := New(context.Background(), Options{
		DataDir:        dataDir,
		SelectionStore: selectionStore,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := service.Complete(context.Background(), CompleteInput{
		Username: "owner",
		Password: "owner1234",
		Database: &DatabaseInput{Driver: database.DriverSQLite},
	}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if err := service.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, configured, loadErr := selectionStore.Load(); loadErr != nil || configured {
		t.Fatalf("selection configured = %t, error = %v; SQLite must reopen from the default file alone", configured, loadErr)
	}

	reopened, err := New(context.Background(), Options{
		DataDir:        dataDir,
		SelectionStore: selectionStore,
	})
	if err != nil {
		t.Fatalf("reopen New() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := reopened.Close(); closeErr != nil {
			t.Fatalf("reopened Close() error = %v", closeErr)
		}
	})
	state := reopened.State()
	if state.Phase != PhaseCompleted || state.OwnerUserID == nil {
		t.Fatalf("reopened state = %+v, want completed installation", state)
	}
}

func TestNewRejectsForeignDefaultSQLite(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	path := filepath.Join(dataDir, "data.db")
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if _, err := sqldb.Exec("CREATE TABLE unrelated (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create unrelated table: %v", err)
	}
	if err := sqldb.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	_, err = New(context.Background(), Options{
		DataDir:        dataDir,
		SelectionStore: database.NewSelectionStore(dataDir),
	})
	if !errors.Is(err, dialect.ErrExistingUnknown) {
		t.Fatalf("New() error = %v, want ErrExistingUnknown", err)
	}
}

func TestConcurrentCompleteRunsOnce(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	var published atomic.Int32
	service, err := New(context.Background(), Options{
		DataDir:        dataDir,
		SelectionStore: database.NewSelectionStore(dataDir),
		Publish: func(*database.Runtime) {
			published.Add(1)
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := service.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	var wait sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, completeErr := service.Complete(context.Background(), CompleteInput{
				Username: "owner",
				Password: "owner1234",
				Database: &DatabaseInput{Driver: database.DriverSQLite},
			})
			results <- completeErr
		}()
	}
	wait.Wait()
	close(results)

	successes := 0
	alreadyCompleted := 0
	for result := range results {
		switch {
		case result == nil:
			successes++
		case errors.Is(result, setup.ErrAlreadyCompleted):
			alreadyCompleted++
		default:
			t.Fatalf("unexpected Complete() error = %v", result)
		}
	}
	if successes != 1 || alreadyCompleted != 1 {
		t.Fatalf("results: success=%d already=%d, want 1 and 1", successes, alreadyCompleted)
	}
	if published.Load() != 1 {
		t.Fatalf("published runtimes = %d, want 1", published.Load())
	}
}

func TestCompleteAbortsWhenPrepareRuntimeFails(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	prepareErr := errors.New("prepare failed")
	var published atomic.Int32
	service, err := New(context.Background(), Options{
		DataDir:        dataDir,
		SelectionStore: database.NewSelectionStore(dataDir),
		PrepareRuntime: func(context.Context, *database.Runtime) error {
			return prepareErr
		},
		Publish: func(*database.Runtime) {
			published.Add(1)
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := service.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	_, err = service.Complete(context.Background(), CompleteInput{
		Username: "owner",
		Password: "owner1234",
		Database: &DatabaseInput{Driver: database.DriverSQLite},
	})
	if !errors.Is(err, prepareErr) {
		t.Fatalf("Complete() error = %v, want the prepare error", err)
	}
	if published.Load() != 0 {
		t.Fatalf("published runtimes = %d, want 0", published.Load())
	}
	if state := service.State(); state.Phase != PhaseDatabaseRequired {
		t.Fatalf("phase = %q, want %q", state.Phase, PhaseDatabaseRequired)
	}
}

func TestPrepareRuntimeRunsOnceBeforePublish(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	runtime, err := database.OpenForSetup(context.Background(), database.Config(database.DefaultSQLiteSelection(dataDir)))
	if err != nil {
		t.Fatalf("OpenForSetup() error = %v", err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatalf("Close() migrated runtime error = %v", err)
	}

	var steps []string
	service, err := New(context.Background(), Options{
		DataDir:        dataDir,
		SelectionStore: database.NewSelectionStore(dataDir),
		PrepareRuntime: func(context.Context, *database.Runtime) error {
			steps = append(steps, "prepare")
			return nil
		},
		Publish: func(*database.Runtime) {
			steps = append(steps, "publish")
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := service.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	if state := service.State(); state.Phase != PhaseFinalizationRequired {
		t.Fatalf("phase = %q, want %q", state.Phase, PhaseFinalizationRequired)
	}
	if _, err := service.Complete(context.Background(), CompleteInput{Username: "owner", Password: "owner1234"}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if got := strings.Join(steps, ","); got != "prepare,publish" {
		t.Fatalf("runtime lifecycle = %q, want prepare once before publish", got)
	}
}

func TestNewAbortsWhenPrepareRuntimeFails(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	selection := database.DefaultSQLiteSelection(dataDir)
	runtime, err := database.OpenForSetup(context.Background(), database.Config(selection))
	if err != nil {
		t.Fatalf("OpenForSetup() error = %v", err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatalf("Close() migrated runtime error = %v", err)
	}

	prepareErr := errors.New("prepare failed")
	_, err = New(context.Background(), Options{
		DataDir:        dataDir,
		SelectionStore: database.NewSelectionStore(dataDir),
		PrepareRuntime: func(context.Context, *database.Runtime) error {
			return prepareErr
		},
	})
	if !errors.Is(err, prepareErr) {
		t.Fatalf("New() error = %v, want the prepare error", err)
	}
}

func TestPostgreSQLDSN(t *testing.T) {
	t.Parallel()

	dsn, err := postgreSQLDSN(PostgreSQLConfig{
		Host:     "2001:db8::1",
		Port:     5432,
		Database: "starter",
		Username: "app user",
		Password: "p@ss word",
		SSLMode:  "verify-full",
	})
	if err != nil {
		t.Fatalf("postgreSQLDSN() error = %v", err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Host != "[2001:db8::1]:5432" || parsed.Path != "/starter" {
		t.Fatalf("parsed DSN host/path = %q %q", parsed.Host, parsed.Path)
	}
	if parsed.Query().Get("sslmode") != "verify-full" {
		t.Fatalf("sslmode = %q, want verify-full", parsed.Query().Get("sslmode"))
	}
	password, ok := parsed.User.Password()
	if !ok || password != "p@ss word" {
		t.Fatalf("DSN password = %q, ok = %t", password, ok)
	}

	for _, invalid := range []PostgreSQLConfig{
		{Host: "", Port: 5432, Database: "app", Username: "user", Password: "secret", SSLMode: "require"},
		{Host: "localhost", Port: 0, Database: "app", Username: "user", Password: "secret", SSLMode: "require"},
		{Host: "localhost", Port: 5432, Database: "app", Username: "user", Password: "secret", SSLMode: "prefer"},
	} {
		if _, err := postgreSQLDSN(invalid); !errors.Is(err, ErrDatabaseConfigInvalid) {
			t.Fatalf("postgreSQLDSN(%+v) error = %v, want ErrDatabaseConfigInvalid", invalid, err)
		}
	}
}

func TestPostgreSQLDSNErrorDoesNotLeakPassword(t *testing.T) {
	t.Parallel()

	_, err := postgreSQLDSN(PostgreSQLConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "app",
		Username: "user",
		Password: "",
		SSLMode:  "require",
	})
	if err == nil {
		t.Fatal("postgreSQLDSN() error = nil, want invalid configuration")
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("error leaks credentials: %v", err)
	}
}
