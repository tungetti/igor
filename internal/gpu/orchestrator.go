package gpu

import (
	"context"
	"sync"
	"time"

	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/gpu/nouveau"
	"github.com/tungetti/igor/internal/gpu/nvidia"
	"github.com/tungetti/igor/internal/gpu/pci"
	"github.com/tungetti/igor/internal/gpu/smi"
	"github.com/tungetti/igor/internal/gpu/validator"
)

// Orchestrator defines the interface for GPU detection orchestration.
// It coordinates all detection components to provide comprehensive system
// GPU information and installation readiness assessment.
type Orchestrator interface {
	// DetectAll runs all detection components and returns comprehensive GPU info.
	DetectAll(ctx context.Context) (*GPUInfo, error)

	// DetectGPUs detects GPU hardware only without driver or validation checks.
	DetectGPUs(ctx context.Context) ([]NVIDIAGPUInfo, error)

	// GetDriverStatus gets the current driver status.
	GetDriverStatus(ctx context.Context) (*DriverInfo, error)

	// ValidateSystem validates installation requirements.
	ValidateSystem(ctx context.Context) (*validator.ValidationReport, error)

	// IsReadyForInstall checks if the system is ready for driver installation.
	// Returns a boolean indicating readiness and a slice of reasons if not ready.
	IsReadyForInstall(ctx context.Context) (bool, []string, error)
}

// OrchestratorImpl is the production implementation of the Orchestrator interface.
type OrchestratorImpl struct {
	pciScanner      pci.Scanner
	gpuDatabase     nvidia.Database
	smiParser       smi.Parser
	nouveauDetector nouveau.Detector
	kernelDetector  kernel.Detector
	systemValidator validator.Validator
	timeout         time.Duration
	skipLspciEnrich bool // Skip lspci name enrichment (for testing)
}

// OrchestratorOption configures the orchestrator.
type OrchestratorOption func(*OrchestratorImpl)

// WithPCIScanner sets the PCI scanner for the orchestrator.
func WithPCIScanner(scanner pci.Scanner) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.pciScanner = scanner
	}
}

// WithGPUDatabase sets the GPU database for the orchestrator.
func WithGPUDatabase(db nvidia.Database) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.gpuDatabase = db
	}
}

// WithSMIParser sets the nvidia-smi parser for the orchestrator.
func WithSMIParser(parser smi.Parser) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.smiParser = parser
	}
}

// WithNouveauDetector sets the Nouveau detector for the orchestrator.
func WithNouveauDetector(detector nouveau.Detector) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.nouveauDetector = detector
	}
}

// WithKernelDetector sets the kernel detector for the orchestrator.
func WithKernelDetector(detector kernel.Detector) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.kernelDetector = detector
	}
}

// WithSystemValidator sets the system validator for the orchestrator.
func WithSystemValidator(v validator.Validator) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.systemValidator = v
	}
}

// WithTimeout sets the overall detection timeout.
func WithTimeout(timeout time.Duration) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.timeout = timeout
	}
}

// WithSkipLspciEnrich disables lspci-based GPU name enrichment (useful for testing).
func WithSkipLspciEnrich(skip bool) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.skipLspciEnrich = skip
	}
}

// DefaultTimeout is the default detection timeout.
const DefaultTimeout = 30 * time.Second

