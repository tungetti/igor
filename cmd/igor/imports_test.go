// cmd/igor/imports_test.go
package main

import (
	"testing"

	// Verify all core dependencies can be imported
	_ "github.com/charmbracelet/bubbles/list"
	_ "github.com/charmbracelet/bubbles/progress"
	_ "github.com/charmbracelet/bubbles/spinner"
	_ "github.com/charmbracelet/bubbletea"
	_ "github.com/charmbracelet/lipgloss"
	_ "github.com/charmbracelet/log"
	_ "github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/require"
	_ "gopkg.in/yaml.v3"
)

func TestImports(t *testing.T) {
	// This test verifies that all core dependencies can be imported.
	// If this test compiles, all imports are valid.
	t.Log("All core dependencies imported successfully")
}
