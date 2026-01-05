# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [6.7.0] - 2026-01-05

### Added
- Uninstall TUI views (`internal/ui/views/uninstall_*.go`)
- UninstallConfirmModel: Confirmation view showing what will be removed
  - Displays driver version, packages, config files to remove
  - Shows nouveau restoration option and warnings
  - Confirm/Cancel buttons with keyboard navigation
  - Key bindings: enter/y confirm, esc/n cancel, left/right navigation
- UninstallProgressModel: Progress view with step-by-step tracking
  - Progress bar with percentage display
  - Step status indicators: ○ pending, ● running, ✓ complete, ✗ failed, - skipped
  - Log output with scrolling
  - Elapsed time display
  - Handles completion, failure, and cancellation states
- Message types for uninstall workflow:
  - UninstallConfirmedMsg, UninstallCancelledMsg
  - UninstallStepStartedMsg, UninstallStepCompletedMsg
  - UninstallLogMsg, UninstallCompleteMsg
  - NavigateToUninstallCompleteMsg, NavigateToUninstallErrorMsg
- UninstallStep struct for tracking step progress
- 98.5% test coverage

## [6.6.0] - 2026-01-05

### Added
- Fallback driver restoration step (`internal/uninstall/steps/nouveau.go`)
- NouveauRestoreStep restores nouveau open-source driver after NVIDIA uninstallation
- Functional options:
  - WithRemoveNouveauBlacklist: Remove blacklist files (default: true)
  - WithLoadNouveauModule: Load nouveau kernel module (default: true)
  - WithRegenerateInitramfs: Regenerate initramfs (default: true)
  - WithNouveauKernelDetector: Custom kernel detector
  - WithBlacklistPaths: Custom blacklist file paths
- Distribution-specific initramfs commands:
  - Debian: update-initramfs -u
  - RHEL/SUSE: dracut --force
  - Arch: mkinitcpio -P
- State tracking: StateNouveauRestored, StateNouveauBlacklistRemoved, StateInitramfsRegenerated, StateNouveauModuleLoaded
- Rollback support: re-creates blacklist, unloads nouveau module
- Graceful degradation when module loading fails
- 91.1% test coverage

## [6.5.0] - 2026-01-05

### Added
- Configuration cleanup step (`internal/uninstall/steps/config.go`)
- ConfigCleanupStep removes NVIDIA configuration files with backup support
- Default paths for nouveau blacklist, X.org config, modprobe config, persistence config
- Functional options:
  - WithConfigPaths: Specific paths to remove
  - WithBackupDir: Backup directory (default: /var/lib/igor/backup/configs)
  - WithCreateBackup: Enable backup before removal (default: true)
  - WithRemoveBlacklist: Remove nouveau blacklist files
  - WithRemoveXorgConf: Remove X.org nvidia config
  - WithRemoveModprobe: Remove modprobe.d nvidia configs
  - WithRemovePersistence: Remove nvidia-persistenced config
- State tracking: StateConfigsCleaned, StateCleanedConfigs, StateBackedUpConfigs, StateBackupDir
- Security: Path traversal prevention, command injection prevention, allowlist-based directory validation
- Rollback support: restores configs from backup directory
- 90.4% test coverage

## [6.4.0] - 2026-01-05

### Added
- Kernel module cleanup step (`internal/uninstall/steps/modunload.go`)
- ModuleUnloadStep unloads NVIDIA kernel modules using `modprobe -r`
- Default modules unloaded in reverse order: nvidia-modeset, nvidia-drm, nvidia-uvm, nvidia
- Functional options:
  - WithUnloadModuleNames: Specific modules to unload
  - WithSkipIfNotLoaded: Skip if not loaded (default: true)
  - WithUnloadKernelDetector: Custom kernel detector
  - WithForceUnload: Force unload with rmmod -f (dangerous)
  - WithUnloadRetry: Retry count and delay for in-use modules
