package smi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/exec"
)

// =============================================================================
// Real-World SMI Outputs
// =============================================================================

// Sample outputs from different nvidia-smi versions and configurations
const (
	// nvidia-smi 535.x output format
	smiOutput535 = `Sat Jan  4 14:30:00 2025       
+-----------------------------------------------------------------------------------------+
| NVIDIA-SMI 535.154.05             Driver Version: 535.154.05   CUDA Version: 12.2     |
|-----------------------------------------+------------------------+----------------------+
| GPU  Name                 Persistence-M | Bus-Id          Disp.A | Volatile Uncorr. ECC |
| Fan  Temp   Perf          Pwr:Usage/Cap |           Memory-Usage | GPU-Util  Compute M. |
|                                         |                        |               MIG M. |
|=========================================+========================+======================|
|   0  NVIDIA GeForce RTX 3080        Off |   00000000:01:00.0 Off |                  N/A |
| 30%   35C    P8             15W /  320W |     512MiB /  10240MiB |      0%      Default |
|                                         |                        |                  N/A |
+-----------------------------------------+------------------------+----------------------+
                                                                                         
+-----------------------------------------------------------------------------------------+
| Processes:                                                                              |
|  GPU   GI   CI        PID   Type   Process name                              GPU Memory |
|        ID   ID                                                               Usage      |
|=========================================================================================|
|  No running processes found                                                             |
+-----------------------------------------------------------------------------------------+`

	// nvidia-smi 550.x output format (latest)
	smiOutput550 = `Mon Jan  6 09:15:30 2025       
+-----------------------------------------------------------------------------------------+
| NVIDIA-SMI 550.54.14              Driver Version: 550.54.14      CUDA Version: 12.4     |
|-----------------------------------------+------------------------+----------------------+
| GPU  Name                 Persistence-M | Bus-Id          Disp.A | Volatile Uncorr. ECC |
| Fan  Temp   Perf          Pwr:Usage/Cap |           Memory-Usage | GPU-Util  Compute M. |
|                                         |                        |               MIG M. |
|=========================================+========================+======================|
|   0  NVIDIA GeForce RTX 4090        Off |   00000000:01:00.0 Off |                  Off |
|  0%   45C    P8             28W /  450W |    1234MiB /  24564MiB |      0%      Default |
|                                         |                        |                  N/A |
+-----------------------------------------+------------------------+----------------------+`

	// nvidia-smi with open kernel module (newer drivers)
	smiOutputOpenKernel = `Driver Version: 555.42.02   CUDA Version: 12.5
(Open kernel module loaded)`

	// Multi-GPU output
	smiOutputMultiGPU = `Sat Jan  4 10:00:00 2025       
+-----------------------------------------------------------------------------------------+
| NVIDIA-SMI 550.54.14              Driver Version: 550.54.14      CUDA Version: 12.4     |
|-----------------------------------------+------------------------+----------------------+
| GPU  Name                 Persistence-M | Bus-Id          Disp.A | Volatile Uncorr. ECC |
| Fan  Temp   Perf          Pwr:Usage/Cap |           Memory-Usage | GPU-Util  Compute M. |
|=========================================+========================+======================|
|   0  NVIDIA GeForce RTX 4090        Off |   00000000:01:00.0 Off |                  N/A |
|  0%   45C    P8             28W /  450W |    1234MiB /  24564MiB |      0%      Default |
+-----------------------------------------+------------------------+----------------------+
|   1  NVIDIA GeForce RTX 3070        Off |   00000000:02:00.0 Off |                  N/A |
| 40%   55C    P2            120W / 220W |    4096MiB /   8192MiB |     65%      Default |
+-----------------------------------------+------------------------+----------------------+`

	// Data center GPU output (H100)
	smiOutputH100 = `Sat Jan  4 10:00:00 2025       
+-----------------------------------------------------------------------------------------+
| NVIDIA-SMI 535.154.05             Driver Version: 535.154.05   CUDA Version: 12.2     |
|-----------------------------------------+------------------------+----------------------+
| GPU  Name                 Persistence-M | Bus-Id          Disp.A | Volatile Uncorr. ECC |
| Fan  Temp   Perf          Pwr:Usage/Cap |           Memory-Usage | GPU-Util  Compute M. |
|=========================================+========================+======================|
|   0  NVIDIA H100 PCIe               On  |   00000000:3B:00.0 Off |                    0 |
| N/A   32C    P0             73W /  350W |   1024MiB /  81920MiB |      0%      Default |
+-----------------------------------------+------------------------+----------------------+`

	// Legacy driver format (470.x)
	smiOutput470 = `NVIDIA-SMI 470.182.03   Driver Version: 470.182.03   CUDA Version: 11.4
GPU 0: NVIDIA GeForce GTX 1080 (UUID: GPU-12345678-1234-1234-1234-123456789abc)
    Memory: 8119 MiB`

	// Very old driver format (390.x)
	smiOutput390 = `NVIDIA-SMI 390.157      Driver Version: 390.157
GPU 0: GeForce GTX 1050 Ti`
)

