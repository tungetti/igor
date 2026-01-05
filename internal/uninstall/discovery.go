// Package uninstall provides the uninstallation workflow framework for Igor.
// This file implements the package discovery logic for finding installed NVIDIA packages.
package uninstall

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/pkg"
)

// DiscoveredPackages contains the results of package discovery.
type DiscoveredPackages struct {
	// DriverPackages are the main NVIDIA driver packages
	DriverPackages []string

	// DriverVersion is the detected driver version (e.g., "550.78")
	DriverVersion string

	// CUDAPackages are CUDA toolkit related packages
	CUDAPackages []string

	// CUDAVersion is the detected CUDA version (e.g., "12.4")
	CUDAVersion string

	// LibraryPackages are NVIDIA library packages (cuDNN, etc.)
	LibraryPackages []string

	// UtilityPackages are utility packages (nvidia-settings, nvidia-smi, etc.)
	UtilityPackages []string

	// KernelModulePackages are kernel module packages (nvidia-dkms, etc.)
	KernelModulePackages []string

	// ConfigPackages are configuration packages
	ConfigPackages []string

	// AllPackages is the complete list of all NVIDIA packages found
	AllPackages []string

	// TotalCount is the total number of packages found
	TotalCount int

	// DiscoveryTime is when the discovery was performed
	DiscoveryTime time.Time
}

// IsEmpty returns true if no packages were discovered.
func (d *DiscoveredPackages) IsEmpty() bool {
	return d.TotalCount == 0
}

// HasDriver returns true if driver packages were found.
func (d *DiscoveredPackages) HasDriver() bool {
	return len(d.DriverPackages) > 0
}

// HasCUDA returns true if CUDA packages were found.
func (d *DiscoveredPackages) HasCUDA() bool {
	return len(d.CUDAPackages) > 0
}

// Discovery discovers installed NVIDIA packages.
type Discovery interface {
	// Discover finds all installed NVIDIA packages on the system.
	Discover(ctx context.Context) (*DiscoveredPackages, error)

	// DiscoverDriver finds only driver-related packages.
	DiscoverDriver(ctx context.Context) ([]string, string, error)

	// DiscoverCUDA finds only CUDA-related packages.
	DiscoverCUDA(ctx context.Context) ([]string, string, error)

	// IsNVIDIAInstalled checks if any NVIDIA packages are installed.
	IsNVIDIAInstalled(ctx context.Context) (bool, error)

	// GetDriverVersion returns the installed driver version, if any.
	GetDriverVersion(ctx context.Context) (string, error)
}

// PackageDiscovery implements the Discovery interface using a package manager.
type PackageDiscovery struct {
	pm     pkg.Manager
	distro *distro.Distribution
	exec   exec.Executor
}

// DiscoveryOption is a functional option for PackageDiscovery.
type DiscoveryOption func(*PackageDiscovery)

// NewPackageDiscovery creates a new package discovery instance.
func NewPackageDiscovery(pm pkg.Manager, opts ...DiscoveryOption) *PackageDiscovery {
	pd := &PackageDiscovery{
		pm: pm,
	}
	for _, opt := range opts {
		opt(pd)
	}
	return pd
}

// WithDiscoveryDistro sets the distribution for discovery.
func WithDiscoveryDistro(dist *distro.Distribution) DiscoveryOption {
	return func(pd *PackageDiscovery) {
		pd.distro = dist
	}
}

// WithDiscoveryExecutor sets the executor for additional commands.
func WithDiscoveryExecutor(executor exec.Executor) DiscoveryOption {
	return func(pd *PackageDiscovery) {
		pd.exec = executor
	}
}

// Discover finds all installed NVIDIA packages on the system.
func (pd *PackageDiscovery) Discover(ctx context.Context) (*DiscoveredPackages, error) {
	if pd.pm == nil {
		return nil, pkg.NewPackageError(pkg.ErrUnsupportedOperation.Code(), "package manager is nil")
	}

	// Get all installed packages
	installedPkgs, err := pd.pm.ListInstalled(ctx)
	if err != nil {
		return nil, err
	}

	// Extract package names
	pkgNames := make([]string, 0, len(installedPkgs))
	for _, p := range installedPkgs {
		pkgNames = append(pkgNames, p.Name)
	}

	// Filter to NVIDIA packages only
	nvidiaPackages := FilterNVIDIAPackages(pkgNames)

	// Get the family for categorization
	family := pd.getFamily()

	// Categorize the packages
	result := CategorizePackages(nvidiaPackages, family)
	result.DiscoveryTime = time.Now()

	return result, nil
}