- State tracking: StateModulesUnloaded, StateUnloadedModules, StateModulesInUse
- Retry logic for modules in use (default: 3 retries, 1s delay)
- Force unload fallback with rmmod -f when enabled
- Rollback support: CanRollback returns true, reloads unloaded modules
- Security: isValidModuleName validates against command injection
- Helper functions: isModuleInUse, getModuleHolders, filterLoadedModules
- 91.7% test coverage

## [6.3.0] - 2026-01-05

### Added
- Package removal step (`internal/uninstall/steps/removal.go`)
- PackageRemovalStep implementing install.Step interface
- Removes NVIDIA packages using package manager with options:
  - WithPackagesToRemove: Specific packages to remove
  - WithRemoveAll: Remove all discovered NVIDIA packages
  - WithPurge: Purge config files (apt purge vs apt remove)
  - WithAutoRemove: Remove orphaned dependencies (default: true)
  - WithRemovalBatchSize: Batch size for removal
  - WithRemovalDiscovery: Discovery instance for removeAll mode
- State tracking:
  - StatePackagesRemoved, StateRemovedPackages, StateFailedPackages, StateRemovalPurged
- Partial failure handling (continues on individual package failures)
- Dry-run mode support
- Batch processing with cancellation checks between batches
- CanRollback returns false (package removal is not reversible)
- MockDiscovery and RemovalMockManager for testing
- 98.2% test coverage

## [6.2.0] - 2026-01-05

### Added
- Installed package discovery (`internal/uninstall/discovery.go`)
- DiscoveredPackages struct with categorized package information:
  - DriverPackages, CUDAPackages, LibraryPackages, UtilityPackages
  - KernelModulePackages, ConfigPackages, AllPackages
  - DriverVersion and CUDAVersion extraction
  - Helper methods: IsEmpty(), HasDriver(), HasCUDA()
- Discovery interface for package discovery operations:
  - Discover(): Full discovery of all NVIDIA packages
  - DiscoverDriver(): Driver-only discovery with version
  - DiscoverCUDA(): CUDA-only discovery with version
  - IsNVIDIAInstalled(): Check for any NVIDIA packages
  - GetDriverVersion(): Get installed driver version
- PackageDiscovery implementation using pkg.Manager
- Distribution-specific package patterns for:
  - Debian: nvidia-driver-*, libnvidia-*, cuda-toolkit-*, nvidia-dkms
  - RHEL: nvidia-driver, kmod-nvidia, akmod-nvidia, cuda-*
  - Arch: nvidia, nvidia-lts, nvidia-open, cuda, nvidia-dkms
  - SUSE: nvidia-driver-G06-kmp-*, nvidia-video-G06, cuda-*
- Helper functions:
  - FilterNVIDIAPackages(): Filter package list to NVIDIA only
  - CategorizePackages(): Categorize by type per distribution
  - GetDriverVersionFromPackages(): Extract driver version
  - GetCUDAVersionFromPackages(): Extract CUDA version
- Functional options: WithDiscoveryDistro, WithDiscoveryExecutor
- 95.0% test coverage

## [6.1.0] - 2026-01-05

### Added
- Uninstallation workflow interface (`internal/uninstall/`)
- UninstallStatus enum: Pending, Running, Completed, Partial, Failed, Cancelled
- UninstallResult struct tracking:
  - RemovedPackages, FailedPackages, CleanedConfigs
  - CompletedSteps, FailedStep, Error
  - NeedsReboot, NouveauRestored flags
- UninstallContext with state management:
  - DistroInfo, InstalledDriver, InstalledPackages
  - PackageManager, Executor, Privilege, Logger
  - Special flags: DryRun, Force, KeepConfig
  - Thread-safe state storage with typed getters
- UninstallStep as type alias for install.Step (code reuse)
- UninstallFuncStep for function-based step implementation
- UninstallWorkflow interface with:
  - Name(), Steps(), AddStep(), Execute(), OnProgress(), Cancel()
- BaseUninstallWorkflow implementation:
  - Progress callbacks using install.StepProgress
  - State synchronization between contexts
  - Partial result tracking
