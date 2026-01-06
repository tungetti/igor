// Package pci provides PCI device scanning capabilities for GPU detection.
package pci

import (
	"bufio"
	"context"
	"os/exec"
	"regexp"
	"strings"
)

// LspciResolver resolves GPU names using the lspci command.
// This provides more reliable names than database lookup as lspci
// uses the system's pci.ids database which is regularly updated.
type LspciResolver struct {
	// commandRunner allows mocking exec.Command for testing
	commandRunner func(name string, args ...string) *exec.Cmd
}

// NewLspciResolver creates a new lspci resolver.
func NewLspciResolver() *LspciResolver {
	return &LspciResolver{
		commandRunner: exec.Command,
	}
}

// lspciEntry represents a parsed line from lspci output.
type lspciEntry struct {
	Address string // PCI address (e.g., "01:00.0")
	Type    string // Device type (e.g., "VGA compatible controller")
	Name    string // Device name (e.g., "NVIDIA Corporation GA102 [GeForce RTX 3090]")
}

// GetGPUNames returns a map of PCI addresses to GPU names for all NVIDIA devices.
// The address format in the map uses the full form (e.g., "0000:01:00.0") with domain
// to match sysfs addresses exactly.
func (r *LspciResolver) GetGPUNames(ctx context.Context) (map[string]string, error) {
	names := make(map[string]string)

	// Run lspci -D to get full domain:bus:device.function format
	// This matches the sysfs address format exactly (e.g., "0000:01:00.0")
	cmd := r.commandRunner("lspci", "-D")
	output, err := cmd.Output()
	if err != nil {
		// lspci not available, return empty map (not an error, just use fallback)
		return names, nil
	}

	// Parse each line
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(strings.ToLower(line), "nvidia") {
			continue
		}

		entry := parseLspciLine(line)
		if entry.Address != "" && entry.Name != "" {
			// Extract just the GPU model name (e.g., "GeForce RTX 3090")
			gpuName := extractGPUModelName(entry.Name)
			if gpuName != "" {
				names[entry.Address] = gpuName
			}
		}
	}

	return names, nil
}

// GetNVIDIAGPUInfo returns a simple description of the first NVIDIA GPU found.
// This mimics the original bash: lspci | grep -i nvidia | head -1
func (r *LspciResolver) GetNVIDIAGPUInfo(ctx context.Context) string {
	// Use -D for full domain format for consistency
	cmd := r.commandRunner("lspci", "-D")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), "nvidia") {
			return line
		}
	}
	return ""
}

// parseLspciLine parses a single line from lspci output.
// Format: "01:00.0 VGA compatible controller: NVIDIA Corporation GA102 [GeForce RTX 3090] (rev a1)"
func parseLspciLine(line string) lspciEntry {
	entry := lspciEntry{}

	// Split on first space to get address
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return entry
	}
	entry.Address = parts[0]

	// The rest contains type and name separated by ": "
	rest := parts[1]
	colonIdx := strings.Index(rest, ": ")
	if colonIdx == -1 {
		return entry
	}

	entry.Type = rest[:colonIdx]
	entry.Name = rest[colonIdx+2:]

	return entry
}

// extractGPUModelName extracts the GPU model name from NVIDIA device description.
// Input: "NVIDIA Corporation GA102 [GeForce RTX 3090] (rev a1)"
// Output: "GeForce RTX 3090"
func extractGPUModelName(description string) string {
	// Try to extract name from brackets first (most reliable)
	// Pattern: [GeForce RTX 3090] or [Tesla V100] etc.
	bracketRe := regexp.MustCompile(`\[([^\]]+)\]`)
	matches := bracketRe.FindStringSubmatch(description)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Fallback: remove "NVIDIA Corporation" prefix and revision suffix
	name := description
	name = strings.TrimPrefix(name, "NVIDIA Corporation ")

	// Remove revision suffix like "(rev a1)"
	if idx := strings.Index(name, " (rev"); idx != -1 {
		name = name[:idx]
	}

	return strings.TrimSpace(name)
}

// NormalizeAddress converts a full PCI address to short form for matching.
// Input: "0000:01:00.0" -> Output: "01:00.0"
func NormalizeAddress(address string) string {
	// Remove domain prefix if present
	parts := strings.Split(address, ":")
	if len(parts) == 3 {
		// Full format: "0000:01:00.0" -> "01:00.0"
		return parts[1] + ":" + parts[2]
	}
	return address
}

// EnrichDevicesWithNames populates the Name field of PCIDevices using lspci.
func (r *LspciResolver) EnrichDevicesWithNames(ctx context.Context, devices []PCIDevice) error {
	names, err := r.GetGPUNames(ctx)
	if err != nil {
		return err
	}

	for i := range devices {
		// Try exact match first (both should be full format like "0000:01:00.0")
		if name, ok := names[devices[i].Address]; ok {
			devices[i].Name = name
			continue
		}
		// Fallback: try normalized address match for compatibility
		shortAddr := NormalizeAddress(devices[i].Address)
		for lspciAddr, name := range names {
			if NormalizeAddress(lspciAddr) == shortAddr {
				devices[i].Name = name
				break
			}
		}
	}

	return nil
}
