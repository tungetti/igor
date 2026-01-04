package smi

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/exec"
)

// Parser defines the interface for nvidia-smi parsing operations.
type Parser interface {
	// Parse runs nvidia-smi and parses the complete output.
	Parse(ctx context.Context) (*SMIInfo, error)

	// IsAvailable checks if nvidia-smi is available and working.
	IsAvailable(ctx context.Context) bool

	// GetDriverVersion returns just the driver version.
	GetDriverVersion(ctx context.Context) (string, error)

	// GetCUDAVersion returns just the CUDA version.
	GetCUDAVersion(ctx context.Context) (string, error)

	// GetGPUCount returns the number of GPUs detected.
	GetGPUCount(ctx context.Context) (int, error)
}

// ParserImpl is the production implementation of Parser.
// It uses the exec.Executor interface for command execution,
// allowing for easy mocking in tests.
type ParserImpl struct {
	executor exec.Executor
}

// NewParser creates a new nvidia-smi parser with the given executor.
func NewParser(executor exec.Executor) *ParserImpl {
	return &ParserImpl{
		executor: executor,
	}
}

// nvidia-smi command and arguments.
const (
	nvidiaSMICommand = "nvidia-smi"

	// Query arguments for CSV output with GPU info.
	queryGPUArgs = "--query-gpu=index,name,uuid,memory.total,memory.used,memory.free,temperature.gpu,power.draw,power.limit,utilization.gpu,utilization.memory,compute_mode,persistence_mode"
	formatArgs   = "--format=csv,noheader,nounits"
)

// Error messages from nvidia-smi.
const (
	errMsgDriverNotLoaded = "NVIDIA-SMI has failed"
	errMsgNotFound        = "command not found"
	errMsgNoDevice        = "No devices were found"
)

// Regular expressions for parsing nvidia-smi output.
var (
	// Matches: "NVIDIA-SMI 550.54.14    Driver Version: 550.54.14    CUDA Version: 12.4"
	headerRegex = regexp.MustCompile(`Driver Version:\s*(\d+\.\d+(?:\.\d+)?)\s+CUDA Version:\s*(\d+\.\d+)`)

	// Alternative pattern for just driver version
	driverVersionRegex = regexp.MustCompile(`Driver Version:\s*(\d+\.\d+(?:\.\d+)?)`)

	// Alternative pattern for just CUDA version
	cudaVersionRegex = regexp.MustCompile(`CUDA Version:\s*(\d+\.\d+)`)
)

// Parse runs nvidia-smi and parses the complete output.
func (p *ParserImpl) Parse(ctx context.Context) (*SMIInfo, error) {
	info := &SMIInfo{
		Available: false,
		GPUs:      []SMIGPUInfo{},
	}

	// First, get driver and CUDA version from the header
	headerResult := p.executor.Execute(ctx, nvidiaSMICommand)
	if err := p.checkExecutionError(headerResult); err != nil {
		return info, err
	}

	// Parse driver and CUDA version from header
	headerOutput := headerResult.StdoutString()
	if matches := headerRegex.FindStringSubmatch(headerOutput); len(matches) == 3 {
		info.DriverVersion = matches[1]
		info.CUDAVersion = matches[2]
	}

	// Now get detailed GPU info using CSV format
	gpuResult := p.executor.Execute(ctx, nvidiaSMICommand, queryGPUArgs, formatArgs)
	if err := p.checkExecutionError(gpuResult); err != nil {
		// If we got driver info but GPU query failed, still return partial info
		if info.DriverVersion != "" {
			info.Available = true
			return info, nil
		}
		return info, err
	}

	// Parse CSV output
	gpus, err := p.parseCSVOutput(gpuResult.StdoutString())
	if err != nil {
		// Return partial info if we have driver version
		if info.DriverVersion != "" {
			info.Available = true
			return info, nil
		}
		return info, err
	}

	info.GPUs = gpus
	info.Available = true
	return info, nil
}

// IsAvailable checks if nvidia-smi is available and working.
func (p *ParserImpl) IsAvailable(ctx context.Context) bool {
	result := p.executor.Execute(ctx, nvidiaSMICommand)
	return p.checkExecutionError(result) == nil
}

// GetDriverVersion returns just the driver version.
func (p *ParserImpl) GetDriverVersion(ctx context.Context) (string, error) {
	result := p.executor.Execute(ctx, nvidiaSMICommand)
	if err := p.checkExecutionError(result); err != nil {
		return "", err
	}

	output := result.StdoutString()
	if matches := driverVersionRegex.FindStringSubmatch(output); len(matches) == 2 {
		return matches[1], nil
	}

	return "", errors.New(errors.NotFound, "driver version not found in nvidia-smi output")
}

// GetCUDAVersion returns just the CUDA version.
func (p *ParserImpl) GetCUDAVersion(ctx context.Context) (string, error) {
	result := p.executor.Execute(ctx, nvidiaSMICommand)
	if err := p.checkExecutionError(result); err != nil {
		return "", err
	}

	output := result.StdoutString()
	if matches := cudaVersionRegex.FindStringSubmatch(output); len(matches) == 2 {
		return matches[1], nil
	}

	return "", errors.New(errors.NotFound, "CUDA version not found in nvidia-smi output")
}

// GetGPUCount returns the number of GPUs detected.
func (p *ParserImpl) GetGPUCount(ctx context.Context) (int, error) {
	result := p.executor.Execute(ctx, nvidiaSMICommand, "--query-gpu=index", formatArgs)
	if err := p.checkExecutionError(result); err != nil {
		return 0, err
	}

	lines := result.StdoutLines()
	return len(lines), nil
}