- State key constants: StatePackagesRemoved, StateRemovedPackages, StateConfigsCleaned, StateCleanedConfigs, StateModulesUnloaded, StateNouveauUnblocked, StateNouveauRestored
- Functional options for context configuration
- Helper functions: SkipUninstallStep, CompleteUninstallStep, FailUninstallStep
- 94.7% test coverage
- **PHASE 6 STARTED!**

## [5.11.0] - 2026-01-05

### Added
- Distribution-specific workflow builder (`internal/install/builder/`)
- WorkflowBuilder struct for assembling installation workflows
- BuilderConfig struct with skip flags for all steps:
  - SkipValidation, SkipRepository, SkipNouveau, SkipDKMS
  - SkipModuleLoad, SkipXorgConfig, SkipVerification
  - CustomSteps for extensibility
  - ValidationChecks and RequiredDiskMB configuration
- Factory functions:
  - NewWorkflowBuilder(dist, opts...): Create builder for a distribution
  - DefaultBuilderConfig(): Returns default configuration
  - BuilderForFamily(family): Convenience function for family-based creation
- Functional options:
  - WithSkipValidation, WithSkipRepository, WithSkipNouveau, WithSkipDKMS
  - WithSkipModuleLoad, WithSkipXorgConfig, WithSkipVerification
  - WithCustomSteps, WithValidationChecks, WithRequiredDiskMB
  - WithBuilderConfig for setting entire config at once
- Build() method creates workflow with name "{distro-family}-nvidia-installation"
- Distribution-specific step ordering:
  - Debian/RHEL/SUSE: All 8 steps (validation, repository, nouveau, packages, dkms, modload, xorg, verify)
  - Arch: 7 steps (auto-skips repository - packages in official repos)
- Error handling for nil distribution and unknown family
- Helper methods: Config() returns defensive copy, Distribution() returns distro info
- 98.7% test coverage
- **PHASE 5 COMPLETE!**

## [5.10.0] - 2026-01-05

### Added
- Workflow orchestrator (`internal/install/orchestrator.go`)
- Orchestrator wraps workflows with additional orchestration features:
  - Automatic rollback on failure
  - Pre/post workflow execution hooks
  - Pre/post step execution hooks
  - Execution event logging
  - Detailed execution reporting
- ExecutionHook and StepHook function types for extensibility
- ExecutionEventType enum for 13 event types
- ExecutionEntry struct for execution event logging
- ExecutionReport struct with comprehensive execution statistics:
  - Steps executed, completed, skipped, failed counts
  - Rollback status and success indicator
  - Total duration and timing information
- Functional options:
  - WithAutoRollback: Enable automatic rollback on failure
  - WithStopOnFirstError: Stop immediately on first error
  - WithOrchestratorDryRun: Execute in dry-run mode
  - WithPreExecuteHook/WithPostExecuteHook: Workflow-level hooks
  - WithPreStepHook/WithPostStepHook: Step-level hooks
  - WithOrchestratorProgress: Progress callback
- Thread-safe implementation with proper mutex usage
- Step-level rollback tracking for proper rollback when using hooks
- 95.2% test coverage

## [5.9.0] - 2026-01-05

### Added
- Post-installation verification step (`internal/install/steps/verify.go`)
- VerificationStep implementing install.Step interface
- Verifies NVIDIA driver installation success via multiple checks:
  - nvidia-smi availability and driver version detection
  - nvidia kernel module loaded check
  - GPU detection via nvidia-smi output parsing
  - X.org configuration file existence check
- VerificationCheck struct for individual check results (Name, Description, Passed, Message, Critical)
- CustomCheckFunc type for user-defined verification functions
- Functional options:
  - WithCheckNvidiaSmi: Enable/disable nvidia-smi check
  - WithCheckModuleLoaded: Enable/disable module check
  - WithCheckGPUDetected: Enable/disable GPU detection
  - WithCheckXorgConfig: Enable/disable X.org config check
  - WithFailOnWarning: Treat non-critical failures as step failures
  - WithVerificationKernelDetector: Set kernel detector for testing
  - WithCustomCheck: Add custom verification functions
