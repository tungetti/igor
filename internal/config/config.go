// Package config provides configuration management for the Igor application.
// It supports loading configuration from YAML files and environment variables,
// with validation and sensible defaults. The package follows XDG Base Directory
// specification for locating configuration files.
package config

import (
	"path/filepath"
	"time"
)

// Config represents the application configuration.
// Configuration values can be set via YAML file or environment variables,
// with environment variables taking precedence.
type Config struct {
	// General settings
	LogLevel string `yaml:"log_level"`
	LogFile  string `yaml:"log_file"`
	DryRun   bool   `yaml:"dry_run"`
	Verbose  bool   `yaml:"verbose"`
	Quiet    bool   `yaml:"quiet"`

	// Directories
	ConfigDir string `yaml:"config_dir"`
	CacheDir  string `yaml:"cache_dir"`

	// Timeouts
	Timeout        time.Duration `yaml:"timeout"`
	NetworkTimeout time.Duration `yaml:"network_timeout"`
	CommandTimeout time.Duration `yaml:"command_timeout"`

	// Installation options
	InstallCUDA   bool   `yaml:"install_cuda"`
	CUDAVersion   string `yaml:"cuda_version"`
	DriverVersion string `yaml:"driver_version"`
	AllowUnsigned bool   `yaml:"allow_unsigned"`

	// Advanced
	ForceInstall bool `yaml:"force_install"`
	SkipReboot   bool `yaml:"skip_reboot"`
	NoBackup     bool `yaml:"no_backup"`
}

// ConfigPath returns the path to the config file.
func (c *Config) ConfigPath() string {
	return filepath.Join(c.ConfigDir, "config.yaml")
}

// CachePath returns a path within the cache directory.
func (c *Config) CachePath(name string) string {
	return filepath.Join(c.CacheDir, name)
}

// IsVerbose returns true if verbose output is enabled and quiet is not.
func (c *Config) IsVerbose() bool {
	return c.Verbose && !c.Quiet
}

// IsSilent returns true if quiet mode is enabled.
func (c *Config) IsSilent() bool {
	return c.Quiet
}

// Clone returns a deep copy of the configuration.
func (c *Config) Clone() *Config {
	clone := *c
	return &clone
}
