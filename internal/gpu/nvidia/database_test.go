package nvidia

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatabase(t *testing.T) {
	db := NewDatabase()
	require.NotNil(t, db)

	// Verify database is populated
	count := db.Count()
	assert.Greater(t, count, 0, "database should contain GPU models")
}

func TestDatabaseLookup(t *testing.T) {
	db := NewDatabase()

	tests := []struct {
		name     string
		deviceID string
		wantName string
		wantArch Architecture
		found    bool
	}{
		{
			name:     "RTX 4090 lowercase",
			deviceID: "2684",
			wantName: "GeForce RTX 4090",
			wantArch: ArchAdaLovelace,
			found:    true,
		},
		{
			name:     "RTX 4090 uppercase",
			deviceID: "2684",
			wantName: "GeForce RTX 4090",
			wantArch: ArchAdaLovelace,
			found:    true,
		},
		{
			name:     "RTX 4090 with 0x prefix",
			deviceID: "0x2684",
			wantName: "GeForce RTX 4090",
			wantArch: ArchAdaLovelace,
			found:    true,
		},
		{
			name:     "RTX 4090 with 0X prefix",
			deviceID: "0X2684",
			wantName: "GeForce RTX 4090",
			wantArch: ArchAdaLovelace,
			found:    true,
		},
		{
			name:     "RTX 3090",
			deviceID: "2205",
			wantName: "GeForce RTX 3090",
			wantArch: ArchAmpere,
			found:    true,
		},
		{
			name:     "RTX 2080 Ti",
			deviceID: "1e04",
			wantName: "GeForce RTX 2080 Ti",
			wantArch: ArchTuring,
			found:    true,
		},
		{
			name:     "GTX 1080 Ti",
			deviceID: "1b06",
			wantName: "GeForce GTX 1080 Ti",
			wantArch: ArchPascal,
			found:    true,
		},
		{
			name:     "Unknown device ID",
			deviceID: "ffff",
			wantName: "",
			wantArch: ArchUnknown,
			found:    false,
		},
		{
			name:     "Empty device ID",
			deviceID: "",
			wantName: "",
			wantArch: ArchUnknown,
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, ok := db.Lookup(tt.deviceID)
			assert.Equal(t, tt.found, ok)

			if tt.found {
				require.NotNil(t, model)
				assert.Equal(t, tt.wantName, model.Name)
				assert.Equal(t, tt.wantArch, model.Architecture)
			} else {
				assert.Nil(t, model)
			}
		})
	}
}

func TestDatabaseLookupByName(t *testing.T) {
	db := NewDatabase()

	tests := []struct {
		name       string
		searchName string
		wantID     string
		found      bool
	}{
		{
			name:       "Exact match",
			searchName: "GeForce RTX 4090",
			wantID:     "2684",
			found:      true,
		},
		{
			name:       "Case insensitive - lowercase",
			searchName: "geforce rtx 4090",
			wantID:     "2684",
			found:      true,
		},
		{
			name:       "Case insensitive - uppercase",
			searchName: "GEFORCE RTX 4090",
			wantID:     "2684",
			found:      true,
		},
		{
			name:       "Case insensitive - mixed",
			searchName: "GeForce RTX 3090",
			wantID:     "2205",
			found:      true,
		},
		{
			name:       "Data center GPU",
			searchName: "NVIDIA H100 PCIe",
			wantID:     "2322",
			found:      true,
		},
		{
			name:       "Unknown name",
			searchName: "GeForce RTX 9999",
			wantID:     "",
			found:      false,
		},
		{
			name:       "Empty name",
			searchName: "",
			wantID:     "",
			found:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, ok := db.LookupByName(tt.searchName)
			assert.Equal(t, tt.found, ok)

			if tt.found {
				require.NotNil(t, model)
				assert.Equal(t, tt.wantID, model.DeviceID)
			} else {
				assert.Nil(t, model)
			}
		})
	}
}