- State keys: StateVerificationPassed, StateDriverVersion, StateNvidiaSmiAvailable, StateModuleLoaded, StateGPUDetected, StateVerificationErrors
- Read-only step (no rollback capability)
- Critical vs non-critical check distinction
- Dry-run mode support
- Cancellation handling at multiple checkpoints
- 91.1% test coverage

## [5.8.0] - 2026-01-05

### Added
- X.org configuration step (`internal/install/steps/xorg.go`)
- XorgConfigStep implementing install.Step interface
- Creates NVIDIA X.org configuration at `/etc/X11/xorg.conf.d/20-nvidia.conf`:
  - OutputClass section with nvidia driver
  - AllowEmptyInitialConfiguration and PrimaryGPU options
- Display server detection (Wayland vs X.org):
  - Detects via XDG_SESSION_TYPE environment variable
  - Optional skip on Wayland sessions
- Functional options:
  - WithXorgConfigDir: Set custom config directory
  - WithXorgConfigFile: Set custom config filename
  - WithXorgConfigContent: Set custom config content
  - WithSkipIfWayland: Skip configuration on Wayland
  - WithBackupExisting: Backup existing config (default: true)
  - WithXorgFileWriter: Set file writer for testing
  - WithDisplayDetector: Set display detector for testing
- State keys: StateXorgConfigured, StateXorgConfigPath, StateXorgBackupPath, StateXorgDisplayServer
- Backs up existing configuration before overwriting
- Full rollback capability (removes config, restores backup)
- Path validation to prevent command injection and path traversal
- Dry-run mode support (directory creation and backup are dry-run aware)
- Cancellation handling at multiple checkpoints
- 90.9% test coverage

## [5.7.0] - 2026-01-05

### Added
- Module loading step (`internal/install/steps/modload.go`)
- ModuleLoadStep implementing install.Step interface
- Loads NVIDIA kernel modules via modprobe:
  - Default modules: nvidia, nvidia-uvm, nvidia-drm, nvidia-modeset
  - Check if module already loaded via kernel.Detector or lsmod fallback
  - Load modules in order, unload in reverse order for rollback
- Functional options:
  - WithModuleNames: Set specific modules to load
  - WithSkipIfLoaded: Skip if nvidia module already loaded (default: true)
  - WithModuleKernelDetector: Set kernel detector for testing
  - WithForceReload: Force unload and reload even if loaded
- State keys: StateModulesLoaded, StateLoadedModules
- Handles empty module list gracefully (returns skip)
- Full rollback capability (modprobe -r in reverse order)
- Input validation for module names (security)
- Dry-run mode support
- Cancellation handling at multiple checkpoints
- Partial failure cleanup during load
- 92.8% test coverage

## [5.6.0] - 2026-01-04

### Added
- DKMS module build step (`internal/install/steps/dkms.go`)
- DKMSBuildStep implementing install.Step interface
- Builds NVIDIA kernel modules using DKMS:
  - `dkms status nvidia` - Check if module is built
  - `dkms build nvidia/<version> -k <kernel_version>` - Build module
  - `dkms install nvidia/<version> -k <kernel_version>` - Install module
- Functional options:
  - WithModuleName: Set DKMS module name (default: "nvidia")
  - WithModuleVersion: Set specific module version
  - WithModuleVersion: Set specific kernel version
  - WithSkipStatusCheck: Skip checking if already built
  - WithKernelDetector: Set kernel detector for testing
  - WithDKMSTimeout: Set build timeout (default: 10 minutes)
- State keys: StateDKMSBuilt, StateDKMSModuleName, StateDKMSModuleVersion, StateDKMSKernelVersion, StateDKMSBuildTime
- Checks if DKMS is available, skips gracefully if not
- Auto-detects nvidia module version from dkms status
- Skips if module already built for current kernel
- Full rollback capability (dkms remove)
- Input validation for module names, versions, and kernel versions (security)
- Dry-run mode support
- Cancellation handling at multiple checkpoints
- 93.7% test coverage

