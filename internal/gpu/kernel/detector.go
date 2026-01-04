package kernel

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/exec"
)

// Default paths for kernel detection.
const (
	// DefaultProcModulesPath is the path to /proc/modules.
	DefaultProcModulesPath = "/proc/modules"

	// DefaultKernelHeadersPrefix is the base path for kernel headers.
	DefaultKernelHeadersPrefix = "/usr/src/linux-headers-"

	// DefaultModulesBuildPath is the alternative path for kernel headers via /lib/modules.
	DefaultModulesBuildPath = "/lib/modules"

	// DefaultEFIVarsPath is the path to EFI variables for Secure Boot detection.
	DefaultEFIVarsPath = "/sys/firmware/efi/efivars"
)

// KernelInfo represents kernel information for the running system.
type KernelInfo struct {
	// Version is the full kernel version string (e.g., "6.5.0-44-generic").
	Version string

	// Release is the major.minor.patch part of the version (e.g., "6.5.0").
	Release string

	// Architecture is the system architecture (e.g., "x86_64").
	Architecture string

	// HeadersPath is the path to the kernel headers directory.
	HeadersPath string

	// HeadersInstalled indicates whether kernel headers are installed.
	HeadersInstalled bool

	// SecureBootEnabled indicates whether Secure Boot is enabled.
	SecureBootEnabled bool
}

// Detector interface for kernel version and module detection.
type Detector interface {
	// GetKernelInfo returns comprehensive kernel information.
	GetKernelInfo(ctx context.Context) (*KernelInfo, error)

	// IsModuleLoaded checks if a kernel module with the given name is loaded.
	IsModuleLoaded(ctx context.Context, name string) (bool, error)

	// GetLoadedModules returns all currently loaded kernel modules.
	GetLoadedModules(ctx context.Context) ([]ModuleInfo, error)

	// GetModule returns information about a specific loaded module.
	// Returns nil if the module is not loaded.
	GetModule(ctx context.Context, name string) (*ModuleInfo, error)

	// AreHeadersInstalled checks if kernel headers are installed for the running kernel.
	AreHeadersInstalled(ctx context.Context) (bool, error)

	// GetHeadersPackage returns the package name for kernel headers based on distribution.
	GetHeadersPackage(ctx context.Context) (string, error)

	// IsSecureBootEnabled checks if Secure Boot is enabled.
	IsSecureBootEnabled(ctx context.Context) (bool, error)
}

// FileSystem abstracts filesystem operations for testing.
type FileSystem interface {
	// ReadDir reads the directory named by dirname and returns a list of directory entries.
	ReadDir(dirname string) ([]fs.DirEntry, error)

	// ReadFile reads the file named by filename and returns the contents.
	ReadFile(filename string) ([]byte, error)

	// Stat returns the FileInfo structure describing file.
	Stat(name string) (fs.FileInfo, error)
}

// RealFileSystem implements FileSystem using the actual operating system.
type RealFileSystem struct{}

// ReadDir reads the directory named by dirname and returns a list of directory entries.
func (RealFileSystem) ReadDir(dirname string) ([]fs.DirEntry, error) {
	return os.ReadDir(dirname)
}

