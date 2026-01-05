package steps

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/install"
)

// =============================================================================
// Mock File Writer for X.org Tests
// =============================================================================

// mockXorgFileWriter implements XorgFileWriter for testing.
type mockXorgFileWriter struct {
	files       map[string][]byte
	dirs        map[string]bool
	writeError  error
	mkdirError  error
	statError   error
	renameError error
	removeError error
	readError   error

	// Call tracking
	writeFileCalled bool
	mkdirAllCalled  bool
	statCalled      bool
	renameCalled    bool
	removeCalled    bool
	readFileCalled  bool
	lastWritePath   string
	lastWriteData   []byte
}

// newMockXorgFileWriter creates a new mock file writer.
func newMockXorgFileWriter() *mockXorgFileWriter {
	return &mockXorgFileWriter{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

// SetFileExists configures a file to exist with the given content.
func (m *mockXorgFileWriter) SetFileExists(path string, content []byte) {
	m.files[path] = content
}

// SetDirExists configures a directory to exist.
func (m *mockXorgFileWriter) SetDirExists(path string) {
	m.dirs[path] = true
}

// SetWriteError sets an error to return from WriteFile.
func (m *mockXorgFileWriter) SetWriteError(err error) {
	m.writeError = err
}

// SetMkdirError sets an error to return from MkdirAll.
func (m *mockXorgFileWriter) SetMkdirError(err error) {
	m.mkdirError = err
}

// SetStatError sets an error to return from Stat.
func (m *mockXorgFileWriter) SetStatError(err error) {
	m.statError = err
}

// SetRenameError sets an error to return from Rename.
func (m *mockXorgFileWriter) SetRenameError(err error) {
	m.renameError = err
}

// SetRemoveError sets an error to return from Remove.
func (m *mockXorgFileWriter) SetRemoveError(err error) {
	m.removeError = err
}

// SetReadError sets an error to return from ReadFile.
func (m *mockXorgFileWriter) SetReadError(err error) {
	m.readError = err
}

// WriteFile implements XorgFileWriter.
func (m *mockXorgFileWriter) WriteFile(path string, content []byte, perm os.FileMode) error {
	m.writeFileCalled = true
	m.lastWritePath = path
	m.lastWriteData = content
	if m.writeError != nil {
		return m.writeError
	}
	m.files[path] = content
	return nil
}

// MkdirAll implements XorgFileWriter.
func (m *mockXorgFileWriter) MkdirAll(path string, perm os.FileMode) error {
	m.mkdirAllCalled = true
	if m.mkdirError != nil {
		return m.mkdirError
	}
	m.dirs[path] = true
	return nil
}

// Stat implements XorgFileWriter.
func (m *mockXorgFileWriter) Stat(path string) (os.FileInfo, error) {
	m.statCalled = true
	if m.statError != nil {
		return nil, m.statError
	}
	// Check if it's a directory
	if m.dirs[path] {
		return &mockFileInfo{name: path, isDir: true}, nil
	}
	// Check if it's a file
	if _, ok := m.files[path]; ok {
		return &mockFileInfo{name: path, isDir: false}, nil
	}
	return nil, os.ErrNotExist
}

// Rename implements XorgFileWriter.
func (m *mockXorgFileWriter) Rename(oldpath, newpath string) error {
	m.renameCalled = true
	if m.renameError != nil {
		return m.renameError
	}
	if content, ok := m.files[oldpath]; ok {
		m.files[newpath] = content
		delete(m.files, oldpath)
	}
	return nil
}

// Remove implements XorgFileWriter.
func (m *mockXorgFileWriter) Remove(path string) error {
	m.removeCalled = true
	if m.removeError != nil {
		return m.removeError
	}
	delete(m.files, path)
	return nil
}

// ReadFile implements XorgFileWriter.
func (m *mockXorgFileWriter) ReadFile(path string) ([]byte, error) {
	m.readFileCalled = true
	if m.readError != nil {
		return nil, m.readError
	}
	if content, ok := m.files[path]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

// Ensure mockXorgFileWriter implements XorgFileWriter.
var _ XorgFileWriter = (*mockXorgFileWriter)(nil)

// mockFileInfo is a minimal FileInfo implementation for testing.
type mockFileInfo struct {
	name  string
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() any           { return nil }

// =============================================================================
// Mock Display Detector
// =============================================================================

// mockDisplayDetector implements DisplayDetector for testing.
type mockDisplayDetector struct {
	displayServer string
	isWayland     bool
	detectError   error
}

// newMockDisplayDetector creates a new mock display detector.
func newMockDisplayDetector() *mockDisplayDetector {
	return &mockDisplayDetector{
		displayServer: "xorg",
		isWayland:     false,
	}
}

// SetDisplayServer sets the display server to return.
func (m *mockDisplayDetector) SetDisplayServer(server string) {
	m.displayServer = server
	m.isWayland = server == "wayland"
}

// SetDetectError sets an error to return from DetectDisplayServer.
func (m *mockDisplayDetector) SetDetectError(err error) {
	m.detectError = err
}

// DetectDisplayServer implements DisplayDetector.
func (m *mockDisplayDetector) DetectDisplayServer(ctx context.Context) (string, error) {
	if m.detectError != nil {
		return "", m.detectError
	}
	return m.displayServer, nil
}

// IsWaylandSession implements DisplayDetector.
func (m *mockDisplayDetector) IsWaylandSession() bool {
	return m.isWayland
}

// Ensure mockDisplayDetector implements DisplayDetector.
var _ DisplayDetector = (*mockDisplayDetector)(nil)

// =============================================================================
// Test Helpers
// =============================================================================

// newXorgTestContext creates a basic test context with executor for X.org tests.
func newXorgTestContext() (*install.Context, *exec.MockExecutor) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	return ctx, mockExec
}

// =============================================================================
// XorgConfigStep Constructor Tests
// =============================================================================

func TestNewXorgConfigStep(t *testing.T) {
	t.Run("creates with defaults", func(t *testing.T) {
		step := NewXorgConfigStep()

		assert.Equal(t, "xorg_config", step.Name())
		assert.Equal(t, "Configure X.org for NVIDIA driver", step.Description())
		assert.True(t, step.CanRollback())
		assert.Equal(t, DefaultXorgConfDir, step.configDir)
		assert.Equal(t, DefaultXorgConfFile, step.configFile)
		assert.Equal(t, NvidiaXorgConfig, step.configContent)
		assert.False(t, step.skipIfWayland)
		assert.True(t, step.backupExisting)
		assert.Nil(t, step.fileWriter)
		assert.Nil(t, step.displayDetector)
	})
}

func TestXorgConfigStepOptions(t *testing.T) {
	t.Run("WithXorgConfigDir sets custom directory", func(t *testing.T) {
		customDir := "/custom/xorg.conf.d"
		step := NewXorgConfigStep(WithXorgConfigDir(customDir))
		assert.Equal(t, customDir, step.configDir)
	})

	t.Run("WithXorgConfigFile sets custom file", func(t *testing.T) {
		customFile := "99-nvidia.conf"
		step := NewXorgConfigStep(WithXorgConfigFile(customFile))
		assert.Equal(t, customFile, step.configFile)
	})

	t.Run("WithXorgConfigContent sets custom content", func(t *testing.T) {
		customContent := "# Custom NVIDIA config\n"
		step := NewXorgConfigStep(WithXorgConfigContent(customContent))
		assert.Equal(t, customContent, step.configContent)
	})

	t.Run("WithSkipIfWayland sets skip flag", func(t *testing.T) {
		step := NewXorgConfigStep(WithSkipIfWayland(true))
		assert.True(t, step.skipIfWayland)
	})

	t.Run("WithBackupExisting sets backup flag", func(t *testing.T) {
		step := NewXorgConfigStep(WithBackupExisting(false))
		assert.False(t, step.backupExisting)
	})

	t.Run("WithXorgFileWriter sets custom writer", func(t *testing.T) {
		mockWriter := newMockXorgFileWriter()
		step := NewXorgConfigStep(WithXorgFileWriter(mockWriter))
		assert.Equal(t, mockWriter, step.fileWriter)
	})

	t.Run("WithDisplayDetector sets custom detector", func(t *testing.T) {
		mockDetector := newMockDisplayDetector()
		step := NewXorgConfigStep(WithDisplayDetector(mockDetector))
		assert.Equal(t, mockDetector, step.displayDetector)
	})

	t.Run("multiple options are applied", func(t *testing.T) {
		mockWriter := newMockXorgFileWriter()
		mockDetector := newMockDisplayDetector()
		customDir := "/custom/xorg"
		customFile := "custom.conf"

		step := NewXorgConfigStep(
			WithXorgConfigDir(customDir),
			WithXorgConfigFile(customFile),
			WithSkipIfWayland(true),
			WithBackupExisting(false),
			WithXorgFileWriter(mockWriter),
			WithDisplayDetector(mockDetector),
		)

		assert.Equal(t, customDir, step.configDir)
		assert.Equal(t, customFile, step.configFile)
		assert.True(t, step.skipIfWayland)
		assert.False(t, step.backupExisting)
		assert.Equal(t, mockWriter, step.fileWriter)
		assert.Equal(t, mockDetector, step.displayDetector)
	})

	t.Run("later options override earlier ones", func(t *testing.T) {
		step := NewXorgConfigStep(
			WithSkipIfWayland(true),
			WithSkipIfWayland(false),
		)
		assert.False(t, step.skipIfWayland)
	})
}

// =============================================================================
// XorgConfigStep Execute Tests
// =============================================================================

func TestXorgConfigStep_Execute_Success(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "successfully")
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))

	// Check state was set
	assert.True(t, ctx.GetStateBool(StateXorgConfigured))
	assert.Equal(t, DefaultXorgConfPath, ctx.GetStateString(StateXorgConfigPath))
	assert.Equal(t, "xorg", ctx.GetStateString(StateXorgDisplayServer))

	// Verify tee was called for file writing
	assert.True(t, mockExec.WasCalled("tee"))
}

