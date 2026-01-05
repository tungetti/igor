package recovery

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewTTYUI Tests
// =============================================================================

func TestNewTTYUI(t *testing.T) {
	t.Run("creates with default options", func(t *testing.T) {
		ui := NewTTYUI()

		assert.NotNil(t, ui)
		assert.NotNil(t, ui.reader)
		assert.NotNil(t, ui.writer)
		assert.Equal(t, 80, ui.width)
	})

	t.Run("with custom reader", func(t *testing.T) {
		reader := strings.NewReader("test input")
		ui := NewTTYUI(WithTTYReader(reader))

		assert.NotNil(t, ui)
		assert.Equal(t, reader, ui.rawReader)
	})

	t.Run("with custom writer", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		assert.NotNil(t, ui)
		assert.Equal(t, &buf, ui.writer)
	})

	t.Run("with custom width", func(t *testing.T) {
		ui := NewTTYUI(WithTTYWidth(120))

		assert.Equal(t, 120, ui.width)
	})

	t.Run("ignores invalid width", func(t *testing.T) {
		ui := NewTTYUI(WithTTYWidth(0))
		assert.Equal(t, 80, ui.width)

		ui = NewTTYUI(WithTTYWidth(-10))
		assert.Equal(t, 80, ui.width)
	})

	t.Run("with all options", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("input")

		ui := NewTTYUI(
			WithTTYReader(reader),
			WithTTYWriter(&buf),
			WithTTYWidth(100),
		)

		assert.NotNil(t, ui)
		assert.Equal(t, reader, ui.rawReader)
		assert.Equal(t, &buf, ui.writer)
		assert.Equal(t, 100, ui.width)
	})
}

// =============================================================================
// Header Tests
// =============================================================================

func TestTTYUI_Header(t *testing.T) {
	t.Run("prints header with separators", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(80))

		ui.Header("Test Header")

		output := buf.String()
		lines := strings.Split(output, "\n")

		assert.GreaterOrEqual(t, len(lines), 3)
		// First line should be separator
		assert.True(t, strings.Contains(lines[0], "----"))
		// Second line should contain header text
		assert.True(t, strings.Contains(lines[1], "Test Header"))
		// Third line should be separator
		assert.True(t, strings.Contains(lines[2], "----"))
	})

	t.Run("centers header text", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(40))

		ui.Header("Hi")

		output := buf.String()
		lines := strings.Split(output, "\n")

		// Check that "Hi" has leading spaces
		headerLine := lines[1]
		assert.True(t, strings.HasPrefix(headerLine, " "))
	})
}

// =============================================================================
// Status Tests
// =============================================================================

func TestTTYUI_Status(t *testing.T) {
	t.Run("prints status with symbol", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		ui.Status("[TEST]", "Test message")

		output := buf.String()
		assert.Contains(t, output, "[TEST]")
		assert.Contains(t, output, "Test message")
	})

	t.Run("truncates long text", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(30))

		longText := strings.Repeat("a", 100)
		ui.Status("[OK]", longText)

		output := buf.String()
		assert.Contains(t, output, "...")
		// Should not exceed width
		lines := strings.Split(strings.TrimSpace(output), "\n")
		assert.LessOrEqual(t, len(lines[0]), 30)
	})
}

func TestTTYUI_Success(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.Success("Operation completed")

	output := buf.String()
	assert.Contains(t, output, SymbolSuccess)
	assert.Contains(t, output, "Operation completed")
}

func TestTTYUI_Error(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.Error("Something went wrong")

	output := buf.String()
	assert.Contains(t, output, SymbolFailed)
	assert.Contains(t, output, "Something went wrong")
}

func TestTTYUI_Warning(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.Warning("Be careful")

	output := buf.String()
	assert.Contains(t, output, SymbolWarning)
	assert.Contains(t, output, "Be careful")
}

func TestTTYUI_Info(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.Info("Some information")

	output := buf.String()
	assert.Contains(t, output, SymbolInfo)
	assert.Contains(t, output, "Some information")
}

func TestTTYUI_Pending(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.Pending("Working on it")

	output := buf.String()
	assert.Contains(t, output, SymbolPending)
	assert.Contains(t, output, "Working on it")
}

// =============================================================================
// Progress Tests
// =============================================================================

func TestTTYUI_Progress(t *testing.T) {
	t.Run("shows progress percentage", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		ui.Progress(50, 100, "Processing")

		output := buf.String()
		assert.Contains(t, output, "50%")
		assert.Contains(t, output, "Processing")
	})

	t.Run("shows progress bar", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		ui.Progress(50, 100, "Test")

		output := buf.String()
		// Should contain hash marks for progress
		assert.Contains(t, output, "#")
		assert.Contains(t, output, "-")
	})

	t.Run("handles 0 total", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		// Should not panic
		ui.Progress(5, 0, "Test")

		output := buf.String()
		assert.NotEmpty(t, output)
	})

	t.Run("handles 100%", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		ui.Progress(100, 100, "Done")

		output := buf.String()
		assert.Contains(t, output, "100%")
	})

	t.Run("uses carriage return for overwriting", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		ui.Progress(25, 100, "Step 1")

		output := buf.String()
		assert.True(t, strings.HasPrefix(output, "\r"))
	})

	t.Run("truncates long text", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(50))

		longText := strings.Repeat("x", 100)
		ui.Progress(50, 100, longText)

		output := buf.String()
		assert.Contains(t, output, "...")
	})
}

