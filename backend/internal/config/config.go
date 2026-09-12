package config
import (
	"fmt"
	"os"
	"strconv"
	"time"
)
type Config struct {
	Port                 string
	DBDriver             string
	DBDSN                string
	JWTSecret            string
	JWTIssuer            string
	JWTExpiry            time.Duration
	LoginLimitPerMinute  int
	RunLimitPerMinute    int
	ShutdownTimeout      time.Duration
	VerificationGraceDay int
}
func Load() (Config, error) {
	cfg := Config{
		Port:                 env("PORT", "8080"),
		DBDriver:             env("DB_DRIVER", "postgres"),
		DBDSN:                env("DB_DSN", "host=localhost user=hazop password=hazop dbname=hazop port=5432 sslmode=disable"),
		JWTSecret:            env("JWT_SECRET", "development-only-change-me"),
		JWTIssuer:            env("JWT_ISSUER", "hazop-safeguard-coverage"),
		JWTExpiry:            durationEnv("JWT_EXPIRY", 8*time.Hour),
		LoginLimitPerMinute:  intEnv("LOGIN_LIMIT_PER_MINUTE", 20),
		RunLimitPerMinute:    intEnv("RUN_LIMIT_PER_MINUTE", 30),
		ShutdownTimeout:      durationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		VerificationGraceDay: intEnv("VERIFICATION_GRACE_DAYS", 0),
	}
	if cfg.Port == "" {
		return Config{}, fmt.Errorf("PORT must not be empty")
	}
	if cfg.DBDriver != "postgres" && cfg.DBDriver != "sqlite" {
		return Config{}, fmt.Errorf("unsupported DB_DRIVER %q", cfg.DBDriver)
	}
	if cfg.DBDSN == "" {
		return Config{}, fmt.Errorf("DB_DSN must not be empty")
	}
	if len(cfg.JWTSecret) < 16 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 16 characters")
	}
	return cfg, nil
}
func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
func intEnv(key string, fallback int) int {
	raw := env(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := env(key, "")
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