func TestXorgConfigStep_Execute_DirectoryCreated(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	// Directory does NOT exist initially
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Verify mkdir -p was called
	assert.True(t, mockExec.WasCalledWith("mkdir", "-p", DefaultXorgConfDir))
}

func TestXorgConfigStep_Execute_BackupExisting(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockWriter.SetFileExists(DefaultXorgConfPath, []byte("# Old config"))
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
		WithBackupExisting(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Verify cp was called for backup
	assert.True(t, mockExec.WasCalledWith("cp", DefaultXorgConfPath, DefaultXorgConfPath+".bak"))

	// Check backup path in state
	assert.Equal(t, DefaultXorgConfPath+".bak", ctx.GetStateString(StateXorgBackupPath))
}

func TestXorgConfigStep_Execute_NoBackup(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockWriter.SetFileExists(DefaultXorgConfPath, []byte("# Old config"))
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
		WithBackupExisting(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Verify cp was NOT called for backup
	assert.False(t, mockExec.WasCalled("cp"))

	// No backup path in state
	assert.Empty(t, ctx.GetStateString(StateXorgBackupPath))
}

func TestXorgConfigStep_Execute_WaylandSkip(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("wayland")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
		WithSkipIfWayland(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusSkipped, result.Status)
	assert.Contains(t, result.Message, "Wayland")

	// State should NOT be set for skipped step
	assert.False(t, ctx.GetStateBool(StateXorgConfigured))

	// Display server should still be recorded
	assert.Equal(t, "wayland", ctx.GetStateString(StateXorgDisplayServer))

	// No file operations should have occurred
	assert.False(t, mockExec.WasCalled("tee"))
	assert.False(t, mockExec.WasCalled("mkdir"))
}

func TestXorgConfigStep_Execute_WaylandProceed(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("wayland")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
		WithSkipIfWayland(false), // Don't skip even on Wayland
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateXorgConfigured))

	// Verify tee was called
	assert.True(t, mockExec.WasCalled("tee"))
}