## [5.5.0] - 2026-01-04

### Added
- Package installation step (`internal/install/steps/packages.go`)
- PackageInstallationStep implementing install.Step interface
- Integrates with nvidia.GetPackageSet for distribution-specific packages
- Supports version-specific driver packages via GetPackagesForVersion
- Maps component strings to nvidia.Component for package lookup
- Automatic package deduplication
- Functional options:
  - WithAdditionalPackages: Add extra packages beyond computed ones
  - WithSkipDependencies: Skip dependency checking (placeholder)
  - WithBatchSize: Install in configurable batches
  - WithPreInstallHook: Hook before installation
  - WithPostInstallHook: Hook after installation
- State keys: StatePackagesInstalled, StateInstalledPackages, StatePackageInstallTime
- Multi-distro support (Debian, RHEL, Arch, SUSE)
- Full rollback capability (removes installed packages)
- Batch installation support with cancellation checks
- Comprehensive error handling with rollback logging
- Dry-run mode support
- Cancellation handling at multiple checkpoints
- 95.9% test coverage

## [5.4.0] - 2026-01-04

### Added
- Nouveau blacklist step (`internal/install/steps/nouveau.go`)
- NouveauBlacklistStep implementing install.Step interface
- Creates `/etc/modprobe.d/blacklist-nouveau.conf` with:
  - `blacklist nouveau`
  - `options nouveau modeset=0`
- Distribution-specific initramfs regeneration:
  - Debian/Ubuntu: `update-initramfs -u`
  - Fedora/RHEL/SUSE: `dracut --force`
  - Arch: `mkinitcpio -P`
- Functional options: WithBlacklistPath, WithNouveauDetector, WithSkipInitramfs, WithFileWriter
- State keys: StateNouveauBlacklisted, StateNouveauBlacklistFile
- Uses `tee` command for privileged file writing
- Full rollback support (removes file, regenerates initramfs)
- Skips if Nouveau already blacklisted
- Dry-run mode support
- Cancellation handling at multiple checkpoints
- 98.3% test coverage

## [5.3.0] - 2026-01-04

### Added
- Repository configuration step (`internal/install/steps/repository.go`)
- RepositoryStep implementing install.Step interface
- Configures NVIDIA repositories based on detected Linux distribution
- Functional option: WithSkipUpdate for batched repository additions
- State keys: StateRepositoryConfigured, StateRepositoryName
- Distribution-specific handling:
  - Debian/Ubuntu: Graphics drivers PPA or CUDA repository
  - Fedora/RHEL: RPM Fusion nonfree repository
  - openSUSE: Official NVIDIA repository
  - Arch Linux: Skips (uses official repos)
- Automatic rollback on update failure
- Dry-run mode support
- Cancellation handling at multiple checkpoints
- Full rollback capability (removes added repository)
- 98.2% test coverage

## [5.2.0] - 2026-01-04

### Added
- Pre-installation validation step (`internal/install/steps/validation.go`)
- ValidationStep implementing install.Step interface
- ValidationCheck enum: Kernel, KernelHeaders, DiskSpace, SecureBoot, BuildTools, NouveauStatus, NVIDIAGPU
- Functional options: WithValidator, WithRequiredDiskMB, WithChecks
- Integration with existing gpu/validator.Validator
- Context state storage for validation results
- Critical checks (fail on error): Kernel, DiskSpace, BuildTools, KernelHeaders
- Warning checks (proceed with warning): Nouveau, SecureBoot
- ValidationReport struct for comprehensive results
- MockValidator for testing
- 98.0% test coverage

## [5.1.0] - 2026-01-04