func TestTTYUI_ProgressComplete(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.ProgressComplete()

	output := buf.String()
	assert.Equal(t, "\n", output)
}

// =============================================================================
// Separator Tests
// =============================================================================

func TestTTYUI_Separator(t *testing.T) {
	t.Run("prints dashes", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(40))

		ui.Separator()

		output := buf.String()
		// Should be 40 dashes + newline
		assert.Equal(t, strings.Repeat("-", 40)+"\n", output)
	})

	t.Run("respects width", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(100))

		ui.Separator()

		output := strings.TrimSpace(buf.String())
		assert.Len(t, output, 100)
	})
}

// =============================================================================
// Confirm Tests
// =============================================================================

func TestTTYUI_Confirm(t *testing.T) {
	t.Run("returns true for 'y'", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("y\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		result := ui.Confirm("Continue?", false)

		assert.True(t, result)
	})

	t.Run("returns true for 'yes'", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("yes\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		result := ui.Confirm("Continue?", false)

		assert.True(t, result)
	})

	t.Run("returns true for 'Y'", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("Y\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		result := ui.Confirm("Continue?", false)

		assert.True(t, result)
	})

	t.Run("returns false for 'n'", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("n\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		result := ui.Confirm("Continue?", true)

		assert.False(t, result)
	})

	t.Run("returns false for 'no'", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("no\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		result := ui.Confirm("Continue?", true)

		assert.False(t, result)
	})

	t.Run("returns default on empty input - defaultYes", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		result := ui.Confirm("Continue?", true)

		assert.True(t, result)
	})

	t.Run("returns default on empty input - defaultNo", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		result := ui.Confirm("Continue?", false)

		assert.False(t, result)
	})

	t.Run("returns default on EOF", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("") // EOF immediately
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		result := ui.Confirm("Continue?", true)

		assert.True(t, result)
	})

	t.Run("shows correct hint for defaultYes", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("y\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		ui.Confirm("Continue?", true)

		output := buf.String()
		assert.Contains(t, output, "[Y/n]")
	})

	t.Run("shows correct hint for defaultNo", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("y\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		ui.Confirm("Continue?", false)

		output := buf.String()
		assert.Contains(t, output, "[y/N]")
	})

	t.Run("retries on invalid input", func(t *testing.T) {
		var buf bytes.Buffer
		// First invalid, then valid
		reader := strings.NewReader("maybe\ny\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		result := ui.Confirm("Continue?", false)

		assert.True(t, result)
		output := buf.String()
		assert.Contains(t, output, "Please enter 'y' or 'n'")
	})
}

// =============================================================================
// ConfirmList Tests
// =============================================================================

func TestTTYUI_ConfirmList(t *testing.T) {
	t.Run("shows header and items", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("y\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		items := []string{"item1", "item2", "item3"}
		result := ui.ConfirmList("Test Header", items, "Confirm?", true)

		assert.True(t, result)
		output := buf.String()
		assert.Contains(t, output, "Test Header")
		assert.Contains(t, output, "* item1")
		assert.Contains(t, output, "* item2")
		assert.Contains(t, output, "* item3")
	})

	t.Run("handles empty header", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("n\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		items := []string{"item1"}
		result := ui.ConfirmList("", items, "Confirm?", true)

		assert.False(t, result)
		output := buf.String()
		assert.Contains(t, output, "* item1")
	})

	t.Run("handles empty items", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("y\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		result := ui.ConfirmList("Header", []string{}, "Confirm?", true)

		assert.True(t, result)
	})
}

// =============================================================================
// ShowPackages Tests
// =============================================================================

func TestTTYUI_ShowPackages(t *testing.T) {
	t.Run("shows package list", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		packages := []string{"nvidia-driver-550", "nvidia-settings", "libnvidia-gl-550"}
		ui.ShowPackages(packages)

		output := buf.String()
		assert.Contains(t, output, "Found 3 package(s)")
		assert.Contains(t, output, "nvidia-driver-550")
		assert.Contains(t, output, "nvidia-settings")
		assert.Contains(t, output, "libnvidia-gl-550")
	})

	t.Run("handles empty list", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		ui.ShowPackages([]string{})

		output := buf.String()
		assert.Contains(t, output, "No packages found")
	})

	t.Run("handles nil list", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		ui.ShowPackages(nil)

		output := buf.String()
		assert.Contains(t, output, "No packages found")
	})

	t.Run("uses columns for many packages", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(80))

		packages := make([]string, 10)
		for i := range packages {
			packages[i] = "package-" + string(rune('a'+i))
		}
		ui.ShowPackages(packages)

		output := buf.String()
		assert.Contains(t, output, "Found 10 package(s)")
	})
}

