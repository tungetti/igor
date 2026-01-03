package pkg

import (
	"context"

	"github.com/tungetti/igor/internal/constants"
)

// Manager defines the package manager interface.
// All distribution-specific implementations (APT, DNF, YUM, Pacman, Zypper)
// must implement this interface to provide a unified package management API.
//
// All methods accept a context.Context as the first parameter to support
// cancellation and timeout handling. Implementations should check for
// context cancellation and return early when appropriate.
//
// Thread Safety: Implementations should be safe for concurrent use from
// multiple goroutines.
type Manager interface {
	// Package Operations

	// Install installs one or more packages.
	// Returns ErrPackageNotFound if a package doesn't exist,
	// or ErrInstallFailed if installation fails.
	Install(ctx context.Context, opts InstallOptions, packages ...string) error

	// Remove removes one or more packages from the system.
	// Returns ErrPackageNotInstalled if a package isn't installed,
	// or ErrRemoveFailed if removal fails.
	Remove(ctx context.Context, opts RemoveOptions, packages ...string) error

	// Update updates the package database/cache.
	// This is equivalent to "apt update", "dnf check-update", etc.
	// Returns ErrUpdateFailed if the update fails.
	Update(ctx context.Context, opts UpdateOptions) error

	// Upgrade upgrades installed packages to their latest versions.
	// If packages are specified, only those packages are upgraded.
	// If no packages are specified, all upgradable packages are upgraded.
	Upgrade(ctx context.Context, opts InstallOptions, packages ...string) error

	// Query Operations

	// IsInstalled checks if a package is currently installed.
	// Returns true if installed, false otherwise.
	// Returns an error only if the check itself fails.
	IsInstalled(ctx context.Context, pkg string) (bool, error)

	// Search searches for packages matching the query.
	// The query can be a package name or description pattern.
	// Returns an empty slice if no packages match.
	Search(ctx context.Context, query string, opts SearchOptions) ([]Package, error)

	// Info returns detailed information about a specific package.
	// Returns ErrPackageNotFound if the package doesn't exist.
	Info(ctx context.Context, pkg string) (*Package, error)

	// ListInstalled returns a list of all installed packages.
	ListInstalled(ctx context.Context) ([]Package, error)

	// ListUpgradable returns a list of packages that can be upgraded.
	ListUpgradable(ctx context.Context) ([]Package, error)

	// Repository Operations

	// AddRepository adds a new package repository.
	// Returns ErrRepositoryExists if the repository already exists.
	AddRepository(ctx context.Context, repo Repository) error

	// RemoveRepository removes a package repository.
	// Returns ErrRepositoryNotFound if the repository doesn't exist.
	RemoveRepository(ctx context.Context, name string) error

	// ListRepositories returns a list of configured repositories.
	ListRepositories(ctx context.Context) ([]Repository, error)

	// EnableRepository enables a disabled repository.
	// Returns ErrRepositoryNotFound if the repository doesn't exist.
	EnableRepository(ctx context.Context, name string) error

	// DisableRepository disables an enabled repository.
	// Returns ErrRepositoryNotFound if the repository doesn't exist.
	DisableRepository(ctx context.Context, name string) error

	// RefreshRepositories refreshes all repository metadata.
	// This is typically called after adding or modifying repositories.
	RefreshRepositories(ctx context.Context) error

	// Utility Operations

	// Clean removes cached package files and temporary data.
	// This frees up disk space used by the package manager.
	Clean(ctx context.Context) error

	// AutoRemove removes automatically installed packages that are no longer needed.
	AutoRemove(ctx context.Context) error

	// Verify verifies the integrity of an installed package.
	// Returns true if the package passes verification, false otherwise.
	// Returns ErrPackageNotInstalled if the package isn't installed.
	Verify(ctx context.Context, pkg string) (bool, error)

	// Metadata

	// Name returns the package manager name (e.g., "apt", "dnf", "pacman").
	Name() string

	// Family returns the distribution family this manager supports.
	Family() constants.DistroFamily

	// IsAvailable checks if this package manager is available on the system.
	// This should check for the presence of the package manager binary.
	IsAvailable() bool
}

// ManagerFactory creates Manager instances based on the detected distribution.
type ManagerFactory interface {
	// Create creates a Manager for the given distribution family.
	// Returns an error if the distribution family is not supported.
	Create(family constants.DistroFamily) (Manager, error)

	// Detect automatically detects the appropriate Manager for the current system.
	// Returns an error if no suitable package manager is found.
	Detect() (Manager, error)
}

// RepositoryManager provides additional repository management capabilities.
// Implementations that support advanced repository features should implement this.
type RepositoryManager interface {
	Manager

	// ImportGPGKey imports a GPG key for repository verification.
	ImportGPGKey(ctx context.Context, keyURL string) error

	// RemoveGPGKey removes a GPG key.
	RemoveGPGKey(ctx context.Context, keyID string) error

	// ListGPGKeys lists all imported GPG keys.
	ListGPGKeys(ctx context.Context) ([]string, error)
}

// LockableManager provides package manager lock management.
// Some package managers (like APT) use locks to prevent concurrent operations.
type LockableManager interface {
	Manager

	// AcquireLock acquires the package manager lock.
	// Returns ErrLockAcquireFailed if the lock cannot be acquired.
	AcquireLock(ctx context.Context) error

	// ReleaseLock releases the package manager lock.
	ReleaseLock(ctx context.Context) error

	// IsLocked checks if the package manager is currently locked.
	IsLocked(ctx context.Context) (bool, error)

	// WaitForLock waits until the package manager lock becomes available.
	// The context can be used to set a timeout for waiting.
	WaitForLock(ctx context.Context) error
}

// TransactionalManager provides transaction support for package operations.
// Implementations that support atomic/reversible operations should implement this.
type TransactionalManager interface {
	Manager

	// BeginTransaction starts a new transaction.
	BeginTransaction(ctx context.Context) error

	// CommitTransaction commits the current transaction.
	CommitTransaction(ctx context.Context) error

	// RollbackTransaction rolls back the current transaction.
	RollbackTransaction(ctx context.Context) error

	// InTransaction returns true if a transaction is currently active.
	InTransaction() bool
}

// HistoryManager provides package operation history.
type HistoryManager interface {
	Manager

	// History returns the package manager operation history.
	History(ctx context.Context, limit int) ([]HistoryEntry, error)

	// Undo undoes a previous operation by ID.
	Undo(ctx context.Context, id string) error

	// Redo redoes a previously undone operation by ID.
	Redo(ctx context.Context, id string) error
}

// HistoryEntry represents a single entry in the package manager history.
type HistoryEntry struct {
	// ID is the unique identifier for this history entry.
	ID string

	// Timestamp is when the operation occurred.
	Timestamp int64

	// Operation is the type of operation (install, remove, upgrade, etc.).
	Operation string

	// Packages lists the packages affected by this operation.
	Packages []string

	// Success indicates whether the operation succeeded.
	Success bool

	// Details contains additional information about the operation.
	Details string
}

// String returns a human-readable representation of the history entry.
func (h HistoryEntry) String() string {
	status := "failed"
	if h.Success {
		status = "success"
	}
	return h.Operation + " [" + status + "]"
}
