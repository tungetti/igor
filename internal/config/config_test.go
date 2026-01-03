package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultConfig tests that DefaultConfig returns valid defaults
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "", cfg.LogFile)
	assert.False(t, cfg.DryRun)
	assert.False(t, cfg.Verbose)
	assert.False(t, cfg.Quiet)
	assert.Equal(t, 5*time.Minute, cfg.Timeout)
	assert.Equal(t, 60*time.Second, cfg.NetworkTimeout)
	assert.Equal(t, 2*time.Minute, cfg.CommandTimeout)
	assert.False(t, cfg.InstallCUDA)
	assert.Equal(t, "", cfg.CUDAVersion)
	assert.Equal(t, "", cfg.DriverVersion)
	assert.False(t, cfg.AllowUnsigned)
	assert.False(t, cfg.ForceInstall)
	assert.False(t, cfg.SkipReboot)
	assert.False(t, cfg.NoBackup)
}

// TestDefaultConfigDirectories tests directory defaults
func TestDefaultConfigDirectories(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotEmpty(t, cfg.ConfigDir)
	assert.NotEmpty(t, cfg.CacheDir)
	assert.Contains(t, cfg.ConfigDir, "igor")
	assert.Contains(t, cfg.CacheDir, "igor")
}

// TestXDGConfigDir tests XDG_CONFIG_HOME compliance
func TestXDGConfigDir(t *testing.T) {
	// Save original value
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	// Test with XDG_CONFIG_HOME set
	testPath := "/tmp/test-xdg-config"
	os.Setenv("XDG_CONFIG_HOME", testPath)

	cfg := DefaultConfig()
	assert.Equal(t, filepath.Join(testPath, "igor"), cfg.ConfigDir)
}

// TestXDGCacheDir tests XDG_CACHE_HOME compliance
func TestXDGCacheDir(t *testing.T) {
	// Save original value
	original := os.Getenv("XDG_CACHE_HOME")
	defer os.Setenv("XDG_CACHE_HOME", original)

	// Test with XDG_CACHE_HOME set
	testPath := "/tmp/test-xdg-cache"
	os.Setenv("XDG_CACHE_HOME", testPath)

	cfg := DefaultConfig()
	assert.Equal(t, filepath.Join(testPath, "igor"), cfg.CacheDir)
}

// TestXDGFallback tests fallback to ~/.config and ~/.cache
func TestXDGFallback(t *testing.T) {
	// Save original values
	origConfig := os.Getenv("XDG_CONFIG_HOME")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_CONFIG_HOME", origConfig)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Clear XDG variables
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_CACHE_HOME")

	cfg := DefaultConfig()
	home, _ := os.UserHomeDir()

	assert.Equal(t, filepath.Join(home, ".config", "igor"), cfg.ConfigDir)
	assert.Equal(t, filepath.Join(home, ".cache", "igor"), cfg.CacheDir)
}

// TestConfigPath tests ConfigPath method
func TestConfigPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ConfigDir = "/test/config"

	assert.Equal(t, "/test/config/config.yaml", cfg.ConfigPath())
}

// TestCachePath tests CachePath method
func TestCachePath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CacheDir = "/test/cache"

	assert.Equal(t, "/test/cache/downloads", cfg.CachePath("downloads"))
}

// TestConfigClone tests Clone method
func TestConfigClone(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LogLevel = "debug"
	cfg.DryRun = true

	clone := cfg.Clone()

	assert.Equal(t, cfg.LogLevel, clone.LogLevel)
	assert.Equal(t, cfg.DryRun, clone.DryRun)

	// Modify clone and verify original is unchanged
	clone.LogLevel = "error"
	assert.Equal(t, "debug", cfg.LogLevel)
}

// TestLoaderLoadDefaults tests loading with no file
func TestLoaderLoadDefaults(t *testing.T) {
	loader := NewLoader("")
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "info", cfg.LogLevel)
}

// TestLoaderLoadFromFile tests loading from YAML file
func TestLoaderLoadFromFile(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
log_level: debug
dry_run: true
verbose: true
timeout: 10m
cuda_version: "12.0"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	loader := NewLoader(configPath)
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.True(t, cfg.DryRun)
	assert.True(t, cfg.Verbose)
	assert.Equal(t, 10*time.Minute, cfg.Timeout)
	assert.Equal(t, "12.0", cfg.CUDAVersion)
}

// TestLoaderFileNotFound tests behavior when file doesn't exist
func TestLoaderFileNotFound(t *testing.T) {
	loader := NewLoader("/nonexistent/path/config.yaml")
	cfg, err := loader.Load()

	// Should not error, should return defaults
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "info", cfg.LogLevel)
}

