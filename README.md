# Igor - NVIDIA Driver Installer for Linux

<p align="center">
  <img src="https://img.shields.io/badge/version-7.7.0-green.svg" alt="Version 7.7.0">
  <img src="https://img.shields.io/badge/go-1.21+-00ADD8.svg" alt="Go 1.21+">
  <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License">
  <img src="https://img.shields.io/badge/coverage-90%2B-brightgreen.svg" alt="Test Coverage 90%+">
  <img src="https://img.shields.io/badge/platforms-linux-lightgrey.svg" alt="Linux">
</p>

A powerful Go-based Terminal User Interface (TUI) application for installing, managing, and uninstalling NVIDIA drivers on Linux systems. Built with the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework, Igor provides an intuitive, safe, and cross-distribution solution for NVIDIA driver management.

---

## Table of Contents

- [Features](#features)
- [Supported Distributions](#supported-distributions)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Tutorial](#tutorial)
- [Command Reference](#command-reference)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

---

## Features

- **Interactive TUI**: Beautiful, intuitive terminal interface with keyboard navigation
- **Multi-Distribution Support**: Works across Debian, RHEL, Arch, and SUSE family distributions
- **Automatic GPU Detection**: Scans PCI devices and uses `lspci` for reliable GPU name resolution
- **Smart Driver Recommendations**: Suggests optimal driver based on GPU architecture
- **Safe Installation**: Built-in validation, rollback mechanisms, and recovery mode
- **CUDA Toolkit Support**: Optional installation of CUDA toolkit and cuDNN
- **Nouveau Handling**: Automatic detection and blacklisting of the Nouveau driver
- **Dry-Run Mode**: Preview changes before applying them
- **Uninstallation**: Clean removal with optional configuration purge
- **Recovery Mode**: TTY-compatible interface for when X.org fails to start
- **90%+ Test Coverage**: Comprehensive test suite ensuring reliability

---

## Supported Distributions

| Family | Distributions | Package Manager |
|--------|---------------|-----------------|
| **Debian** | Ubuntu, Debian, Linux Mint, Pop!_OS, Elementary OS | APT |
| **RHEL** | Fedora, CentOS, RHEL, Rocky Linux, AlmaLinux | DNF/YUM |
| **Arch** | Arch Linux, Manjaro, EndeavourOS, Garuda | Pacman |
| **SUSE** | openSUSE Leap, openSUSE Tumbleweed, SLES | Zypper |

---

## Quick Start

```bash
# Download the latest release (or build from source)
wget https://github.com/tungetti/igor/releases/download/v7.7.0/igor-linux-amd64
chmod +x igor-linux-amd64
sudo mv igor-linux-amd64 /usr/local/bin/igor

# Run the interactive installer
sudo igor

# Or detect GPUs first
igor detect
```

---

## Installation

### Pre-built Binaries

Download the appropriate binary for your system from the [Releases](https://github.com/tungetti/igor/releases) page:

| Architecture | Binary |
|--------------|--------|
| x86_64 (64-bit) | `igor-linux-amd64` |
| ARM64 (64-bit) | `igor-linux-arm64` |
| x86 (32-bit) | `igor-linux-386` |

```bash
# Example for x86_64
wget https://github.com/tungetti/igor/releases/download/v7.7.0/igor-linux-amd64
chmod +x igor-linux-amd64
sudo mv igor-linux-amd64 /usr/local/bin/igor
```

### Building from Source

**Prerequisites:**
- Go 1.21 or later
- Git

```bash
# Clone the repository
git clone https://github.com/tungetti/igor.git
cd igor

# Build for current platform
make build

# Or build for all platforms
./scripts/build.sh --all

# Install system-wide (optional)
sudo cp igor /usr/local/bin/
```

---

## Tutorial

### Step 1: Check Your System

Before installing drivers, verify that Igor can detect your NVIDIA GPU:

```bash
igor detect
```

**Example output:**
```
NVIDIA GPU Detection Results
=============================

GPU #1: NVIDIA GeForce RTX 4090
  - PCI Address: 01:00.0
  - Architecture: Ada Lovelace
  - Memory: 24GB GDDR6X
  - Recommended Driver: 550.xx (Latest)

Driver Status:
  - Current Driver: Not installed
  - Nouveau: Active (will be blacklisted)

System Information:
  - Distribution: Ubuntu 24.04 LTS
  - Kernel: 6.5.0-44-generic
  - Kernel Headers: Installed
  - Secure Boot: Disabled

Validation: PASSED (0 errors, 1 warning)
  - Warning: Nouveau driver is currently active
```

For JSON output (useful for scripting):
```bash
igor detect --json
```

### Step 2: List Available Drivers

See what driver versions are available for your GPU:

```bash
igor list
```

**Example output:**
```
Available NVIDIA Drivers for GeForce RTX 4090
=============================================

  Version    Branch       Status         Notes
  -------    ------       ------         -----
  550.120    Latest       Recommended    Latest features
  550.107    Latest       Available      
  545.29     Production   Available      Stable for production
  535.183    LTS          Available      Long-term support
```

### Step 3: Install Drivers (Interactive Mode)

Run Igor with sudo to start the interactive installer:

```bash
sudo igor
```

This launches the TUI where you can:
1. **Welcome Screen**: Press `Enter` to begin or `q` to quit
2. **Detection Screen**: Automatic GPU and system scanning
3. **Driver Selection**: Choose driver version and optional components
4. **Confirmation**: Review selections before installation
5. **Progress**: Watch real-time installation progress
6. **Complete**: Reboot prompt after successful installation

**Navigation:**
- `Arrow keys` / `j/k`: Navigate options
- `Tab`: Switch between panels
- `Space`: Toggle selection
- `Enter`: Confirm selection
- `Esc`: Go back
- `q`: Quit
- `?`: Show help

### Step 4: Install Drivers (Command Line)

For automated or scriptable installations:

```bash
# Install recommended driver
sudo igor install

# Install specific driver version
sudo igor install --driver 550.120

# Install with CUDA toolkit
sudo igor install --with-cuda

# Install specific CUDA version
sudo igor install --cuda 12.4

# Force reinstall
sudo igor install --force

# Preview without making changes
sudo igor install --dry-run
```

### Step 5: Verify Installation

After rebooting, verify the installation:

```bash
# Check driver version
nvidia-smi

# Or use Igor
igor detect
```

**Expected nvidia-smi output:**
```
+-----------------------------------------------------------------------------+
| NVIDIA-SMI 550.120    Driver Version: 550.120    CUDA Version: 12.4        |
|-------------------------------+----------------------+----------------------+
| GPU  Name        Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |
| Fan  Temp  Perf  Pwr:Usage/Cap|         Memory-Usage | GPU-Util  Compute M. |
|===============================+======================+======================|
|   0  NVIDIA GeForce ...  Off  | 00000000:01:00.0 Off |                  N/A |
|  0%   45C    P8    15W / 450W |    512MiB / 24564MiB |      0%      Default |
+-------------------------------+----------------------+----------------------+
```

### Step 6: Uninstalling Drivers

To remove NVIDIA drivers:

```bash
# Remove drivers (keep configuration)
sudo igor uninstall

# Remove drivers and all configuration
sudo igor uninstall --purge
```

---

## Command Reference

### Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Enable detailed output |
| `--quiet` | `-q` | Suppress non-essential output |
| `--dry-run` | `-n` | Show what would be done without changes |
| `--config` | `-c` | Path to configuration file |
| `--log-file` | | Path to log file |
| `--log-level` | | Log level: debug, info, warn, error |
| `--no-color` | | Disable colored output |

### Commands

#### `igor` (no command)
Launch the interactive TUI installer.

```bash
sudo igor
```

#### `igor install`
Install NVIDIA drivers.

| Flag | Description |
|------|-------------|
| `--driver VERSION` | Install specific driver version |
| `--cuda VERSION` | Install specific CUDA version |
| `--with-cuda` | Install latest compatible CUDA toolkit |
| `--force`, `-f` | Force installation |
| `--skip-reboot` | Don't prompt for reboot |

**Examples:**
```bash
sudo igor install
sudo igor install --driver 550.120
sudo igor install --with-cuda
sudo igor install --driver 550.120 --cuda 12.4
sudo igor --dry-run install
```

#### `igor uninstall`
Remove NVIDIA drivers.

| Flag | Description |
|------|-------------|
| `--purge` | Also remove configuration files |
| `--keep-config` | Keep configuration (default) |

**Examples:**
```bash
sudo igor uninstall
sudo igor uninstall --purge
```

#### `igor detect`
Detect NVIDIA GPUs and system information.

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--brief`, `-b` | Show condensed output |

**Examples:**
```bash
igor detect
igor detect --json
igor detect --brief
```

#### `igor list`
List available or installed drivers.

| Flag | Description |
|------|-------------|
| `--installed` | Show only installed drivers |
| `--available` | Show available drivers (default) |
| `--json` | Output as JSON |

**Examples:**
```bash
igor list
igor list --installed
igor list --json
```

#### `igor version`
Show version information.

```bash
igor version
```

**Output:**
```
Igor v7.6.0
Build Time: 2026-01-05T12:00:00Z
Git Commit: 1c38de3
```

#### `igor help`
Show help for commands.

```bash
igor help
igor help install
igor help uninstall
```

---

## Configuration

### Configuration File

Igor follows the XDG Base Directory specification:

- **Config File**: `~/.config/igor/config.yaml`
- **Cache Directory**: `~/.cache/igor/`
- **Log Directory**: `~/.local/share/igor/logs/`

### Example Configuration

```yaml
# ~/.config/igor/config.yaml

# Logging
log_level: info          # debug, info, warn, error
log_file: ""             # Empty = stderr

# Behavior
dry_run: false           # Preview mode
verbose: false           # Detailed output
quiet: false             # Minimal output

# Timeouts
timeout: 5m              # Overall timeout
network_timeout: 60s     # Network operations
command_timeout: 2m      # Shell commands

# Installation defaults
install_cuda: false      # Default CUDA installation
driver_version: ""       # Preferred driver version
cuda_version: ""         # Preferred CUDA version

# Advanced
force_install: false     # Force installation
skip_reboot: false       # Skip reboot prompt
allow_unsigned: false    # Allow unsigned packages
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `IGOR_CONFIG` | Path to config file |
| `IGOR_LOG_LEVEL` | Override log level |
| `IGOR_DRY_RUN` | Enable dry-run mode (set to "true") |
| `IGOR_APP_MODE` | Set to "service" for daemon mode |

---

## Troubleshooting

### Common Issues

#### 1. "No NVIDIA GPU detected"

**Cause**: The PCI scanner couldn't find an NVIDIA device.

**Solutions**:
- Verify GPU is physically installed: `lspci | grep -i nvidia`
- Check if GPU is enabled in BIOS
- Ensure GPU power connectors are attached
- Run Igor with sudo for full PCI access: `sudo igor`

#### 1a. GPU shows as "Unknown GPU"

**Cause**: The GPU name couldn't be resolved from `lspci` or the internal database.

**Solutions**:
- Ensure `pciutils` package is installed (provides `lspci`):
  ```bash
  # Debian/Ubuntu
  sudo apt install pciutils
  
  # Fedora/RHEL
  sudo dnf install pciutils
  
  # Arch
  sudo pacman -S pciutils
  
  # openSUSE
  sudo zypper install pciutils
  ```
- Verify lspci can see your GPU: `lspci -D | grep -i nvidia`
- If the GPU is very new, it may not be in the pci.ids database - update it:
  ```bash
  sudo update-pciids
  ```

#### 2. "Nouveau driver is active"

**Cause**: The open-source Nouveau driver is loaded.

**Solution**: Igor will automatically blacklist Nouveau during installation. A reboot is required for changes to take effect.

Manual blacklist:
```bash
echo "blacklist nouveau" | sudo tee /etc/modprobe.d/blacklist-nouveau.conf
sudo update-initramfs -u  # Debian/Ubuntu
sudo dracut -f            # Fedora/RHEL
```

#### 3. "Kernel headers not found"

**Cause**: DKMS requires kernel headers to build modules.

**Solutions**:
```bash
# Debian/Ubuntu
sudo apt install linux-headers-$(uname -r)

# Fedora/RHEL
sudo dnf install kernel-devel-$(uname -r)

# Arch
sudo pacman -S linux-headers

# openSUSE
sudo zypper install kernel-devel
```

#### 4. "Secure Boot is enabled"

**Cause**: Secure Boot requires signed kernel modules.

**Solutions**:
1. Disable Secure Boot in BIOS (simplest)
2. Sign the NVIDIA modules with your own key (advanced)
3. Use distribution-provided signed drivers (if available)

#### 5. Installation fails at DKMS

**Cause**: Kernel module compilation failed.

**Solutions**:
- Check DKMS logs: `cat /var/lib/dkms/nvidia/*/build/make.log`
- Ensure kernel headers match running kernel
- Try a different driver version

#### 6. Black screen after installation

**Solutions**:
1. Switch to TTY: `Ctrl+Alt+F2`
2. Run Igor in recovery mode: `sudo igor`
3. Uninstall drivers: `sudo igor uninstall`
4. Reboot: `sudo reboot`

### Getting Help

```bash
# Show detailed help
igor help

# Enable verbose output
igor -v detect

# Check logs
cat ~/.local/share/igor/logs/igor.log
```

---

## Development

### Project Structure

```
igor/
├── cmd/igor/                 # Application entry point
│   ├── main.go               # Main function
│   ├── cli.go                # CLI execution
│   └── version.go            # Version variables
├── internal/
│   ├── app/                  # Application lifecycle
│   ├── cli/                  # CLI argument parsing
│   ├── config/               # Configuration management
│   ├── constants/            # Project constants
│   ├── distro/               # Distribution detection
│   ├── errors/               # Custom error types
│   ├── exec/                 # Command execution
│   ├── gpu/                  # GPU detection subsystem
│   │   ├── kernel/           # Kernel module detection
│   │   ├── nouveau/          # Nouveau driver detection
│   │   ├── nvidia/           # NVIDIA GPU database
│   │   ├── pci/              # PCI device scanner
│   │   ├── smi/              # nvidia-smi parser
│   │   └── validator/        # System validation
│   ├── install/              # Installation workflow
│   │   ├── builder/          # Workflow builder
│   │   └── steps/            # Installation steps
│   ├── logging/              # Structured logging
│   ├── pkg/                  # Package managers
│   │   ├── apt/              # APT (Debian)
│   │   ├── dnf/              # DNF (Fedora/RHEL 8+)
│   │   ├── factory/          # Package manager factory
│   │   ├── nvidia/           # NVIDIA package mappings
│   │   ├── pacman/           # Pacman (Arch)
│   │   ├── yum/              # YUM (CentOS 7/RHEL 7)
│   │   └── zypper/           # Zypper (openSUSE)
│   ├── privilege/            # Privilege escalation
│   ├── recovery/             # Recovery mode
│   ├── testing/              # Test utilities
│   ├── ui/                   # TUI components
│   │   ├── components/       # Reusable UI widgets
│   │   ├── theme/            # Theming system
│   │   └── views/            # Screen views
│   └── uninstall/            # Uninstallation workflow
│       └── steps/            # Uninstall steps
├── scripts/                  # Build scripts
├── Makefile                  # Build automation
├── go.mod                    # Go module
└── VERSION                   # Version file
```

### GPU Detection Architecture

Igor uses a multi-source approach for GPU detection:

1. **PCI Scanner** (`internal/gpu/pci/`): Scans `/sys/bus/pci/devices/` to find NVIDIA devices by vendor ID (`10de`)
2. **lspci Resolver** (`internal/gpu/pci/lspci.go`): Runs `lspci -D` to get human-readable GPU names from the system's `pci.ids` database
3. **GPU Database** (`internal/gpu/nvidia/database.go`): Internal mapping of device IDs to GPU models (fallback)
4. **nvidia-smi Parser** (`internal/gpu/smi/`): Extracts runtime information when NVIDIA drivers are loaded

**Name Resolution Priority:**
1. `lspci` name (most reliable, always current with system's pci.ids)
2. Internal GPU database (may be outdated for new GPUs)
3. nvidia-smi name (only available if driver is loaded)
4. Fallback: Device ID

### Building

```bash
# Build for current platform
make build

# Build for all platforms
./scripts/build.sh --all

# Run tests
make test

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test -v ./internal/install/...
```

### Test Coverage

| Package | Coverage |
|---------|----------|
| internal/constants | 100% |
| internal/errors | 100% |
| internal/install/builder | 98.7% |
| internal/install | 97.4% |
| internal/pkg/nvidia | 97.9% |
| internal/distro | 96.8% |
| internal/uninstall | 96.0% |
| internal/recovery | 98.0% |
| **Average** | **90%+** |

---

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes with tests
4. Ensure tests pass: `make test`
5. Commit changes: `git commit -m 'Add amazing feature'`
6. Push to branch: `git push origin feature/amazing-feature`
7. Open a Pull Request

### Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Maintain >85% test coverage for new code
- Document exported functions and types

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Author

**Tommaso Maria Ungetti**

---

## Acknowledgments

- [Charm](https://charm.sh/) for the excellent TUI libraries (Bubble Tea, Lip Gloss, Bubbles)
- The NVIDIA and Linux communities for driver documentation
- All contributors and testers

---

<p align="center">
  Made with Go and Bubble Tea
</p>