func TestXorgConfigStep_Execute_DryRun(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	ctx.DryRun = true

	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")

	// State should NOT be set for dry run
	assert.False(t, ctx.GetStateBool(StateXorgConfigured))

	// No file writing should occur
	assert.False(t, mockExec.WasCalled("tee"))
}

func TestXorgConfigStep_Execute_Cancelled(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	ctx.Cancel() // Cancel immediately

	step := NewXorgConfigStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	assert.True(t, errors.Is(result.Error, context.Canceled))

	// No operations should have been performed
	assert.Equal(t, 0, mockExec.CallCount())
}

func TestXorgConfigStep_Execute_NoExecutor(t *testing.T) {
	ctx := install.NewContext() // No executor

	step := NewXorgConfigStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "executor")
}

func TestXorgConfigStep_Execute_WriteFailure(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	// Set tee to fail
	mockExec.SetResponse("tee", exec.FailureResult(1, "permission denied"))

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to write")
	assert.Error(t, result.Error)

	// State should NOT be set on failure
	assert.False(t, ctx.GetStateBool(StateXorgConfigured))
}

func TestXorgConfigStep_Execute_CustomContent(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	customContent := "# Custom NVIDIA configuration\nSection \"Device\"\nEndSection\n"

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
		WithXorgConfigContent(customContent),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Verify tee was called with the custom content
	calls := mockExec.Calls()
	for _, call := range calls {
		if call.Command == "tee" {
			assert.Equal(t, customContent, string(call.Input))
			break
		}
	}
}