// =============================================================================
// Integration Test: Real-World SMI Outputs
// =============================================================================

func TestParser_RealWorldSMIOutputs(t *testing.T) {
	t.Run("nvidia-smi 535.x output", func(t *testing.T) {
		mock := exec.NewMockExecutor()
		parser := NewParser(mock)
		ctx := context.Background()

		mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(smiOutput535))

		info, err := parser.Parse(ctx)

		require.NoError(t, err)
		assert.True(t, info.Available)
		assert.Equal(t, "535.154.05", info.DriverVersion)
		assert.Equal(t, "12.2", info.CUDAVersion)
	})

	t.Run("nvidia-smi 550.x output (latest)", func(t *testing.T) {
		mock := exec.NewMockExecutor()
		parser := NewParser(mock)
		ctx := context.Background()

		mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(smiOutput550))

		info, err := parser.Parse(ctx)

		require.NoError(t, err)
		assert.True(t, info.Available)
		assert.Equal(t, "550.54.14", info.DriverVersion)
		assert.Equal(t, "12.4", info.CUDAVersion)
	})

	t.Run("multi-GPU configuration", func(t *testing.T) {
		mock := exec.NewMockExecutor()
		parser := NewParser(mock)
		ctx := context.Background()

		// Header for first call
		mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(smiOutputMultiGPU))

		info, err := parser.Parse(ctx)

		require.NoError(t, err)
		assert.True(t, info.Available)
		assert.Equal(t, "550.54.14", info.DriverVersion)
	})

	t.Run("data center GPU (H100)", func(t *testing.T) {
		mock := exec.NewMockExecutor()
		parser := NewParser(mock)
		ctx := context.Background()

		mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(smiOutputH100))

		info, err := parser.Parse(ctx)

		require.NoError(t, err)
		assert.True(t, info.Available)
		assert.Equal(t, "535.154.05", info.DriverVersion)
		assert.Equal(t, "12.2", info.CUDAVersion)
	})

	t.Run("legacy 470.x driver", func(t *testing.T) {
		mock := exec.NewMockExecutor()
		parser := NewParser(mock)
		ctx := context.Background()

		mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(smiOutput470))

		info, err := parser.Parse(ctx)

		require.NoError(t, err)
		assert.True(t, info.Available)
		assert.Equal(t, "470.182.03", info.DriverVersion)
		assert.Equal(t, "11.4", info.CUDAVersion)
	})

	t.Run("very old 390.x driver", func(t *testing.T) {
		mock := exec.NewMockExecutor()
		parser := NewParser(mock)
		ctx := context.Background()

		mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(smiOutput390))

		version, err := parser.GetDriverVersion(ctx)

		require.NoError(t, err)
		assert.Equal(t, "390.157", version)
	})
}