// ReadFile reads the file named by filename and returns the contents.
func (RealFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// Stat returns the FileInfo structure describing file.
func (RealFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// DetectorImpl is the production implementation of the Detector interface.
type DetectorImpl struct {
	executor          exec.Executor
	fs                FileSystem
	distroFamily      constants.DistroFamily
	procModulesPath   string
	kernelHeadersPath string
	modulesBuildPath  string
	efiVarsPath       string
}

// DetectorOption configures the detector.
type DetectorOption func(*DetectorImpl)

// WithExecutor sets a custom executor (useful for testing).
func WithExecutor(executor exec.Executor) DetectorOption {
	return func(d *DetectorImpl) {
		d.executor = executor
	}
}

// WithFileSystem sets a custom filesystem implementation (useful for testing).
func WithFileSystem(fs FileSystem) DetectorOption {
	return func(d *DetectorImpl) {
		d.fs = fs
	}
}

// WithDistroFamily sets the distribution family for package name detection.
func WithDistroFamily(family constants.DistroFamily) DetectorOption {
	return func(d *DetectorImpl) {
		d.distroFamily = family
	}
}

// WithProcModulesPath sets a custom path for /proc/modules.
func WithProcModulesPath(path string) DetectorOption {
	return func(d *DetectorImpl) {
		d.procModulesPath = path
	}
}

// WithKernelHeadersPath sets a custom base path for kernel headers.
func WithKernelHeadersPath(path string) DetectorOption {
	return func(d *DetectorImpl) {
		d.kernelHeadersPath = path
	}
}

// WithModulesBuildPath sets a custom path for /lib/modules.
func WithModulesBuildPath(path string) DetectorOption {
	return func(d *DetectorImpl) {
		d.modulesBuildPath = path
	}
}

// WithEFIVarsPath sets a custom path for EFI variables.
func WithEFIVarsPath(path string) DetectorOption {
	return func(d *DetectorImpl) {
		d.efiVarsPath = path
	}
}

// NewDetector creates a new kernel detector with the given options.
func NewDetector(opts ...DetectorOption) *DetectorImpl {
	d := &DetectorImpl{
		fs:                RealFileSystem{},
		distroFamily:      constants.FamilyUnknown,
		procModulesPath:   DefaultProcModulesPath,
		kernelHeadersPath: DefaultKernelHeadersPrefix,
		modulesBuildPath:  DefaultModulesBuildPath,
		efiVarsPath:       DefaultEFIVarsPath,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// GetKernelInfo returns comprehensive kernel information.
func (d *DetectorImpl) GetKernelInfo(ctx context.Context) (*KernelInfo, error) {
	const op = "kernel.GetKernelInfo"

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.GPUDetection, "kernel info detection cancelled", ctx.Err()).WithOp(op)
	default:
	}

	info := &KernelInfo{}

	// Get kernel version via uname -r
	version, err := d.getKernelVersion(ctx)
	if err != nil {
		return nil, err
	}
	info.Version = version
	info.Release = extractRelease(version)

	// Get architecture via uname -m
	arch, err := d.getArchitecture(ctx)
	if err != nil {
		return nil, err
	}
	info.Architecture = arch

	// Check headers installation
	headersInstalled, headersPath, err := d.checkHeadersInstalled(ctx, version)
	if err != nil {
		// Non-fatal error, continue with other checks
		headersInstalled = false
	}
	info.HeadersInstalled = headersInstalled
	info.HeadersPath = headersPath

	// Check Secure Boot status
	secureBootEnabled, err := d.IsSecureBootEnabled(ctx)
	if err != nil {
		// Non-fatal error, assume not enabled
		secureBootEnabled = false
	}
	info.SecureBootEnabled = secureBootEnabled

	return info, nil
}

// getKernelVersion runs 'uname -r' to get the kernel version.
func (d *DetectorImpl) getKernelVersion(ctx context.Context) (string, error) {
	const op = "kernel.getKernelVersion"

	if d.executor == nil {
		return "", errors.New(errors.GPUDetection, "no executor available").WithOp(op)
	}

	result := d.executor.Execute(ctx, "uname", "-r")
	if !result.Success() {
		return "", errors.Wrap(errors.GPUDetection, "failed to get kernel version", result.Error).WithOp(op)
	}

	version := strings.TrimSpace(result.StdoutString())
	if version == "" {
		return "", errors.New(errors.GPUDetection, "empty kernel version returned").WithOp(op)
	}

	return version, nil
}

// getArchitecture runs 'uname -m' to get the system architecture.
func (d *DetectorImpl) getArchitecture(ctx context.Context) (string, error) {
	const op = "kernel.getArchitecture"

	if d.executor == nil {
		return "", errors.New(errors.GPUDetection, "no executor available").WithOp(op)
	}

	result := d.executor.Execute(ctx, "uname", "-m")
	if !result.Success() {
		return "", errors.Wrap(errors.GPUDetection, "failed to get architecture", result.Error).WithOp(op)
	}

	arch := strings.TrimSpace(result.StdoutString())
	if arch == "" {
		return "", errors.New(errors.GPUDetection, "empty architecture returned").WithOp(op)
	}

	return arch, nil
}

// extractRelease extracts the release version (major.minor.patch) from a kernel version string.
// e.g., "6.5.0-44-generic" -> "6.5.0"
func extractRelease(version string) string {
	// Match major.minor.patch at the beginning of the string
	re := regexp.MustCompile(`^(\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) >= 2 {
		return matches[1]
	}

	// Fallback: try to extract just major.minor
	re = regexp.MustCompile(`^(\d+\.\d+)`)
	matches = re.FindStringSubmatch(version)
	if len(matches) >= 2 {
		return matches[1]
	}

	return version
}

// checkHeadersInstalled checks if kernel headers are installed for the given kernel version.
// Returns whether headers are installed and the path to them.
func (d *DetectorImpl) checkHeadersInstalled(ctx context.Context, kernelVersion string) (bool, string, error) {
	const op = "kernel.checkHeadersInstalled"

	// Check context cancellation
	select {
	case <-ctx.Done():
		return false, "", errors.Wrap(errors.GPUDetection, "headers check cancelled", ctx.Err()).WithOp(op)
	default:
	}

	// Primary check: /usr/src/linux-headers-<version>
	headersPath := d.kernelHeadersPath + kernelVersion
	_, err := d.fs.Stat(headersPath)
	if err == nil {
		return true, headersPath, nil
	}

	// Secondary check: /lib/modules/<version>/build symlink
	buildPath := filepath.Join(d.modulesBuildPath, kernelVersion, "build")
	info, err := d.fs.Stat(buildPath)
	if err == nil {
		// Check if it's a valid directory (or symlink to a directory)
		if info.IsDir() {
			return true, buildPath, nil
		}
	}

	return false, "", nil
}

// IsModuleLoaded checks if a kernel module with the given name is loaded.
func (d *DetectorImpl) IsModuleLoaded(ctx context.Context, name string) (bool, error) {
	const op = "kernel.IsModuleLoaded"

	modules, err := d.GetLoadedModules(ctx)
	if err != nil {
		return false, errors.Wrap(errors.GPUDetection, "failed to check if module is loaded", err).WithOp(op)
	}

	return IsModuleInList(modules, name), nil
}

// GetLoadedModules returns all currently loaded kernel modules.
func (d *DetectorImpl) GetLoadedModules(ctx context.Context) ([]ModuleInfo, error) {
	const op = "kernel.GetLoadedModules"

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.GPUDetection, "get loaded modules cancelled", ctx.Err()).WithOp(op)
	default:
	}

	content, err := d.fs.ReadFile(d.procModulesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrap(errors.NotFound, "proc modules file not found", err).WithOp(op)
		}
		if os.IsPermission(err) {
			return nil, errors.Wrap(errors.Permission, "permission denied reading proc modules", err).WithOp(op)
		}
		return nil, errors.Wrap(errors.GPUDetection, "failed to read proc modules", err).WithOp(op)
	}

	modules, err := ParseModulesContent(content)
	if err != nil {
		return nil, errors.Wrap(errors.GPUDetection, "failed to parse proc modules", err).WithOp(op)
	}

	return modules, nil
}

// GetModule returns information about a specific loaded module.
// Returns nil if the module is not loaded.
func (d *DetectorImpl) GetModule(ctx context.Context, name string) (*ModuleInfo, error) {
	const op = "kernel.GetModule"

	modules, err := d.GetLoadedModules(ctx)
	if err != nil {
		return nil, errors.Wrap(errors.GPUDetection, "failed to get module info", err).WithOp(op)
	}

	return FindModule(modules, name), nil
}

// AreHeadersInstalled checks if kernel headers are installed for the running kernel.
func (d *DetectorImpl) AreHeadersInstalled(ctx context.Context) (bool, error) {
	const op = "kernel.AreHeadersInstalled"

	version, err := d.getKernelVersion(ctx)
	if err != nil {
		return false, errors.Wrap(errors.GPUDetection, "failed to get kernel version for headers check", err).WithOp(op)
	}

	installed, _, err := d.checkHeadersInstalled(ctx, version)
	if err != nil {
		return false, err
	}

	return installed, nil
}

// GetHeadersPackage returns the package name for kernel headers based on distribution.
// Package names vary by distribution family:
//   - Debian/Ubuntu: linux-headers-$(uname -r)
//   - Fedora/RHEL: kernel-devel-$(uname -r)
//   - Arch: linux-headers
//   - openSUSE: kernel-default-devel
func (d *DetectorImpl) GetHeadersPackage(ctx context.Context) (string, error) {
	const op = "kernel.GetHeadersPackage"

	version, err := d.getKernelVersion(ctx)
	if err != nil {
		return "", errors.Wrap(errors.GPUDetection, "failed to get kernel version for headers package", err).WithOp(op)
	}

	return d.getHeadersPackageForVersion(version), nil
}

// getHeadersPackageForVersion returns the headers package name for a given kernel version.
func (d *DetectorImpl) getHeadersPackageForVersion(kernelVersion string) string {
	switch d.distroFamily {
	case constants.FamilyDebian:
		return "linux-headers-" + kernelVersion
	case constants.FamilyRHEL:
		return "kernel-devel-" + kernelVersion
	case constants.FamilyArch:
		// Arch uses a generic package name, but may need linux-lts-headers for LTS kernel
		if strings.Contains(kernelVersion, "lts") {
			return "linux-lts-headers"
		}
		return "linux-headers"
	case constants.FamilySUSE:
		// openSUSE uses kernel-<flavor>-devel, default is most common
		return "kernel-default-devel"
	default:
		// Generic fallback
		return "linux-headers-" + kernelVersion
	}
}

// IsSecureBootEnabled checks if Secure Boot is enabled.
// It first tries mokutil --sb-state, then falls back to checking EFI variables.
func (d *DetectorImpl) IsSecureBootEnabled(ctx context.Context) (bool, error) {
	const op = "kernel.IsSecureBootEnabled"

	// Check context cancellation
	select {
	case <-ctx.Done():
		return false, errors.Wrap(errors.GPUDetection, "secure boot check cancelled", ctx.Err()).WithOp(op)
	default:
	}

	// Try mokutil first if executor is available
	if d.executor != nil {
		enabled, err := d.checkSecureBootViaMokutil(ctx)
		if err == nil {
			return enabled, nil
		}
		// mokutil failed, try EFI variable fallback
	}

	// Fallback: check EFI variables
	return d.checkSecureBootViaEFI(ctx)
}

// checkSecureBootViaMokutil checks Secure Boot status using mokutil.
func (d *DetectorImpl) checkSecureBootViaMokutil(ctx context.Context) (bool, error) {
	result := d.executor.Execute(ctx, "mokutil", "--sb-state")

	// mokutil returns non-zero if Secure Boot is not supported
	// We check the output regardless of exit code
	output := result.StdoutString()
	stderrOutput := result.StderrString()
	combinedOutput := output + stderrOutput

	// Check for enabled status
	if strings.Contains(combinedOutput, "SecureBoot enabled") {
		return true, nil
	}

	// Check for disabled status
	if strings.Contains(combinedOutput, "SecureBoot disabled") {
		return false, nil
	}

	// If mokutil command failed completely (not installed)
	if result.Error != nil {
		return false, result.Error
	}

	// Unknown output - assume not supported/enabled
	return false, nil
}

// checkSecureBootViaEFI checks Secure Boot status by reading EFI variables.
func (d *DetectorImpl) checkSecureBootViaEFI(ctx context.Context) (bool, error) {
	const op = "kernel.checkSecureBootViaEFI"

	// Check context cancellation
	select {
	case <-ctx.Done():
		return false, errors.Wrap(errors.GPUDetection, "EFI check cancelled", ctx.Err()).WithOp(op)
	default:
	}

	// Look for SecureBoot-* variable in /sys/firmware/efi/efivars
	entries, err := d.fs.ReadDir(d.efiVarsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No EFI firmware = no Secure Boot
			return false, nil
		}
		// Can't read EFI vars, assume not enabled
		return false, nil
	}

	// Find the SecureBoot variable
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "SecureBoot-") {
			// Read the variable content
			varPath := filepath.Join(d.efiVarsPath, entry.Name())
			content, err := d.fs.ReadFile(varPath)
			if err != nil {
				// Can't read, assume not enabled
				return false, nil
			}

			// The EFI variable format has 4 bytes of attributes followed by the value
			// SecureBoot is 1 byte: 0x00 = disabled, 0x01 = enabled
			if len(content) >= 5 {
				// Skip the 4-byte attribute header
				value := content[4]
				return value == 0x01, nil
			}
		}
	}

	// No SecureBoot variable found
	return false, nil
}

// Ensure DetectorImpl implements Detector interface.
var _ Detector = (*DetectorImpl)(nil)