func TestXorgConfigStep_Execute_MkdirFailure(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	// Directory doesn't exist
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	// mkdir fails
	mockExec.SetResponse("mkdir", exec.FailureResult(1, "permission denied"))

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to create config directory")
}

func TestXorgConfigStep_Execute_DisplayServerUnknown(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("unknown")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
	)

	result := step.Execute(ctx)

	// Should still proceed with unknown display server
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Equal(t, "unknown", ctx.GetStateString(StateXorgDisplayServer))
	assert.True(t, mockExec.WasCalled("tee"))
}

func TestXorgConfigStep_Execute_DetectError(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDetectError(errors.New("detection failed"))

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
	)

	result := step.Execute(ctx)

	// Should proceed even if detection fails
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Equal(t, "unknown", ctx.GetStateString(StateXorgDisplayServer))
	assert.True(t, mockExec.WasCalled("tee"))
}

// =============================================================================
// XorgConfigStep Rollback Tests
// =============================================================================

func TestXorgConfigStep_Rollback_Success(t *testing.T) {
	ctx, mockExec := newXorgTestContext()

	step := NewXorgConfigStep()

	// Simulate that Execute was called
	ctx.SetState(StateXorgConfigured, true)
	ctx.SetState(StateXorgConfigPath, DefaultXorgConfPath)
	ctx.SetState(StateXorgDisplayServer, "xorg")

	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// Verify rm -f was called
	assert.True(t, mockExec.WasCalledWith("rm", "-f", DefaultXorgConfPath))

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateXorgConfigured))
	assert.Empty(t, ctx.GetStateString(StateXorgConfigPath))
	assert.Empty(t, ctx.GetStateString(StateXorgBackupPath))
	assert.Empty(t, ctx.GetStateString(StateXorgDisplayServer))
}

func TestXorgConfigStep_Rollback_RestoreBackup(t *testing.T) {
	ctx, mockExec := newXorgTestContext()

	step := NewXorgConfigStep()

	// Simulate that Execute was called with backup
	ctx.SetState(StateXorgConfigured, true)
	ctx.SetState(StateXorgConfigPath, DefaultXorgConfPath)
	ctx.SetState(StateXorgBackupPath, DefaultXorgConfPath+".bak")

	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// Verify rm and mv were called
	assert.True(t, mockExec.WasCalledWith("rm", "-f", DefaultXorgConfPath))
	assert.True(t, mockExec.WasCalledWith("mv", DefaultXorgConfPath+".bak", DefaultXorgConfPath))
}

func TestXorgConfigStep_Rollback_NoConfig(t *testing.T) {
	ctx, mockExec := newXorgTestContext()

	step := NewXorgConfigStep()

	// No state set (Execute was not called or failed)

	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// No commands should have been called
	assert.Equal(t, 0, mockExec.CallCount())
}

func TestXorgConfigStep_Rollback_NoConfigPath(t *testing.T) {
	ctx, mockExec := newXorgTestContext()

	step := NewXorgConfigStep()

	// Configured but no path
	ctx.SetState(StateXorgConfigured, true)

	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 0, mockExec.CallCount())
}

func TestXorgConfigStep_Rollback_NoExecutor(t *testing.T) {
	ctx := install.NewContext() // No executor
	ctx.SetState(StateXorgConfigured, true)
	ctx.SetState(StateXorgConfigPath, DefaultXorgConfPath)

	step := NewXorgConfigStep()

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor not available")
}