// =============================================================================
// Integration Test: Driver Version Extraction
// =============================================================================

func TestParser_DriverVersionExtraction(t *testing.T) {
	tests := []struct {
		name            string
		output          string
		expectedVersion string
		expectError     bool
	}{
		{
			name:            "standard format X.Y.Z",
			output:          "Driver Version: 550.54.14   CUDA Version: 12.4",
			expectedVersion: "550.54.14",
		},
		{
			name:            "two-part version X.Y",
			output:          "Driver Version: 535.54   CUDA Version: 12.2",
			expectedVersion: "535.54",
		},
		{
			name:            "full nvidia-smi header",
			output:          "| NVIDIA-SMI 550.54.14              Driver Version: 550.54.14      CUDA Version: 12.4     |",
			expectedVersion: "550.54.14",
		},
		{
			name:            "open kernel module format",
			output:          "Driver Version: 555.42.02   CUDA Version: 12.5\n(Open kernel module loaded)",
			expectedVersion: "555.42.02",
		},
		{
			name:            "legacy 390.x format",
			output:          "NVIDIA-SMI 390.157      Driver Version: 390.157",
			expectedVersion: "390.157",
		},
		{
			name:            "legacy 340.x format",
			output:          "NVIDIA-SMI 340.108     Driver Version: 340.108",
			expectedVersion: "340.108",
		},
		{
			name:        "no driver version",
			output:      "Some other output without version",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := exec.NewMockExecutor()
			parser := NewParser(mock)
			ctx := context.Background()

			mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(tc.output))

			version, err := parser.GetDriverVersion(ctx)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedVersion, version)
			}
		})
	}
}

// =============================================================================
// Integration Test: GPU Memory Parsing
// =============================================================================

func TestParser_GPUMemoryParsing(t *testing.T) {
	tests := []struct {
		name           string
		csvOutput      string
		expectedMemory int64
		gpuName        string
	}{
		{
			name:           "RTX 4090 24GB",
			csvOutput:      "0, NVIDIA GeForce RTX 4090, GPU-uuid, 24564, 1234, 23330, 45, 120.50, 450.00, 35, 12, Default, Disabled",
			expectedMemory: 24564,
			gpuName:        "NVIDIA GeForce RTX 4090",
		},
		{
			name:           "RTX 3090 24GB",
			csvOutput:      "0, NVIDIA GeForce RTX 3090, GPU-uuid, 24576, 2048, 22528, 50, 150.00, 350.00, 40, 15, Default, Disabled",
			expectedMemory: 24576,
			gpuName:        "NVIDIA GeForce RTX 3090",
		},
		{
			name:           "RTX 3080 10GB",
			csvOutput:      "0, NVIDIA GeForce RTX 3080, GPU-uuid, 10240, 512, 9728, 45, 100.00, 320.00, 30, 10, Default, Disabled",
			expectedMemory: 10240,
			gpuName:        "NVIDIA GeForce RTX 3080",
		},
		{
			name:           "H100 80GB",
			csvOutput:      "0, NVIDIA H100 PCIe, GPU-uuid, 81920, 1024, 80896, 32, 73.00, 350.00, 5, 2, Default, Enabled",
			expectedMemory: 81920,
			gpuName:        "NVIDIA H100 PCIe",
		},
		{
			name:           "A100 40GB",
			csvOutput:      "0, NVIDIA A100-PCIE-40GB, GPU-uuid, 40960, 512, 40448, 35, 50.00, 250.00, 10, 5, Default, Enabled",
			expectedMemory: 40960,
			gpuName:        "NVIDIA A100-PCIE-40GB",
		},
		{
			name:           "GTX 1650 4GB",
			csvOutput:      "0, NVIDIA GeForce GTX 1650, GPU-uuid, 4096, 256, 3840, 55, 40.00, 75.00, 20, 8, Default, Disabled",
			expectedMemory: 4096,
			gpuName:        "NVIDIA GeForce GTX 1650",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser(exec.NewMockExecutor())

			gpus, err := parser.parseCSVOutput(tc.csvOutput)

			require.NoError(t, err)
			require.Len(t, gpus, 1)
			assert.Equal(t, tc.expectedMemory, gpus[0].MemoryTotalMiB)
			assert.Equal(t, tc.gpuName, gpus[0].Name)
			assert.Equal(t, tc.expectedMemory, gpus[0].MemoryTotalMiB)
		})
	}
}