// =============================================================================
// ShowResult Tests
// =============================================================================

func TestTTYUI_ShowResult(t *testing.T) {
	t.Run("shows success result", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		ui.ShowResult(true, "Operation completed", nil)

		output := buf.String()
		assert.Contains(t, output, SymbolSuccess)
		assert.Contains(t, output, "Operation completed")
	})

	t.Run("shows failure result", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		ui.ShowResult(false, "Operation failed", nil)

		output := buf.String()
		assert.Contains(t, output, SymbolFailed)
		assert.Contains(t, output, "Operation failed")
	})

	t.Run("shows details", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		details := []string{"Detail 1", "Detail 2"}
		ui.ShowResult(true, "Done", details)

		output := buf.String()
		assert.Contains(t, output, "Detail 1")
		assert.Contains(t, output, "Detail 2")
	})

	t.Run("has separators", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(40))

		ui.ShowResult(true, "Done", nil)

		output := buf.String()
		// Should have separators (dashes)
		assert.GreaterOrEqual(t, strings.Count(output, strings.Repeat("-", 40)), 2)
	})
}

// =============================================================================
// ShowError Tests
// =============================================================================

func TestTTYUI_ShowError(t *testing.T) {
	t.Run("shows error message", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		ui.ShowError("Something went wrong", nil)

		output := buf.String()
		assert.Contains(t, output, "ERROR:")
		assert.Contains(t, output, "Something went wrong")
	})

	t.Run("shows suggestions", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		suggestions := []string{"Try this", "Or try that"}
		ui.ShowError("Failed", suggestions)

		output := buf.String()
		assert.Contains(t, output, "Suggestions:")
		assert.Contains(t, output, "Try this")
		assert.Contains(t, output, "Or try that")
	})
}

// =============================================================================
// ShowWarning Tests
// =============================================================================

func TestTTYUI_ShowWarning(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.ShowWarning("Be careful here")

	output := buf.String()
	assert.Contains(t, output, "WARNING:")
	assert.Contains(t, output, "Be careful here")
}

// =============================================================================
// Step Display Tests
// =============================================================================

func TestTTYUI_ShowStep(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.ShowStep(2, 5, "Processing files")

	output := buf.String()
	assert.Contains(t, output, "[2/5]")
	assert.Contains(t, output, "Processing files")
}

func TestTTYUI_StepSuccess(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.StepSuccess(3, 5, "Files processed")

	output := buf.String()
	assert.Contains(t, output, "[3/5]")
	assert.Contains(t, output, SymbolSuccess)
	assert.Contains(t, output, "Files processed")
}

func TestTTYUI_StepFailed(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.StepFailed(3, 5, "Processing failed")

	output := buf.String()
	assert.Contains(t, output, "[3/5]")
	assert.Contains(t, output, SymbolFailed)
	assert.Contains(t, output, "Processing failed")
}

// =============================================================================
// Utility Method Tests
// =============================================================================

func TestTTYUI_Blank(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.Blank()

	assert.Equal(t, "\n", buf.String())
}

func TestTTYUI_Print(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.Print("Hello world")

	assert.Equal(t, "Hello world\n", buf.String())
}

func TestTTYUI_Printf(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))

	ui.Printf("Count: %d\n", 42)

	assert.Equal(t, "Count: 42\n", buf.String())
}

func TestTTYUI_ReadLine(t *testing.T) {
	t.Run("reads line with prompt", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("test input\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		input, err := ui.ReadLine("Enter: ")

		require.NoError(t, err)
		assert.Equal(t, "test input", input)
		assert.Contains(t, buf.String(), "Enter: ")
	})

	t.Run("trims whitespace", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("  trimmed  \n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		input, err := ui.ReadLine("")

		require.NoError(t, err)
		assert.Equal(t, "trimmed", input)
	})

	t.Run("handles EOF", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))

		_, err := ui.ReadLine("")

		assert.Error(t, err)
	})
}

func TestTTYUI_Width(t *testing.T) {
	ui := NewTTYUI(WithTTYWidth(120))

	assert.Equal(t, 120, ui.Width())
}

// =============================================================================
// Symbol Constants Tests
// =============================================================================

func TestSymbolConstants(t *testing.T) {
	// Verify symbols are ASCII-only (no Unicode)
	symbols := []string{SymbolSuccess, SymbolFailed, SymbolPending, SymbolWarning, SymbolInfo}

	for _, sym := range symbols {
		for _, r := range sym {
			assert.Less(t, r, rune(128), "Symbol should be ASCII: %s", sym)
		}
	}

	// Verify expected values
	assert.Equal(t, "[OK]", SymbolSuccess)
	assert.Equal(t, "[FAIL]", SymbolFailed)
	assert.Equal(t, "[..]", SymbolPending)
	assert.Equal(t, "[!]", SymbolWarning)
	assert.Equal(t, "[i]", SymbolInfo)
}
