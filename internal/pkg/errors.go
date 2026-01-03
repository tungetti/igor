package pkg

import (
	"errors"
	"fmt"

	igorerrors "github.com/tungetti/igor/internal/errors"
)

// Package manager specific sentinel errors.
// These errors can be checked using errors.Is() and provide
// specific context for package manager operations.
var (
	// ErrPackageNotFound indicates the requested package does not exist in any repository.
	ErrPackageNotFound = &PackageError{
		code:    igorerrors.NotFound,
		message: "package not found",
	}

	// ErrPackageInstalled indicates the package is already installed.
	ErrPackageInstalled = &PackageError{
		code:    igorerrors.AlreadyExists,
		message: "package already installed",
	}

	// ErrPackageNotInstalled indicates the package is not installed.
	ErrPackageNotInstalled = &PackageError{
		code:    igorerrors.NotFound,
		message: "package not installed",
	}

	// ErrRepositoryExists indicates the repository already exists.
	ErrRepositoryExists = &PackageError{
		code:    igorerrors.AlreadyExists,
		message: "repository already exists",
	}

	// ErrRepositoryNotFound indicates the repository does not exist.
	ErrRepositoryNotFound = &PackageError{
		code:    igorerrors.NotFound,
		message: "repository not found",
	}

	// ErrUpdateFailed indicates the package database update failed.
	ErrUpdateFailed = &PackageError{
		code:    igorerrors.PackageManager,
		message: "package update failed",
	}

	// ErrInstallFailed indicates the package installation failed.
	ErrInstallFailed = &PackageError{
		code:    igorerrors.Installation,
		message: "package installation failed",
	}

	// ErrRemoveFailed indicates the package removal failed.
	ErrRemoveFailed = &PackageError{
		code:    igorerrors.PackageManager,
		message: "package removal failed",
	}

	// ErrLockAcquireFailed indicates the package manager lock could not be acquired.
	// This typically happens when another package manager instance is running.
	ErrLockAcquireFailed = &PackageError{
		code:    igorerrors.PackageManager,
		message: "failed to acquire package manager lock",
	}

	// ErrDependencyConflict indicates a dependency conflict prevents the operation.
	ErrDependencyConflict = &PackageError{
		code:    igorerrors.PackageManager,
		message: "dependency conflict",
	}

	// ErrGPGVerificationFailed indicates GPG signature verification failed.
	ErrGPGVerificationFailed = &PackageError{
		code:    igorerrors.Validation,
		message: "GPG verification failed",
	}

	// ErrUnsupportedOperation indicates the operation is not supported by this package manager.
	ErrUnsupportedOperation = &PackageError{
		code:    igorerrors.Unsupported,
		message: "operation not supported",
	}

	// ErrNetworkUnavailable indicates network access is required but unavailable.
	ErrNetworkUnavailable = &PackageError{
		code:    igorerrors.Network,
		message: "network unavailable",
	}

	// ErrInsufficientSpace indicates there is not enough disk space for the operation.
	ErrInsufficientSpace = &PackageError{
		code:    igorerrors.PackageManager,
		message: "insufficient disk space",
	}
)

// PackageError is a package manager specific error type that integrates
// with the igor error system while providing package-specific context.
type PackageError struct {
	code        igorerrors.Code
	message     string
	packageName string
	cause       error
	op          string
}

// Error implements the error interface.
func (e *PackageError) Error() string {
	var result string

	if e.op != "" {
		result = e.op + ": "
	}

	result += e.message

	if e.packageName != "" {
		result += fmt.Sprintf(" [%s]", e.packageName)
	}

	if e.cause != nil {
		result += ": " + e.cause.Error()
	}

	return result
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *PackageError) Unwrap() error {
	return e.cause
}

// Is checks if the target error matches this error.
// Two PackageErrors match if they have the same message (indicating the same sentinel).
func (e *PackageError) Is(target error) bool {
	var t *PackageError
	if errors.As(target, &t) {
		return e.message == t.message
	}
	return false
}

// Code returns the igor error code associated with this error.
func (e *PackageError) Code() igorerrors.Code {
	return e.code
}

// PackageName returns the package name associated with this error, if any.
func (e *PackageError) PackageName() string {
	return e.packageName
}

// WithPackage returns a new error with the specified package name.
func (e *PackageError) WithPackage(name string) *PackageError {
	return &PackageError{
		code:        e.code,
		message:     e.message,
		packageName: name,
		cause:       e.cause,
		op:          e.op,
	}
}

// WithOp returns a new error with the specified operation context.
func (e *PackageError) WithOp(op string) *PackageError {
	return &PackageError{
		code:        e.code,
		message:     e.message,
		packageName: e.packageName,
		cause:       e.cause,
		op:          op,
	}
}

// WithCause returns a new error with the specified underlying cause.
func (e *PackageError) WithCause(cause error) *PackageError {
	return &PackageError{
		code:        e.code,
		message:     e.message,
		packageName: e.packageName,
		cause:       cause,
		op:          e.op,
	}
}

// Wrap creates a new PackageError wrapping an existing error.
func Wrap(sentinel *PackageError, cause error) *PackageError {
	return &PackageError{
		code:        sentinel.code,
		message:     sentinel.message,
		packageName: sentinel.packageName,
		cause:       cause,
		op:          sentinel.op,
	}
}

// WrapWithPackage creates a new PackageError wrapping an existing error with package context.
func WrapWithPackage(sentinel *PackageError, pkgName string, cause error) *PackageError {
	return &PackageError{
		code:        sentinel.code,
		message:     sentinel.message,
		packageName: pkgName,
		cause:       cause,
		op:          sentinel.op,
	}
}

// NewPackageError creates a new PackageError with the given code and message.
func NewPackageError(code igorerrors.Code, message string) *PackageError {
	return &PackageError{
		code:    code,
		message: message,
	}
}

// NewPackageErrorf creates a new PackageError with a formatted message.
func NewPackageErrorf(code igorerrors.Code, format string, args ...interface{}) *PackageError {
	return &PackageError{
		code:    code,
		message: fmt.Sprintf(format, args...),
	}
}

// IsPackageError checks if an error is a PackageError.
func IsPackageError(err error) bool {
	var pe *PackageError
	return errors.As(err, &pe)
}

// GetPackageError extracts a PackageError from an error chain.
// Returns nil if no PackageError is found.
func GetPackageError(err error) *PackageError {
	var pe *PackageError
	if errors.As(err, &pe) {
		return pe
	}
	return nil
}

// ToIgorError converts a PackageError to an igor Error for consistent error handling.
func (e *PackageError) ToIgorError() *igorerrors.Error {
	return igorerrors.Wrap(e.code, e.message, e.cause).WithOp(e.op)
}