// =============================================================================
// Integration Test: Error Conditions
// =============================================================================

func TestParser_ErrorConditions(t *testing.T) {
	t.Run("driver not loaded", func(t *testing.T) {
		mock := exec.NewMockExecutor()
		parser := NewParser(mock)
		ctx := context.Background()

		errorOutput := `NVIDIA-SMI has failed because it couldn't communicate with the NVIDIA driver. Make sure that the latest NVIDIA driver is installed and running.`
		mock.SetResponse(nvidiaSMICommand, exec.FailureResult(1, errorOutput))

		info, err := parser.Parse(ctx)

		require.Error(t, err)
		assert.False(t, info.Available)
		assert.Contains(t, err.Error(), "driver is not loaded")
		assert.True(t, errors.IsCode(err, errors.GPUDetection))
	})

	t.Run("GPU not found", func(t *testing.T) {
		mock := exec.NewMockExecutor()
		parser := NewParser(mock)
		ctx := context.Background()

		errorOutput := `No devices were found`
		mock.SetResponse(nvidiaSMICommand, exec.FailureResult(6, errorOutput))

		info, err := parser.Parse(ctx)

		require.Error(t, err)
		assert.False(t, info.Available)
		assert.Contains(t, err.Error(), "no NVIDIA devices found")
	})

	t.Run("permission denied", func(t *testing.T) {
		mock := exec.NewMockExecutor()
		parser := NewParser(mock)
		ctx := context.Background()

		// Simulate permission error via execution failure
		execErr := errors.New(errors.Permission, "permission denied accessing GPU")
		mock.SetResponse(nvidiaSMICommand, exec.ErrorResult(execErr))

		info, err := parser.Parse(ctx)

		require.Error(t, err)
		assert.False(t, info.Available)
	})

	t.Run("nvidia-smi command not found", func(t *testing.T) {
		mock := exec.NewMockExecutor()
		parser := NewParser(mock)
		ctx := context.Background()

		// Simulate command not found
		result := &exec.Result{
			ExitCode: 127,
			Stderr:   []byte("bash: nvidia-smi: command not found"),
			Error:    errors.New(errors.Execution, "exec failed"),
		}
		mock.SetResponse(nvidiaSMICommand, result)

		info, err := parser.Parse(ctx)

		require.Error(t, err)
		assert.False(t, info.Available)
		assert.Contains(t, err.Error(), "nvidia-smi not found")
	})

	t.Run("GPU in recovery mode", func(t *testing.T) {
		mock := exec.NewMockExecutor()
		parser := NewParser(mock)
		ctx := context.Background()

		errorOutput := `Unable to determine the device handle for GPU 0000:01:00.0: GPU is lost.`
		mock.SetResponse(nvidiaSMICommand, exec.FailureResult(2, errorOutput))

		info, err := parser.Parse(ctx)

		require.Error(t, err)
		assert.False(t, info.Available)
	})

	t.Run("insufficient permissions for nvidia-smi", func(t *testing.T) {
		mock := exec.NewMockExecutor()
		parser := NewParser(mock)
		ctx := context.Background()

		errorOutput := `Insufficient Permissions. Run as root (or use sudo).`
		mock.SetResponse(nvidiaSMICommand, exec.FailureResult(1, errorOutput))

		// Should still try to work with the error
		info, err := parser.Parse(ctx)

		require.Error(t, err)
		assert.False(t, info.Available)
	})
}

