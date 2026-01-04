// Package nouveau provides detection of the Nouveau open-source NVIDIA driver.
// It checks if the Nouveau kernel module is loaded, in use, and whether it
// has been blacklisted. This is critical because Nouveau must be blacklisted
// before installing proprietary NVIDIA drivers.
package nouveau

import (
	"bufio"
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/gpu/pci"
)

// Default paths for Nouveau detection.
const (
	// DefaultModulePath is the path to check if nouveau module is loaded.
	DefaultModulePath = "/sys/module/nouveau"

	// DefaultProcModulesPath is the path to /proc/modules for checking module usage.
	DefaultProcModulesPath = "/proc/modules"

	// DefaultModprobePath is the directory containing modprobe configuration files.
	DefaultModprobePath = "/etc/modprobe.d"
)

// Blacklist patterns to look for in modprobe configuration files.
var blacklistPatterns = []string{
	"blacklist nouveau",
	"options nouveau modeset=0",
}

// Status represents Nouveau driver status on the system.
type Status struct {
	// Loaded indicates whether the nouveau kernel module is loaded.
	Loaded bool

	// InUse indicates whether the nouveau module is currently being used by GPU devices.
	InUse bool

	// BoundDevices contains PCI addresses of devices bound to the nouveau driver.
	BoundDevices []string

	// BlacklistExists indicates whether a nouveau blacklist configuration exists.
	BlacklistExists bool

	// BlacklistFiles contains paths to files that contain nouveau blacklist entries.
	BlacklistFiles []string
}

// Detector interface for checking Nouveau driver status.
type Detector interface {
	// Detect performs a comprehensive check of Nouveau driver status.
	Detect(ctx context.Context) (*Status, error)

	// IsLoaded checks if the nouveau kernel module is loaded.
	IsLoaded(ctx context.Context) (bool, error)

	// IsBlacklisted checks if nouveau is blacklisted in modprobe configuration.
	IsBlacklisted(ctx context.Context) (bool, error)

	// GetBoundDevices returns PCI addresses of devices using the nouveau driver.
	GetBoundDevices(ctx context.Context) ([]string, error)
}

// FileSystem abstracts filesystem operations for testing.
// This interface is compatible with pci.FileSystem.
type FileSystem interface {
	// ReadDir reads the directory named by dirname and returns a list of directory entries.
	ReadDir(dirname string) ([]fs.DirEntry, error)

	// ReadFile reads the file named by filename and returns the contents.
	ReadFile(filename string) ([]byte, error)

	// Readlink returns the destination of the named symbolic link.
	Readlink(name string) (string, error)

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

// Readlink returns the destination of the named symbolic link.
func (RealFileSystem) Readlink(name string) (string, error) {
	return os.Readlink(name)
}

// Stat returns the FileInfo structure describing file.
func (RealFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// DetectorImpl is the production implementation of the Detector interface.
type DetectorImpl struct {
	fs              FileSystem
	pciScanner      pci.Scanner
	modulePath      string
	procModulesPath string
	modprobePath    string
}

// DetectorOption configures the detector.
type DetectorOption func(*DetectorImpl)

// WithFileSystem sets a custom filesystem implementation (useful for testing).
func WithFileSystem(fs FileSystem) DetectorOption {
	return func(d *DetectorImpl) {
		d.fs = fs
	}
}

// WithPCIScanner sets a custom PCI scanner (useful for testing).
func WithPCIScanner(scanner pci.Scanner) DetectorOption {
	return func(d *DetectorImpl) {
		d.pciScanner = scanner
	}
}

// WithModulePath sets a custom path for the nouveau module directory.
func WithModulePath(path string) DetectorOption {
	return func(d *DetectorImpl) {
		d.modulePath = path
	}
}

// WithProcModulesPath sets a custom path for /proc/modules.
func WithProcModulesPath(path string) DetectorOption {
	return func(d *DetectorImpl) {
		d.procModulesPath = path
	}
}

// WithModprobePath sets a custom path for the modprobe.d directory.
func WithModprobePath(path string) DetectorOption {
	return func(d *DetectorImpl) {
		d.modprobePath = path
	}
}

// NewDetector creates a new Nouveau detector with the given options.
func NewDetector(opts ...DetectorOption) *DetectorImpl {
	d := &DetectorImpl{
		fs:              RealFileSystem{},
		modulePath:      DefaultModulePath,
		procModulesPath: DefaultProcModulesPath,
		modprobePath:    DefaultModprobePath,
	}
	for _, opt := range opts {
		opt(d)
	}
	// Create default PCI scanner if not provided
	if d.pciScanner == nil {
		d.pciScanner = pci.NewScanner()
	}
	return d
}

// Detect performs a comprehensive check of Nouveau driver status.
func (d *DetectorImpl) Detect(ctx context.Context) (*Status, error) {
	// Check context cancellation before starting
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.GPUDetection, "nouveau detection cancelled", ctx.Err())
	default:
	}

	status := &Status{}

	// Check if module is loaded
	loaded, err := d.IsLoaded(ctx)
	if err != nil {
		return nil, err
	}
	status.Loaded = loaded

	// Check for bound devices
	boundDevices, err := d.GetBoundDevices(ctx)
	if err != nil {
		return nil, err
	}
	status.BoundDevices = boundDevices
	status.InUse = len(boundDevices) > 0

	// Check for blacklist configuration
	blacklisted, err := d.IsBlacklisted(ctx)
	if err != nil {
		return nil, err
	}
	status.BlacklistExists = blacklisted

	// Get list of blacklist files
	blacklistFiles, err := d.findBlacklistFiles(ctx)
	if err != nil {
		// Non-fatal: we still have the blacklist status
		blacklistFiles = nil
	}
	status.BlacklistFiles = blacklistFiles

	return status, nil
}

