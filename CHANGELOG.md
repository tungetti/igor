# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.2.0] - 2026-01-03

### Added
- Distribution detection core with /etc/os-release parsing
- 8-level fallback chain for distribution detection
- Support for 15+ Linux distributions (Ubuntu, Debian, Fedora, RHEL, Rocky, AlmaLinux, CentOS, Arch, Manjaro, EndeavourOS, openSUSE, Linux Mint, Pop!_OS)
- Distribution family detection (Debian, RHEL, Arch, SUSE)
- FileReader interface for mockable filesystem access
- Distribution helper methods (IsDebian, IsRHEL, IsArch, IsSUSE, MajorVersion, IsRolling)
- 96% test coverage

## [2.1.0] - 2026-01-03

### Added
- Package manager interface (Manager) for cross-distribution support
- Package and Repository types with helper methods
- Extended interfaces: RepositoryManager, LockableManager, TransactionalManager, HistoryManager
- Install/Update/Remove/Search options types with factory functions
- Package manager error types with errors.Is/As support
- Integration with internal/errors package
- 100% test coverage

## [1.9.0] - 2026-01-03

### Added
- Application bootstrap and lifecycle management
- Dependency injection container (Config, Logger, Executor, Privilege)
- Signal handling (SIGINT, SIGTERM) with graceful shutdown
- Panic recovery with stack trace logging
- LIFO shutdown order for cleanup functions
- 93% test coverage

### Changed
- Updated main.go to optionally use app package

## [1.8.0] - 2026-01-03

### Added
- Command executor interface with output capture
- Execute, ExecuteElevated, ExecuteWithInput, Stream methods
- Result struct with stdout, stderr, exit code, duration
- MockExecutor for testing with call recording
- Integration with privilege manager
- Context support for timeout/cancellation
- 98.4% test coverage

## [1.7.0] - 2026-01-03

### Added
- Root privilege handler with sudo/pkexec/doas support
- Root detection using os.Geteuid()
- Environment sanitization for security
- Non-interactive sudo mode (-n flag)
- Safe PATH enforcement
- Context support for timeout/cancellation
- 91% test coverage

## [1.6.0] - 2026-01-03

### Added
- CLI argument parser with subcommands
- Commands: install, uninstall, detect, list, version, help
- Global flags: --verbose, --quiet, --dry-run, --config, --no-color
- Command-specific flags for each subcommand
- Command aliases (i for install, ls for list, etc.)
- Integrated help system with per-command documentation
- 97% test coverage

## [1.5.0] - 2026-01-03

### Added
- Configuration management system with YAML support
- Environment variable overrides (IGOR_* prefix)
- XDG Base Directory compliance
- Configuration validation with detailed error messages
- Helper methods: Clone(), ConfigPath(), IsVerbose()
- 89.4% test coverage

## [1.4.0] - 2026-01-03

### Added
- Centralized logging infrastructure with charmbracelet/log
- Logger interface for mockable testing
- Log levels: Debug, Info, Warn, Error with filtering
- Thread-safe logging with mutex protection
- File logging support (TUI-safe, no colors)
- Structured logging with key-value pairs
- WithPrefix/WithFields fluent API
- NopLogger and MultiLogger implementations
- 98.9% test coverage

## [1.3.0] - 2026-01-03

### Added
- Custom error types with 14 error codes
- Error wrapping with `errors.Is()` and `errors.As()` support
- 5 sentinel errors (ErrNotRoot, ErrNoGPU, ErrUnsupportedOS, ErrTimeout, ErrCancelled)
- Typed application constants (ExitCode, DistroFamily, timeouts, paths)
- 100% test coverage for errors and constants packages

## [1.2.0] - 2026-01-03

### Added
- Core dependencies: Bubble Tea, Bubbles, Lipgloss, Log, Testify, YAML
- Import verification test

## [1.1.0] - 2026-01-03

### Added
- Initial project structure
- Go module initialization
- Basic CLI with version info
- Makefile with build targets
