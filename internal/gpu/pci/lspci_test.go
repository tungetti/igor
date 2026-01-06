package pci

import (
	"context"
	"testing"
)

func TestParseLspciLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected lspciEntry
	}{
		{
			name:  "GeForce RTX 3090 with domain",
			input: "0000:01:00.0 VGA compatible controller: NVIDIA Corporation GA102 [GeForce RTX 3090] (rev a1)",
			expected: lspciEntry{
				Address: "0000:01:00.0",
				Type:    "VGA compatible controller",
				Name:    "NVIDIA Corporation GA102 [GeForce RTX 3090] (rev a1)",
			},
		},
		{
			name:  "GeForce GTX 1650 Mobile with domain",
			input: "0000:01:00.0 3D controller: NVIDIA Corporation TU117M [GeForce GTX 1650 Mobile / Max-Q] (rev a1)",
			expected: lspciEntry{
				Address: "0000:01:00.0",
				Type:    "3D controller",
				Name:    "NVIDIA Corporation TU117M [GeForce GTX 1650 Mobile / Max-Q] (rev a1)",
			},
		},
		{
			name:  "Tesla V100 with domain",
			input: "0000:3b:00.0 3D controller: NVIDIA Corporation GV100GL [Tesla V100 PCIe 32GB] (rev a1)",
			expected: lspciEntry{
				Address: "0000:3b:00.0",
				Type:    "3D controller",
				Name:    "NVIDIA Corporation GV100GL [Tesla V100 PCIe 32GB] (rev a1)",
			},
		},
		{
			name:  "Short format without domain (fallback)",
			input: "01:00.0 VGA compatible controller: NVIDIA Corporation GA102 [GeForce RTX 3090] (rev a1)",
			expected: lspciEntry{
				Address: "01:00.0",
				Type:    "VGA compatible controller",
				Name:    "NVIDIA Corporation GA102 [GeForce RTX 3090] (rev a1)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLspciLine(tt.input)
			if result.Address != tt.expected.Address {
				t.Errorf("Address: got %q, want %q", result.Address, tt.expected.Address)
			}
			if result.Type != tt.expected.Type {
				t.Errorf("Type: got %q, want %q", result.Type, tt.expected.Type)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Name: got %q, want %q", result.Name, tt.expected.Name)
			}
		})
	}
}

func TestExtractGPUModelName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "GeForce RTX 3090 with brackets",
			input:    "NVIDIA Corporation GA102 [GeForce RTX 3090] (rev a1)",
			expected: "GeForce RTX 3090",
		},
		{
			name:     "GeForce GTX 1650 Mobile with brackets",
			input:    "NVIDIA Corporation TU117M [GeForce GTX 1650 Mobile / Max-Q] (rev a1)",
			expected: "GeForce GTX 1650 Mobile / Max-Q",
		},
		{
			name:     "Tesla V100 with brackets",
			input:    "NVIDIA Corporation GV100GL [Tesla V100 PCIe 32GB] (rev a1)",
			expected: "Tesla V100 PCIe 32GB",
		},
		{
			name:     "Quadro RTX 8000",
			input:    "NVIDIA Corporation TU102GL [Quadro RTX 8000] (rev a1)",
			expected: "Quadro RTX 8000",
		},
		{
			name:     "No brackets - fallback",
			input:    "NVIDIA Corporation Some GPU (rev a1)",
			expected: "Some GPU",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractGPUModelName(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNormalizeAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0000:01:00.0", "01:00.0"},
		{"0000:3b:00.0", "3b:00.0"},
		{"01:00.0", "01:00.0"},
		{"3b:00.0", "3b:00.0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeAddress(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestLspciResolverIntegration tests the lspci resolver on the actual system.
// This test is skipped if lspci is not available.
func TestLspciResolverIntegration(t *testing.T) {
	resolver := NewLspciResolver()
	ctx := context.Background()

	// Test GetNVIDIAGPUInfo
	info := resolver.GetNVIDIAGPUInfo(ctx)
	t.Logf("NVIDIA GPU Info: %s", info)

	// Test GetGPUNames
	names, err := resolver.GetGPUNames(ctx)
	if err != nil {
		t.Fatalf("GetGPUNames error: %v", err)
	}
	t.Logf("GPU Names map: %v", names)

	// If we got any names, verify they look valid
	for addr, name := range names {
		if addr == "" {
			t.Error("Got empty address")
		}
		if name == "" {
			t.Error("Got empty name for address:", addr)
		}
		t.Logf("  %s -> %s", addr, name)
	}
}
