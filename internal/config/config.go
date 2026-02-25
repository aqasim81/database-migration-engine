package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Default values for configuration fields.
const (
	DefaultMigrationsDir    = "./migrations"
	DefaultLockTimeout      = 5 * time.Second
	DefaultStatementTimeout = 30 * time.Second
	DefaultTargetPGVersion  = 14
	DefaultFormat           = "text"
)

// Config holds the application configuration loaded from file, environment, and flags.
type Config struct {
	DatabaseURL      string
	MigrationsDir    string
	LockTimeout      time.Duration
	StatementTimeout time.Duration
	TargetPGVersion  int
	Format           string
}

// yamlConfig is the raw YAML file representation with string durations.
type yamlConfig struct {
	DatabaseURL      string `yaml:"database_url"`
	MigrationsDir    string `yaml:"migrations_dir"`
	LockTimeout      string `yaml:"lock_timeout"`
	StatementTimeout string `yaml:"statement_timeout"`
	TargetPGVersion  int    `yaml:"target_pg_version"`
	Format           string `yaml:"format"`
}

// New returns a Config populated with default values.
func New() *Config {
	return &Config{
		MigrationsDir:    DefaultMigrationsDir,
		LockTimeout:      DefaultLockTimeout,
		StatementTimeout: DefaultStatementTimeout,
		TargetPGVersion:  DefaultTargetPGVersion,
		Format:           DefaultFormat,
	}
}

// Load reads a YAML configuration file and returns a Config.
// If allowMissing is true and the file does not exist, defaults are returned.
func Load(path string, allowMissing bool) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && allowMissing {
			return New(), nil
		}

		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var raw yamlConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return fromYAML(&raw)
}

// fromYAML converts the raw YAML representation to a Config with defaults applied.
func fromYAML(raw *yamlConfig) (*Config, error) {
	cfg := New()

	if raw.DatabaseURL != "" {
		cfg.DatabaseURL = raw.DatabaseURL
	}

	if raw.MigrationsDir != "" {
		cfg.MigrationsDir = raw.MigrationsDir
	}

	if raw.LockTimeout != "" {
		d, err := time.ParseDuration(raw.LockTimeout)
		if err != nil {
			return nil, fmt.Errorf("parsing lock_timeout %q: %w", raw.LockTimeout, err)
		}

		cfg.LockTimeout = d
	}

	if raw.StatementTimeout != "" {
		d, err := time.ParseDuration(raw.StatementTimeout)
		if err != nil {
			return nil, fmt.Errorf("parsing statement_timeout %q: %w", raw.StatementTimeout, err)
		}

		cfg.StatementTimeout = d
	}

	if raw.TargetPGVersion != 0 {
		cfg.TargetPGVersion = raw.TargetPGVersion
	}

	if raw.Format != "" {
		cfg.Format = raw.Format
	}

	return cfg, nil
}

// MergeEnv overrides config fields from MIGRATE_* environment variables.
func MergeEnv(cfg *Config) {
	if v := os.Getenv("MIGRATE_DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}

	if v := os.Getenv("MIGRATE_MIGRATIONS_DIR"); v != "" {
		cfg.MigrationsDir = v
	}

	if v := os.Getenv("MIGRATE_LOCK_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.LockTimeout = d
		}
	}

	if v := os.Getenv("MIGRATE_STATEMENT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.StatementTimeout = d
		}
	}
}
