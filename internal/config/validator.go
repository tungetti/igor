package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tungetti/igor/internal/errors"
)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation: %s: %s", e.Field, e.Message)
}

// Validator validates configuration.
type Validator struct {
	// validLogLevels defines the accepted log level values.
	validLogLevels map[string]bool
}

// NewValidator creates a new validator.
func NewValidator() *Validator {
	return &Validator{
		validLogLevels: map[string]bool{
			"debug": true,
			"info":  true,
			"warn":  true,
			"error": true,
		},
	}
}

// Validate validates the configuration and returns all errors.
// This allows collecting all validation errors at once rather than
// failing on the first error.
func (v *Validator) Validate(cfg *Config) []error {
	var errs []error

	// Validate log level
	if !v.validLogLevels[strings.ToLower(cfg.LogLevel)] {
		errs = append(errs, &ValidationError{
			Field:   "log_level",
			Message: fmt.Sprintf("invalid log level %q: must be one of: debug, info, warn, error", cfg.LogLevel),
		})
	}

	// Validate timeouts are positive
	if cfg.Timeout <= 0 {
		errs = append(errs, &ValidationError{
			Field:   "timeout",
			Message: "timeout must be positive",
		})
	}
	if cfg.NetworkTimeout <= 0 {
		errs = append(errs, &ValidationError{
			Field:   "network_timeout",
			Message: "network timeout must be positive",
		})
	}
	if cfg.CommandTimeout <= 0 {
		errs = append(errs, &ValidationError{
			Field:   "command_timeout",
			Message: "command timeout must be positive",
		})
	}

	// Validate conflicting options
	if cfg.Verbose && cfg.Quiet {
		errs = append(errs, &ValidationError{
			Field:   "verbose/quiet",
			Message: "verbose and quiet cannot both be true",
		})
	}

	// Validate log file directory exists (if log file is specified)
	if cfg.LogFile != "" {
		dir := filepath.Dir(cfg.LogFile)
		if dir != "" && dir != "." {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				errs = append(errs, &ValidationError{
					Field:   "log_file",
					Message: fmt.Sprintf("directory does not exist: %s", dir),
				})
			}
		}
	}

	// Validate CUDA version format if specified
	if cfg.CUDAVersion != "" {
		if !isValidVersionFormat(cfg.CUDAVersion) {
			errs = append(errs, &ValidationError{
				Field:   "cuda_version",
				Message: fmt.Sprintf("invalid version format: %s", cfg.CUDAVersion),
			})
		}
	}

	// Validate driver version format if specified
	if cfg.DriverVersion != "" {
		if !isValidVersionFormat(cfg.DriverVersion) {
			errs = append(errs, &ValidationError{
				Field:   "driver_version",
				Message: fmt.Sprintf("invalid version format: %s", cfg.DriverVersion),
			})
		}
	}

	// Validate directories are not empty
	if cfg.ConfigDir == "" {
		errs = append(errs, &ValidationError{
			Field:   "config_dir",
			Message: "config directory cannot be empty",
		})
	}
	if cfg.CacheDir == "" {
		errs = append(errs, &ValidationError{
			Field:   "cache_dir",
			Message: "cache directory cannot be empty",
		})
	}

	return errs
}

// ValidateOrError validates and returns a single wrapped error.
// If there are no validation errors, nil is returned.
func (v *Validator) ValidateOrError(cfg *Config) error {
	errs := v.Validate(cfg)
	if len(errs) == 0 {
		return nil
	}

	// Combine all errors into a single message
	msgs := make([]string, len(errs))
	for i, err := range errs {
		msgs[i] = err.Error()
	}

	return errors.New(errors.Configuration, strings.Join(msgs, "; ")).
		WithOp("config.Validate")
}

// IsValid returns true if the configuration is valid.
func (v *Validator) IsValid(cfg *Config) bool {
	return len(v.Validate(cfg)) == 0
}

// isValidVersionFormat checks if a version string has a valid format.
// Valid formats: "X", "X.Y", "X.Y.Z" where X, Y, Z are numbers.
func isValidVersionFormat(version string) bool {
	if version == "" {
		return false
	}

	parts := strings.Split(version, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}

	return true
}

// ValidateField validates a single field and returns an error if invalid.
// This is useful for validating individual values before setting them.
func ValidateField(field, value string) error {
	v := NewValidator()

	switch field {
	case "log_level":
		if !v.validLogLevels[strings.ToLower(value)] {
			return &ValidationError{
				Field:   field,
				Message: fmt.Sprintf("invalid log level %q", value),
			}
		}
	case "cuda_version", "driver_version":
		if value != "" && !isValidVersionFormat(value) {
			return &ValidationError{
				Field:   field,
				Message: fmt.Sprintf("invalid version format: %s", value),
			}
		}
	}

	return nil
}
