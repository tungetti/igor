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
├── cmd/igor/          # Application entry point
├── internal/          # Private application code
│   ├── app/           # Application orchestration
│   ├── cli/           # CLI parsing
│   ├── config/        # Configuration management
│   ├── constants/     # Application constants
│   ├── distro/        # Linux distribution detection
│   ├── errors/        # Custom error types
│   ├── exec/          # Command execution
│   ├── gpu/           # GPU detection
│   ├── install/       # Driver installation
│   ├── logging/       # Logging utilities
│   ├── pkg/           # Package manager abstraction
│   ├── privilege/     # Privilege escalation
│   ├── recovery/      # Rollback mechanisms
│   ├── testing/       # Test utilities
│   ├── ui/            # TUI components
│   └── uninstall/     # Driver uninstallation
├── pkg/               # Public library code
├── scripts/           # Build and utility scripts
└── Makefile           # Build automation
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