// DiscoverDriver finds only driver-related packages.
func (pd *PackageDiscovery) DiscoverDriver(ctx context.Context) ([]string, string, error) {
	discovered, err := pd.Discover(ctx)
	if err != nil {
		return nil, "", err
	}
	return discovered.DriverPackages, discovered.DriverVersion, nil
}

// DiscoverCUDA finds only CUDA-related packages.
func (pd *PackageDiscovery) DiscoverCUDA(ctx context.Context) ([]string, string, error) {
	discovered, err := pd.Discover(ctx)
	if err != nil {
		return nil, "", err
	}
	return discovered.CUDAPackages, discovered.CUDAVersion, nil
}

// IsNVIDIAInstalled checks if any NVIDIA packages are installed.
func (pd *PackageDiscovery) IsNVIDIAInstalled(ctx context.Context) (bool, error) {
	discovered, err := pd.Discover(ctx)
	if err != nil {
		return false, err
	}
	return discovered.TotalCount > 0, nil
}

// GetDriverVersion returns the installed driver version, if any.
func (pd *PackageDiscovery) GetDriverVersion(ctx context.Context) (string, error) {
	discovered, err := pd.Discover(ctx)
	if err != nil {
		return "", err
	}
	return discovered.DriverVersion, nil
}

// getFamily returns the distribution family for package categorization.
func (pd *PackageDiscovery) getFamily() constants.DistroFamily {
	if pd.distro != nil {
		return pd.distro.Family
	}
	if pd.pm != nil {
		return pd.pm.Family()
	}
	return constants.FamilyUnknown
}

// Compile-time check that PackageDiscovery implements Discovery.
var _ Discovery = (*PackageDiscovery)(nil)

// =============================================================================
// Helper Functions
// =============================================================================