### Added
- Installation workflow interface (`internal/install/`)
- StepStatus enum: Pending, Running, Completed, Failed, Skipped, RolledBack
- WorkflowStatus enum: Pending, Running, Completed, Failed, Cancelled, RollingBack, RolledBack
- Step interface with Name, Description, Execute, Rollback, CanRollback, Validate
- BaseStep and FuncStep implementations for reusable step creation
- Workflow interface with Steps, AddStep, Execute, Rollback, OnProgress, Cancel
- BaseWorkflow implementation with thread-safe state management
- Context struct with GPU info, distro info, driver version, components
- Integration with PackageManager, Executor, Privilege, Logger
- Thread-safe state storage with typed getters (GetStateString, GetStateInt, GetStateBool)
- Cancellation support via context.Context
- Dry run mode support
- Functional options pattern for Context construction
- Builder pattern for StepResult and WorkflowResult
- Helper functions: SkipStep, CompleteStep, FailStep
- 97.2% test coverage

## [4.11.0] - 2026-01-04

### Added
- View navigation and state machine (`internal/ui/app.go`)
- Full integration of all view models (welcome, detection, selection, confirmation, progress, complete, error)
- Navigation state machine handling all view transitions
- View initialization helpers for lazy view creation
- Window size propagation to all initialized views
- Spinner tick delegation to active view (detection, progress)
- NewWithVersion and NewWithContextAndVersion constructors
- Shared state management (gpuInfo, driver, components) between views
- Complete navigation flow: Welcome → Detection → Selection → Confirmation → Progress → Complete/Error
- 83.3% test coverage for ui package

### Changed
- Model struct now includes theme, styles, version, and all view models
- Update method handles all view-specific navigation messages
- View method renders actual view models instead of placeholders

### Removed
- Deprecated renderError method (replaced by ErrorModel view)

## [4.10.0] - 2026-01-04

### Added
- Error/help view (`internal/ui/views/error.go`)
- ErrorModel displaying error screen when installation fails
- Error banner with X icon and "Installation Failed" title
- Failed step name and error message display
- Context-aware troubleshooting tips based on failed step:
  - blacklist: Nouveau driver tips
  - update: Network/repository tips
  - install_*: Disk space/package tips
  - configure: Permission tips
  - verify: Driver loading/dmesg tips
- Two buttons: "Retry" and "Exit"
- RetryRequestedMsg and ErrorExitRequestedMsg for navigation flow
- ErrorKeyMap with help.KeyMap interface (enter/r, q/esc, arrows, c for copy)
- buildTroubleshootingTips function for step-specific help
- Graceful nil error handling with "Unknown error occurred" fallback
- 98.2% test coverage

## [4.9.0] - 2026-01-04

### Added
- Completion/result view (`internal/ui/views/complete.go`)
- CompleteModel displaying success screen after driver installation
- Success banner with checkmark icon and "Installation Complete!" title
- Installed driver version and branch display
- List of installed components with checkmarks (✓)
- Next steps section with nvidia-smi verification instruction
- Reboot recommendation when Nouveau driver was blacklisted
- Two buttons: "Reboot Now" (primary) and "Exit"
- RebootRequestedMsg and ExitRequestedMsg for navigation flow
- CompleteKeyMap with help.KeyMap interface (enter/r, q/esc, arrows)
- needsReboot logic based on NouveauStatus.Loaded
- "(none)" placeholder for empty components list
- 98.1% test coverage

## [4.8.0] - 2026-01-04

### Added
- Installation progress view (`internal/ui/views/progress.go`)
- ProgressModel with spinner, progress bar, and step tracking
- StepStatus enum: Pending, Running, Complete, Failed, Skipped
- InstallationStep struct with name, description, status, timing, error
- buildInstallationSteps generating steps from selected components
- Progress bar showing overall completion percentage
- Steps list with status indicators (✓ ● ✗ ○) and duration
- Log output area showing recent installation messages
- InstallationStepStart/Complete/Failed messages for step tracking
- InstallationLogMsg for real-time output display
- Cancel with Ctrl+C during installation
- NavigateToCompleteMsg and NavigateToErrorMsg for flow control
- ProgressKeyMap with help.KeyMap interface
- 97.8% test coverage

## [4.7.0] - 2026-01-04

