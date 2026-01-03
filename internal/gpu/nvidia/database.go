package nvidia

import (
	"strings"
	"sync"

	"github.com/tungetti/igor/internal/errors"
)

// Database provides GPU model lookup functionality.
type Database interface {
	// Lookup returns the GPU model for a device ID.
	// Returns nil and false if the device ID is not found.
	Lookup(deviceID string) (*GPUModel, bool)

	// LookupByName returns the GPU model by name.
	// The lookup is case-insensitive.
	// Returns nil and false if the name is not found.
	LookupByName(name string) (*GPUModel, bool)

	// ListByArchitecture returns all GPUs of a given architecture.
	// Returns an empty slice if no GPUs match the architecture.
	ListByArchitecture(arch Architecture) []GPUModel

	// GetMinDriverVersion returns minimum driver version for a device.
	// Returns an error if the device ID is not found.
	GetMinDriverVersion(deviceID string) (string, error)

	// AllModels returns all GPU models in the database.
	AllModels() []GPUModel

	// Count returns the number of models in the database.
	Count() int
}

// DatabaseImpl implements the Database interface with map-based storage.
type DatabaseImpl struct {
	mu         sync.RWMutex
	byDeviceID map[string]*GPUModel
	byName     map[string]*GPUModel
}

// NewDatabase creates a new Database populated with NVIDIA GPU data.
func NewDatabase() Database {
	db := &DatabaseImpl{
		byDeviceID: make(map[string]*GPUModel),
		byName:     make(map[string]*GPUModel),
	}

	// Populate the database with GPU models
	for _, model := range allGPUModels() {
		m := model // Create a copy to avoid pointer issues
		db.byDeviceID[strings.ToLower(m.DeviceID)] = &m
		db.byName[strings.ToLower(m.Name)] = &m
	}

	return db
}

// Lookup returns the GPU model for a device ID.
func (db *DatabaseImpl) Lookup(deviceID string) (*GPUModel, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Normalize the device ID: remove 0x prefix and convert to lowercase
	deviceID = normalizeDeviceID(deviceID)

	model, ok := db.byDeviceID[deviceID]
	if !ok {
		return nil, false
	}
	// Return a copy to prevent modification of internal state
	copy := *model
	return &copy, true
}

// LookupByName returns the GPU model by name (case-insensitive).
func (db *DatabaseImpl) LookupByName(name string) (*GPUModel, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	model, ok := db.byName[strings.ToLower(name)]
	if !ok {
		return nil, false
	}
	// Return a copy to prevent modification of internal state
	copy := *model
	return &copy, true
}

// ListByArchitecture returns all GPUs of a given architecture.
func (db *DatabaseImpl) ListByArchitecture(arch Architecture) []GPUModel {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var result []GPUModel
	for _, model := range db.byDeviceID {
		if model.Architecture == arch {
			result = append(result, *model)
		}
	}
	return result
}

// GetMinDriverVersion returns minimum driver version for a device.
func (db *DatabaseImpl) GetMinDriverVersion(deviceID string) (string, error) {
	model, ok := db.Lookup(deviceID)
	if !ok {
		return "", errors.Newf(errors.NotFound, "device ID %q not found in database", deviceID)
	}
	return model.MinDriverVersion, nil
}

// AllModels returns all GPU models in the database.
func (db *DatabaseImpl) AllModels() []GPUModel {
	db.mu.RLock()
	defer db.mu.RUnlock()

	result := make([]GPUModel, 0, len(db.byDeviceID))
	for _, model := range db.byDeviceID {
		result = append(result, *model)
	}
	return result
}

// Count returns the number of models in the database.
func (db *DatabaseImpl) Count() int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return len(db.byDeviceID)
}

// normalizeDeviceID normalizes a device ID by removing the 0x prefix
// and converting to lowercase.
func normalizeDeviceID(deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	deviceID = strings.TrimPrefix(deviceID, "0x")
	deviceID = strings.TrimPrefix(deviceID, "0X")
	return strings.ToLower(deviceID)
}

