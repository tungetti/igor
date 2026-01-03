// Package pci provides PCI device scanning capabilities for GPU detection.
// It reads from /sys/bus/pci/devices to enumerate PCI devices and identify
// NVIDIA GPUs based on vendor ID and device class.
package pci

import (
	"fmt"
	"strings"
)

// Standard PCI constants for GPU identification.
const (
	// VendorNVIDIA is the PCI vendor ID for NVIDIA Corporation.
	VendorNVIDIA = "10de"

	// ClassVGA is the PCI class code for VGA compatible controller.
	ClassVGA = "0300"

	// Class3DController is the PCI class code for 3D controller (compute GPUs).
	Class3DController = "0302"

	// ClassDisplayController is the PCI class code for display controller.
	ClassDisplayController = "0380"
)

// Common driver names for GPU devices.
const (
	DriverNVIDIA  = "nvidia"
	DriverNouveau = "nouveau"
	DriverVFIOPCI = "vfio-pci"
	DriverNone    = ""
)

// PCIDevice represents a PCI device discovered from sysfs.
type PCIDevice struct {
	// Address is the PCI bus address (e.g., "0000:01:00.0")
	Address string

	// VendorID is the PCI vendor ID (e.g., "10de" for NVIDIA)
	VendorID string

	// DeviceID is the PCI device ID (e.g., "2684" for RTX 4090)
	DeviceID string

	// Class is the PCI class code (e.g., "0300" for VGA controller)
	Class string

	// SubVendorID is the PCI subsystem vendor ID
	SubVendorID string

	// SubDeviceID is the PCI subsystem device ID
	SubDeviceID string

	// Driver is the currently bound kernel driver (nvidia, nouveau, vfio-pci, or empty)
	Driver string

	// Revision is the device revision ID
	Revision string
}

// IsNVIDIA returns true if this device is manufactured by NVIDIA.
func (d *PCIDevice) IsNVIDIA() bool {
	return strings.EqualFold(d.VendorID, VendorNVIDIA)
}

// IsGPU returns true if this device is a GPU (VGA, 3D controller, or display controller).
func (d *PCIDevice) IsGPU() bool {
	// Class codes are typically 6 digits, we check the first 4 (base class + sub class)
	classPrefix := d.classPrefix()
	return classPrefix == ClassVGA || classPrefix == Class3DController || classPrefix == ClassDisplayController
}

// IsNVIDIAGPU returns true if this device is an NVIDIA GPU.
func (d *PCIDevice) IsNVIDIAGPU() bool {
	return d.IsNVIDIA() && d.IsGPU()
}

// HasDriver returns true if a driver is currently bound to this device.
func (d *PCIDevice) HasDriver() bool {
	return d.Driver != ""
}

// IsUsingProprietaryDriver returns true if the NVIDIA proprietary driver is bound.
func (d *PCIDevice) IsUsingProprietaryDriver() bool {
	return d.Driver == DriverNVIDIA
}

// IsUsingNouveau returns true if the nouveau open-source driver is bound.
func (d *PCIDevice) IsUsingNouveau() bool {
	return d.Driver == DriverNouveau
}

// IsUsingVFIO returns true if the device is bound to vfio-pci for passthrough.
func (d *PCIDevice) IsUsingVFIO() bool {
	return d.Driver == DriverVFIOPCI
}

// classPrefix returns the first 4 characters of the class code (base class + sub class).
func (d *PCIDevice) classPrefix() string {
	if len(d.Class) >= 4 {
		return d.Class[:4]
	}
	return d.Class
}

// String returns a human-readable representation of the device.
func (d *PCIDevice) String() string {
	driverInfo := "no driver"
	if d.Driver != "" {
		driverInfo = fmt.Sprintf("driver: %s", d.Driver)
	}
	return fmt.Sprintf("PCI %s [%s:%s] class %s (%s)", d.Address, d.VendorID, d.DeviceID, d.Class, driverInfo)
}

// PCIID returns the full PCI ID in the format "vendor:device:subvendor:subdevice".
func (d *PCIDevice) PCIID() string {
	return fmt.Sprintf("%s:%s:%s:%s", d.VendorID, d.DeviceID, d.SubVendorID, d.SubDeviceID)
}

// ShortID returns the PCI ID in the format "vendor:device".
func (d *PCIDevice) ShortID() string {
	return fmt.Sprintf("%s:%s", d.VendorID, d.DeviceID)
}

// ParseHexID normalizes a hex ID string by removing "0x" prefix and converting to lowercase.
func ParseHexID(s string) string {
	s = strings.TrimSpace(s)
	// Handle both lowercase and uppercase hex prefix
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	s = strings.ToLower(s)
	return s
}

// MatchesVendor checks if the device matches the given vendor ID.
// The vendor ID can be with or without "0x" prefix.
func (d *PCIDevice) MatchesVendor(vendorID string) bool {
	return strings.EqualFold(d.VendorID, ParseHexID(vendorID))
}

// MatchesClass checks if the device matches the given class code prefix.
// The class can be with or without "0x" prefix.
// This matches if the device class starts with the given prefix.
func (d *PCIDevice) MatchesClass(classCode string) bool {
	classCode = ParseHexID(classCode)
	return strings.HasPrefix(strings.ToLower(d.Class), classCode)
}