// TestLoaderInvalidYAML tests behavior with invalid YAML
func TestLoaderInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	loader := NewLoader(configPath)
	_, err = loader.Load()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

// TestLoaderEnvironmentOverrides tests environment variable overrides
func TestLoaderEnvironmentOverrides(t *testing.T) {
	// Create temp config file with some values
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
log_level: info
dry_run: false
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment variables
	envVars := map[string]string{
		"IGOR_LOG_LEVEL":       "debug",
		"IGOR_DRY_RUN":         "true",
		"IGOR_VERBOSE":         "yes",
		"IGOR_TIMEOUT":         "15m",
		"IGOR_NETWORK_TIMEOUT": "2m",
		"IGOR_COMMAND_TIMEOUT": "5m",
		"IGOR_INSTALL_CUDA":    "1",
		"IGOR_CUDA_VERSION":    "12.1",
		"IGOR_DRIVER_VERSION":  "535.104",
		"IGOR_ALLOW_UNSIGNED":  "on",
		"IGOR_FORCE_INSTALL":   "true",
		"IGOR_SKIP_REBOOT":     "true",
		"IGOR_NO_BACKUP":       "true",
	}

	// Set and defer cleanup
	for k, v := range envVars {
		original := os.Getenv(k)
		os.Setenv(k, v)
		defer os.Setenv(k, original)
	}

	loader := NewLoader(configPath)
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.True(t, cfg.DryRun)
	assert.True(t, cfg.Verbose)
	assert.Equal(t, 15*time.Minute, cfg.Timeout)
	assert.Equal(t, 2*time.Minute, cfg.NetworkTimeout)
	assert.Equal(t, 5*time.Minute, cfg.CommandTimeout)
	assert.True(t, cfg.InstallCUDA)
	assert.Equal(t, "12.1", cfg.CUDAVersion)
	assert.Equal(t, "535.104", cfg.DriverVersion)
	assert.True(t, cfg.AllowUnsigned)
	assert.True(t, cfg.ForceInstall)
	assert.True(t, cfg.SkipReboot)
	assert.True(t, cfg.NoBackup)
}

// TestLoaderEnvironmentDirectories tests directory environment overrides
func TestLoaderEnvironmentDirectories(t *testing.T) {
	// Set environment variables
	origConfig := os.Getenv("IGOR_CONFIG_DIR")
	origCache := os.Getenv("IGOR_CACHE_DIR")
	defer func() {
		os.Setenv("IGOR_CONFIG_DIR", origConfig)
		os.Setenv("IGOR_CACHE_DIR", origCache)
	}()

	os.Setenv("IGOR_CONFIG_DIR", "/custom/config")
	os.Setenv("IGOR_CACHE_DIR", "/custom/cache")

	loader := NewLoader("")
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "/custom/config", cfg.ConfigDir)
	assert.Equal(t, "/custom/cache", cfg.CacheDir)
}

// TestLoaderWithCustomPrefix tests custom environment prefix
func TestLoaderWithCustomPrefix(t *testing.T) {
	original := os.Getenv("MYAPP_LOG_LEVEL")
	defer os.Setenv("MYAPP_LOG_LEVEL", original)

	os.Setenv("MYAPP_LOG_LEVEL", "debug")

	loader := NewLoaderWithPrefix("", "MYAPP_")
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
}

// TestLoaderLoadAndValidate tests combined load and validate
func TestLoaderLoadAndValidate(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
log_level: info
timeout: 5m
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	loader := NewLoader(configPath)
	cfg, err := loader.LoadAndValidate()

	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

// TestLoaderLoadAndValidateInvalid tests validation failure
func TestLoaderLoadAndValidateInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
log_level: invalid
timeout: -1s
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	loader := NewLoader(configPath)
	_, err = loader.LoadAndValidate()

	assert.Error(t, err)
}

// TestValidatorValidConfig tests validation of valid config
func TestValidatorValidConfig(t *testing.T) {
	cfg := DefaultConfig()
	validator := NewValidator()

	errs := validator.Validate(cfg)
	assert.Empty(t, errs)
}

// TestValidatorInvalidLogLevel tests invalid log level detection
func TestValidatorInvalidLogLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LogLevel = "invalid"
	validator := NewValidator()

	errs := validator.Validate(cfg)
	assert.NotEmpty(t, errs)

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "log_level") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected log_level validation error")
}

// TestValidatorInvalidTimeouts tests invalid timeout detection
func TestValidatorInvalidTimeouts(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Timeout = -1 * time.Second
	cfg.NetworkTimeout = 0
	cfg.CommandTimeout = -5 * time.Minute
	validator := NewValidator()

	errs := validator.Validate(cfg)
	assert.Len(t, errs, 3)
}

