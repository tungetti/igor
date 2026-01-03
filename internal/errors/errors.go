// Package errors provides custom error types and utilities for the Igor application.
// It implements structured errors with error codes, operation context, and proper
// support for Go's error wrapping with errors.Is() and errors.As().
package errors

import (
	"errors"
	"fmt"
)

// Code represents error categories for classifying different types of failures.
type Code int

const (
	// Unknown indicates an unclassified error.
	Unknown Code = iota
	// DistroDetection indicates a failure to detect the Linux distribution.
	DistroDetection
	// GPUDetection indicates a failure to detect NVIDIA GPU hardware.
	GPUDetection
	// PackageManager indicates a package manager operation failure.
	PackageManager
	// Installation indicates a driver installation failure.
	Installation
	// Permission indicates insufficient permissions for an operation.
	Permission
	// Network indicates a network-related failure.
	Network
	// Configuration indicates a configuration error.
	Configuration
	// Validation indicates a validation failure.
	Validation
	// Execution indicates a command execution failure.
	Execution
	// Timeout indicates an operation exceeded its time limit.
	Timeout
	// NotFound indicates a required resource was not found.
	NotFound
	// AlreadyExists indicates a resource already exists.
	AlreadyExists
	// Unsupported indicates an unsupported operation or platform.
	Unsupported
)

// String returns the string representation of the error code.
func (c Code) String() string {
	switch c {
	case Unknown:
		return "Unknown"
	case DistroDetection:
		return "DistroDetection"
	case GPUDetection:
		return "GPUDetection"
	case PackageManager:
		return "PackageManager"
	case Installation:
		return "Installation"
	case Permission:
		return "Permission"
	case Network:
		return "Network"
	case Configuration:
		return "Configuration"
	case Validation:
		return "Validation"
	case Execution:
		return "Execution"
	case Timeout:
		return "Timeout"
	case NotFound:
		return "NotFound"
	case AlreadyExists:
		return "AlreadyExists"
	case Unsupported:
		return "Unsupported"
	default:
		return fmt.Sprintf("Code(%d)", c)
	}
}

// Error represents a structured application error with code, message,
// operation context, and optional cause for error chaining.
type Error struct {
	Code    Code   // Error category
	Message string // Human-readable error message
	Op      string // Operation that failed (e.g., "gpu.Detect")
	Cause   error  // Underlying error, if any
}

// New creates a new Error with the given code and message.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Newf creates a new Error with a formatted message.
func Newf(code Code, format string, args ...interface{}) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap wraps an existing error with additional context.
func Wrap(code Code, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

// Wrapf wraps an existing error with a formatted message.
func Wrapf(code Code, cause error, format string, args ...interface{}) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...), Cause: cause}
}

// WithOp adds operation context to the error and returns the modified error.
// This allows for fluent chaining: errors.New(...).WithOp("operation").
func (e *Error) WithOp(op string) *Error {
	e.Op = op
	return e
}

// Error implements the error interface.
// The format varies based on whether Op and Cause are set:
//   - With Op and Cause: "op: message: cause"
//   - With Op only: "op: message"
//   - With Cause only: "message: cause"
//   - Message only: "message"
func (e *Error) Error() string {
	if e.Op != "" {
		if e.Cause != nil {
			return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Cause)
		}
		return fmt.Sprintf("%s: %s", e.Op, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error for errors.Is/As support.
// This enables error chain traversal with the standard library.
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is checks if the target error matches this error's code.
// This enables errors.Is() to match errors by their code.
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// GetCode extracts the error code from an error.
// Returns Unknown if the error is not an *Error type.
func GetCode(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return Unknown
}

// IsCode checks if an error has a specific code.
// This is a convenience function for common error code checks.
func IsCode(err error, code Code) bool {
	return GetCode(err) == code
}

// Sentinel errors for common cases.
// These can be used directly or wrapped with additional context.
var (
	// ErrNotRoot indicates the application must be run with root privileges.
	ErrNotRoot = New(Permission, "must be run as root")
	// ErrNoGPU indicates no NVIDIA GPU was detected on the system.
	ErrNoGPU = New(GPUDetection, "no NVIDIA GPU detected")
	// ErrUnsupportedOS indicates the operating system is not supported.
	ErrUnsupportedOS = New(Unsupported, "unsupported operating system")
	// ErrTimeout indicates an operation exceeded its allowed time.
	ErrTimeout = New(Timeout, "operation timed out")
	// ErrCancelled indicates an operation was cancelled by the user.
	ErrCancelled = New(Unknown, "operation cancelled")
)
