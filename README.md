# Igor - NVIDIA TUI Installer

A powerful Go-based Terminal User Interface (TUI) application for installing NVIDIA drivers on Linux systems.

## Overview

Igor simplifies the often complex process of installing NVIDIA drivers on Linux by providing an intuitive terminal-based interface. It handles driver detection, system compatibility checks, package management, and installation with full rollback capabilities.

## Features

- **Interactive TUI**: Beautiful terminal interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Multi-Distribution Support**: Works across major Linux distributions
- **GPU Detection**: Automatic detection of NVIDIA graphics cards
- **Driver Management**: Install, update, and uninstall NVIDIA drivers
- **Safe Installation**: Built-in recovery and rollback mechanisms
- **Privilege Handling**: Secure sudo/root privilege management

## Supported Distributions

- Ubuntu / Debian
- Fedora / Red Hat / CentOS / Rocky Linux / AlmaLinux
- Arch Linux / Manjaro
- openSUSE

## Quick Start

### Prerequisites

- Go 1.21 or later (for building from source)
- Linux operating system
- NVIDIA GPU
- Root/sudo access for driver installation

### Installation

#### From Source

```bash
git clone https://github.com/tommasomariaungetti/igor.git
cd igor
make build
sudo ./igor
```

#### Pre-built Binaries

Pre-built binaries will be available in future releases.

## Usage

```bash
# Run the installer (requires root privileges)
sudo ./igor

# Show version information
./igor --version

# Show help
./igor --help
```

## Current Status

**Version:** 3.5.0 | **Phase:** 3 of 7 | **Progress:** 23 of 62 sprints complete

### Completed Phases
- **Phase 1:** Project Foundation & Architecture (9 sprints) - CLI, config, logging, errors, exec, privilege
- **Phase 2:** Distribution Detection & Package Manager Abstraction (9 sprints) - APT, DNF, YUM, Pacman, Zypper
- **Phase 3:** GPU Detection & System Analysis (5 of 7 sprints) - PCI scanner, GPU database, nvidia-smi, Nouveau, kernel

### In Progress
- **P3-MS6:** System Requirements Validator
- **P3-MS7:** GPU Detection Orchestrator

### Test Coverage
Average >90% across all packages (23 packages, 100+ test files)

## Development

### Building

```bash
# Build for current platform
make build

# Run tests
make test

# Clean build artifacts
make clean
```

### Project Structure

```
igor/
├── cmd/igor/              # Application entry point
├── internal/              # Private application code
│   ├── app/               # Application orchestration and lifecycle
│   ├── cli/               # CLI argument parsing with subcommands
│   ├── config/            # YAML configuration with env overrides
│   ├── constants/         # Typed constants and exit codes
│   ├── distro/            # Linux distribution detection (8-level fallback)
│   ├── errors/            # Custom error types with codes
│   ├── exec/              # Command execution with mocking support
│   ├── gpu/               # GPU detection subsystem
│   │   ├── kernel/        # Kernel version and module detection
│   │   ├── nouveau/       # Nouveau driver detector
│   │   ├── nvidia/        # NVIDIA GPU database (54 models)
│   │   ├── pci/           # PCI device scanner
│   │   └── smi/           # nvidia-smi parser
│   ├── install/           # Driver installation (planned)
│   ├── logging/           # Structured logging with levels
│   ├── pkg/               # Package manager abstraction
│   │   ├── apt/           # APT (Debian/Ubuntu)
│   │   ├── dnf/           # DNF (Fedora/RHEL 8+)
│   │   ├── factory/       # Auto-detection factory
│   │   ├── nvidia/        # NVIDIA package mappings
│   │   ├── pacman/        # Pacman (Arch)
│   │   ├── yum/           # YUM (CentOS 7/RHEL 7)
│   │   └── zypper/        # Zypper (openSUSE)
│   ├── privilege/         # Sudo/pkexec privilege handling
│   ├── recovery/          # Rollback mechanisms (planned)
│   ├── testing/           # Test utilities (planned)
│   ├── ui/                # TUI components (planned)
│   └── uninstall/         # Driver uninstallation (planned)
├── pkg/                   # Public library code
├── scripts/               # Build and utility scripts
└── Makefile               # Build automation
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
./scripts/test.sh
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Author

Tommaso Maria Ungetti

## Acknowledgments

- [Charm](https://charm.sh/) for the excellent TUI libraries
- The Linux community for driver documentation and support
