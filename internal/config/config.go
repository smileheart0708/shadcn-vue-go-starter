package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ysmood/goe"
)

type Config struct {
	DataDir              string
	Port                 int
	APIRequestLogEnabled bool
	// JWTSecret is an optional first-run seed. The signing secret itself lives in
	// the database and always takes precedence over this value.
	JWTSecret          string
	JWTTTL             time.Duration
	RefreshIdleTTL     time.Duration
	RefreshAbsoluteTTL time.Duration
	RefreshCookieName  string
}

func Load() error {
	err := goe.Load(false, true, goe.DOTENV)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("load .env: %w", err)
	}

	return nil
}

func LoadConfig() (Config, error) {
	dataDir, err := getEnv("DATA_DIR", ".data")
	if err != nil {
		return Config{}, err
	}

	port, err := getEnv("PORT", 8080)
	if err != nil {
		return Config{}, err
	}

	apiRequestLogEnabled, err := getEnv("API_REQUEST_LOG_ENABLED", false)
	if err != nil {
		return Config{}, err
	}

	jwtSecret, err := getEnv("JWT_SECRET", "")
	if err != nil {
		return Config{}, err
	}

	jwtTTL, err := getEnv("JWT_TTL", 10*time.Minute)
	if err != nil {
		return Config{}, err
	}

	refreshIdleTTL, err := getEnv("REFRESH_IDLE_TTL", 7*24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	refreshAbsoluteTTL, err := getEnv("REFRESH_ABSOLUTE_TTL", 30*24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	refreshCookieName, err := getEnv("REFRESH_COOKIE_NAME", "")
	if err != nil {
		return Config{}, err
	}

	return Config{
		DataDir:              dataDir,
		Port:                 port,
		APIRequestLogEnabled: apiRequestLogEnabled,
		JWTSecret:            jwtSecret,
		JWTTTL:               jwtTTL,
		RefreshIdleTTL:       refreshIdleTTL,
		RefreshAbsoluteTTL:   refreshAbsoluteTTL,
		RefreshCookieName:    refreshCookieName,
	}, nil
}

func getEnv[T goe.EnvType](name string, fallback T) (T, error) {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}

	value, err := goe.Parse[T](raw)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("config %s: %w", name, err)
	}

	return value, nil
}
