package nvidia

import (
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Integration Test: All Architectures
// =============================================================================

func TestDatabase_AllArchitectures(t *testing.T) {
	db := NewDatabase()

	architectureTests := []struct {
		arch          Architecture
		expectedCount int // minimum count
		sampleGPU     string
		minDriver     string
		computeCap    string
	}{
		{
			arch:          ArchAdaLovelace,
			expectedCount: 5,
			sampleGPU:     "GeForce RTX 4090",
			minDriver:     "525.60",
			computeCap:    "8.9",
		},
		{
			arch:          ArchAmpere,
			expectedCount: 10,
			sampleGPU:     "GeForce RTX 3090",
			minDriver:     "455.23",
			computeCap:    "8.6",
		},
		{
			arch:          ArchTuring,
			expectedCount: 10,
			sampleGPU:     "GeForce RTX 2080 Ti",
			minDriver:     "418.39",
			computeCap:    "7.5",
		},
		{
			arch:          ArchPascal,
			expectedCount: 5,
			sampleGPU:     "GeForce GTX 1080 Ti",
			minDriver:     "384.59",
			computeCap:    "6.1",
		},
		{
			arch:          ArchVolta,
			expectedCount: 3,
			sampleGPU:     "NVIDIA V100 PCIe 16GB",
			minDriver:     "396.24",
			computeCap:    "7.0",
		},
		{
			arch:          ArchHopper,
			expectedCount: 2,
			sampleGPU:     "NVIDIA H100 PCIe",
			minDriver:     "525.60",
			computeCap:    "9.0",
		},
		{
			arch:          ArchBlackwell,
			expectedCount: 2,
			sampleGPU:     "NVIDIA B200",
			minDriver:     "560.00",
			computeCap:    "10.0",
		},
	}

	for _, tc := range architectureTests {
		t.Run(tc.arch.String(), func(t *testing.T) {
			models := db.ListByArchitecture(tc.arch)

			assert.GreaterOrEqual(t, len(models), tc.expectedCount,
				"Expected at least %d GPUs for %s", tc.expectedCount, tc.arch)

			// Verify all models have correct architecture
			for _, m := range models {
				assert.Equal(t, tc.arch, m.Architecture)
				assert.NotEmpty(t, m.DeviceID)
				assert.NotEmpty(t, m.Name)
			}

			// Verify sample GPU exists
			found := false
			for _, m := range models {
				if m.Name == tc.sampleGPU {
					found = true
					assert.Equal(t, tc.minDriver, m.MinDriverVersion)
					assert.Equal(t, tc.computeCap, m.ComputeCapability)
					break
				}
			}
			assert.True(t, found, "Sample GPU %s not found in %s architecture", tc.sampleGPU, tc.arch)
		})
	}

	t.Run("unknown architecture returns empty", func(t *testing.T) {
		models := db.ListByArchitecture(ArchUnknown)
		assert.Empty(t, models)
	})

	t.Run("architecture validation", func(t *testing.T) {
		for _, arch := range AllArchitectures() {
			assert.True(t, arch.IsValid())
			assert.NotEmpty(t, arch.MinDriverVersion())
			assert.NotEmpty(t, arch.ComputeCapability())
		}
	})
}

// =============================================================================
// Integration Test: Driver Recommendations
// =============================================================================

func TestDatabase_DriverRecommendation(t *testing.T) {
	db := NewDatabase()

	t.Run("minimum driver per architecture", func(t *testing.T) {
		archDrivers := map[Architecture]string{
			ArchBlackwell:   "560.00",
			ArchHopper:      "525.60",
			ArchAdaLovelace: "525.60",
			ArchAmpere:      "455.23",
			ArchTuring:      "418.39",
			ArchVolta:       "396.24",
			ArchPascal:      "384.59",
			ArchMaxwell:     "340.21",
			ArchKepler:      "304.64",
		}

		for arch, expectedMin := range archDrivers {
			assert.Equal(t, expectedMin, arch.MinDriverVersion(),
				"Wrong min driver for %s", arch)
		}
	})

	t.Run("minimum driver per GPU", func(t *testing.T) {
		gpuDrivers := []struct {
			deviceID  string
			minDriver string
		}{
			{"2684", "525.60"}, // RTX 4090
			{"2205", "455.23"}, // RTX 3090
			{"1e04", "418.39"}, // RTX 2080 Ti
			{"1b06", "384.59"}, // GTX 1080 Ti
			{"2322", "525.60"}, // H100 PCIe
			{"20b2", "455.23"}, // A100 80GB
			{"1db1", "396.24"}, // V100 PCIe
		}

		for _, tc := range gpuDrivers {
			driver, err := db.GetMinDriverVersion(tc.deviceID)
			require.NoError(t, err)
			assert.Equal(t, tc.minDriver, driver,
				"Wrong min driver for device %s", tc.deviceID)
		}
	})

	t.Run("GPU with required driver version", func(t *testing.T) {
		model, found := db.Lookup("2684")
		require.True(t, found)

		assert.Equal(t, "525.60", model.MinDriverVersion)
		assert.Equal(t, "8.9", model.ComputeCapability)
	})

	t.Run("data center GPUs require specific drivers", func(t *testing.T) {
		dcGPUs := []string{"2322", "2324", "20b2", "1db1"} // H100, A100, V100

		for _, deviceID := range dcGPUs {
			model, found := db.Lookup(deviceID)
			require.True(t, found, "DC GPU %s not found", deviceID)
			assert.True(t, model.IsDataCenter)
			assert.NotEmpty(t, model.MinDriverVersion)
		}
	})
}

// =============================================================================
// Integration Test: Legacy GPUs
// =============================================================================

func TestDatabase_LegacyGPUs(t *testing.T) {
	db := NewDatabase()

	t.Run("Pascal GPUs with legacy-capable drivers", func(t *testing.T) {
		pascalGPUs := db.ListByArchitecture(ArchPascal)
		require.NotEmpty(t, pascalGPUs)

		for _, gpu := range pascalGPUs {
			// Pascal requires at least 384.59
			assert.Equal(t, "384.59", gpu.MinDriverVersion)
			assert.Equal(t, "6.1", gpu.ComputeCapability)
			assert.False(t, gpu.IsDataCenter)
		}
	})

	t.Run("Turing GTX 16xx series", func(t *testing.T) {
		turingGPUs := db.ListByArchitecture(ArchTuring)

		gtx16xxCount := 0
		for _, gpu := range turingGPUs {
			if gpu.Name[:3] == "GeF" && gpu.Name[12:14] == "16" {
				gtx16xxCount++
				assert.Equal(t, "7.5", gpu.ComputeCapability)
			}
		}

		assert.GreaterOrEqual(t, gtx16xxCount, 4, "Expected at least 4 GTX 16xx models")
	})

	t.Run("architecture compute capabilities are ordered", func(t *testing.T) {
		// Compute capabilities should increase with newer architectures
		// Note: We compare numerically since "10.0" > "9.0" numerically but not lexicographically
		archOrder := []Architecture{
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

		parseComputeCap := func(cap string) float64 {
			parts := strings.Split(cap, ".")
			if len(parts) >= 2 {
				major, _ := strconv.Atoi(parts[0])
				minor, _ := strconv.Atoi(parts[1])
				return float64(major) + float64(minor)/10.0
			}
			return 0
		}

		prevCap := 0.0
		for _, arch := range archOrder {
			cap := parseComputeCap(arch.ComputeCapability())
			if prevCap != 0 {
				assert.Greater(t, cap, prevCap,
					"Compute capability should increase from previous to %s", arch)
			}
			prevCap = cap
		}
	})
}

// =============================================================================
// Integration Test: Lookup Operations
// =============================================================================

func TestDatabase_LookupOperations(t *testing.T) {
	db := NewDatabase()

	t.Run("lookup all consumer RTX 40 series", func(t *testing.T) {
		rtx40 := []string{
			"2684", // RTX 4090
			"2704", // RTX 4080
			"2782", // RTX 4070 Ti
			"2786", // RTX 4070
			"2860", // RTX 4060
		}

		for _, deviceID := range rtx40 {
			model, found := db.Lookup(deviceID)
			require.True(t, found, "RTX 40 device %s not found", deviceID)
			assert.Equal(t, ArchAdaLovelace, model.Architecture)
			assert.False(t, model.IsDataCenter)
		}
	})

	t.Run("lookup all consumer RTX 30 series", func(t *testing.T) {
		rtx30 := []string{
			"2204", // RTX 3090 Ti
			"2205", // RTX 3090
			"2206", // RTX 3080
			"2488", // RTX 3070
			"2503", // RTX 3060
		}

		for _, deviceID := range rtx30 {
			model, found := db.Lookup(deviceID)
			require.True(t, found, "RTX 30 device %s not found", deviceID)
			assert.Equal(t, ArchAmpere, model.Architecture)
		}
	})

	t.Run("lookup by name case insensitive", func(t *testing.T) {
		testCases := []string{
			"GeForce RTX 4090",
			"GEFORCE RTX 4090",
			"geforce rtx 4090",
		}

		for _, name := range testCases {
			model, found := db.LookupByName(name)
			require.True(t, found, "GPU %s not found", name)
			assert.Equal(t, "2684", model.DeviceID)
		}
	})

	t.Run("lookup with 0x prefix variations", func(t *testing.T) {
		testCases := []string{
			"2684",
			"0x2684",
			"0X2684",
			"  2684  ",
		}

		for _, id := range testCases {
			model, found := db.Lookup(id)
			require.True(t, found, "Device ID %s not found", id)
			assert.Equal(t, "GeForce RTX 4090", model.Name)
		}
	})
}

// =============================================================================
// Integration Test: Thread Safety
// =============================================================================

func TestDatabase_ThreadSafety(t *testing.T) {
	db := NewDatabase()

	t.Run("concurrent reads", func(t *testing.T) {
		var wg sync.WaitGroup
		const numGoroutines = 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				switch id % 6 {
				case 0:
					db.Lookup("2684")
				case 1:
					db.LookupByName("GeForce RTX 4090")
				case 2:
					db.ListByArchitecture(ArchAdaLovelace)
				case 3:
					db.GetMinDriverVersion("2684")
				case 4:
					db.AllModels()
				case 5:
					db.Count()
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent lookups same device", func(t *testing.T) {
		var wg sync.WaitGroup
		const numGoroutines = 50

		results := make([]*GPUModel, numGoroutines)
		found := make([]bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx], found[idx] = db.Lookup("2684")
			}(i)
		}

		wg.Wait()

		// All should succeed with same result
		for i := 0; i < numGoroutines; i++ {
			assert.True(t, found[i])
			assert.Equal(t, "GeForce RTX 4090", results[i].Name)
		}
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkDatabase_Lookup(b *testing.B) {
	db := NewDatabase()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Lookup("2684")
	}
}

func BenchmarkDatabase_LookupWithPrefix(b *testing.B) {
	db := NewDatabase()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Lookup("0x2684")
	}
}

func BenchmarkDatabase_LookupByName(b *testing.B) {
	db := NewDatabase()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.LookupByName("GeForce RTX 4090")
	}
}

func BenchmarkDatabase_ListByArchitecture(b *testing.B) {
	db := NewDatabase()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.ListByArchitecture(ArchAmpere)
	}
}

func BenchmarkDatabase_GetMinDriverVersion(b *testing.B) {
	db := NewDatabase()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = db.GetMinDriverVersion("2684")
	}
}

func BenchmarkDatabase_AllModels(b *testing.B) {
	db := NewDatabase()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.AllModels()
	}
}

func BenchmarkDatabase_Count(b *testing.B) {
	db := NewDatabase()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Count()
	}
}

func BenchmarkDatabase_ConcurrentLookup(b *testing.B) {
	db := NewDatabase()
	deviceIDs := []string{"2684", "2206", "1e04", "1b06", "2322"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			db.Lookup(deviceIDs[i%len(deviceIDs)])
			i++
		}
	})
}

func BenchmarkArchitecture_MinDriverVersion(b *testing.B) {
	arch := ArchAdaLovelace

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = arch.MinDriverVersion()
	}
}

func BenchmarkArchitecture_ComputeCapability(b *testing.B) {
	arch := ArchAdaLovelace

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = arch.ComputeCapability()
	}
}

func BenchmarkNormalizeDeviceID(b *testing.B) {
	inputs := []string{"2684", "0x2684", "  2684  ", "0X2684"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			normalizeDeviceID(input)
		}
	}
}