### Added
- Installation confirmation view (`internal/ui/views/confirmation.go`)
- ConfirmationModel displaying installation summary before proceeding
- GPU section showing target GPU name with fallback handling
- Driver section showing selected version and branch
- Components section listing selected components with checkmarks (✓)
- Warnings section with system warnings (Nouveau, Secure Boot, headers, existing driver)
- buildWarnings function checking system state for potential issues
- Two buttons: "Install" and "Go Back" with keyboard navigation
- StartInstallationMsg containing GPUInfo, Driver, and Components
- NavigateBackToSelectionMsg for returning to driver selection
- ConfirmationKeyMap with multiple key options (enter/y, esc/n, vim-style)
- 98.3% test coverage

## [4.6.0] - 2026-01-04

### Added
- Driver selection view (`internal/ui/views/selection.go`)
- SelectionModel with two-panel layout: drivers and components
- DriverOption struct with version, branch, description, recommended flag
- ComponentOption struct with name, ID, description, selected, required flags
- Four driver versions: 550 (Latest), 545 (Production), 535 (LTS), 470 (Legacy)
- Four components: Driver (required), CUDA Toolkit, cuDNN, NVIDIA Settings
- Section navigation with Tab key, Up/Down navigation within sections
- Space to toggle component selection, required components protected
- SelectionKeyMap with vim-style navigation and help.KeyMap interface
- NavigateToConfirmationMsg with selected driver and components
- Responsive two-column layout
- 97.8% test coverage

## [4.5.0] - 2026-01-04

### Added
- System detection view (`internal/ui/views/detection.go`)
- DetectionModel with three states: Detecting, Complete, Error
- Animated spinner with 6 detection step messages
- GPU section displaying detected NVIDIA GPUs by name
- Driver section showing type, version, CUDA version
- System section with kernel, headers, secure boot, nouveau status
- Validation section with error/warning counts
- DetectionKeyMap with state-aware key bindings
- Navigation messages: NavigateToDriverSelectionMsg, NavigateToWelcomeMsg
- Detection messages: DetectionStepMsg, DetectionCompleteMsg, DetectionErrorMsg
- Integration with gpu.GPUInfo types from Phase 3
- 98.8% test coverage

## [4.4.0] - 2026-01-04

### Added
- Welcome screen view (`internal/ui/views`)
- WelcomeModel with header, footer, and button group components
- ASCII art IGOR logo
- Welcome description and feature list display
- Two buttons: "Start Installation" and "Exit" with keyboard navigation
- WelcomeKeyMap with vim-style navigation (h/l) and help.KeyMap interface
- StartDetectionMsg for navigation to detection screen
- Responsive layout adapting to terminal dimensions
- 100% test coverage

## [4.3.0] - 2026-01-04

### Added
- Common UI components (`internal/ui/components`)
- SpinnerModel with configurable message, Show/Hide, multiple spinner types
- ProgressModel with SetProgress, SetLabel, percent tracking, animation support
- ListModel wrapping bubbles list with item selection, SelectedItem/Index
- ButtonModel with Focus/Blur, Enable/Disable states
- ButtonGroup for keyboard navigation with Next/Previous and wrap-around
- PanelModel for styled containers with title, content, focus border
- HeaderModel with title, subtitle, version, responsive width layout
- FooterModel with help integration, status display (info/success/warning/error)
- All components integrate with theme.Styles for consistent appearance
- 94.3% test coverage

## [4.2.0] - 2026-01-04

### Added
- Theme and styling system (`internal/ui/theme`)
- NVIDIA-inspired color palette with primary green (#76B900)
- AdaptiveColor support for automatic light/dark terminal detection
- Three theme variants: DefaultTheme (dark), LightTheme, HighContrastTheme
- Theme struct with semantic colors (success, warning, error, info)
- Styles struct with 48 pre-built lipgloss styles for all UI components
- Helper functions: RenderBox, RenderStatusLine, RenderProgressBar, RenderProgressBarWithLabel
- RenderKeyValue, RenderGPUCard, RenderButton, RenderDialog, RenderList helpers
- Styles.Copy(), WithWidth(), WithHeight() for responsive layouts
- GetTheme() factory and AvailableThemes() for theme discovery
- 97.4% test coverage

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
