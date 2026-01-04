# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [4.1.0] - 2026-01-04

### Added
- Base TUI application structure (`internal/ui`)
- Model struct implementing tea.Model interface with ViewState, dimensions, Ready, Quitting, Error
- ViewState enum: Welcome, Detecting, SystemInfo, DriverSelection, Confirmation, Installing, Complete, Error
- New() and NewWithContext() constructors with context cancellation support
- Init() returns tea.EnterAltScreen command
- Update() handles WindowSizeMsg, KeyMsg, QuitMsg, ErrorMsg, NavigateMsg, WindowReadyMsg
- View() renders placeholder views for each state
- KeyMap with vim-style navigation (hjkl) plus arrow keys, implements help.KeyMap interface
- Message types: QuitMsg, ErrorMsg, NavigateMsg, WindowReadyMsg, TickMsg, StatusMsg, ProgressMsg
- Command constructors: Navigate(), ReportError(), Quit(), SendProgress(), SendStatus()
- Helper methods: Context(), Shutdown(), IsReady(), IsQuitting(), NavigateTo()
- 98.5% test coverage
- **PHASE 4 STARTED!**

## [3.7.0] - 2026-01-04

### Added
- GPU detection orchestrator (`internal/gpu`)
- Orchestrator interface with DetectAll, DetectGPUs, GetDriverStatus, ValidateSystem, IsReadyForInstall
- GPUInfo struct aggregating all detection results (PCIDevices, NVIDIAGPUs, DriverInfo, NouveauStatus, KernelInfo, ValidationReport)
- NVIDIAGPUInfo combining hardware detection with database lookup
- DriverInfo struct for installed driver status
- Concurrent detection with graceful partial failure handling
- Integration of all Phase 3 components (pci, nvidia, smi, nouveau, kernel, validator)
- Shared types in `internal/gpu/types.go`
- 95.8% test coverage
- **PHASE 3 COMPLETE!**

## [3.6.0] - 2026-01-04

### Added
- System requirements validator (`internal/gpu/validator`)
- Validator interface with Validate, ValidateKernel, ValidateDiskSpace, ValidateSecureBoot
- ValidateKernelHeaders, ValidateBuildTools, ValidateNouveauStatus methods
- CheckResult struct with Name, Passed, Message, Severity, Remediation
- ValidationReport with aggregated Errors, Warnings, Infos
- Severity levels: Error, Warning, Info
- Configurable thresholds for disk space, kernel version, required tools
- Integration with kernel.Detector and nouveau.Detector
- Remediation suggestions for all failed checks
- 97.7% test coverage

## [3.5.0] - 2026-01-04

### Added
- Kernel version and module detection (`internal/gpu/kernel`)
- Detector interface with GetKernelInfo, IsModuleLoaded, GetLoadedModules, GetModule
- AreHeadersInstalled, GetHeadersPackage, IsSecureBootEnabled methods
- KernelInfo struct with Version, Release, Architecture, HeadersPath, HeadersInstalled, SecureBootEnabled
- ModuleInfo struct with Name, Size, UsedBy, UsedCount, State
- /proc/modules parsing with comprehensive error handling
- Distribution-aware kernel headers package names (Debian, RHEL, Arch, SUSE)
- Secure Boot detection via mokutil and EFI variables fallback
- 92.2% test coverage

## [3.4.0] - 2026-01-04