// allGPUModels returns a slice of all GPU models to populate the database.
func allGPUModels() []GPUModel {
	return []GPUModel{
		// =========================================================================
		// Ada Lovelace (RTX 40xx) - Compute Capability 8.9, Min Driver 525.60
		// =========================================================================
		{DeviceID: "2684", Name: "GeForce RTX 4090", Architecture: ArchAdaLovelace, MinDriverVersion: "525.60", ComputeCapability: "8.9", MemorySize: "24GB", IsDataCenter: false},
		{DeviceID: "2702", Name: "GeForce RTX 4080 Super", Architecture: ArchAdaLovelace, MinDriverVersion: "525.60", ComputeCapability: "8.9", MemorySize: "16GB", IsDataCenter: false},
		{DeviceID: "2704", Name: "GeForce RTX 4080", Architecture: ArchAdaLovelace, MinDriverVersion: "525.60", ComputeCapability: "8.9", MemorySize: "16GB", IsDataCenter: false},
		{DeviceID: "2705", Name: "GeForce RTX 4070 Ti Super", Architecture: ArchAdaLovelace, MinDriverVersion: "525.60", ComputeCapability: "8.9", MemorySize: "16GB", IsDataCenter: false},
		{DeviceID: "2782", Name: "GeForce RTX 4070 Ti", Architecture: ArchAdaLovelace, MinDriverVersion: "525.60", ComputeCapability: "8.9", MemorySize: "12GB", IsDataCenter: false},
		{DeviceID: "2783", Name: "GeForce RTX 4070 Super", Architecture: ArchAdaLovelace, MinDriverVersion: "525.60", ComputeCapability: "8.9", MemorySize: "12GB", IsDataCenter: false},
		{DeviceID: "2786", Name: "GeForce RTX 4070", Architecture: ArchAdaLovelace, MinDriverVersion: "525.60", ComputeCapability: "8.9", MemorySize: "12GB", IsDataCenter: false},
		{DeviceID: "2882", Name: "GeForce RTX 4060 Ti", Architecture: ArchAdaLovelace, MinDriverVersion: "525.60", ComputeCapability: "8.9", MemorySize: "8GB", IsDataCenter: false},
		{DeviceID: "2860", Name: "GeForce RTX 4060", Architecture: ArchAdaLovelace, MinDriverVersion: "525.60", ComputeCapability: "8.9", MemorySize: "8GB", IsDataCenter: false},

		// =========================================================================
		// Ampere (RTX 30xx) - Compute Capability 8.6, Min Driver 455.23
		// =========================================================================
		{DeviceID: "2204", Name: "GeForce RTX 3090 Ti", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.6", MemorySize: "24GB", IsDataCenter: false},
		{DeviceID: "2205", Name: "GeForce RTX 3090", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.6", MemorySize: "24GB", IsDataCenter: false},
		{DeviceID: "2208", Name: "GeForce RTX 3080 Ti", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.6", MemorySize: "12GB", IsDataCenter: false},
		{DeviceID: "2206", Name: "GeForce RTX 3080", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.6", MemorySize: "10GB", IsDataCenter: false},
		{DeviceID: "2484", Name: "GeForce RTX 3070 Ti", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.6", MemorySize: "8GB", IsDataCenter: false},
		{DeviceID: "2488", Name: "GeForce RTX 3070", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.6", MemorySize: "8GB", IsDataCenter: false},
		{DeviceID: "2489", Name: "GeForce RTX 3060 Ti", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.6", MemorySize: "8GB", IsDataCenter: false},
		{DeviceID: "2503", Name: "GeForce RTX 3060", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.6", MemorySize: "12GB", IsDataCenter: false},
		{DeviceID: "2584", Name: "GeForce RTX 3050", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.6", MemorySize: "8GB", IsDataCenter: false},

		// =========================================================================
		// Turing (RTX 20xx) - Compute Capability 7.5, Min Driver 418.39
		// =========================================================================
		{DeviceID: "1e04", Name: "GeForce RTX 2080 Ti", Architecture: ArchTuring, MinDriverVersion: "418.39", ComputeCapability: "7.5", MemorySize: "11GB", IsDataCenter: false},
		{DeviceID: "1e81", Name: "GeForce RTX 2080 Super", Architecture: ArchTuring, MinDriverVersion: "418.39", ComputeCapability: "7.5", MemorySize: "8GB", IsDataCenter: false},
		{DeviceID: "1e82", Name: "GeForce RTX 2080", Architecture: ArchTuring, MinDriverVersion: "418.39", ComputeCapability: "7.5", MemorySize: "8GB", IsDataCenter: false},
		{DeviceID: "1e84", Name: "GeForce RTX 2070 Super", Architecture: ArchTuring, MinDriverVersion: "418.39", ComputeCapability: "7.5", MemorySize: "8GB", IsDataCenter: false},
		{DeviceID: "1f02", Name: "GeForce RTX 2070", Architecture: ArchTuring, MinDriverVersion: "418.39", ComputeCapability: "7.5", MemorySize: "8GB", IsDataCenter: false},
		{DeviceID: "1f06", Name: "GeForce RTX 2060 Super", Architecture: ArchTuring, MinDriverVersion: "418.39", ComputeCapability: "7.5", MemorySize: "8GB", IsDataCenter: false},
		{DeviceID: "1f08", Name: "GeForce RTX 2060", Architecture: ArchTuring, MinDriverVersion: "418.39", ComputeCapability: "7.5", MemorySize: "6GB", IsDataCenter: false},

		// =========================================================================
		// Turing (GTX 16xx) - Compute Capability 7.5, Min Driver 418.39
		// =========================================================================
		{DeviceID: "2182", Name: "GeForce GTX 1660 Ti", Architecture: ArchTuring, MinDriverVersion: "418.39", ComputeCapability: "7.5", MemorySize: "6GB", IsDataCenter: false},
		{DeviceID: "21c4", Name: "GeForce GTX 1660 Super", Architecture: ArchTuring, MinDriverVersion: "418.39", ComputeCapability: "7.5", MemorySize: "6GB", IsDataCenter: false},
		{DeviceID: "2184", Name: "GeForce GTX 1660", Architecture: ArchTuring, MinDriverVersion: "418.39", ComputeCapability: "7.5", MemorySize: "6GB", IsDataCenter: false},
		{DeviceID: "2188", Name: "GeForce GTX 1650 Super", Architecture: ArchTuring, MinDriverVersion: "418.39", ComputeCapability: "7.5", MemorySize: "4GB", IsDataCenter: false},
		{DeviceID: "1f82", Name: "GeForce GTX 1650", Architecture: ArchTuring, MinDriverVersion: "418.39", ComputeCapability: "7.5", MemorySize: "4GB", IsDataCenter: false},

		// =========================================================================
		// Pascal (GTX 10xx) - Compute Capability 6.1, Min Driver 384.59
		// =========================================================================
		{DeviceID: "1b06", Name: "GeForce GTX 1080 Ti", Architecture: ArchPascal, MinDriverVersion: "384.59", ComputeCapability: "6.1", MemorySize: "11GB", IsDataCenter: false},
		{DeviceID: "1b80", Name: "GeForce GTX 1080", Architecture: ArchPascal, MinDriverVersion: "384.59", ComputeCapability: "6.1", MemorySize: "8GB", IsDataCenter: false},
		{DeviceID: "1b82", Name: "GeForce GTX 1070 Ti", Architecture: ArchPascal, MinDriverVersion: "384.59", ComputeCapability: "6.1", MemorySize: "8GB", IsDataCenter: false},
		{DeviceID: "1b81", Name: "GeForce GTX 1070", Architecture: ArchPascal, MinDriverVersion: "384.59", ComputeCapability: "6.1", MemorySize: "8GB", IsDataCenter: false},
		{DeviceID: "1c03", Name: "GeForce GTX 1060 6GB", Architecture: ArchPascal, MinDriverVersion: "384.59", ComputeCapability: "6.1", MemorySize: "6GB", IsDataCenter: false},
		{DeviceID: "1c81", Name: "GeForce GTX 1050 Ti", Architecture: ArchPascal, MinDriverVersion: "384.59", ComputeCapability: "6.1", MemorySize: "4GB", IsDataCenter: false},
		{DeviceID: "1c82", Name: "GeForce GTX 1050", Architecture: ArchPascal, MinDriverVersion: "384.59", ComputeCapability: "6.1", MemorySize: "2GB", IsDataCenter: false},

		// =========================================================================
		// Blackwell (Data Center) - Compute Capability 10.0, Min Driver 560.00
		// =========================================================================
		{DeviceID: "2900", Name: "NVIDIA B200", Architecture: ArchBlackwell, MinDriverVersion: "560.00", ComputeCapability: "10.0", MemorySize: "192GB", IsDataCenter: true},
		{DeviceID: "2901", Name: "NVIDIA B100", Architecture: ArchBlackwell, MinDriverVersion: "560.00", ComputeCapability: "10.0", MemorySize: "192GB", IsDataCenter: true},

		// =========================================================================
		// Hopper (Data Center) - Compute Capability 9.0, Min Driver 525.60
		// =========================================================================
		{DeviceID: "2330", Name: "NVIDIA H200", Architecture: ArchHopper, MinDriverVersion: "525.60", ComputeCapability: "9.0", MemorySize: "141GB", IsDataCenter: true},
		{DeviceID: "2322", Name: "NVIDIA H100 PCIe", Architecture: ArchHopper, MinDriverVersion: "525.60", ComputeCapability: "9.0", MemorySize: "80GB", IsDataCenter: true},
		{DeviceID: "2324", Name: "NVIDIA H100 SXM", Architecture: ArchHopper, MinDriverVersion: "525.60", ComputeCapability: "9.0", MemorySize: "80GB", IsDataCenter: true},

		// =========================================================================
		// Ampere (Data Center) - Compute Capability 8.0, Min Driver 455.23
		// =========================================================================
		{DeviceID: "20b0", Name: "NVIDIA A100 PCIe 40GB", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.0", MemorySize: "40GB", IsDataCenter: true},
		{DeviceID: "20b2", Name: "NVIDIA A100 PCIe 80GB", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.0", MemorySize: "80GB", IsDataCenter: true},
		{DeviceID: "20b5", Name: "NVIDIA A100 SXM4 40GB", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.0", MemorySize: "40GB", IsDataCenter: true},
		{DeviceID: "20b7", Name: "NVIDIA A100 SXM4 80GB", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.0", MemorySize: "80GB", IsDataCenter: true},
		{DeviceID: "2235", Name: "NVIDIA A40", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.6", MemorySize: "48GB", IsDataCenter: true},
		{DeviceID: "20b8", Name: "NVIDIA A30", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.0", MemorySize: "24GB", IsDataCenter: true},
		{DeviceID: "2236", Name: "NVIDIA A10", Architecture: ArchAmpere, MinDriverVersion: "455.23", ComputeCapability: "8.6", MemorySize: "24GB", IsDataCenter: true},

		// =========================================================================
		// Volta (Data Center) - Compute Capability 7.0, Min Driver 396.24
		// =========================================================================
		{DeviceID: "1db1", Name: "NVIDIA V100 PCIe 16GB", Architecture: ArchVolta, MinDriverVersion: "396.24", ComputeCapability: "7.0", MemorySize: "16GB", IsDataCenter: true},
		{DeviceID: "1db4", Name: "NVIDIA V100 PCIe 32GB", Architecture: ArchVolta, MinDriverVersion: "396.24", ComputeCapability: "7.0", MemorySize: "32GB", IsDataCenter: true},
		{DeviceID: "1db5", Name: "NVIDIA V100 SXM2 16GB", Architecture: ArchVolta, MinDriverVersion: "396.24", ComputeCapability: "7.0", MemorySize: "16GB", IsDataCenter: true},
		{DeviceID: "1db6", Name: "NVIDIA V100 SXM2 32GB", Architecture: ArchVolta, MinDriverVersion: "396.24", ComputeCapability: "7.0", MemorySize: "32GB", IsDataCenter: true},
	}
}

// DefaultDatabase is the default GPU database instance.
var defaultDatabase Database

// init initializes the default database.
func init() {
	defaultDatabase = NewDatabase()
}

// GetDefaultDatabase returns the default GPU database instance.
func GetDefaultDatabase() Database {
	return defaultDatabase
}

// LookupDevice is a convenience function to lookup a device in the default database.
func LookupDevice(deviceID string) (*GPUModel, bool) {
	return defaultDatabase.Lookup(deviceID)
}

// LookupDeviceByName is a convenience function to lookup a device by name in the default database.
func LookupDeviceByName(name string) (*GPUModel, bool) {
	return defaultDatabase.LookupByName(name)
}