// nvidiaPackagePatterns defines regex patterns for identifying NVIDIA packages.
var nvidiaPackagePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^nvidia`),
	regexp.MustCompile(`(?i)^libnvidia`),
	regexp.MustCompile(`(?i)^cuda`),
	regexp.MustCompile(`(?i)^libcuda`),
	regexp.MustCompile(`(?i)^libcudnn`),
	regexp.MustCompile(`(?i)^libnccl`),
	regexp.MustCompile(`(?i)^cudnn`),
	regexp.MustCompile(`(?i)^nccl`),
	regexp.MustCompile(`(?i)^kmod-nvidia`),
	regexp.MustCompile(`(?i)^akmod-nvidia`),
	regexp.MustCompile(`(?i)^dkms-nvidia`),
	regexp.MustCompile(`(?i)^xorg-x11-drv-nvidia`),
	regexp.MustCompile(`(?i)^x11-video-nvidia`),
}

// FilterNVIDIAPackages filters a package list to only include NVIDIA-related packages.
func FilterNVIDIAPackages(packages []string) []string {
	result := make([]string, 0)
	for _, pkg := range packages {
		if isNVIDIAPackage(pkg) {
			result = append(result, pkg)
		}
	}
	// Sort for consistent ordering
	sort.Strings(result)
	return result
}

// isNVIDIAPackage checks if a package name matches NVIDIA patterns.
func isNVIDIAPackage(name string) bool {
	for _, pattern := range nvidiaPackagePatterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// CategorizePackages categorizes packages into driver, cuda, library, utility, kernel groups.
func CategorizePackages(packages []string, family constants.DistroFamily) *DiscoveredPackages {
	result := &DiscoveredPackages{
		DriverPackages:       make([]string, 0),
		CUDAPackages:         make([]string, 0),
		LibraryPackages:      make([]string, 0),
		UtilityPackages:      make([]string, 0),
		KernelModulePackages: make([]string, 0),
		ConfigPackages:       make([]string, 0),
		AllPackages:          make([]string, 0, len(packages)),
	}

	for _, pkg := range packages {
		result.AllPackages = append(result.AllPackages, pkg)

		category := categorizePackage(pkg, family)
		switch category {
		case categoryDriver:
			result.DriverPackages = append(result.DriverPackages, pkg)
		case categoryCUDA:
			result.CUDAPackages = append(result.CUDAPackages, pkg)
		case categoryLibrary:
			result.LibraryPackages = append(result.LibraryPackages, pkg)
		case categoryUtility:
			result.UtilityPackages = append(result.UtilityPackages, pkg)
		case categoryKernel:
			result.KernelModulePackages = append(result.KernelModulePackages, pkg)
		case categoryConfig:
			result.ConfigPackages = append(result.ConfigPackages, pkg)
		}
	}

	result.TotalCount = len(result.AllPackages)
	result.DriverVersion = GetDriverVersionFromPackages(result.DriverPackages)
	result.CUDAVersion = GetCUDAVersionFromPackages(result.CUDAPackages)

	return result
}

// packageCategory represents the category of a package.
type packageCategory int

const (
	categoryUnknown packageCategory = iota
	categoryDriver
	categoryCUDA
	categoryLibrary
	categoryUtility
	categoryKernel
	categoryConfig
)

// categorizePackage determines the category of a package based on its name and the distribution family.
func categorizePackage(name string, family constants.DistroFamily) packageCategory {
	lowerName := strings.ToLower(name)

	// Check kernel module packages first (they're specific)
	if isKernelPackage(lowerName, family) {
		return categoryKernel
	}

	// Check for CUDA packages
	if isCUDAPackage(lowerName, family) {
		return categoryCUDA
	}

	// Check for library packages
	if isLibraryPackage(lowerName, family) {
		return categoryLibrary
	}

	// Check for utility packages
	if isUtilityPackage(lowerName, family) {
		return categoryUtility
	}

	// Check for config packages
	if isConfigPackage(lowerName, family) {
		return categoryConfig
	}

	// Check for driver packages
	if isDriverPackage(lowerName, family) {
		return categoryDriver
	}

	return categoryUnknown
}

// isDriverPackage checks if a package is a driver package.
func isDriverPackage(name string, family constants.DistroFamily) bool {
	patterns := getDriverPatterns(family)
	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// isCUDAPackage checks if a package is a CUDA package.
func isCUDAPackage(name string, family constants.DistroFamily) bool {
	patterns := getCUDAPatterns(family)
	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// isLibraryPackage checks if a package is a library package.
func isLibraryPackage(name string, family constants.DistroFamily) bool {
	patterns := getLibraryPatterns(family)
	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// isUtilityPackage checks if a package is a utility package.
func isUtilityPackage(name string, family constants.DistroFamily) bool {
	patterns := getUtilityPatterns(family)
	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// isKernelPackage checks if a package is a kernel module package.
func isKernelPackage(name string, family constants.DistroFamily) bool {
	patterns := getKernelPatterns(family)
	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// isConfigPackage checks if a package is a configuration package.
func isConfigPackage(name string, _ constants.DistroFamily) bool {
	configPatterns := []*regexp.Regexp{
		regexp.MustCompile(`nvidia.*config`),
		regexp.MustCompile(`nvidia.*prime`),
	}
	for _, pattern := range configPatterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// =============================================================================
// Distribution-Specific Patterns
// =============================================================================

// getDriverPatterns returns driver package patterns for the given family.
func getDriverPatterns(family constants.DistroFamily) []*regexp.Regexp {
	switch family {
	case constants.FamilyDebian:
		return []*regexp.Regexp{
			regexp.MustCompile(`^nvidia-driver`),
			regexp.MustCompile(`^nvidia-\d+-driver`),
			regexp.MustCompile(`^libnvidia-`),
		}
	case constants.FamilyRHEL:
		return []*regexp.Regexp{
			regexp.MustCompile(`^nvidia-driver`),
			regexp.MustCompile(`^kmod-nvidia`),
			regexp.MustCompile(`^akmod-nvidia`),
			regexp.MustCompile(`^xorg-x11-drv-nvidia`),
		}
	case constants.FamilyArch:
		return []*regexp.Regexp{
			regexp.MustCompile(`^nvidia$`),
			regexp.MustCompile(`^nvidia-lts$`),
			regexp.MustCompile(`^nvidia-open$`),
			regexp.MustCompile(`^nvidia-open-dkms$`),
			regexp.MustCompile(`^lib32-nvidia-utils`),
		}
	case constants.FamilySUSE:
		return []*regexp.Regexp{
			regexp.MustCompile(`(?i)^nvidia-driver-g06`),
			regexp.MustCompile(`(?i)^nvidia-video-g06`),
			regexp.MustCompile(`(?i)^x11-video-nvidiag06`),
		}
	default:
		return []*regexp.Regexp{
			regexp.MustCompile(`^nvidia-driver`),
			regexp.MustCompile(`^nvidia.*driver`),
		}
	}
}

// getCUDAPatterns returns CUDA package patterns for the given family.
func getCUDAPatterns(family constants.DistroFamily) []*regexp.Regexp {
	switch family {
	case constants.FamilyDebian:
		return []*regexp.Regexp{
			regexp.MustCompile(`^cuda-`),
			regexp.MustCompile(`^nvidia-cuda-`),
			regexp.MustCompile(`^cuda-toolkit`),
			regexp.MustCompile(`^libcuda`),
		}
	case constants.FamilyRHEL:
		return []*regexp.Regexp{
			regexp.MustCompile(`^cuda-`),
			regexp.MustCompile(`^nvidia-cuda-`),
		}
	case constants.FamilyArch:
		return []*regexp.Regexp{
			regexp.MustCompile(`^cuda$`),
			regexp.MustCompile(`^cuda-tools$`),
		}
	case constants.FamilySUSE:
		return []*regexp.Regexp{
			regexp.MustCompile(`^cuda-`),
		}
	default:
		return []*regexp.Regexp{
			regexp.MustCompile(`^cuda`),
		}
	}
}

// getLibraryPatterns returns library package patterns for the given family.
func getLibraryPatterns(family constants.DistroFamily) []*regexp.Regexp {
	switch family {
	case constants.FamilyDebian:
		return []*regexp.Regexp{
			regexp.MustCompile(`^libcudnn`),
			regexp.MustCompile(`^libnccl`),
		}
	case constants.FamilyRHEL:
		return []*regexp.Regexp{
			regexp.MustCompile(`^cudnn`),
			regexp.MustCompile(`^nccl`),
			regexp.MustCompile(`^libcudnn`),
			regexp.MustCompile(`^libnccl`),
		}
	case constants.FamilyArch:
		return []*regexp.Regexp{
			regexp.MustCompile(`^cudnn$`),
			regexp.MustCompile(`^nccl$`),
		}
	case constants.FamilySUSE:
		return []*regexp.Regexp{
			regexp.MustCompile(`^libcudnn`),
		}
	default:
		return []*regexp.Regexp{
			regexp.MustCompile(`cudnn`),
			regexp.MustCompile(`nccl`),
		}
	}
}

// getUtilityPatterns returns utility package patterns for the given family.
func getUtilityPatterns(family constants.DistroFamily) []*regexp.Regexp {
	switch family {
	case constants.FamilyDebian:
		return []*regexp.Regexp{
			regexp.MustCompile(`^nvidia-settings$`),
			regexp.MustCompile(`^nvidia-utils-`),
			regexp.MustCompile(`^nvidia-smi$`),
		}
	case constants.FamilyRHEL:
		return []*regexp.Regexp{
			regexp.MustCompile(`^nvidia-settings$`),
			regexp.MustCompile(`^nvidia-xconfig$`),
		}
	case constants.FamilyArch:
		return []*regexp.Regexp{
			regexp.MustCompile(`^nvidia-settings$`),
			regexp.MustCompile(`^nvidia-utils$`),
		}
	case constants.FamilySUSE:
		return []*regexp.Regexp{
			regexp.MustCompile(`^nvidia-settings$`),
		}
	default:
		return []*regexp.Regexp{
			regexp.MustCompile(`nvidia-settings`),
			regexp.MustCompile(`nvidia-utils`),
		}
	}
}

// getKernelPatterns returns kernel module package patterns for the given family.
func getKernelPatterns(family constants.DistroFamily) []*regexp.Regexp {
	switch family {
	case constants.FamilyDebian:
		return []*regexp.Regexp{
			regexp.MustCompile(`^nvidia-dkms`),
			regexp.MustCompile(`^nvidia-kernel-`),
		}
	case constants.FamilyRHEL:
		return []*regexp.Regexp{
			regexp.MustCompile(`^nvidia-kmod$`),
			regexp.MustCompile(`^dkms-nvidia`),
			regexp.MustCompile(`^kmod-nvidia`),
			regexp.MustCompile(`^akmod-nvidia`),
		}
	case constants.FamilyArch:
		return []*regexp.Regexp{
			regexp.MustCompile(`^nvidia-dkms$`),
		}
	case constants.FamilySUSE:
		return []*regexp.Regexp{
			regexp.MustCompile(`(?i)^nvidia-gfxg06-kmp-`),
		}
	default:
		return []*regexp.Regexp{
			regexp.MustCompile(`nvidia-dkms`),
			regexp.MustCompile(`nvidia-kmod`),
			regexp.MustCompile(`nvidia-kernel`),
		}
	}
}

// =============================================================================
// Version Extraction
// =============================================================================

// driverVersionPatterns are regex patterns to extract driver versions from package names.
var driverVersionPatterns = []*regexp.Regexp{
	// nvidia-driver-550, nvidia-driver-550-server
	regexp.MustCompile(`nvidia-driver-(\d+)`),
	// nvidia-550-driver
	regexp.MustCompile(`nvidia-(\d+)-driver`),
	// nvidia-utils-550
	regexp.MustCompile(`nvidia-utils-(\d+)`),
	// libnvidia-gl-550
	regexp.MustCompile(`libnvidia-[a-z]+-(\d+)`),
	// xorg-x11-drv-nvidia-550
	regexp.MustCompile(`xorg-x11-drv-nvidia-?(\d+)`),
	// nvidia-driver-G06-kmp-550
	regexp.MustCompile(`nvidia-driver-G\d+-kmp-.*?(\d+)`),
}

// GetDriverVersionFromPackages extracts the driver version from package names.
// For example, "nvidia-driver-550" -> "550"
func GetDriverVersionFromPackages(packages []string) string {
	for _, pkg := range packages {
		for _, pattern := range driverVersionPatterns {
			matches := pattern.FindStringSubmatch(pkg)
			if len(matches) >= 2 {
				return matches[1]
			}
		}
	}
	return ""
}

// cudaVersionPatterns are regex patterns to extract CUDA versions from package names.
var cudaVersionPatterns = []*regexp.Regexp{
	// cuda-toolkit-12-4
	regexp.MustCompile(`cuda-toolkit-(\d+)-(\d+)`),
	// cuda-12-4
	regexp.MustCompile(`cuda-(\d+)-(\d+)`),
	// cuda-toolkit-12.4
	regexp.MustCompile(`cuda-toolkit-(\d+)\.(\d+)`),
	// cuda-12.4
	regexp.MustCompile(`^cuda-(\d+)\.(\d+)`),
	// cuda-runtime-12-4
	regexp.MustCompile(`cuda-runtime-(\d+)-(\d+)`),
	// cuda-libraries-12-4
	regexp.MustCompile(`cuda-libraries-(\d+)-(\d+)`),
}

// GetCUDAVersionFromPackages extracts the CUDA version from package names.
// For example, "cuda-toolkit-12-4" -> "12.4"
func GetCUDAVersionFromPackages(packages []string) string {
	for _, pkg := range packages {
		for _, pattern := range cudaVersionPatterns {
			matches := pattern.FindStringSubmatch(pkg)
			if len(matches) >= 3 {
				return matches[1] + "." + matches[2]
			}
		}
	}
	return ""
}