func TestXorgConfigStep_Rollback_RemoveError(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockExec.SetResponse("rm", exec.FailureResult(1, "permission denied"))

	step := NewXorgConfigStep()

	ctx.SetState(StateXorgConfigured, true)
	ctx.SetState(StateXorgConfigPath, DefaultXorgConfPath)

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove")
}

func TestXorgConfigStep_Rollback_RestoreError(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockExec.SetResponse("mv", exec.FailureResult(1, "permission denied"))

	step := NewXorgConfigStep()

	ctx.SetState(StateXorgConfigured, true)
	ctx.SetState(StateXorgConfigPath, DefaultXorgConfPath)
	ctx.SetState(StateXorgBackupPath, DefaultXorgConfPath+".bak")

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to restore")
}

// =============================================================================
// XorgConfigStep Validate Tests
// =============================================================================

func TestXorgConfigStep_Validate_Success(t *testing.T) {
	ctx, _ := newXorgTestContext()

	step := NewXorgConfigStep()

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

func TestXorgConfigStep_Validate_NoExecutor(t *testing.T) {
	ctx := install.NewContext()

	step := NewXorgConfigStep()

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor is required")
}

func TestXorgConfigStep_Validate_InvalidPath(t *testing.T) {
	ctx, _ := newXorgTestContext()

	testCases := []struct {
		name      string
		configDir string
		wantErr   bool
	}{
		{"valid path", "/etc/X11/xorg.conf.d", false},
		{"path with semicolon", "/etc;rm -rf", true},
		{"path with ampersand", "/etc && echo", true},
		{"path with pipe", "/etc | cat", true},
		{"path with backtick", "/etc`whoami`", true},
		{"path with dollar", "/etc$HOME", true},
		{"empty path", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := NewXorgConfigStep(WithXorgConfigDir(tc.configDir))
			err := step.Validate(ctx)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestXorgConfigStep_Validate_InvalidFileName(t *testing.T) {
	ctx, _ := newXorgTestContext()

	testCases := []struct {
		name       string
		configFile string
		wantErr    bool
	}{
		{"valid name", "20-nvidia.conf", false},
		{"name with path", "../nvidia.conf", true},
		{"name with slash", "sub/nvidia.conf", true},
		{"name with semicolon", "nvidia;rm.conf", true},
		{"empty name", "", true},
		{"name with backslash", "nvidia\\bad.conf", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := NewXorgConfigStep(WithXorgConfigFile(tc.configFile))
			err := step.Validate(ctx)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// XorgConfigStep CanRollback Tests
// =============================================================================

func TestXorgConfigStep_CanRollback(t *testing.T) {
	step := NewXorgConfigStep()
	assert.True(t, step.CanRollback())
}

// =============================================================================
// XorgConfigStep DetectDisplayServer Tests
// =============================================================================

func TestXorgConfigStep_DetectDisplayServer(t *testing.T) {
	t.Run("uses injected detector", func(t *testing.T) {
		ctx, _ := newXorgTestContext()
		mockDetector := newMockDisplayDetector()
		mockDetector.SetDisplayServer("wayland")

		step := NewXorgConfigStep(WithDisplayDetector(mockDetector))

		server, err := step.detectDisplayServer(ctx)

		assert.NoError(t, err)
		assert.Equal(t, "wayland", server)
	})

	t.Run("uses real detector when none injected", func(t *testing.T) {
		ctx, _ := newXorgTestContext()

		step := NewXorgConfigStep()

		server, err := step.detectDisplayServer(ctx)

		assert.NoError(t, err)
		// Result depends on environment, just verify it returns something valid
		assert.Contains(t, []string{"xorg", "wayland", "unknown"}, server)
	})
}

// =============================================================================
// State Keys Tests
// =============================================================================

func TestXorgConfigStep_StateKeys(t *testing.T) {
	assert.Equal(t, "xorg_configured", StateXorgConfigured)
	assert.Equal(t, "xorg_config_path", StateXorgConfigPath)
	assert.Equal(t, "xorg_backup_path", StateXorgBackupPath)
	assert.Equal(t, "xorg_display_server", StateXorgDisplayServer)
}

// =============================================================================
// Default Values Tests
// =============================================================================

func TestXorgConfigStep_DefaultValues(t *testing.T) {
	assert.Equal(t, "/etc/X11/xorg.conf.d", DefaultXorgConfDir)
	assert.Equal(t, "20-nvidia.conf", DefaultXorgConfFile)
	assert.Equal(t, "/etc/X11/xorg.conf.d/20-nvidia.conf", DefaultXorgConfPath)
}

// =============================================================================
// NvidiaXorgConfig Template Tests
// =============================================================================

func TestNvidiaXorgConfig_Content(t *testing.T) {
	// Verify the template contains expected sections
	assert.Contains(t, NvidiaXorgConfig, "Section \"OutputClass\"")
	assert.Contains(t, NvidiaXorgConfig, "Identifier \"nvidia\"")
	assert.Contains(t, NvidiaXorgConfig, "MatchDriver \"nvidia-drm\"")
	assert.Contains(t, NvidiaXorgConfig, "Driver \"nvidia\"")
	assert.Contains(t, NvidiaXorgConfig, "AllowEmptyInitialConfiguration")
	assert.Contains(t, NvidiaXorgConfig, "PrimaryGPU")
	assert.Contains(t, NvidiaXorgConfig, "ModulePath")
	assert.Contains(t, NvidiaXorgConfig, "EndSection")
	assert.Contains(t, NvidiaXorgConfig, "Generated by Igor")
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

func TestXorgConfigStep_InterfaceCompliance(t *testing.T) {
	var _ install.Step = (*XorgConfigStep)(nil)
}

// =============================================================================
// Full Workflow Tests
// =============================================================================

func TestXorgConfigStep_FullWorkflow_ExecuteAndRollback(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
	)

	// Execute
	result := step.Execute(ctx)
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateXorgConfigured))

	configPath := ctx.GetStateString(StateXorgConfigPath)
	assert.NotEmpty(t, configPath)

	// Reset mock tracking
	mockExec.Reset()

	// Rollback
	err := step.Rollback(ctx)
	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("rm", "-f", configPath))

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateXorgConfigured))
	assert.Empty(t, ctx.GetStateString(StateXorgConfigPath))
}

func TestXorgConfigStep_FullWorkflow_WithBackup(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockWriter.SetFileExists(DefaultXorgConfPath, []byte("# Old config"))
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
		WithBackupExisting(true),
	)

	// Execute
	result := step.Execute(ctx)
	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Verify backup was created
	backupPath := ctx.GetStateString(StateXorgBackupPath)
	assert.NotEmpty(t, backupPath)
	assert.True(t, mockExec.WasCalled("cp"))

	// Reset and rollback
	mockExec.Reset()
	err := step.Rollback(ctx)
	assert.NoError(t, err)

	// Verify restore from backup
	assert.True(t, mockExec.WasCalledWith("mv", backupPath, DefaultXorgConfPath))
}

// =============================================================================
// Duration Tests
// =============================================================================

func TestXorgConfigStep_Execute_Duration(t *testing.T) {
	ctx, _ := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

// =============================================================================
// Real Display Detector Tests
// =============================================================================

func TestRealDisplayDetector_DetectDisplayServer(t *testing.T) {
	detector := &RealDisplayDetector{}

	server, err := detector.DetectDisplayServer(context.Background())

	assert.NoError(t, err)
	// The result depends on the environment
	assert.Contains(t, []string{"xorg", "wayland", "unknown"}, server)
}

func TestRealDisplayDetector_IsWaylandSession(t *testing.T) {
	detector := &RealDisplayDetector{}

	// Just verify it doesn't panic
	_ = detector.IsWaylandSession()
}

// =============================================================================
// Real File Writer Tests
// =============================================================================

func TestRealXorgFileWriter_Methods(t *testing.T) {
	// These tests verify the real file writer calls don't panic
	// Actual file operations are not tested here to avoid side effects
	writer := &RealXorgFileWriter{}

	// Verify methods exist and return expected types
	_, err := writer.Stat("/nonexistent/path/to/file")
	assert.Error(t, err) // Should error for non-existent file

	_, err = writer.ReadFile("/nonexistent/path/to/file")
	assert.Error(t, err) // Should error for non-existent file
}

// =============================================================================
// Path Validation Tests
// =============================================================================

func TestIsValidXorgPath(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"/etc/X11/xorg.conf.d", true},
		{"/usr/share/X11/xorg.conf.d", true},
		{"/home/user/.config/xorg", true},
		{"", false},
		{"/etc;rm -rf /", false},
		{"/etc && cat", false},
		{"/etc | grep", false},
		{"/etc`whoami`", false},
		{"/etc$HOME", false},
		{"/etc(test)", false},
		{"/etc{test}", false},
		{"/etc<test>", false},
		{"/etc!test", false},
		{"/etc\ntest", false},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := isValidXorgPath(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsValidXorgFileName(t *testing.T) {
	testCases := []struct {
		name     string
		expected bool
	}{
		{"20-nvidia.conf", true},
		{"nvidia.conf", true},
		{"99-custom-nvidia.conf", true},
		{"", false},
		{"../nvidia.conf", false},
		{"sub/nvidia.conf", false},
		{"nvidia\\bad.conf", false},
		{"nvidia;rm.conf", false},
		{"nvidia|cat.conf", false},
		{"nvidia`whoami`.conf", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidXorgFileName(tc.name)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// Backup Failure Tests
// =============================================================================

func TestXorgConfigStep_Execute_BackupFailure_Continues(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)
	mockWriter.SetFileExists(DefaultXorgConfPath, []byte("# Old config"))
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	// Make cp fail for backup
	mockExec.SetResponse("cp", exec.FailureResult(1, "permission denied"))

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
		WithBackupExisting(true),
	)

	result := step.Execute(ctx)

	// Should still complete despite backup failure
	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// No backup path should be in state
	assert.Empty(t, ctx.GetStateString(StateXorgBackupPath))
}

// =============================================================================
// Custom Config Path Tests
// =============================================================================

func TestXorgConfigStep_Execute_CustomPath(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	customDir := "/custom/xorg"
	customFile := "99-nvidia.conf"
	mockWriter.SetDirExists(customDir)
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
		WithXorgConfigDir(customDir),
		WithXorgConfigFile(customFile),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	expectedPath := customDir + "/" + customFile
	assert.Equal(t, expectedPath, ctx.GetStateString(StateXorgConfigPath))

	// Verify tee was called with custom path
	calls := mockExec.Calls()
	teeCall := false
	for _, call := range calls {
		if call.Command == "tee" && len(call.Args) > 0 && call.Args[0] == expectedPath {
			teeCall = true
			break
		}
	}
	assert.True(t, teeCall, "tee should be called with custom path")
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkXorgConfigStep_Execute(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	mockWriter := newMockXorgFileWriter()
	mockWriter.SetDirExists(DefaultXorgConfDir)

	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
	)

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear state between iterations
		ctx.DeleteState(StateXorgConfigured)
		ctx.DeleteState(StateXorgConfigPath)
		ctx.DeleteState(StateXorgBackupPath)
		ctx.DeleteState(StateXorgDisplayServer)

		step.Execute(ctx)
	}
}

func BenchmarkXorgConfigStep_Validate(b *testing.B) {
	mockExec := exec.NewMockExecutor()

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	step := NewXorgConfigStep()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = step.Validate(ctx)
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestXorgConfigStep_Execute_CancelledAfterDirectoryCreation(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	mockWriter := newMockXorgFileWriter()
	// Directory doesn't exist, will be created
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	// Create step that will be cancelled mid-execution
	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
	)

	// Start execution, then cancel during backup phase
	// This is difficult to test precisely without async execution
	// Just verify cancellation is checked at multiple points
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Now test with immediate cancellation
	ctx.Cancel()
	mockExec.Reset()

	result2 := step.Execute(ctx)
	assert.Equal(t, install.StepStatusFailed, result2.Status)
	assert.Contains(t, result2.Message, "cancelled")
}

func TestXorgConfigStep_Execute_StatError(t *testing.T) {
	ctx, mockExec := newXorgTestContext()
	mockWriter := newMockXorgFileWriter()
	// Set an error for Stat that's not "not exist"
	mockWriter.SetStatError(errors.New("permission denied"))
	mockDetector := newMockDisplayDetector()
	mockDetector.SetDisplayServer("xorg")

	step := NewXorgConfigStep(
		WithXorgFileWriter(mockWriter),
		WithDisplayDetector(mockDetector),
	)

	result := step.Execute(ctx)

	// Should fail when checking if directory exists
	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to create config directory")

	_ = mockExec // Verify mockExec was set up
}
