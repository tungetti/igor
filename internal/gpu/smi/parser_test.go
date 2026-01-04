package smi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/exec"
)

// Sample nvidia-smi outputs for testing.
const (
	// Successful header output with driver and CUDA version
	sampleHeaderOutput = `Sat Jan  4 10:00:00 2025       
+-----------------------------------------------------------------------------------------+
| NVIDIA-SMI 550.54.14              Driver Version: 550.54.14      CUDA Version: 12.4     |
|-----------------------------------------+------------------------+----------------------+
| GPU  Name                 Persistence-M | Bus-Id          Disp.A | Volatile Uncorr. ECC |
| Fan  Temp   Perf          Pwr:Usage/Cap |           Memory-Usage | GPU-Util  Compute M. |
|                                         |                        |               MIG M. |
|=========================================+========================+======================|
|   0  NVIDIA GeForce RTX 4090        Off |   00000000:01:00.0 Off |                  N/A |
|  0%   45C    P8             28W /  450W |     123MiB /  24564MiB |      0%      Default |
|                                         |                        |                  N/A |
+-----------------------------------------+------------------------+----------------------+
                                                                                         
+-----------------------------------------------------------------------------------------+
| Processes:                                                                              |
|  GPU   GI   CI        PID   Type   Process name                              GPU Memory |
|        ID   ID                                                               Usage      |
|=========================================================================================|
|  No running processes found                                                             |
+-----------------------------------------------------------------------------------------+`

	// Single GPU CSV output
	sampleSingleGPUCSV = `0, NVIDIA GeForce RTX 4090, GPU-12345678-1234-1234-1234-123456789012, 24564, 1234, 23330, 45, 120.50, 450.00, 35, 12, Default, Disabled`

	// Multi GPU CSV output
	sampleMultiGPUCSV = `0, NVIDIA GeForce RTX 4090, GPU-12345678-1234-1234-1234-123456789012, 24564, 1234, 23330, 45, 120.50, 450.00, 35, 12, Default, Disabled
1, NVIDIA GeForce RTX 3080, GPU-87654321-4321-4321-4321-210987654321, 10240, 512, 9728, 52, 85.00, 320.00, 78, 45, Default, Enabled`

	// Driver not loaded error
	sampleDriverNotLoaded = `NVIDIA-SMI has failed because it couldn't communicate with the NVIDIA driver. Make sure that the latest NVIDIA driver is installed and running.`

	// No devices found
	sampleNoDevices = `No devices were found`

	// Older driver format
	sampleOlderDriverHeader = `NVIDIA-SMI 470.182.03   Driver Version: 470.182.03   CUDA Version: 11.4`
)

// --- Parser Tests ---

func TestNewParser(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)

	assert.NotNil(t, parser)
	assert.Equal(t, mock, parser.executor)
}

func TestParser_Parse_SingleGPU(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	// Set up mock responses
	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(sampleHeaderOutput))

	info, err := parser.Parse(ctx)

	require.NoError(t, err)
	assert.True(t, info.Available)
	assert.Equal(t, "550.54.14", info.DriverVersion)
	assert.Equal(t, "12.4", info.CUDAVersion)
}

func TestParser_Parse_WithGPUDetails(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	// We need to handle two calls - first for header, second for CSV query
	// Since MockExecutor uses command name as key, we set the same response
	// In practice, the mock will return this for all nvidia-smi calls
	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(sampleHeaderOutput+"\n"+sampleSingleGPUCSV))

	info, err := parser.Parse(ctx)

	require.NoError(t, err)
	assert.True(t, info.Available)
	assert.Equal(t, "550.54.14", info.DriverVersion)
	assert.Equal(t, "12.4", info.CUDAVersion)
}

func TestParser_Parse_MultipleGPUs(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	// Set response that includes both header and GPU CSV
	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(sampleHeaderOutput+"\n"+sampleMultiGPUCSV))

	info, err := parser.Parse(ctx)

	require.NoError(t, err)
	assert.True(t, info.Available)
	assert.Equal(t, "550.54.14", info.DriverVersion)
	assert.Equal(t, "12.4", info.CUDAVersion)
}

func TestParser_Parse_DriverNotLoaded(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.FailureResult(1, sampleDriverNotLoaded))

	info, err := parser.Parse(ctx)

	require.Error(t, err)
	assert.False(t, info.Available)
	assert.Contains(t, err.Error(), "driver is not loaded")
	assert.True(t, errors.IsCode(err, errors.GPUDetection))
}