// TestValidatorConflictingOptions tests verbose/quiet conflict detection
func TestValidatorConflictingOptions(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Verbose = true
	cfg.Quiet = true
	validator := NewValidator()

	errs := validator.Validate(cfg)
	assert.NotEmpty(t, errs)

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "verbose") && strings.Contains(err.Error(), "quiet") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected verbose/quiet conflict error")
}

// TestValidatorLogFileDirectory tests log file directory validation
func TestValidatorLogFileDirectory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LogFile = "/nonexistent/directory/app.log"
	validator := NewValidator()

	errs := validator.Validate(cfg)
	assert.NotEmpty(t, errs)

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "log_file") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected log_file validation error")
}

// TestValidatorLogFileCurrentDir tests log file in current directory
func TestValidatorLogFileCurrentDir(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LogFile = "app.log" // Current directory, should be valid
	validator := NewValidator()

	errs := validator.Validate(cfg)
	// Should not have log_file error
	for _, err := range errs {
		assert.NotContains(t, err.Error(), "log_file")
	}
}

// TestValidatorInvalidVersionFormat tests version format validation
func TestValidatorInvalidVersionFormat(t *testing.T) {
	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{"empty", "", true}, // Empty is valid (not specified)
		{"simple", "12", true},
		{"major.minor", "12.0", true},
		{"semver", "535.104.05", true},
		{"with_text", "12.0-beta", false},
		{"leading_dot", ".12.0", false},
		{"trailing_dot", "12.0.", false},
		{"double_dot", "12..0", false},
		{"letters", "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.CUDAVersion = tt.version
			validator := NewValidator()

			errs := validator.Validate(cfg)

			hasVersionError := false
			for _, err := range errs {
				if strings.Contains(err.Error(), "cuda_version") {
					hasVersionError = true
					break
				}
			}

			if tt.valid {
				assert.False(t, hasVersionError, "Version %q should be valid", tt.version)
			} else {
				assert.True(t, hasVersionError, "Version %q should be invalid", tt.version)
			}
		})
	}
}

// TestValidatorEmptyDirectories tests empty directory validation
func TestValidatorEmptyDirectories(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ConfigDir = ""
	cfg.CacheDir = ""
	validator := NewValidator()

	errs := validator.Validate(cfg)
	assert.Len(t, errs, 2)
}

// TestValidatorValidateOrError tests ValidateOrError method
func TestValidatorValidateOrError(t *testing.T) {
	validator := NewValidator()

	// Valid config
	cfg := DefaultConfig()
	err := validator.ValidateOrError(cfg)
	assert.NoError(t, err)

	// Invalid config
	cfg.LogLevel = "invalid"
	cfg.Timeout = -1
	err = validator.ValidateOrError(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "log_level")
	assert.Contains(t, err.Error(), "timeout")
}

// TestValidatorIsValid tests IsValid method
func TestValidatorIsValid(t *testing.T) {
	validator := NewValidator()

	cfg := DefaultConfig()
	assert.True(t, validator.IsValid(cfg))

	cfg.LogLevel = "invalid"
	assert.False(t, validator.IsValid(cfg))
}

// TestValidateField tests individual field validation
func TestValidateField(t *testing.T) {
	// Valid log level
	err := ValidateField("log_level", "debug")
	assert.NoError(t, err)

	// Invalid log level
	err = ValidateField("log_level", "invalid")
	assert.Error(t, err)

	// Valid version
	err = ValidateField("cuda_version", "12.0")
	assert.NoError(t, err)

	// Invalid version
	err = ValidateField("driver_version", "abc")
	assert.Error(t, err)

	// Empty version is valid
	err = ValidateField("cuda_version", "")
	assert.NoError(t, err)
}

// TestParseBool tests parseBool function
func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"invalid", false},
		{"  true  ", true}, // With whitespace
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseBool(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConfigHelperMethods tests IsVerbose and IsSilent
func TestConfigHelperMethods(t *testing.T) {
	cfg := DefaultConfig()

	// Default state
	assert.False(t, cfg.IsVerbose())
	assert.False(t, cfg.IsSilent())

	// Verbose only
	cfg.Verbose = true
	assert.True(t, cfg.IsVerbose())
	assert.False(t, cfg.IsSilent())

	// Both verbose and quiet (quiet wins)
	cfg.Quiet = true
	assert.False(t, cfg.IsVerbose())
	assert.True(t, cfg.IsSilent())

	// Quiet only
	cfg.Verbose = false
	assert.False(t, cfg.IsVerbose())
	assert.True(t, cfg.IsSilent())
}

// TestSaveConfig tests SaveConfig function
func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test", "config.yaml")

	cfg := DefaultConfig()
	cfg.ConfigDir = filepath.Join(tmpDir, "test")
	cfg.LogLevel = "debug"
	cfg.DryRun = true

	err := SaveConfig(cfg, configPath)
	require.NoError(t, err)

	// Verify file was created
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "log_level: debug")
	assert.Contains(t, string(data), "dry_run: true")
}