// NewOrchestrator creates a new GPU detection orchestrator with the given options.
func NewOrchestrator(opts ...OrchestratorOption) *OrchestratorImpl {
	o := &OrchestratorImpl{
		timeout: DefaultTimeout,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// DetectAll runs all detection components and returns comprehensive GPU info.
// It handles partial failures gracefully, continuing detection even if some
// components fail and collecting errors for reporting.
func (o *OrchestratorImpl) DetectAll(ctx context.Context) (*GPUInfo, error) {
	const op = "gpu.DetectAll"

	startTime := time.Now()

	// Apply timeout if configured
	if o.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.GPUDetection, "GPU detection cancelled", ctx.Err()).WithOp(op)
	default:
	}

	info := &GPUInfo{
		DetectionTime: startTime,
		Errors:        make([]error, 0),
	}

	// Use a WaitGroup and mutex for concurrent detection
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Helper to safely append errors
	appendError := func(err error) {
		if err != nil {
			mu.Lock()
			info.Errors = append(info.Errors, err)
			mu.Unlock()
		}
	}

	// 1. Detect PCI devices and NVIDIA GPUs
	wg.Add(1)
	go func() {
		defer wg.Done()
		gpus, pciDevices, err := o.detectGPUsInternal(ctx)
		mu.Lock()
		info.PCIDevices = pciDevices
		info.NVIDIAGPUs = gpus
		mu.Unlock()
		appendError(err)
	}()

	// 2. Get driver status
	wg.Add(1)
	go func() {
		defer wg.Done()
		driverInfo, err := o.GetDriverStatus(ctx)
		mu.Lock()
		info.InstalledDriver = driverInfo
		mu.Unlock()
		appendError(err)
	}()

	// 3. Detect Nouveau status
	wg.Add(1)
	go func() {
		defer wg.Done()
		if o.nouveauDetector == nil {
			return
		}
		status, err := o.nouveauDetector.Detect(ctx)
		mu.Lock()
		info.NouveauStatus = status
		mu.Unlock()
		appendError(err)
	}()

	// 4. Get kernel info
	wg.Add(1)
	go func() {
		defer wg.Done()
		if o.kernelDetector == nil {
			return
		}
		kernelInfo, err := o.kernelDetector.GetKernelInfo(ctx)
		mu.Lock()
		info.KernelInfo = kernelInfo
		mu.Unlock()
		appendError(err)
	}()

	// 5. Validate system
	wg.Add(1)
	go func() {
		defer wg.Done()
		if o.systemValidator == nil {
			return
		}
		report, err := o.systemValidator.Validate(ctx)
		mu.Lock()
		info.ValidationReport = report
		mu.Unlock()
		appendError(err)
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// Calculate duration
	info.Duration = time.Since(startTime)

	// Enhance NVIDIA GPU info with SMI data if available
	o.enrichWithSMIData(ctx, info)

	return info, nil
}

// detectGPUsInternal detects GPU hardware and enriches with database info.
func (o *OrchestratorImpl) detectGPUsInternal(ctx context.Context) ([]NVIDIAGPUInfo, []pci.PCIDevice, error) {
	const op = "gpu.detectGPUsInternal"

	if o.pciScanner == nil {
		return nil, nil, errors.New(errors.GPUDetection, "PCI scanner not configured").WithOp(op)
	}

	// Scan for NVIDIA GPUs
	nvidiaDevices, err := o.pciScanner.ScanNVIDIA(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(errors.GPUDetection, "failed to scan NVIDIA GPUs", err).WithOp(op)
	}

	// Enrich devices with GPU names from lspci (unless disabled for testing)
	// This provides more reliable names than database lookup
	if !o.skipLspciEnrich {
		lspciResolver := pci.NewLspciResolver()
		if err := lspciResolver.EnrichDevicesWithNames(ctx, nvidiaDevices); err != nil {
			// Non-fatal: continue without lspci names, will fall back to database
		}
	}

	// Build NVIDIAGPUInfo for each device
	gpuInfos := make([]NVIDIAGPUInfo, 0, len(nvidiaDevices))
	for _, device := range nvidiaDevices {
		gpuInfo := NVIDIAGPUInfo{
			PCIDevice: device,
		}

		// Lookup in database if available (as fallback for name)
		if o.gpuDatabase != nil {
			if model, found := o.gpuDatabase.Lookup(device.DeviceID); found {
				gpuInfo.Model = model
			}
		}

		gpuInfos = append(gpuInfos, gpuInfo)
	}

	return gpuInfos, nvidiaDevices, nil
}

// enrichWithSMIData adds nvidia-smi information to detected GPUs.
func (o *OrchestratorImpl) enrichWithSMIData(ctx context.Context, info *GPUInfo) {
	if o.smiParser == nil || len(info.NVIDIAGPUs) == 0 {
		return
	}

	smiInfo, err := o.smiParser.Parse(ctx)
	if err != nil || smiInfo == nil || !smiInfo.Available {
		// nvidia-smi not available, skip enrichment
		return
	}

	// Match SMI GPUs to PCI devices by index
	// Note: This assumes nvidia-smi GPU index matches detection order
	for i := range info.NVIDIAGPUs {
		if i < len(smiInfo.GPUs) {
			gpuCopy := smiInfo.GPUs[i]
			info.NVIDIAGPUs[i].SMIInfo = &gpuCopy
		}
	}
}

// DetectGPUs detects GPU hardware only without driver or validation checks.
func (o *OrchestratorImpl) DetectGPUs(ctx context.Context) ([]NVIDIAGPUInfo, error) {
	const op = "gpu.DetectGPUs"

	// Apply timeout if configured
	if o.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}

	gpus, _, err := o.detectGPUsInternal(ctx)
	if err != nil {
		return nil, errors.Wrap(errors.GPUDetection, "failed to detect GPUs", err).WithOp(op)
	}

	// Try to enrich with SMI data
	if o.smiParser != nil && len(gpus) > 0 {
		smiInfo, smiErr := o.smiParser.Parse(ctx)
		if smiErr == nil && smiInfo != nil && smiInfo.Available {
			for i := range gpus {
				if i < len(smiInfo.GPUs) {
					gpuCopy := smiInfo.GPUs[i]
					gpus[i].SMIInfo = &gpuCopy
				}
			}
		}
	}

	return gpus, nil
}