func TestParser_Parse_NvidiaSMINotFound(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	// Simulate command not found
	execErr := errors.New(errors.Execution, "command not found")
	mock.SetResponse(nvidiaSMICommand, exec.ErrorResult(execErr))

	info, err := parser.Parse(ctx)

	require.Error(t, err)
	assert.False(t, info.Available)
	assert.True(t, errors.IsCode(err, errors.Execution))
}

func TestParser_Parse_NoDevices(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.FailureResult(1, sampleNoDevices))

	info, err := parser.Parse(ctx)

	require.Error(t, err)
	assert.False(t, info.Available)
	assert.Contains(t, err.Error(), "no NVIDIA devices found")
}

func TestParser_Parse_OlderDriver(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(sampleOlderDriverHeader))

	info, err := parser.Parse(ctx)

	require.NoError(t, err)
	assert.True(t, info.Available)
	assert.Equal(t, "470.182.03", info.DriverVersion)
	assert.Equal(t, "11.4", info.CUDAVersion)
}

func TestParser_Parse_ContextCancelled(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The mock doesn't actually respect context, but we test the flow
	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(sampleHeaderOutput))

	info, err := parser.Parse(ctx)

	// With mock, this still succeeds since mock doesn't check context
	require.NoError(t, err)
	assert.True(t, info.Available)
}

// --- IsAvailable Tests ---

func TestParser_IsAvailable_True(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(sampleHeaderOutput))

	assert.True(t, parser.IsAvailable(ctx))
}

func TestParser_IsAvailable_False_NotFound(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	execErr := errors.New(errors.Execution, "command not found")
	mock.SetResponse(nvidiaSMICommand, exec.ErrorResult(execErr))

	assert.False(t, parser.IsAvailable(ctx))
}

func TestParser_IsAvailable_False_DriverNotLoaded(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.FailureResult(1, sampleDriverNotLoaded))

	assert.False(t, parser.IsAvailable(ctx))
}

// --- GetDriverVersion Tests ---

func TestParser_GetDriverVersion_Success(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(sampleHeaderOutput))

	version, err := parser.GetDriverVersion(ctx)

	require.NoError(t, err)
	assert.Equal(t, "550.54.14", version)
}

func TestParser_GetDriverVersion_OlderFormat(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(sampleOlderDriverHeader))

	version, err := parser.GetDriverVersion(ctx)

	require.NoError(t, err)
	assert.Equal(t, "470.182.03", version)
}

func TestParser_GetDriverVersion_NotFound(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	// Output without driver version
	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult("some random output"))

	_, err := parser.GetDriverVersion(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "driver version not found")
}

func TestParser_GetDriverVersion_ExecutionError(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	execErr := errors.New(errors.Execution, "command failed")
	mock.SetResponse(nvidiaSMICommand, exec.ErrorResult(execErr))

	_, err := parser.GetDriverVersion(ctx)

	require.Error(t, err)
}

// --- GetCUDAVersion Tests ---

func TestParser_GetCUDAVersion_Success(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(sampleHeaderOutput))

	version, err := parser.GetCUDAVersion(ctx)

	require.NoError(t, err)
	assert.Equal(t, "12.4", version)
}

func TestParser_GetCUDAVersion_OlderFormat(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(sampleOlderDriverHeader))

	version, err := parser.GetCUDAVersion(ctx)

	require.NoError(t, err)
	assert.Equal(t, "11.4", version)
}

func TestParser_GetCUDAVersion_NotFound(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	// Output without CUDA version
	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult("some random output"))

	_, err := parser.GetCUDAVersion(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CUDA version not found")
}

// --- GetGPUCount Tests ---

func TestParser_GetGPUCount_SingleGPU(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult("0"))

	count, err := parser.GetGPUCount(ctx)

	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestParser_GetGPUCount_MultipleGPUs(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult("0\n1\n2\n3"))

	count, err := parser.GetGPUCount(ctx)

	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

func TestParser_GetGPUCount_NoGPUs(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(""))

	count, err := parser.GetGPUCount(ctx)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestParser_GetGPUCount_ExecutionError(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.FailureResult(1, sampleDriverNotLoaded))

	_, err := parser.GetGPUCount(ctx)

	require.Error(t, err)
}