func TestDatabaseListByArchitecture(t *testing.T) {
	db := NewDatabase()

	tests := []struct {
		name      string
		arch      Architecture
		wantEmpty bool
	}{
		{
			name:      "Ada Lovelace GPUs",
			arch:      ArchAdaLovelace,
			wantEmpty: false,
		},
		{
			name:      "Ampere GPUs",
			arch:      ArchAmpere,
			wantEmpty: false,
		},
		{
			name:      "Turing GPUs",
			arch:      ArchTuring,
			wantEmpty: false,
		},
		{
			name:      "Pascal GPUs",
			arch:      ArchPascal,
			wantEmpty: false,
		},
		{
			name:      "Hopper GPUs",
			arch:      ArchHopper,
			wantEmpty: false,
		},
		{
			name:      "Blackwell GPUs",
			arch:      ArchBlackwell,
			wantEmpty: false,
		},
		{
			name:      "Volta GPUs",
			arch:      ArchVolta,
			wantEmpty: false,
		},
		{
			name:      "Unknown architecture",
			arch:      ArchUnknown,
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			models := db.ListByArchitecture(tt.arch)
			if tt.wantEmpty {
				assert.Empty(t, models)
			} else {
				assert.NotEmpty(t, models)
				// Verify all returned models have the correct architecture
				for _, m := range models {
					assert.Equal(t, tt.arch, m.Architecture)
				}
			}
		})
	}
}

func TestDatabaseGetMinDriverVersion(t *testing.T) {
	db := NewDatabase()

	tests := []struct {
		name       string
		deviceID   string
		wantDriver string
		wantErr    bool
	}{
		{
			name:       "RTX 4090 driver version",
			deviceID:   "2684",
			wantDriver: "525.60",
			wantErr:    false,
		},
		{
			name:       "RTX 3090 driver version",
			deviceID:   "2205",
			wantDriver: "455.23",
			wantErr:    false,
		},
		{
			name:       "GTX 1080 Ti driver version",
			deviceID:   "1b06",
			wantDriver: "384.59",
			wantErr:    false,
		},
		{
			name:       "Unknown device",
			deviceID:   "ffff",
			wantDriver: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver, err := db.GetMinDriverVersion(tt.deviceID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantDriver, driver)
			}
		})
	}
}

func TestDatabaseAllModels(t *testing.T) {
	db := NewDatabase()
	models := db.AllModels()

	assert.NotEmpty(t, models)
	assert.Equal(t, db.Count(), len(models))

	// Verify each model has required fields
	for _, m := range models {
		assert.NotEmpty(t, m.DeviceID, "model should have a device ID")
		assert.NotEmpty(t, m.Name, "model should have a name")
		assert.NotEqual(t, ArchUnknown, m.Architecture, "model should have a valid architecture")
		assert.NotEmpty(t, m.MinDriverVersion, "model should have a minimum driver version")
		assert.NotEmpty(t, m.ComputeCapability, "model should have a compute capability")
	}
}

func TestDatabaseCount(t *testing.T) {
	db := NewDatabase()
	count := db.Count()

	// We expect at least the number of models we defined
	assert.GreaterOrEqual(t, count, 40, "database should contain at least 40 GPU models")
}

func TestDatabaseThreadSafety(t *testing.T) {
	db := NewDatabase()
	var wg sync.WaitGroup

	// Run concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Mix of different operations
			switch id % 5 {
			case 0:
				db.Lookup("2684")
			case 1:
				db.LookupByName("GeForce RTX 4090")
			case 2:
				db.ListByArchitecture(ArchAdaLovelace)
			case 3:
				db.AllModels()
			case 4:
				db.Count()
			}
		}(i)
	}

	wg.Wait()
}

func TestGPUModelString(t *testing.T) {
	consumerGPU := GPUModel{
		DeviceID:     "2684",
		Name:         "GeForce RTX 4090",
		Architecture: ArchAdaLovelace,
		IsDataCenter: false,
	}
	assert.Equal(t, "GeForce RTX 4090", consumerGPU.String())

	dataCenterGPU := GPUModel{
		DeviceID:     "2322",
		Name:         "NVIDIA H100 PCIe",
		Architecture: ArchHopper,
		IsDataCenter: true,
	}
	assert.Equal(t, "NVIDIA H100 PCIe (Data Center)", dataCenterGPU.String())
}

