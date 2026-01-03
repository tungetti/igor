package config

import (
	"os"
	"path/filepath"
	"time"
)

const (
	// AppName is the application name used for directory paths.
	AppName = "igor"

	// DefaultLogLevel is the default logging level.
	DefaultLogLevel = "info"

	// DefaultTimeout is the default overall operation timeout.
	DefaultTimeout = 5 * time.Minute

	// DefaultNetworkTimeout is the default network operation timeout.
	DefaultNetworkTimeout = 60 * time.Second

	// DefaultCommandTimeout is the default command execution timeout.
	DefaultCommandTimeout = 2 * time.Minute
)

// DefaultConfig returns a Config with sensible defaults.
// All default values follow best practices for system administration tools.
func DefaultConfig() *Config {
	return &Config{
		LogLevel:       DefaultLogLevel,
		LogFile:        "",
		DryRun:         false,
		Verbose:        false,
		Quiet:          false,
		ConfigDir:      defaultConfigDir(),
		CacheDir:       defaultCacheDir(),
		Timeout:        DefaultTimeout,
		NetworkTimeout: DefaultNetworkTimeout,
		CommandTimeout: DefaultCommandTimeout,
		InstallCUDA:    false,
		CUDAVersion:    "",
		DriverVersion:  "",
		AllowUnsigned:  false,
		ForceInstall:   false,
		SkipReboot:     false,
		NoBackup:       false,
	}
}

// defaultConfigDir returns the XDG config directory for igor.
// Falls back to ~/.config/igor if XDG_CONFIG_HOME is not set.
func defaultConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, AppName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home can't be determined
		return filepath.Join(".", ".config", AppName)
	}
	return filepath.Join(home, ".config", AppName)
}

// defaultCacheDir returns the XDG cache directory for igor.
// Falls back to ~/.cache/igor if XDG_CACHE_HOME is not set.
func defaultCacheDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, AppName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home can't be determined
		return filepath.Join(".", ".cache", AppName)
	}
	return filepath.Join(home, ".cache", AppName)
}

// GetConfigDir returns the configuration directory, respecting XDG.
// This is exported for use by other packages that need the config path.
func GetConfigDir() string {
	return defaultConfigDir()
}

// GetCacheDir returns the cache directory, respecting XDG.
// This is exported for use by other packages that need the cache path.
func GetCacheDir() string {
	return defaultCacheDir()
}