// TestLoadDefaultConfig tests LoadDefaultConfig function
func TestLoadDefaultConfig(t *testing.T) {
	// This should work even if no config file exists
	cfg, err := LoadDefaultConfig()

	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "info", cfg.LogLevel)
}

// TestValidationErrorString tests ValidationError.Error method
func TestValidationErrorString(t *testing.T) {
	err := &ValidationError{
		Field:   "test_field",
		Message: "test message",
	}

	expected := "config validation: test_field: test message"
	assert.Equal(t, expected, err.Error())
}

// TestGetConfigDir tests GetConfigDir function
func TestGetConfigDir(t *testing.T) {
	dir := GetConfigDir()
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, "igor")
}

// TestGetCacheDir tests GetCacheDir function
func TestGetCacheDir(t *testing.T) {
	dir := GetCacheDir()
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, "igor")
}

// TestLoaderWithAllTimeoutEnvVars tests all timeout environment variables
func TestLoaderWithAllTimeoutEnvVars(t *testing.T) {
	// Save and defer cleanup
	envVars := []string{"IGOR_TIMEOUT", "IGOR_NETWORK_TIMEOUT", "IGOR_COMMAND_TIMEOUT"}
	originals := make(map[string]string)
	for _, key := range envVars {
		originals[key] = os.Getenv(key)
	}
	defer func() {
		for k, v := range originals {
			os.Setenv(k, v)
		}
	}()

	os.Setenv("IGOR_TIMEOUT", "20m")
	os.Setenv("IGOR_NETWORK_TIMEOUT", "3m")
	os.Setenv("IGOR_COMMAND_TIMEOUT", "10m")

	loader := NewLoader("")
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 20*time.Minute, cfg.Timeout)
	assert.Equal(t, 3*time.Minute, cfg.NetworkTimeout)
	assert.Equal(t, 10*time.Minute, cfg.CommandTimeout)
}

// TestLoaderWithInvalidDuration tests behavior with invalid duration env var
func TestLoaderWithInvalidDuration(t *testing.T) {
	original := os.Getenv("IGOR_TIMEOUT")
	defer os.Setenv("IGOR_TIMEOUT", original)

	os.Setenv("IGOR_TIMEOUT", "not-a-duration")

	loader := NewLoader("")
	cfg, err := loader.Load()

	// Should not error, should use default
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, cfg.Timeout)
}

// TestFullConfigYAML tests loading a complete YAML config
func TestFullConfigYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
log_level: debug
log_file: /var/log/igor.log
dry_run: true
verbose: false
quiet: false
config_dir: /etc/igor
cache_dir: /var/cache/igor
timeout: 10m
network_timeout: 30s
command_timeout: 5m
install_cuda: true
cuda_version: "12.2"
driver_version: "535.86.10"
allow_unsigned: true
force_install: true
skip_reboot: true
no_backup: true
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	loader := NewLoader(configPath)
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "/var/log/igor.log", cfg.LogFile)
	assert.True(t, cfg.DryRun)
	assert.False(t, cfg.Verbose)
	assert.False(t, cfg.Quiet)
	assert.Equal(t, "/etc/igor", cfg.ConfigDir)
	assert.Equal(t, "/var/cache/igor", cfg.CacheDir)
	assert.Equal(t, 10*time.Minute, cfg.Timeout)
	assert.Equal(t, 30*time.Second, cfg.NetworkTimeout)
	assert.Equal(t, 5*time.Minute, cfg.CommandTimeout)
	assert.True(t, cfg.InstallCUDA)
	assert.Equal(t, "12.2", cfg.CUDAVersion)
	assert.Equal(t, "535.86.10", cfg.DriverVersion)
	assert.True(t, cfg.AllowUnsigned)
	assert.True(t, cfg.ForceInstall)
	assert.True(t, cfg.SkipReboot)
	assert.True(t, cfg.NoBackup)
}

// TestLoaderWithLogFile tests loading log file from env
func TestLoaderWithLogFile(t *testing.T) {
	original := os.Getenv("IGOR_LOG_FILE")
	defer os.Setenv("IGOR_LOG_FILE", original)

	os.Setenv("IGOR_LOG_FILE", "/tmp/igor.log")

	loader := NewLoader("")
	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "/tmp/igor.log", cfg.LogFile)
}
