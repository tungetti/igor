// Package constants defines application-wide constants for the Igor application.
// All constants are typed to ensure type safety and prevent accidental misuse.
package constants

import "time"

// Application metadata
const (
	// AppName is the application name used in logs, configs, and user messages.
	AppName string = "igor"
	// AppDescription is a short description of the application.
	AppDescription string = "NVIDIA TUI Installer for Linux"
)

// ExitCode represents process exit codes for different termination scenarios.
type ExitCode int

const (
	// ExitSuccess indicates the application completed successfully.
	ExitSuccess ExitCode = iota
	// ExitError indicates a general error occurred.
	ExitError
	// ExitPermission indicates insufficient permissions (e.g., not root).
	ExitPermission
	// ExitValidation indicates invalid input or configuration.
	ExitValidation
	// ExitInstallation indicates the installation process failed.
	ExitInstallation
	// ExitUserAbort indicates the user cancelled the operation.
	ExitUserAbort
)

// Int returns the exit code as an int for use with os.Exit().
func (e ExitCode) Int() int {
	return int(e)
}

// Timeouts for various operations. These are tuned for typical system
// responsiveness while allowing for slower hardware or network conditions.
const (
	// DefaultTimeout is the standard timeout for most operations.
	DefaultTimeout time.Duration = 5 * time.Minute
	// ShortTimeout is for quick operations that should complete rapidly.
	ShortTimeout time.Duration = 30 * time.Second
	// LongTimeout is for operations that may take extended time (e.g., large downloads).
	LongTimeout time.Duration = 15 * time.Minute
	// NetworkTimeout is for network operations like HTTP requests.
	NetworkTimeout time.Duration = 60 * time.Second
	// CommandTimeout is for shell command execution.
	CommandTimeout time.Duration = 2 * time.Minute
)

// File paths relative to user's home directory
const (
	// DefaultConfigDir is the default configuration directory relative to $HOME.
	DefaultConfigDir string = ".config/igor"
	// DefaultCacheDir is the default cache directory relative to $HOME.
	DefaultCacheDir string = ".cache/igor"
	// DefaultLogFile is the default log file name.
	DefaultLogFile string = "igor.log"
	// ConfigFileName is the configuration file name.
	ConfigFileName string = "config.yaml"
)

// System paths for Linux system files and directories
const (
	// OSReleasePath is the path to the os-release file for distro detection.
	OSReleasePath string = "/etc/os-release"
	// LSBReleasePath is the path to the LSB release file (alternative distro detection).
	LSBReleasePath string = "/etc/lsb-release"
	// ModprobeDir is the directory for kernel module configuration.
	ModprobeDir string = "/etc/modprobe.d"
	// XorgConfDir is the directory for X.org configuration files.
	XorgConfDir string = "/etc/X11/xorg.conf.d"
	// SysClassDRM is the sysfs path for DRM devices.
	SysClassDRM string = "/sys/class/drm"
	// ProcModules is the path to the list of loaded kernel modules.
	ProcModules string = "/proc/modules"
)

// NVIDIA-specific constants for driver and module management
const (
	// NouveauModuleName is the name of the open-source NVIDIA driver.
	NouveauModuleName string = "nouveau"
	// NvidiaModuleName is the name of the proprietary NVIDIA driver module.
	NvidiaModuleName string = "nvidia"
	// NvidiaDRMModule is the NVIDIA DRM kernel module.
	NvidiaDRMModule string = "nvidia_drm"
	// NvidiaModeset is the NVIDIA modesetting kernel module.
	NvidiaModeset string = "nvidia_modeset"
	// NvidiaBlacklistFile is the filename for the nouveau blacklist configuration.
	NvidiaBlacklistFile string = "blacklist-nouveau.conf"
)

// Package manager command names
const (
	// AptGet is the apt-get package manager command.
	AptGet string = "apt-get"
	// Apt is the apt package manager command.
	Apt string = "apt"
	// Dpkg is the Debian package tool command.
	Dpkg string = "dpkg"
	// Dnf is the DNF package manager command (Fedora/RHEL 8+).
	Dnf string = "dnf"
	// Yum is the YUM package manager command (RHEL/CentOS 7).
	Yum string = "yum"
	// Rpm is the RPM package tool command.
	Rpm string = "rpm"
	// Pacman is the Arch Linux package manager command.
	Pacman string = "pacman"
	// Zypper is the openSUSE package manager command.
	Zypper string = "zypper"
)

// DistroFamily represents Linux distribution families for package management.
type DistroFamily string

const (
	// FamilyDebian includes Debian, Ubuntu, Linux Mint, Pop!_OS, etc.
	FamilyDebian DistroFamily = "debian"
	// FamilyRHEL includes RHEL, CentOS, Fedora, Rocky Linux, AlmaLinux, etc.
	FamilyRHEL DistroFamily = "rhel"
	// FamilyArch includes Arch Linux, Manjaro, EndeavourOS, etc.
	FamilyArch DistroFamily = "arch"
	// FamilySUSE includes openSUSE, SUSE Linux Enterprise, etc.
	FamilySUSE DistroFamily = "suse"
	// FamilyUnknown indicates an unrecognized distribution family.
	FamilyUnknown DistroFamily = "unknown"
)

// String returns the string representation of the distro family.
func (f DistroFamily) String() string {
	return string(f)
}
