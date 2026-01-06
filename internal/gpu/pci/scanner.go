package pci

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tungetti/igor/internal/errors"
)

// DefaultSysfsPath is the default path to the sysfs PCI devices directory.
const DefaultSysfsPath = "/sys/bus/pci/devices"

// FileSystem abstracts filesystem operations for testing.
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

// Scanner interface for PCI device discovery.
type Scanner interface {
	// ScanAll returns all PCI devices found in sysfs.
	ScanAll(ctx context.Context) ([]PCIDevice, error)

	// ScanByVendor returns devices matching the given vendor ID.
	ScanByVendor(ctx context.Context, vendorID string) ([]PCIDevice, error)

	// ScanByClass returns devices matching the given class code prefix.
	ScanByClass(ctx context.Context, classCode string) ([]PCIDevice, error)

	// ScanNVIDIA returns all NVIDIA GPU devices (VGA and 3D controllers).
	ScanNVIDIA(ctx context.Context) ([]PCIDevice, error)
}

// ScannerImpl is the production implementation of the Scanner interface.
type ScannerImpl struct {
	fs        FileSystem
	sysfsPath string
}

// ScannerOption configures the scanner.
type ScannerOption func(*ScannerImpl)

// WithFileSystem sets a custom filesystem implementation (useful for testing).
func WithFileSystem(fs FileSystem) ScannerOption {
	return func(s *ScannerImpl) {
		s.fs = fs
	}
}

// WithSysfsPath sets a custom sysfs path (useful for testing).
func WithSysfsPath(path string) ScannerOption {
	return func(s *ScannerImpl) {
		s.sysfsPath = path
	}
}

// NewScanner creates a new PCI device scanner with the given options.
func NewScanner(opts ...ScannerOption) *ScannerImpl {
	s := &ScannerImpl{
		fs:        RealFileSystem{},
		sysfsPath: DefaultSysfsPath,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ScanAll returns all PCI devices found in sysfs.
func (s *ScannerImpl) ScanAll(ctx context.Context) ([]PCIDevice, error) {
	return s.scan(ctx, nil)
}

// ScanByVendor returns devices matching the given vendor ID.
func (s *ScannerImpl) ScanByVendor(ctx context.Context, vendorID string) ([]PCIDevice, error) {
	normalizedVendor := ParseHexID(vendorID)
	return s.scan(ctx, func(d *PCIDevice) bool {
		return d.MatchesVendor(normalizedVendor)
	})
}

// ScanByClass returns devices matching the given class code prefix.
func (s *ScannerImpl) ScanByClass(ctx context.Context, classCode string) ([]PCIDevice, error) {
	normalizedClass := ParseHexID(classCode)
	return s.scan(ctx, func(d *PCIDevice) bool {
		return d.MatchesClass(normalizedClass)
	})
}

// ScanNVIDIA returns all NVIDIA GPU devices (VGA and 3D controllers).
func (s *ScannerImpl) ScanNVIDIA(ctx context.Context) ([]PCIDevice, error) {
	return s.scan(ctx, func(d *PCIDevice) bool {
		return d.IsNVIDIAGPU()
	})
}

// scan performs the actual scanning with an optional filter function.
func (s *ScannerImpl) scan(ctx context.Context, filter func(*PCIDevice) bool) ([]PCIDevice, error) {
	// Check context cancellation before starting
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.GPUDetection, "PCI scan cancelled", ctx.Err())
	default:
	}

	entries, err := s.fs.ReadDir(s.sysfsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrap(errors.NotFound, "sysfs PCI devices directory not found", err).WithOp("pci.Scan")
		}
		if os.IsPermission(err) {
			return nil, errors.Wrap(errors.Permission, "permission denied reading PCI devices", err).WithOp("pci.Scan")
		}
		return nil, errors.Wrap(errors.GPUDetection, "failed to read PCI devices directory", err).WithOp("pci.Scan")
	}

	var devices []PCIDevice
	for _, entry := range entries {
		// Check for context cancellation during iteration
		select {
		case <-ctx.Done():
			return nil, errors.Wrap(errors.GPUDetection, "PCI scan cancelled", ctx.Err())
		default:
		}

		// In /sys/bus/pci/devices/, entries are symlinks to directories
		// We need to check if the entry is a directory OR a symlink
		// Note: entry.IsDir() returns false for symlinks even if they point to directories
		if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			continue
		}

		device, err := s.readDevice(entry.Name())
		if err != nil {
			// Skip devices we can't read (may be permission issues for specific devices)
			continue
		}

		if filter == nil || filter(&device) {
			devices = append(devices, device)
		}
	}

	return devices, nil
}

// readDevice reads device information from sysfs for a given PCI address.
func (s *ScannerImpl) readDevice(address string) (PCIDevice, error) {
	devicePath := filepath.Join(s.sysfsPath, address)

	device := PCIDevice{
		Address: address,
	}

	// Read vendor ID (required)
	vendorID, err := s.readSysfsFile(devicePath, "vendor")
	if err != nil {
		return device, errors.Wrap(errors.GPUDetection, "failed to read vendor ID", err)
	}
	device.VendorID = ParseHexID(vendorID)

	// Read device ID (required)
	deviceID, err := s.readSysfsFile(devicePath, "device")
	if err != nil {
		return device, errors.Wrap(errors.GPUDetection, "failed to read device ID", err)
	}
	device.DeviceID = ParseHexID(deviceID)

	// Read class (required)
	class, err := s.readSysfsFile(devicePath, "class")
	if err != nil {
		return device, errors.Wrap(errors.GPUDetection, "failed to read device class", err)
	}
	device.Class = ParseHexID(class)

	// Read subsystem vendor (optional)
	subVendor, err := s.readSysfsFile(devicePath, "subsystem_vendor")
	if err == nil {
		device.SubVendorID = ParseHexID(subVendor)
	}

	// Read subsystem device (optional)
	subDevice, err := s.readSysfsFile(devicePath, "subsystem_device")
	if err == nil {
		device.SubDeviceID = ParseHexID(subDevice)
	}

	// Read revision (optional)
	revision, err := s.readSysfsFile(devicePath, "revision")
	if err == nil {
		device.Revision = ParseHexID(revision)
	}

	// Read driver via symlink (optional)
	device.Driver = s.readDriverLink(devicePath)

	return device, nil
}

// readSysfsFile reads a single sysfs file and returns its trimmed content.
func (s *ScannerImpl) readSysfsFile(devicePath, filename string) (string, error) {
	content, err := s.fs.ReadFile(filepath.Join(devicePath, filename))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// readDriverLink reads the driver symlink and returns the driver name.
// Returns empty string if no driver is bound.
func (s *ScannerImpl) readDriverLink(devicePath string) string {
	driverPath := filepath.Join(devicePath, "driver")

	// Check if driver symlink exists
	_, err := s.fs.Stat(driverPath)
	if err != nil {
		return ""
	}

	// Read the symlink target
	target, err := s.fs.Readlink(driverPath)
	if err != nil {
		return ""
	}

	// The target is something like "../../../bus/pci/drivers/nvidia"
	// We want just the driver name (last path component)
	return filepath.Base(target)
}

// Ensure ScannerImpl implements Scanner.
var _ Scanner = (*ScannerImpl)(nil)