// GetDriverStatus gets the current driver status.
func (o *OrchestratorImpl) GetDriverStatus(ctx context.Context) (*DriverInfo, error) {
	const op = "gpu.GetDriverStatus"

	driverInfo := &DriverInfo{
		Installed: false,
		Type:      DriverTypeNone,
	}

	// Check for Nouveau first via PCI devices
	if o.pciScanner != nil {
		devices, err := o.pciScanner.ScanNVIDIA(ctx)
		if err == nil {
			for _, device := range devices {
				if device.IsUsingNouveau() {
					driverInfo.Installed = true
					driverInfo.Type = DriverTypeNouveau
					return driverInfo, nil
				}
				if device.IsUsingProprietaryDriver() {
					driverInfo.Installed = true
					driverInfo.Type = DriverTypeNVIDIA
					// Continue to get version from nvidia-smi
					break
				}
			}
		}
	}

	// Check for NVIDIA proprietary driver via nvidia-smi
	if o.smiParser != nil {
		if o.smiParser.IsAvailable(ctx) {
			driverInfo.Installed = true
			driverInfo.Type = DriverTypeNVIDIA

			version, err := o.smiParser.GetDriverVersion(ctx)
			if err == nil {
				driverInfo.Version = version
			}

			cudaVersion, err := o.smiParser.GetCUDAVersion(ctx)
			if err == nil {
				driverInfo.CUDAVersion = cudaVersion
			}

			return driverInfo, nil
		}
	}

	// Fallback: check Nouveau detector
	if o.nouveauDetector != nil {
		status, err := o.nouveauDetector.Detect(ctx)
		if err == nil && status != nil && status.Loaded {
			driverInfo.Installed = true
			driverInfo.Type = DriverTypeNouveau
			return driverInfo, nil
		}
	}

	return driverInfo, nil
}

// ValidateSystem validates installation requirements.
func (o *OrchestratorImpl) ValidateSystem(ctx context.Context) (*validator.ValidationReport, error) {
	const op = "gpu.ValidateSystem"

	if o.systemValidator == nil {
		return nil, errors.New(errors.GPUDetection, "system validator not configured").WithOp(op)
	}

	// Apply timeout if configured
	if o.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}

	return o.systemValidator.Validate(ctx)
}

// IsReadyForInstall checks if the system is ready for driver installation.
// Returns:
//   - ready: true if installation can proceed
//   - reasons: slice of reasons why installation cannot proceed (if not ready)
//   - error: any error that occurred during the check
func (o *OrchestratorImpl) IsReadyForInstall(ctx context.Context) (bool, []string, error) {
	const op = "gpu.IsReadyForInstall"

	// Apply timeout if configured
	if o.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}

	reasons := make([]string, 0)

	// 1. Check for NVIDIA GPUs
	gpus, err := o.DetectGPUs(ctx)
	if err != nil {
		reasons = append(reasons, "Failed to detect GPUs: "+err.Error())
		return false, reasons, nil
	}
	if len(gpus) == 0 {
		reasons = append(reasons, "No NVIDIA GPUs detected")
		return false, reasons, nil
	}

	// 2. Run system validation
	if o.systemValidator != nil {
		report, err := o.systemValidator.Validate(ctx)
		if err != nil {
			reasons = append(reasons, "System validation failed: "+err.Error())
			return false, reasons, nil
		}

		if report != nil && report.HasErrors() {
			for _, checkErr := range report.Errors {
				reason := checkErr.Name.String() + ": " + checkErr.Message
				if checkErr.Remediation != "" {
					reason += " (" + checkErr.Remediation + ")"
				}
				reasons = append(reasons, reason)
			}
		}
	}

	// 3. Check Nouveau status (warning, not blocking)
	if o.nouveauDetector != nil {
		status, err := o.nouveauDetector.Detect(ctx)
		if err == nil && status != nil && status.Loaded {
			// Nouveau is a warning, not a blocker, but we note it
			reasons = append(reasons, "Warning: Nouveau driver is currently loaded (will need to be disabled)")
		}
	}

	// Determine readiness
	// If we have GPUs and no blocking errors, we're ready
	// Nouveau being loaded is a warning, not a blocker
	hasBlockingIssues := false
	for _, reason := range reasons {
		// Check if it's not just a warning
		if len(reason) < 8 || reason[:8] != "Warning:" {
			hasBlockingIssues = true
			break
		}
	}

	if len(gpus) > 0 && !hasBlockingIssues {
		// Filter out warnings for the "ready" case but still return them
		return true, reasons, nil
	}

	return false, reasons, nil
}

// Ensure OrchestratorImpl implements Orchestrator interface.
var _ Orchestrator = (*OrchestratorImpl)(nil)
