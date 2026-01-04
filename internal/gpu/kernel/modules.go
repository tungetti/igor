// Package kernel provides kernel version and module detection for the Igor application.
// It detects kernel information, loaded kernel modules, and kernel headers installation
// status. This is critical for verifying DKMS compatibility and kernel module requirements
// when installing NVIDIA drivers.
package kernel

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
)

// ModuleInfo represents a kernel module loaded in the system.
// This information is parsed from /proc/modules.
type ModuleInfo struct {
	// Name is the module name (e.g., "nvidia", "nouveau").
	Name string

	// Size is the memory size of the module in bytes.
	Size int64

	// UsedBy is a list of modules that depend on this module.
	UsedBy []string

	// UsedCount is the reference count for this module.
	UsedCount int

	// State indicates the module state: "Live", "Loading", or "Unloading".
	State string
}

// ModuleState constants for kernel module states.
const (
	// ModuleStateLive indicates the module is loaded and active.
	ModuleStateLive = "Live"

	// ModuleStateLoading indicates the module is being loaded.
	ModuleStateLoading = "Loading"

	// ModuleStateUnloading indicates the module is being unloaded.
	ModuleStateUnloading = "Unloading"
)

// ParseModulesContent parses the content of /proc/modules and returns a list of ModuleInfo.
// The format of /proc/modules is:
//
//	name size used_count dependencies state address
//
// Example:
//
//	nvidia 55123456 10 - Live 0xffffffffc0a00000
//	nouveau 1234567 0 - Live 0xffffffffc0800000
//	nvidia_drm 65536 5 nvidia - Live 0xffffffffc0700000
func ParseModulesContent(content []byte) ([]ModuleInfo, error) {
	var modules []ModuleInfo

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		module, err := parseModuleLine(line)
		if err != nil {
			// Skip lines that can't be parsed
			continue
		}

		modules = append(modules, module)
	}

	if err := scanner.Err(); err != nil {
		return modules, err
	}

	return modules, nil
}

// parseModuleLine parses a single line from /proc/modules.
// Format: name size used_count dependencies state address
func parseModuleLine(line string) (ModuleInfo, error) {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return ModuleInfo{}, &parseError{line: line, reason: "not enough fields"}
	}

	module := ModuleInfo{
		Name: fields[0],
	}

	// Parse size
	size, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return ModuleInfo{}, &parseError{line: line, reason: "invalid size"}
	}
	module.Size = size

	// Parse used count
	usedCount, err := strconv.Atoi(fields[2])
	if err != nil {
		return ModuleInfo{}, &parseError{line: line, reason: "invalid used count"}
	}
	module.UsedCount = usedCount

	// Parse dependencies (field 3)
	// Dependencies are comma-separated, or "-" if none
	deps := fields[3]
	if deps != "-" && deps != "" {
		// Remove trailing comma if present
		deps = strings.TrimSuffix(deps, ",")
		if deps != "" {
			module.UsedBy = strings.Split(deps, ",")
		}
	}

	// Parse state (field 4)
	if len(fields) >= 5 {
		module.State = fields[4]
	}

	return module, nil
}

// parseError represents an error parsing a module line.
type parseError struct {
	line   string
	reason string
}

func (e *parseError) Error() string {
	return "failed to parse module line: " + e.reason + ": " + e.line
}

// FindModule searches for a module by name in a list of modules.
// Returns nil if not found.
func FindModule(modules []ModuleInfo, name string) *ModuleInfo {
	for i := range modules {
		if modules[i].Name == name {
			return &modules[i]
		}
	}
	return nil
}

// IsModuleInList checks if a module with the given name exists in the list.
func IsModuleInList(modules []ModuleInfo, name string) bool {
	return FindModule(modules, name) != nil
}

// FilterModulesByState filters modules by their state.
func FilterModulesByState(modules []ModuleInfo, state string) []ModuleInfo {
	var filtered []ModuleInfo
	for _, m := range modules {
		if m.State == state {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// GetNVIDIAModules returns all NVIDIA-related modules from the list.
// This includes modules with names starting with "nvidia".
func GetNVIDIAModules(modules []ModuleInfo) []ModuleInfo {
	var nvidiaModules []ModuleInfo
	for _, m := range modules {
		if strings.HasPrefix(m.Name, "nvidia") {
			nvidiaModules = append(nvidiaModules, m)
		}
	}
	return nvidiaModules
}
