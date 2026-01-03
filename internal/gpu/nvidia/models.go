// Package nvidia provides NVIDIA GPU identification and database functionality.
// It maps PCI device IDs to GPU models, architectures, and driver requirements,
// enabling automatic detection and validation of NVIDIA hardware.
package nvidia

// Architecture represents an NVIDIA GPU architecture generation.
type Architecture string

// NVIDIA GPU architecture constants.
const (
	// ArchKepler represents the Kepler architecture (GTX 600/700 series).
	ArchKepler Architecture = "kepler"

	// ArchMaxwell represents the Maxwell architecture (GTX 900 series).
	ArchMaxwell Architecture = "maxwell"

	// ArchPascal represents the Pascal architecture (GTX 10xx series).
	ArchPascal Architecture = "pascal"

	// ArchTuring represents the Turing architecture (RTX 20xx, GTX 16xx series).
	ArchTuring Architecture = "turing"

	// ArchAmpere represents the Ampere architecture (RTX 30xx series).
	ArchAmpere Architecture = "ampere"

	// ArchAdaLovelace represents the Ada Lovelace architecture (RTX 40xx series).
	ArchAdaLovelace Architecture = "ada"

	// ArchHopper represents the Hopper architecture (H100 series).
	ArchHopper Architecture = "hopper"

	// ArchBlackwell represents the Blackwell architecture (B100/B200 series).
	ArchBlackwell Architecture = "blackwell"

	// ArchVolta represents the Volta architecture (V100 series).
	ArchVolta Architecture = "volta"

	// ArchUnknown represents an unknown or unsupported architecture.
	ArchUnknown Architecture = "unknown"
)

// String returns the string representation of the architecture.
func (a Architecture) String() string {
	return string(a)
}

// IsValid returns true if the architecture is a known valid architecture.
func (a Architecture) IsValid() bool {
	switch a {
	case ArchKepler, ArchMaxwell, ArchPascal, ArchTuring, ArchAmpere,
		ArchAdaLovelace, ArchHopper, ArchBlackwell, ArchVolta:
		return true
	default:
		return false
	}
}

// MinDriverVersion returns the minimum driver version required for this architecture.
func (a Architecture) MinDriverVersion() string {
	return minDriverVersions[a]
}

// ComputeCapability returns the compute capability for this architecture.
func (a Architecture) ComputeCapability() string {
	return computeCapabilities[a]
}

// minDriverVersions maps architectures to their minimum supported driver versions.
var minDriverVersions = map[Architecture]string{
	ArchBlackwell:   "560.00",
	ArchHopper:      "525.60",
	ArchAdaLovelace: "525.60",
	ArchAmpere:      "455.23",
	ArchTuring:      "418.39",
	ArchVolta:       "396.24",
	ArchPascal:      "384.59",
	ArchMaxwell:     "340.21",
	ArchKepler:      "304.64",
	ArchUnknown:     "",
}

// computeCapabilities maps architectures to their CUDA compute capability.
var computeCapabilities = map[Architecture]string{
	ArchBlackwell:   "10.0",
	ArchHopper:      "9.0",
	ArchAdaLovelace: "8.9",
	ArchAmpere:      "8.6",
	ArchTuring:      "7.5",
	ArchVolta:       "7.0",
	ArchPascal:      "6.1",
	ArchMaxwell:     "5.2",
	ArchKepler:      "3.5",
	ArchUnknown:     "",
}

// GPUModel represents an NVIDIA GPU model with its specifications.
type GPUModel struct {
	// DeviceID is the PCI device ID without "0x" prefix (e.g., "2684")
	DeviceID string

	// Name is the marketing name of the GPU (e.g., "GeForce RTX 4090")
	Name string

	// Architecture is the GPU architecture generation
	Architecture Architecture

	// MinDriverVersion is the minimum supported driver version
	MinDriverVersion string

	// ComputeCapability is the CUDA compute capability (e.g., "8.9")
	ComputeCapability string

	// MemorySize is the GPU memory size (e.g., "24GB")
	MemorySize string

	// IsDataCenter indicates if this is a data center GPU (H100, A100, etc.)
	IsDataCenter bool
}

// String returns a human-readable representation of the GPU model.
func (m *GPUModel) String() string {
	if m.IsDataCenter {
		return m.Name + " (Data Center)"
	}
	return m.Name
}

// AllArchitectures returns a slice of all known architectures in chronological order.
func AllArchitectures() []Architecture {
	return []Architecture{
		ArchKepler,
		ArchMaxwell,
		ArchPascal,
		ArchVolta,
		ArchTuring,
		ArchAmpere,
		ArchAdaLovelace,
		ArchHopper,
		ArchBlackwell,
	}
}