// checkExecutionError checks the result for common nvidia-smi errors.
func (p *ParserImpl) checkExecutionError(result *exec.Result) error {
	// Check for execution error (command not found)
	if result.Error != nil {
		combined := result.CombinedString()
		if strings.Contains(strings.ToLower(combined), errMsgNotFound) {
			return errors.New(errors.NotFound, "nvidia-smi not found: NVIDIA drivers may not be installed")
		}
		return errors.Wrap(errors.Execution, "failed to execute nvidia-smi", result.Error)
	}

	// Check for non-zero exit code
	if result.ExitCode != 0 {
		combined := result.CombinedString()

		// Check for driver not loaded error
		if strings.Contains(combined, errMsgDriverNotLoaded) {
			return errors.New(errors.GPUDetection, "NVIDIA driver is not loaded: nvidia-smi cannot communicate with the driver")
		}

		// Check for no devices
		if strings.Contains(combined, errMsgNoDevice) {
			return errors.New(errors.NotFound, "no NVIDIA devices found")
		}

		return errors.Newf(errors.Execution, "nvidia-smi exited with code %d: %s",
			result.ExitCode, strings.TrimSpace(combined))
	}

	return nil
}

// parseCSVOutput parses the CSV output from nvidia-smi --query-gpu.
// Expected format (each line): index, name, uuid, memory.total, memory.used, memory.free,
// temperature.gpu, power.draw, power.limit, utilization.gpu, utilization.memory, compute_mode, persistence_mode
func (p *ParserImpl) parseCSVOutput(output string) ([]SMIGPUInfo, error) {
	var gpus []SMIGPUInfo

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		gpu, err := p.parseGPULine(line)
		if err != nil {
			// Skip malformed lines but log would be nice
			continue
		}
		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// parseGPULine parses a single CSV line of GPU info.
func (p *ParserImpl) parseGPULine(line string) (SMIGPUInfo, error) {
	var gpu SMIGPUInfo

	// Split by comma, but be careful with GPU names that might contain commas
	fields := splitCSVLine(line)
	if len(fields) < 13 {
		return gpu, errors.Newf(errors.Validation, "expected 13 fields, got %d", len(fields))
	}

	// Parse index
	idx, err := strconv.Atoi(strings.TrimSpace(fields[0]))
	if err != nil {
		return gpu, errors.Wrap(errors.Validation, "invalid GPU index", err)
	}
	gpu.Index = idx

	// Name
	gpu.Name = strings.TrimSpace(fields[1])

	// UUID
	gpu.UUID = strings.TrimSpace(fields[2])

	// Memory (values are in MiB when using nounits)
	memTotal := strings.TrimSpace(fields[3])
	memUsed := strings.TrimSpace(fields[4])
	memFree := strings.TrimSpace(fields[5])

	gpu.MemoryTotal = memTotal + " MiB"
	gpu.MemoryUsed = memUsed + " MiB"
	gpu.MemoryFree = memFree + " MiB"

	// Parse memory values
	if val, err := strconv.ParseInt(memTotal, 10, 64); err == nil {
		gpu.MemoryTotalMiB = val
	}
	if val, err := strconv.ParseInt(memUsed, 10, 64); err == nil {
		gpu.MemoryUsedMiB = val
	}
	if val, err := strconv.ParseInt(memFree, 10, 64); err == nil {
		gpu.MemoryFreeMiB = val
	}

	// Temperature
	tempStr := strings.TrimSpace(fields[6])
	if temp, err := strconv.Atoi(tempStr); err == nil {
		gpu.Temperature = temp
	}

	// Power (values are in Watts when using nounits)
	powerDraw := strings.TrimSpace(fields[7])
	powerLimit := strings.TrimSpace(fields[8])

	gpu.PowerDraw = powerDraw + " W"
	gpu.PowerLimit = powerLimit + " W"

	// Parse power values
	if val, err := strconv.ParseFloat(powerDraw, 64); err == nil {
		gpu.PowerDrawWatts = val
	}
	if val, err := strconv.ParseFloat(powerLimit, 64); err == nil {
		gpu.PowerLimitWatts = val
	}

	// Utilization (values are percentages when using nounits)
	utilGPU := strings.TrimSpace(fields[9])
	utilMem := strings.TrimSpace(fields[10])

	if val, err := strconv.Atoi(utilGPU); err == nil {
		gpu.UtilizationGPU = val
	}
	if val, err := strconv.Atoi(utilMem); err == nil {
		gpu.UtilizationMem = val
	}

	// Compute mode
	gpu.ComputeMode = strings.TrimSpace(fields[11])

	// Persistence mode
	persistenceStr := strings.TrimSpace(fields[12])
	gpu.PersistenceMode = strings.EqualFold(persistenceStr, "enabled") ||
		strings.EqualFold(persistenceStr, "on") ||
		strings.EqualFold(persistenceStr, "1")

	return gpu, nil
}

// splitCSVLine splits a CSV line by comma, handling potential edge cases.
// This is a simple implementation that handles the nvidia-smi output format.
func splitCSVLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for _, r := range line {
		switch {
		case r == '"':
			inQuotes = !inQuotes
		case r == ',' && !inQuotes:
			fields = append(fields, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	// Add the last field
	fields = append(fields, current.String())

	return fields
}

// Ensure ParserImpl implements Parser interface.
var _ Parser = (*ParserImpl)(nil)