// =============================================================================
// Integration Test: Multi-GPU CSV Parsing
// =============================================================================

func TestParser_MultiGPUCSVParsing(t *testing.T) {
	t.Run("4x A100 compute node", func(t *testing.T) {
		parser := NewParser(exec.NewMockExecutor())

		csvOutput := `0, NVIDIA A100-PCIE-80GB, GPU-a1b2c3d4, 81920, 1024, 80896, 32, 50.00, 300.00, 0, 0, Default, Enabled
1, NVIDIA A100-PCIE-80GB, GPU-e5f6g7h8, 81920, 2048, 79872, 35, 75.00, 300.00, 25, 10, Default, Enabled
2, NVIDIA A100-PCIE-80GB, GPU-i9j0k1l2, 81920, 4096, 77824, 38, 100.00, 300.00, 50, 20, Default, Enabled
3, NVIDIA A100-PCIE-80GB, GPU-m3n4o5p6, 81920, 8192, 73728, 40, 150.00, 300.00, 75, 30, Default, Enabled`

		gpus, err := parser.parseCSVOutput(csvOutput)

		require.NoError(t, err)
		require.Len(t, gpus, 4)

		for i, gpu := range gpus {
			assert.Equal(t, i, gpu.Index)
			assert.Equal(t, "NVIDIA A100-PCIE-80GB", gpu.Name)
			assert.Equal(t, int64(81920), gpu.MemoryTotalMiB)
			assert.True(t, gpu.PersistenceMode)
		}
	})

	t.Run("mixed consumer GPUs", func(t *testing.T) {
		parser := NewParser(exec.NewMockExecutor())

		csvOutput := `0, NVIDIA GeForce RTX 4090, GPU-a1, 24564, 1234, 23330, 45, 120.50, 450.00, 10, 5, Default, Disabled
1, NVIDIA GeForce RTX 3080, GPU-b2, 10240, 512, 9728, 55, 100.00, 320.00, 20, 10, Default, Disabled`

		gpus, err := parser.parseCSVOutput(csvOutput)

		require.NoError(t, err)
		require.Len(t, gpus, 2)

		assert.Equal(t, "NVIDIA GeForce RTX 4090", gpus[0].Name)
		assert.Equal(t, int64(24564), gpus[0].MemoryTotalMiB)

		assert.Equal(t, "NVIDIA GeForce RTX 3080", gpus[1].Name)
		assert.Equal(t, int64(10240), gpus[1].MemoryTotalMiB)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkParser_ParseSMI(b *testing.B) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(smiOutput550))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(ctx)
	}
}

func BenchmarkParser_GetDriverVersion(b *testing.B) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(smiOutput550))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.GetDriverVersion(ctx)
	}
}

func BenchmarkParser_GetCUDAVersion(b *testing.B) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(smiOutput550))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.GetCUDAVersion(ctx)
	}
}

func BenchmarkParser_ParseCSVOutput(b *testing.B) {
	parser := NewParser(exec.NewMockExecutor())

	csvOutput := `0, NVIDIA GeForce RTX 4090, GPU-uuid, 24564, 1234, 23330, 45, 120.50, 450.00, 35, 12, Default, Disabled
1, NVIDIA GeForce RTX 3080, GPU-uuid, 10240, 512, 9728, 55, 100.00, 320.00, 20, 10, Default, Disabled`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.parseCSVOutput(csvOutput)
	}
}

func BenchmarkParser_IsAvailable(b *testing.B) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(smiOutput550))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.IsAvailable(ctx)
	}
}

func BenchmarkSplitCSVLine(b *testing.B) {
	line := `0, NVIDIA GeForce RTX 4090, GPU-12345678-1234-1234-1234-123456789012, 24564, 1234, 23330, 45, 120.50, 450.00, 35, 12, Default, Disabled`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = splitCSVLine(line)
	}
}
