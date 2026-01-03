# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