// --- CSV Parsing Tests ---

func TestParser_ParseCSVOutput_SingleLine(t *testing.T) {
	parser := NewParser(exec.NewMockExecutor())

	gpus, err := parser.parseCSVOutput(sampleSingleGPUCSV)

	require.NoError(t, err)
	require.Len(t, gpus, 1)

	gpu := gpus[0]
	assert.Equal(t, 0, gpu.Index)
	assert.Equal(t, "NVIDIA GeForce RTX 4090", gpu.Name)
	assert.Equal(t, "GPU-12345678-1234-1234-1234-123456789012", gpu.UUID)
	assert.Equal(t, "24564 MiB", gpu.MemoryTotal)
	assert.Equal(t, "1234 MiB", gpu.MemoryUsed)
	assert.Equal(t, "23330 MiB", gpu.MemoryFree)
	assert.Equal(t, int64(24564), gpu.MemoryTotalMiB)
	assert.Equal(t, int64(1234), gpu.MemoryUsedMiB)
	assert.Equal(t, int64(23330), gpu.MemoryFreeMiB)
	assert.Equal(t, 45, gpu.Temperature)
	assert.Equal(t, "120.50 W", gpu.PowerDraw)
	assert.Equal(t, "450.00 W", gpu.PowerLimit)
	assert.InDelta(t, 120.50, gpu.PowerDrawWatts, 0.01)
	assert.InDelta(t, 450.00, gpu.PowerLimitWatts, 0.01)
	assert.Equal(t, 35, gpu.UtilizationGPU)
	assert.Equal(t, 12, gpu.UtilizationMem)
	assert.Equal(t, "Default", gpu.ComputeMode)
	assert.False(t, gpu.PersistenceMode)
}

func TestParser_ParseCSVOutput_MultipleLines(t *testing.T) {
	parser := NewParser(exec.NewMockExecutor())

	gpus, err := parser.parseCSVOutput(sampleMultiGPUCSV)

	require.NoError(t, err)
	require.Len(t, gpus, 2)

	// First GPU
	assert.Equal(t, 0, gpus[0].Index)
	assert.Equal(t, "NVIDIA GeForce RTX 4090", gpus[0].Name)
	assert.False(t, gpus[0].PersistenceMode)

	// Second GPU
	assert.Equal(t, 1, gpus[1].Index)
	assert.Equal(t, "NVIDIA GeForce RTX 3080", gpus[1].Name)
	assert.True(t, gpus[1].PersistenceMode)
	assert.Equal(t, 78, gpus[1].UtilizationGPU)
	assert.Equal(t, 45, gpus[1].UtilizationMem)
}

func TestParser_ParseCSVOutput_EmptyInput(t *testing.T) {
	parser := NewParser(exec.NewMockExecutor())

	gpus, err := parser.parseCSVOutput("")

	require.NoError(t, err)
	assert.Empty(t, gpus)
}

func TestParser_ParseCSVOutput_WhitespaceOnly(t *testing.T) {
	parser := NewParser(exec.NewMockExecutor())

	gpus, err := parser.parseCSVOutput("   \n   \n   ")

	require.NoError(t, err)
	assert.Empty(t, gpus)
}

func TestParser_ParseCSVOutput_MalformedLine(t *testing.T) {
	parser := NewParser(exec.NewMockExecutor())

	// Line with too few fields - should be skipped
	gpus, err := parser.parseCSVOutput("0, GPU Name, UUID")

	require.NoError(t, err)
	assert.Empty(t, gpus) // Malformed line skipped
}

func TestParser_ParseCSVOutput_MixedValidAndInvalid(t *testing.T) {
	parser := NewParser(exec.NewMockExecutor())

	input := "0, GPU Name, UUID\n" + sampleSingleGPUCSV + "\ninvalid line"

	gpus, err := parser.parseCSVOutput(input)

	require.NoError(t, err)
	require.Len(t, gpus, 1) // Only the valid line parsed
}

func TestParser_ParseGPULine_InvalidIndex(t *testing.T) {
	parser := NewParser(exec.NewMockExecutor())

	_, err := parser.parseGPULine("not_a_number, NVIDIA GPU, UUID, 1000, 100, 900, 50, 100, 200, 10, 5, Default, Disabled")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid GPU index")
}