func TestArchitectureString(t *testing.T) {
	tests := []struct {
		arch     Architecture
		expected string
	}{
		{ArchAdaLovelace, "ada"},
		{ArchAmpere, "ampere"},
		{ArchTuring, "turing"},
		{ArchPascal, "pascal"},
		{ArchHopper, "hopper"},
		{ArchBlackwell, "blackwell"},
		{ArchVolta, "volta"},
		{ArchKepler, "kepler"},
		{ArchMaxwell, "maxwell"},
		{ArchUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.arch.String())
		})
	}
}

func TestArchitectureIsValid(t *testing.T) {
	validArchitectures := []Architecture{
		ArchAdaLovelace, ArchAmpere, ArchTuring, ArchPascal,
		ArchHopper, ArchBlackwell, ArchVolta, ArchKepler, ArchMaxwell,
	}

	for _, arch := range validArchitectures {
		t.Run(arch.String(), func(t *testing.T) {
			assert.True(t, arch.IsValid())
		})
	}

	assert.False(t, ArchUnknown.IsValid())
	assert.False(t, Architecture("invalid").IsValid())
}

func TestArchitectureMinDriverVersion(t *testing.T) {
	tests := []struct {
		arch     Architecture
		expected string
	}{
		{ArchBlackwell, "560.00"},
		{ArchHopper, "525.60"},
		{ArchAdaLovelace, "525.60"},
		{ArchAmpere, "455.23"},
		{ArchTuring, "418.39"},
		{ArchVolta, "396.24"},
		{ArchPascal, "384.59"},
		{ArchMaxwell, "340.21"},
		{ArchKepler, "304.64"},
		{ArchUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.arch.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.arch.MinDriverVersion())
		})
	}
}

func TestArchitectureComputeCapability(t *testing.T) {
	tests := []struct {
		arch     Architecture
		expected string
	}{
		{ArchBlackwell, "10.0"},
		{ArchHopper, "9.0"},
		{ArchAdaLovelace, "8.9"},
		{ArchAmpere, "8.6"},
		{ArchTuring, "7.5"},
		{ArchVolta, "7.0"},
		{ArchPascal, "6.1"},
		{ArchMaxwell, "5.2"},
		{ArchKepler, "3.5"},
		{ArchUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.arch.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.arch.ComputeCapability())
		})
	}
}

func TestAllArchitectures(t *testing.T) {
	archs := AllArchitectures()
	assert.NotEmpty(t, archs)

	// Verify all architectures are valid
	for _, arch := range archs {
		assert.True(t, arch.IsValid())
	}

	// Verify Unknown is not included
	for _, arch := range archs {
		assert.NotEqual(t, ArchUnknown, arch)
	}
}

