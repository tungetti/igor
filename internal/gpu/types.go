// Package gpu provides GPU detection and orchestration capabilities for Igor.
// It coordinates various detection components to provide comprehensive system
// GPU information and installation readiness assessment.
package gpu

import (
	"time"

	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/gpu/nouveau"
	"github.com/tungetti/igor/internal/gpu/nvidia"
	"github.com/tungetti/igor/internal/gpu/pci"
	"github.com/tungetti/igor/internal/gpu/smi"
	"github.com/tungetti/igor/internal/gpu/validator"
)

// DriverType represents the type of GPU driver.
type DriverType string

// Driver type constants.
const (
	// DriverTypeNVIDIA indicates the NVIDIA proprietary driver.
	DriverTypeNVIDIA DriverType = "nvidia"
	// DriverTypeNouveau indicates the Nouveau open-source driver.
	DriverTypeNouveau DriverType = "nouveau"
	// DriverTypeNone indicates no driver is installed/loaded.
	DriverTypeNone DriverType = "none"
)

// String returns the string representation of the driver type.
func (d DriverType) String() string {
	return string(d)
}

// DriverInfo represents installed driver information.
type DriverInfo struct {
	// Installed indicates whether any NVIDIA driver is installed.
	Installed bool

	// Type indicates the driver type ("nvidia", "nouveau", "none").
	Type DriverType

	// Version is the driver version (e.g., "550.54.14").
	// Empty if driver is not installed or unavailable.
	Version string

	// CUDAVersion is the supported CUDA version (e.g., "12.4").
	// Empty if CUDA is not available.
	CUDAVersion string
}

// NVIDIAGPUInfo combines hardware and database info for an NVIDIA GPU.
type NVIDIAGPUInfo struct {
	// PCIDevice contains the raw PCI device information.
	PCIDevice pci.PCIDevice

	// Model contains GPU model details from the database.
	// May be nil if the GPU is not in the database.
	Model *nvidia.GPUModel

	// SMIInfo contains runtime information from nvidia-smi.
	// May be nil if nvidia-smi is unavailable or driver not loaded.
	SMIInfo *smi.SMIGPUInfo
}

// Name returns the GPU name, preferring Model name over SMI name over device ID.
func (g *NVIDIAGPUInfo) Name() string {
	if g.Model != nil {
		return g.Model.Name
	}
	if g.SMIInfo != nil && g.SMIInfo.Name != "" {
		return g.SMIInfo.Name
	}
	return "NVIDIA GPU (Device ID: " + g.PCIDevice.DeviceID + ")"
}

// Architecture returns the GPU architecture if known.
func (g *NVIDIAGPUInfo) Architecture() string {
	if g.Model != nil {
		return g.Model.Architecture.String()
	}
	return "unknown"
}

// GPUInfo represents complete GPU information collected from all detection components.
type GPUInfo struct {
	// Hardware detection
	// PCIDevices contains all GPU PCI devices found (including non-NVIDIA).
	PCIDevices []pci.PCIDevice

	// NVIDIAGPUs contains detailed information for each NVIDIA GPU.
	NVIDIAGPUs []NVIDIAGPUInfo

	// Driver status
	// InstalledDriver contains information about the currently installed driver.
	InstalledDriver *DriverInfo

	// NouveauStatus contains the status of the Nouveau driver.
	NouveauStatus *nouveau.Status

	// System info
	// KernelInfo contains kernel version and module information.
	KernelInfo *kernel.KernelInfo

	// ValidationReport contains the system validation results.
	ValidationReport *validator.ValidationReport

	// Detection metadata
	// DetectionTime is when the detection was performed.
	DetectionTime time.Time

	// Duration is how long the detection took.
	Duration time.Duration

	// Errors contains any non-fatal errors encountered during detection.
	// Detection continues even if some components fail.
	Errors []error
}

// HasNVIDIAGPUs returns true if at least one NVIDIA GPU was detected.
func (g *GPUInfo) HasNVIDIAGPUs() bool {
	return len(g.NVIDIAGPUs) > 0
}

// GPUCount returns the number of NVIDIA GPUs detected.
func (g *GPUInfo) GPUCount() int {
	return len(g.NVIDIAGPUs)
}

// HasErrors returns true if any errors occurred during detection.
func (g *GPUInfo) HasErrors() bool {
	return len(g.Errors) > 0
}

// IsDriverInstalled returns true if any NVIDIA driver is installed.
func (g *GPUInfo) IsDriverInstalled() bool {
	return g.InstalledDriver != nil && g.InstalledDriver.Installed
}

// IsNouveauLoaded returns true if the Nouveau driver is currently loaded.
func (g *GPUInfo) IsNouveauLoaded() bool {
	return g.NouveauStatus != nil && g.NouveauStatus.Loaded
}

// HasValidationErrors returns true if the validation report contains errors.
func (g *GPUInfo) HasValidationErrors() bool {
	return g.ValidationReport != nil && g.ValidationReport.HasErrors()
}

// HasValidationWarnings returns true if the validation report contains warnings.
func (g *GPUInfo) HasValidationWarnings() bool {
	return g.ValidationReport != nil && g.ValidationReport.HasWarnings()
}
