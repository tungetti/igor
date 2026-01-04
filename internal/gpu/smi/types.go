// Package smi provides nvidia-smi output parsing capabilities for GPU detection.
// It parses nvidia-smi command output to extract driver information, CUDA version,
// and per-GPU details including memory usage, temperature, and utilization.
package smi

import "fmt"

// SMIInfo represents the complete output from nvidia-smi.
type SMIInfo struct {
	// DriverVersion is the installed NVIDIA driver version (e.g., "550.54.14")
	DriverVersion string

	// CUDAVersion is the supported CUDA version (e.g., "12.4")
	CUDAVersion string

	// GPUs contains information about each detected GPU
	GPUs []SMIGPUInfo

	// Available indicates whether nvidia-smi is working and driver is loaded
	Available bool
}

// GPUCount returns the number of detected GPUs.
func (s *SMIInfo) GPUCount() int {
	return len(s.GPUs)
}

// HasGPUs returns true if at least one GPU was detected.
func (s *SMIInfo) HasGPUs() bool {
	return len(s.GPUs) > 0
}

// TotalMemory returns the total memory across all GPUs in MiB.
// Returns 0 if memory information is not available.
func (s *SMIInfo) TotalMemory() int64 {
	var total int64
	for _, gpu := range s.GPUs {
		total += gpu.MemoryTotalMiB
	}
	return total
}

// String returns a human-readable summary of the SMI info.
func (s *SMIInfo) String() string {
	if !s.Available {
		return "nvidia-smi: not available"
	}
	return fmt.Sprintf("nvidia-smi: Driver %s, CUDA %s, %d GPU(s)",
		s.DriverVersion, s.CUDAVersion, len(s.GPUs))
}

// SMIGPUInfo represents per-GPU information from nvidia-smi.
type SMIGPUInfo struct {
	// Index is the GPU index (0-based)
	Index int

	// Name is the GPU model name (e.g., "NVIDIA GeForce RTX 4090")
	Name string

	// UUID is the unique identifier for the GPU
	UUID string

	// MemoryTotal is the total GPU memory as a string (e.g., "24564 MiB")
	MemoryTotal string

	// MemoryUsed is the used GPU memory as a string (e.g., "1234 MiB")
	MemoryUsed string

	// MemoryFree is the free GPU memory as a string (e.g., "23330 MiB")
	MemoryFree string

	// MemoryTotalMiB is the total GPU memory in MiB (parsed value)
	MemoryTotalMiB int64

	// MemoryUsedMiB is the used GPU memory in MiB (parsed value)
	MemoryUsedMiB int64

	// MemoryFreeMiB is the free GPU memory in MiB (parsed value)
	MemoryFreeMiB int64

	// Temperature is the GPU temperature in Celsius
	Temperature int

	// PowerDraw is the current power draw (e.g., "120.50 W")
	PowerDraw string

	// PowerLimit is the power limit (e.g., "450.00 W")
	PowerLimit string

	// PowerDrawWatts is the current power draw in Watts (parsed value)
	PowerDrawWatts float64

	// PowerLimitWatts is the power limit in Watts (parsed value)
	PowerLimitWatts float64

	// UtilizationGPU is the GPU utilization percentage (0-100)
	UtilizationGPU int

	// UtilizationMem is the memory controller utilization percentage (0-100)
	UtilizationMem int

	// ComputeMode is the compute mode (e.g., "Default", "Exclusive_Process")
	ComputeMode string

	// PersistenceMode indicates whether persistence mode is enabled
	PersistenceMode bool
}

// MemoryUsagePercent returns the memory usage as a percentage.
func (g *SMIGPUInfo) MemoryUsagePercent() float64 {
	if g.MemoryTotalMiB == 0 {
		return 0
	}
	return float64(g.MemoryUsedMiB) / float64(g.MemoryTotalMiB) * 100
}

// PowerUsagePercent returns the power usage as a percentage of the limit.
func (g *SMIGPUInfo) PowerUsagePercent() float64 {
	if g.PowerLimitWatts == 0 {
		return 0
	}
	return g.PowerDrawWatts / g.PowerLimitWatts * 100
}

// IsIdle returns true if the GPU has low utilization.
func (g *SMIGPUInfo) IsIdle() bool {
	return g.UtilizationGPU < 5 && g.UtilizationMem < 5
}

// String returns a human-readable summary of the GPU.
func (g *SMIGPUInfo) String() string {
	return fmt.Sprintf("GPU %d: %s (%s, %d%% util, %d°C)",
		g.Index, g.Name, g.MemoryTotal, g.UtilizationGPU, g.Temperature)
}

// ComputeModeType represents the compute mode of a GPU.
type ComputeModeType string

// Compute mode constants.
const (
	ComputeModeDefault          ComputeModeType = "Default"
	ComputeModeExclusiveThread  ComputeModeType = "Exclusive_Thread"
	ComputeModeExclusiveProcess ComputeModeType = "Exclusive_Process"
	ComputeModeProhibited       ComputeModeType = "Prohibited"
)

// IsExclusive returns true if the compute mode is exclusive.
func (c ComputeModeType) IsExclusive() bool {
	return c == ComputeModeExclusiveThread || c == ComputeModeExclusiveProcess
}

// String returns the string representation of the compute mode.
func (c ComputeModeType) String() string {
	return string(c)
}