// IsLoaded checks if the nouveau kernel module is loaded.
// It checks if /sys/module/nouveau directory exists.
func (d *DetectorImpl) IsLoaded(ctx context.Context) (bool, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return false, errors.Wrap(errors.GPUDetection, "nouveau loaded check cancelled", ctx.Err())
	default:
	}

	_, err := d.fs.Stat(d.modulePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		if os.IsPermission(err) {
			return false, errors.Wrap(errors.Permission, "permission denied checking nouveau module", err).WithOp("nouveau.IsLoaded")
		}
		return false, errors.Wrap(errors.GPUDetection, "failed to check nouveau module status", err).WithOp("nouveau.IsLoaded")
	}

	return true, nil
}

// IsBlacklisted checks if nouveau is blacklisted in modprobe configuration.
// It scans /etc/modprobe.d/*.conf files for blacklist patterns.
func (d *DetectorImpl) IsBlacklisted(ctx context.Context) (bool, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return false, errors.Wrap(errors.GPUDetection, "nouveau blacklist check cancelled", ctx.Err())
	default:
	}

	files, err := d.findBlacklistFiles(ctx)
	if err != nil {
		return false, err
	}

	return len(files) > 0, nil
}

// GetBoundDevices returns PCI addresses of devices using the nouveau driver.
func (d *DetectorImpl) GetBoundDevices(ctx context.Context) ([]string, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.GPUDetection, "nouveau bound devices check cancelled", ctx.Err())
	default:
	}

	// Scan all PCI devices
	devices, err := d.pciScanner.ScanAll(ctx)
	if err != nil {
		return nil, errors.Wrap(errors.GPUDetection, "failed to scan PCI devices", err).WithOp("nouveau.GetBoundDevices")
	}

	// Filter devices bound to nouveau
	var boundDevices []string
	for _, device := range devices {
		if device.IsUsingNouveau() {
			boundDevices = append(boundDevices, device.Address)
		}
	}

	return boundDevices, nil
}

// findBlacklistFiles returns paths to files containing nouveau blacklist entries.
func (d *DetectorImpl) findBlacklistFiles(ctx context.Context) ([]string, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.GPUDetection, "nouveau blacklist file search cancelled", ctx.Err())
	default:
	}

	entries, err := d.fs.ReadDir(d.modprobePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No modprobe.d directory means no blacklist
			return nil, nil
		}
		if os.IsPermission(err) {
			return nil, errors.Wrap(errors.Permission, "permission denied reading modprobe.d", err).WithOp("nouveau.findBlacklistFiles")
		}
		return nil, errors.Wrap(errors.GPUDetection, "failed to read modprobe.d directory", err).WithOp("nouveau.findBlacklistFiles")
	}

	var blacklistFiles []string
	for _, entry := range entries {
		// Check context cancellation during iteration
		select {
		case <-ctx.Done():
			return nil, errors.Wrap(errors.GPUDetection, "nouveau blacklist file search cancelled", ctx.Err())
		default:
		}

		// Only check .conf files
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".conf") {
			continue
		}

		filePath := filepath.Join(d.modprobePath, entry.Name())
		content, err := d.fs.ReadFile(filePath)
		if err != nil {
			// Skip files we can't read
			continue
		}

		if containsBlacklistPattern(content) {
			blacklistFiles = append(blacklistFiles, filePath)
		}
	}

	return blacklistFiles, nil
}

// containsBlacklistPattern checks if the content contains any nouveau blacklist pattern.
func containsBlacklistPattern(content []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Normalize whitespace for comparison
		normalizedLine := strings.Join(strings.Fields(line), " ")

		for _, pattern := range blacklistPatterns {
			if strings.HasPrefix(normalizedLine, pattern) {
				return true
			}
		}
	}
	return false
}

// Ensure DetectorImpl implements Detector interface.
var _ Detector = (*DetectorImpl)(nil)
