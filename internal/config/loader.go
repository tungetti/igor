package config

import (
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tungetti/igor/internal/errors"
)

const (
	// EnvPrefix is the prefix for environment variables.
	EnvPrefix = "IGOR_"
)

// Loader handles configuration loading from multiple sources.
// It loads configuration in order: defaults -> file -> environment variables,
// with later sources overriding earlier ones.
type Loader struct {
	configPath string
	envPrefix  string
}

// NewLoader creates a new configuration loader.
// If configPath is empty, only defaults and environment variables are used.
func NewLoader(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
		envPrefix:  EnvPrefix,
	}
}

// NewLoaderWithPrefix creates a new loader with a custom environment variable prefix.
func NewLoaderWithPrefix(configPath, envPrefix string) *Loader {
	return &Loader{
		configPath: configPath,
		envPrefix:  envPrefix,
	}
}

// Load loads configuration from file and environment.
// The loading order is: defaults -> file -> environment variables.
// Returns an error if the file exists but cannot be parsed.
func (l *Loader) Load() (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Load from file if path is specified
	if l.configPath != "" {
		if err := l.loadFromFile(cfg); err != nil {
			return nil, err
		}
	}

	// Override with environment variables
	l.loadFromEnv(cfg)

	return cfg, nil
}

// LoadAndValidate loads configuration and validates it.
// This is a convenience method that combines Load and Validate.
func (l *Loader) LoadAndValidate() (*Config, error) {
	cfg, err := l.Load()
	if err != nil {
		return nil, err
	}

	validator := NewValidator()
	if err := validator.ValidateOrError(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadFromFile loads config from YAML file.
func (l *Loader) loadFromFile(cfg *Config) error {
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, use defaults - this is not an error
			return nil
		}
		return errors.Wrap(errors.Configuration, "failed to read config file", err).
			WithOp("config.loadFromFile")
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return errors.Wrap(errors.Configuration, "failed to parse config file", err).
			WithOp("config.loadFromFile")
	}

	return nil
}

// loadFromEnv loads config from environment variables.
// Environment variables take precedence over file config.
func (l *Loader) loadFromEnv(cfg *Config) {
	// General settings
	if v := os.Getenv(l.envPrefix + "LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv(l.envPrefix + "LOG_FILE"); v != "" {
		cfg.LogFile = v
	}
	if v := os.Getenv(l.envPrefix + "DRY_RUN"); v != "" {
		cfg.DryRun = parseBool(v)
	}
	if v := os.Getenv(l.envPrefix + "VERBOSE"); v != "" {
		cfg.Verbose = parseBool(v)
	}
	if v := os.Getenv(l.envPrefix + "QUIET"); v != "" {
		cfg.Quiet = parseBool(v)
	}

	// Directories
	if v := os.Getenv(l.envPrefix + "CONFIG_DIR"); v != "" {
		cfg.ConfigDir = v
	}
	if v := os.Getenv(l.envPrefix + "CACHE_DIR"); v != "" {
		cfg.CacheDir = v
	}

	// Timeouts
	if v := os.Getenv(l.envPrefix + "TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
		}
	}
	if v := os.Getenv(l.envPrefix + "NETWORK_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.NetworkTimeout = d
		}
	}
	if v := os.Getenv(l.envPrefix + "COMMAND_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.CommandTimeout = d
		}
	}

	// Installation options
	if v := os.Getenv(l.envPrefix + "INSTALL_CUDA"); v != "" {
		cfg.InstallCUDA = parseBool(v)
	}
	if v := os.Getenv(l.envPrefix + "CUDA_VERSION"); v != "" {
		cfg.CUDAVersion = v
	}
	if v := os.Getenv(l.envPrefix + "DRIVER_VERSION"); v != "" {
		cfg.DriverVersion = v
	}
	if v := os.Getenv(l.envPrefix + "ALLOW_UNSIGNED"); v != "" {
		cfg.AllowUnsigned = parseBool(v)
	}

	// Advanced options
	if v := os.Getenv(l.envPrefix + "FORCE_INSTALL"); v != "" {
		cfg.ForceInstall = parseBool(v)
	}
	if v := os.Getenv(l.envPrefix + "SKIP_REBOOT"); v != "" {
		cfg.SkipReboot = parseBool(v)
	}
	if v := os.Getenv(l.envPrefix + "NO_BACKUP"); v != "" {
		cfg.NoBackup = parseBool(v)
	}
}

// parseBool parses a string as a boolean value.
// Accepts: true, 1, yes, on (case-insensitive) as true.
// All other values are treated as false.
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// SaveConfig saves the configuration to a YAML file.
// The directory is created if it doesn't exist.
func SaveConfig(cfg *Config, path string) error {
	// Ensure directory exists
	dir := cfg.ConfigDir
	if path != "" {
		dir = path[:strings.LastIndex(path, "/")]
		if dir == "" {
			dir = "."
		}
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(errors.Configuration, "failed to create config directory", err).
			WithOp("config.SaveConfig")
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrap(errors.Configuration, "failed to marshal config", err).
			WithOp("config.SaveConfig")
	}

	targetPath := path
	if targetPath == "" {
		targetPath = cfg.ConfigPath()
	}

	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return errors.Wrap(errors.Configuration, "failed to write config file", err).
			WithOp("config.SaveConfig")
	}

	return nil
}

// LoadDefaultConfig loads configuration from the default location.
// It looks for config.yaml in the XDG config directory.
func LoadDefaultConfig() (*Config, error) {
	configPath := DefaultConfig().ConfigPath()
	loader := NewLoader(configPath)
	return loader.Load()
}

// LoadDefaultConfigAndValidate loads and validates configuration from the default location.
func LoadDefaultConfigAndValidate() (*Config, error) {
	configPath := DefaultConfig().ConfigPath()
	loader := NewLoader(configPath)
	return loader.LoadAndValidate()
}