### Added
- Nouveau driver detector (`internal/gpu/nouveau`)
- Detector interface with Detect, IsLoaded, IsBlacklisted, GetBoundDevices
- Status struct with Loaded, InUse, BoundDevices, BlacklistExists, BlacklistFiles
- Module detection via /sys/module/nouveau
- Bound device detection via pci.Scanner integration
- Blacklist detection scanning /etc/modprobe.d/*.conf files
- FileSystem abstraction for testability
- 94.9% test coverage

## [3.3.0] - 2026-01-04

### Added
- nvidia-smi parser for runtime GPU detection (`internal/gpu/smi`)
- Parser interface with Parse, IsAvailable, GetDriverVersion, GetCUDAVersion, GetGPUCount
- SMIInfo struct with driver version, CUDA version, and GPU list
- SMIGPUInfo struct with 13 fields (memory, temperature, power, utilization, etc.)
- Robust CSV parsing with quoted field handling
- Error handling for nvidia-smi not found, driver not loaded, no devices
- Helper methods: MemoryUsagePercent, PowerUsagePercent, IsIdle, TotalMemory
- Uses exec.Executor for testability
- 93.7% test coverage

## [3.2.0] - 2026-01-04

### Added
- NVIDIA GPU database with 54 GPU models (`internal/gpu/nvidia`)
- Database interface with Lookup, LookupByName, ListByArchitecture, GetMinDriverVersion
- Architecture constants: Blackwell, Hopper, Ada Lovelace, Ampere, Turing, Pascal, Maxwell, Kepler, Volta
- Consumer GPUs: RTX 40xx, 30xx, 20xx series, GTX 16xx, 10xx series
- Data center GPUs: B200, B100, H200, H100, A100, A40, A30, A10, V100
- Minimum driver version requirements per architecture
- Compute capability mappings
- Thread-safe database implementation
- 100% test coverage with duplicate ID detection

## [3.1.0] - 2026-01-04

### Added
- PCI device scanner for GPU detection (`internal/gpu/pci`)
- Scanner interface with ScanAll, ScanByVendor, ScanByClass, ScanNVIDIA methods
- PCIDevice struct with Address, VendorID, DeviceID, Class, SubVendorID, SubDeviceID, Driver, Revision
- FileSystem abstraction for testability
- Helper methods: IsNVIDIA(), IsGPU(), IsNVIDIAGPU(), HasDriver(), IsUsingProprietaryDriver(), IsUsingNouveau(), IsUsingVFIO()
- Constants for NVIDIA vendor ID (0x10de) and GPU class codes (0300, 0302, 0380)
- 96.2% test coverage
- **Phase 3 Started!**

## [2.9.0] - 2026-01-03

### Added
- Distribution-specific NVIDIA package mappings
- Component types: driver, driver-dkms, cuda, cudnn, nvcc, utils, settings, opencl, vulkan
- Package mappings for Debian/Ubuntu, Fedora/RHEL, Arch, openSUSE families
- Distribution-specific overrides (Ubuntu, Pop!_OS, Fedora, Manjaro, Tumbleweed, Leap)
- Version-specific driver packages (550, 545, 535 LTS, 525, 470 legacy)
- Repository URLs with GPG keys for all distributions
- GetAllPackages, GetMinimalPackages, GetDevelopmentPackages, GetGraphicsPackages helpers
- 97.9% test coverage
- **Phase 2 Complete!**

## [2.8.0] - 2026-01-03

### Added
- Package manager factory for automatic distribution detection
- Factory.Create() auto-detects distribution and returns correct manager
- Factory.CreateForFamily() for explicit family selection
- Factory.CreateForDistribution() for precise version-based selection
- Handles YUM vs DNF for CentOS/RHEL 7 vs 8+
- Fedora always returns DNF regardless of version
- AvailableManagers() and SupportedFamilies() helper functions
- 97% test coverage

## [2.7.0] - 2026-01-03

### Added
- Zypper package manager implementation for openSUSE
- Full pkg.Manager interface implementation for Zypper
- Supports openSUSE Leap, Tumbleweed, and SLES
- Uses --non-interactive for unattended operation
- Uses dist-upgrade for full system upgrade
- NVIDIA repository support for Tumbleweed and Leap versions
- Repository management via zypper addrepo/modifyrepo
- rpm -q integration for package queries (shared with DNF/YUM)
- 90.7% test coverage

## [2.6.0] - 2026-01-03

### Added
- Pacman package manager implementation for Arch Linux and derivatives
- Full pkg.Manager interface implementation for Pacman
- Supports Arch, Manjaro, EndeavourOS, Garuda, Artix
- Uses pacman -S/-R/-Q with --noconfirm for non-interactive operation
- Repository management via pacman.conf
- GPG key management via pacman-key (--recv-keys, --lsign-key)
- Orphan package removal via pacman -Qdtq
- 90.6% test coverage

## [2.5.0] - 2026-01-03

### Added
- YUM package manager implementation for CentOS 7/RHEL 7
- Full pkg.Manager interface implementation for YUM
- Correct handling of yum check-update exit code 100
- Uses yum update (not upgrade) per YUM conventions
- Uses yum-config-manager for repository management
- EPEL repository support with RHEL fallback URL
- ELRepo repository support for kernel modules
- NVIDIA CUDA repository support
- RPM Fusion EL7 repository support
- Fallback to package-cleanup for older YUM versions
- 87% test coverage

## [2.4.0] - 2026-01-03

### Added
- DNF package manager implementation for Fedora/RHEL 8+/Rocky/AlmaLinux
- Full pkg.Manager interface implementation for DNF
- Correct handling of dnf check-update exit code 100 (updates available)
- rpm -q integration for package status checks
- RPM Fusion repository support (free and nonfree)
- RPM Fusion EL support for RHEL/Rocky/AlmaLinux with EPEL
- GPG key import via rpm --import
- dnf config-manager integration for repository management
- 93.8% test coverage

## [2.3.0] - 2026-01-03

### Added
- APT package manager implementation for Debian/Ubuntu
- Full pkg.Manager interface implementation (Install, Remove, Update, Upgrade, Search, Info)
- dpkg-query integration for package status checks
- apt-cache integration for package search and info
- PPA support via add-apt-repository
- Modern GPG key handling (/etc/apt/keyrings) with legacy apt-key fallback
- Repository management (Add, Remove, List, Enable, Disable)
- Non-interactive mode with DEBIAN_FRONTEND=noninteractive
- Input validation to prevent shell injection in GPG key handling
- 94% test coverage

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