func TestParser_ParseGPULine_PersistenceModesVariations(t *testing.T) {
	parser := NewParser(exec.NewMockExecutor())

	tests := []struct {
		persistenceValue string
		expected         bool
	}{
		{"Enabled", true},
		{"enabled", true},
		{"ENABLED", true},
		{"On", true},
		{"on", true},
		{"1", true},
		{"Disabled", false},
		{"disabled", false},
		{"Off", false},
		{"off", false},
		{"0", false},
	}

	for _, tt := range tests {
		t.Run(tt.persistenceValue, func(t *testing.T) {
			line := "0, GPU, UUID, 1000, 100, 900, 50, 100.0, 200.0, 10, 5, Default, " + tt.persistenceValue
			gpu, err := parser.parseGPULine(line)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, gpu.PersistenceMode)
		})
	}
}

// --- CSV Split Tests ---

func TestSplitCSVLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple fields",
			input:    "a, b, c",
			expected: []string{"a", " b", " c"},
		},
		{
			name:     "with quotes",
			input:    `a, "b, c", d`,
			expected: []string{"a", " b, c", " d"},
		},
		{
			name:     "empty field",
			input:    "a,,c",
			expected: []string{"a", "", "c"},
		},
		{
			name:     "single field",
			input:    "value",
			expected: []string{"value"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitCSVLine(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- Types Tests ---

func TestSMIInfo_GPUCount(t *testing.T) {
	info := &SMIInfo{
		GPUs: []SMIGPUInfo{{}, {}, {}},
	}
	assert.Equal(t, 3, info.GPUCount())
}

func TestSMIInfo_HasGPUs(t *testing.T) {
	assert.False(t, (&SMIInfo{}).HasGPUs())
	assert.True(t, (&SMIInfo{GPUs: []SMIGPUInfo{{}}}).HasGPUs())
}

func TestSMIInfo_TotalMemory(t *testing.T) {
	info := &SMIInfo{
		GPUs: []SMIGPUInfo{
			{MemoryTotalMiB: 24564},
			{MemoryTotalMiB: 10240},
		},
	}
	assert.Equal(t, int64(34804), info.TotalMemory())
}

func TestSMIInfo_TotalMemory_NoGPUs(t *testing.T) {
	info := &SMIInfo{}
	assert.Equal(t, int64(0), info.TotalMemory())
}

func TestSMIInfo_String(t *testing.T) {
	t.Run("not available", func(t *testing.T) {
		info := &SMIInfo{Available: false}
		assert.Equal(t, "nvidia-smi: not available", info.String())
	})

	t.Run("available", func(t *testing.T) {
		info := &SMIInfo{
			Available:     true,
			DriverVersion: "550.54.14",
			CUDAVersion:   "12.4",
			GPUs:          []SMIGPUInfo{{}, {}},
		}
		s := info.String()
		assert.Contains(t, s, "550.54.14")
		assert.Contains(t, s, "12.4")
		assert.Contains(t, s, "2 GPU(s)")
	})
}

func TestSMIGPUInfo_MemoryUsagePercent(t *testing.T) {
	gpu := SMIGPUInfo{
		MemoryTotalMiB: 24564,
		MemoryUsedMiB:  12282,
	}
	assert.InDelta(t, 50.0, gpu.MemoryUsagePercent(), 0.01)
}

func TestSMIGPUInfo_MemoryUsagePercent_ZeroTotal(t *testing.T) {
	gpu := SMIGPUInfo{
		MemoryTotalMiB: 0,
		MemoryUsedMiB:  100,
	}
	assert.Equal(t, 0.0, gpu.MemoryUsagePercent())
}

func TestSMIGPUInfo_PowerUsagePercent(t *testing.T) {
	gpu := SMIGPUInfo{
		PowerDrawWatts:  225.0,
		PowerLimitWatts: 450.0,
	}
	assert.InDelta(t, 50.0, gpu.PowerUsagePercent(), 0.01)
}

func TestSMIGPUInfo_PowerUsagePercent_ZeroLimit(t *testing.T) {
	gpu := SMIGPUInfo{
		PowerDrawWatts:  100.0,
		PowerLimitWatts: 0,
	}
	assert.Equal(t, 0.0, gpu.PowerUsagePercent())
}

func TestSMIGPUInfo_IsIdle(t *testing.T) {
	tests := []struct {
		name     string
		gpuUtil  int
		memUtil  int
		expected bool
	}{
		{"idle", 0, 0, true},
		{"low utilization", 4, 4, true},
		{"high GPU util", 50, 2, false},
		{"high mem util", 2, 50, false},
		{"both high", 50, 50, false},
		{"at threshold", 5, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gpu := SMIGPUInfo{
				UtilizationGPU: tt.gpuUtil,
				UtilizationMem: tt.memUtil,
			}
			assert.Equal(t, tt.expected, gpu.IsIdle())
		})
	}
}

func TestSMIGPUInfo_String(t *testing.T) {
	gpu := SMIGPUInfo{
		Index:          0,
		Name:           "NVIDIA GeForce RTX 4090",
		MemoryTotal:    "24564 MiB",
		UtilizationGPU: 35,
		Temperature:    45,
	}
	s := gpu.String()
	assert.Contains(t, s, "GPU 0")
	assert.Contains(t, s, "RTX 4090")
	assert.Contains(t, s, "35%")
	assert.Contains(t, s, "45°C")
}

func TestComputeModeType_IsExclusive(t *testing.T) {
	assert.False(t, ComputeModeDefault.IsExclusive())
	assert.True(t, ComputeModeExclusiveThread.IsExclusive())
	assert.True(t, ComputeModeExclusiveProcess.IsExclusive())
	assert.False(t, ComputeModeProhibited.IsExclusive())
}

func TestComputeModeType_String(t *testing.T) {
	assert.Equal(t, "Default", ComputeModeDefault.String())
	assert.Equal(t, "Exclusive_Thread", ComputeModeExclusiveThread.String())
	assert.Equal(t, "Exclusive_Process", ComputeModeExclusiveProcess.String())
	assert.Equal(t, "Prohibited", ComputeModeProhibited.String())
}

// --- Interface Compliance Tests ---

func TestParserImplementsInterface(t *testing.T) {
	var _ Parser = (*ParserImpl)(nil)
}

// --- Edge Cases ---

func TestParser_Parse_EmptyStdout(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(""))

	info, err := parser.Parse(ctx)

	require.NoError(t, err)
	assert.True(t, info.Available)
	assert.Empty(t, info.DriverVersion)
	assert.Empty(t, info.CUDAVersion)
}

func TestParser_Parse_GenericExitError(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.FailureResult(255, "some unknown error"))

	info, err := parser.Parse(ctx)

	require.Error(t, err)
	assert.False(t, info.Available)
	assert.Contains(t, err.Error(), "255")
}