func TestNormalizeDeviceID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2684", "2684"},
		{"0x2684", "2684"},
		{"0X2684", "2684"},
		{"2684 ", "2684"},
		{" 2684", "2684"},
		{"ABCD", "abcd"},
		{"0xABCD", "abcd"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeDeviceID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLookupReturnsACopy(t *testing.T) {
	db := NewDatabase()

	// Get the model
	model1, ok := db.Lookup("2684")
	require.True(t, ok)

	// Modify the returned model
	model1.Name = "Modified Name"

	// Get the model again
	model2, ok := db.Lookup("2684")
	require.True(t, ok)

	// The second lookup should return the original name
	assert.Equal(t, "GeForce RTX 4090", model2.Name)
	assert.NotEqual(t, model1.Name, model2.Name)
}

func TestDataCenterGPUs(t *testing.T) {
	db := NewDatabase()

	dataCenterGPUs := []string{
		"NVIDIA H100 PCIe",
		"NVIDIA H100 SXM",
		"NVIDIA A100 PCIe 40GB",
		"NVIDIA A100 PCIe 80GB",
		"NVIDIA V100 PCIe 16GB",
	}

	for _, name := range dataCenterGPUs {
		t.Run(name, func(t *testing.T) {
			model, ok := db.LookupByName(name)
			require.True(t, ok, "GPU %s should be in database", name)
			assert.True(t, model.IsDataCenter, "GPU %s should be marked as data center", name)
		})
	}
}

func TestConsumerGPUs(t *testing.T) {
	db := NewDatabase()

	consumerGPUs := []string{
		"GeForce RTX 4090",
		"GeForce RTX 3090",
		"GeForce RTX 2080 Ti",
		"GeForce GTX 1080 Ti",
		"GeForce GTX 1660 Ti",
	}

	for _, name := range consumerGPUs {
		t.Run(name, func(t *testing.T) {
			model, ok := db.LookupByName(name)
			require.True(t, ok, "GPU %s should be in database", name)
			assert.False(t, model.IsDataCenter, "GPU %s should not be marked as data center", name)
		})
	}
}

func TestGetDefaultDatabase(t *testing.T) {
	db := GetDefaultDatabase()
	require.NotNil(t, db)
	assert.Greater(t, db.Count(), 0)
}

func TestLookupDeviceConvenience(t *testing.T) {
	model, ok := LookupDevice("2684")
	require.True(t, ok)
	assert.Equal(t, "GeForce RTX 4090", model.Name)
}

func TestLookupDeviceByNameConvenience(t *testing.T) {
	model, ok := LookupDeviceByName("GeForce RTX 4090")
	require.True(t, ok)
	assert.Equal(t, "2684", model.DeviceID)
}

func TestDatabaseContainsExpectedGPUs(t *testing.T) {
	db := NewDatabase()

	// Test Ada Lovelace GPUs
	adaGPUs := []string{"RTX 4090", "RTX 4080", "RTX 4070", "RTX 4060"}
	for _, gpu := range adaGPUs {
		models := db.ListByArchitecture(ArchAdaLovelace)
		found := false
		for _, m := range models {
			if strings.Contains(m.Name, gpu) {
				found = true
				break
			}
		}
		assert.True(t, found, "Ada Lovelace should contain %s", gpu)
	}

	// Test Ampere GPUs
	ampereGPUs := []string{"RTX 3090", "RTX 3080", "RTX 3070", "RTX 3060"}
	for _, gpu := range ampereGPUs {
		models := db.ListByArchitecture(ArchAmpere)
		found := false
		for _, m := range models {
			if strings.Contains(m.Name, gpu) {
				found = true
				break
			}
		}
		assert.True(t, found, "Ampere should contain %s", gpu)
	}

	// Test Turing GPUs
	turingGPUs := []string{"RTX 2080", "RTX 2070", "GTX 1660", "GTX 1650"}
	for _, gpu := range turingGPUs {
		models := db.ListByArchitecture(ArchTuring)
		found := false
		for _, m := range models {
			if strings.Contains(m.Name, gpu) {
				found = true
				break
			}
		}
		assert.True(t, found, "Turing should contain %s", gpu)
	}

	// Test Pascal GPUs
	pascalGPUs := []string{"GTX 1080", "GTX 1070", "GTX 1060", "GTX 1050"}
	for _, gpu := range pascalGPUs {
		models := db.ListByArchitecture(ArchPascal)
		found := false
		for _, m := range models {
			if strings.Contains(m.Name, gpu) {
				found = true
				break
			}
		}
		assert.True(t, found, "Pascal should contain %s", gpu)
	}
}

func TestMemorySizePopulated(t *testing.T) {
	db := NewDatabase()
	models := db.AllModels()

	for _, m := range models {
		assert.NotEmpty(t, m.MemorySize, "GPU %s should have memory size", m.Name)
		assert.True(t, strings.HasSuffix(m.MemorySize, "GB"), "Memory size should end with GB: %s", m.MemorySize)
	}
}

func TestNoDuplicateDeviceIDs(t *testing.T) {
	models := allGPUModels()
	seen := make(map[string]string) // deviceID -> name

	for _, m := range models {
		id := strings.ToLower(m.DeviceID)
		if existing, ok := seen[id]; ok {
			t.Errorf("Duplicate device ID %s: %s and %s", id, existing, m.Name)
		}
		seen[id] = m.Name
	}
}