func TestParser_CommandNotFoundInStderr(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	// Create error result with "command not found" in output
	result := &exec.Result{
		ExitCode: -1,
		Stderr:   []byte("bash: nvidia-smi: command not found"),
		Error:    errors.New(errors.Execution, "exec failed"),
	}
	mock.SetResponse(nvidiaSMICommand, result)

	info, err := parser.Parse(ctx)

	require.Error(t, err)
	assert.False(t, info.Available)
	assert.Contains(t, err.Error(), "nvidia-smi not found")
}

// --- Verify Calls Tests ---

func TestParser_VerifiesCorrectCommandsCalled(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetDefaultResponse(exec.SuccessResult(sampleHeaderOutput))

	_, _ = parser.Parse(ctx)

	// Verify nvidia-smi was called
	assert.True(t, mock.WasCalled(nvidiaSMICommand))
	assert.GreaterOrEqual(t, mock.CallCount(), 1)
}

func TestParser_GetDriverVersion_VerifiesCommand(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(sampleHeaderOutput))

	_, _ = parser.GetDriverVersion(ctx)

	assert.True(t, mock.WasCalled(nvidiaSMICommand))
}

func TestParser_GetCUDAVersion_VerifiesCommand(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult(sampleHeaderOutput))

	_, _ = parser.GetCUDAVersion(ctx)

	assert.True(t, mock.WasCalled(nvidiaSMICommand))
}

func TestParser_GetGPUCount_VerifiesArgs(t *testing.T) {
	mock := exec.NewMockExecutor()
	parser := NewParser(mock)
	ctx := context.Background()

	mock.SetResponse(nvidiaSMICommand, exec.SuccessResult("0\n1"))

	_, _ = parser.GetGPUCount(ctx)

	calls := mock.Calls()
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0].Args, "--query-gpu=index")
	assert.Contains(t, calls[0].Args, "--format=csv,noheader,nounits")
}
