package steps

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/gpu/nouveau"
	"github.com/tungetti/igor/internal/install"
)

// =============================================================================
// Mock Nouveau Detector
// =============================================================================

// MockNouveauDetector implements nouveau.Detector for testing.
type MockNouveauDetector struct {
	isBlacklisted bool
	isLoaded      bool
	boundDevices  []string
	status        *nouveau.Status

	// Error injection
	detectErr        error
	isLoadedErr      error
	isBlacklistedErr error
	getBoundErr      error

	// Call tracking
	detectCalled        bool
	isLoadedCalled      bool
	isBlacklistedCalled bool
	getBoundCalled      bool
}

// NewMockNouveauDetector creates a new mock detector with default values.
func NewMockNouveauDetector() *MockNouveauDetector {
	return &MockNouveauDetector{
		isBlacklisted: false,
		isLoaded:      true,
		boundDevices:  []string{},
	}
}

// SetBlacklisted sets the blacklist status returned by IsBlacklisted.
func (m *MockNouveauDetector) SetBlacklisted(blacklisted bool) {
	m.isBlacklisted = blacklisted
}

// SetLoaded sets the loaded status returned by IsLoaded.
func (m *MockNouveauDetector) SetLoaded(loaded bool) {
	m.isLoaded = loaded
}

// SetIsBlacklistedError sets an error to return from IsBlacklisted.
func (m *MockNouveauDetector) SetIsBlacklistedError(err error) {
	m.isBlacklistedErr = err
}

// Detect implements nouveau.Detector.
func (m *MockNouveauDetector) Detect(ctx context.Context) (*nouveau.Status, error) {
	m.detectCalled = true
	if m.detectErr != nil {
		return nil, m.detectErr
	}
	if m.status != nil {
		return m.status, nil
	}
	return &nouveau.Status{
		Loaded:          m.isLoaded,
		InUse:           len(m.boundDevices) > 0,
		BoundDevices:    m.boundDevices,
		BlacklistExists: m.isBlacklisted,
	}, nil
}

// IsLoaded implements nouveau.Detector.
func (m *MockNouveauDetector) IsLoaded(ctx context.Context) (bool, error) {
	m.isLoadedCalled = true
	if m.isLoadedErr != nil {
		return false, m.isLoadedErr
	}
	return m.isLoaded, nil
}

// IsBlacklisted implements nouveau.Detector.
func (m *MockNouveauDetector) IsBlacklisted(ctx context.Context) (bool, error) {
	m.isBlacklistedCalled = true
	if m.isBlacklistedErr != nil {
		return false, m.isBlacklistedErr
	}
	return m.isBlacklisted, nil
}

// GetBoundDevices implements nouveau.Detector.
func (m *MockNouveauDetector) GetBoundDevices(ctx context.Context) ([]string, error) {
	m.getBoundCalled = true
	if m.getBoundErr != nil {
		return nil, m.getBoundErr
	}
	return m.boundDevices, nil
}

// Ensure MockNouveauDetector implements nouveau.Detector.
var _ nouveau.Detector = (*MockNouveauDetector)(nil)

// =============================================================================
// Mock File Writer
// =============================================================================

// MockFileWriter implements FileWriter for testing.
type MockFileWriter struct {
	writeErr  error
	removeErr error

	writeCalled      bool
	removeCalled     bool
	lastWritePath    string
	lastWriteContent string
	lastRemovePath   string
}

// NewMockFileWriter creates a new mock file writer.
func NewMockFileWriter() *MockFileWriter {
	return &MockFileWriter{}
}

// SetWriteError sets an error to return from WriteFile.
func (m *MockFileWriter) SetWriteError(err error) {
	m.writeErr = err
}

// SetRemoveError sets an error to return from RemoveFile.
func (m *MockFileWriter) SetRemoveError(err error) {
	m.removeErr = err
}

// WriteFile implements FileWriter.
func (m *MockFileWriter) WriteFile(ctx context.Context, path, content string, executor interface{}) error {
	m.writeCalled = true
	m.lastWritePath = path
	m.lastWriteContent = content
	return m.writeErr
}

// RemoveFile implements FileWriter.
func (m *MockFileWriter) RemoveFile(ctx context.Context, path string, executor interface{}) error {
	m.removeCalled = true
	m.lastRemovePath = path
	return m.removeErr
}

// Ensure MockFileWriter implements FileWriter.
var _ FileWriter = (*MockFileWriter)(nil)

// =============================================================================
// Test Helpers
// =============================================================================

// newDebianDistro creates a test Debian distribution.
func newDebianDistro() *distro.Distribution {
	return &distro.Distribution{
		ID:         "debian",
		Name:       "Debian GNU/Linux",
		VersionID:  "12",
		PrettyName: "Debian GNU/Linux 12 (bookworm)",
		Family:     constants.FamilyDebian,
	}
}

// newTestContext creates a basic test context with executor and distro.
func newTestContext() (*install.Context, *exec.MockExecutor) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newDebianDistro()),
	)

	return ctx, mockExec
}

// =============================================================================
// NouveauBlacklistStep Constructor Tests
// =============================================================================

func TestNewNouveauBlacklistStep(t *testing.T) {
	t.Run("creates with defaults", func(t *testing.T) {
		step := NewNouveauBlacklistStep()

		assert.Equal(t, "nouveau_blacklist", step.Name())
		assert.Equal(t, "Blacklist Nouveau driver", step.Description())
		assert.True(t, step.CanRollback())
		assert.Equal(t, DefaultBlacklistPath, step.blacklistPath)
		assert.False(t, step.skipInitramfs)
		assert.Nil(t, step.detector)
	})

	t.Run("creates with WithBlacklistPath", func(t *testing.T) {
		customPath := "/custom/path/blacklist.conf"
		step := NewNouveauBlacklistStep(WithBlacklistPath(customPath))

		assert.Equal(t, customPath, step.blacklistPath)
	})

	t.Run("creates with WithNouveauDetector", func(t *testing.T) {
		mockDetector := NewMockNouveauDetector()
		step := NewNouveauBlacklistStep(WithNouveauDetector(mockDetector))

		assert.Equal(t, mockDetector, step.detector)
	})

	t.Run("creates with WithSkipInitramfs true", func(t *testing.T) {
		step := NewNouveauBlacklistStep(WithSkipInitramfs(true))

		assert.True(t, step.skipInitramfs)
	})

	t.Run("creates with WithSkipInitramfs false", func(t *testing.T) {
		step := NewNouveauBlacklistStep(WithSkipInitramfs(false))

		assert.False(t, step.skipInitramfs)
	})

	t.Run("creates with WithFileWriter", func(t *testing.T) {
		mockWriter := NewMockFileWriter()
		step := NewNouveauBlacklistStep(WithFileWriter(mockWriter))

		assert.Equal(t, mockWriter, step.fileWriter)
	})

	t.Run("applies multiple options", func(t *testing.T) {
		mockDetector := NewMockNouveauDetector()
		mockWriter := NewMockFileWriter()
		customPath := "/custom/path.conf"

		step := NewNouveauBlacklistStep(
			WithBlacklistPath(customPath),
			WithNouveauDetector(mockDetector),
			WithSkipInitramfs(true),
			WithFileWriter(mockWriter),
		)

		assert.Equal(t, customPath, step.blacklistPath)
		assert.Equal(t, mockDetector, step.detector)
		assert.True(t, step.skipInitramfs)
		assert.Equal(t, mockWriter, step.fileWriter)
	})

	t.Run("later options override earlier ones", func(t *testing.T) {
		step := NewNouveauBlacklistStep(
			WithSkipInitramfs(true),
			WithSkipInitramfs(false), // This should override the first
		)

		assert.False(t, step.skipInitramfs)
	})
}

// =============================================================================
// NouveauBlacklistStep Execute Tests
// =============================================================================

func TestNouveauBlacklistStep_Execute_Success(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		WithSkipInitramfs(true), // Skip initramfs for simpler testing
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "blacklisted successfully")
	assert.True(t, mockDetector.isBlacklistedCalled)
	assert.True(t, mockWriter.writeCalled)
	assert.Equal(t, DefaultBlacklistPath, mockWriter.lastWritePath)
	assert.Contains(t, mockWriter.lastWriteContent, "blacklist nouveau")
	assert.Contains(t, mockWriter.lastWriteContent, "options nouveau modeset=0")

	// Check state was set
	assert.True(t, ctx.GetStateBool(StateNouveauBlacklisted))
	assert.Equal(t, DefaultBlacklistPath, ctx.GetStateString(StateNouveauBlacklistFile))

	// Executor should not have been called since we use mock writer
	_ = mockExec // Acknowledge mockExec is set up
}

func TestNouveauBlacklistStep_Execute_WithInitramfsUpdate(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		// skipInitramfs defaults to false
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockWriter.writeCalled)

	// Should have called initramfs update
	assert.True(t, mockExec.WasCalled("update-initramfs"))
}

func TestNouveauBlacklistStep_Execute_AlreadyBlacklisted(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockDetector.SetBlacklisted(true)
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusSkipped, result.Status)
	assert.Contains(t, result.Message, "already blacklisted")
	assert.True(t, mockDetector.isBlacklistedCalled)
	assert.False(t, mockWriter.writeCalled)
	assert.Equal(t, 0, mockExec.CallCount())

	// State should not be set for skipped step
	assert.False(t, ctx.GetStateBool(StateNouveauBlacklisted))
}

func TestNouveauBlacklistStep_Execute_DryRun(t *testing.T) {
	ctx, mockExec := newTestContext()
	ctx.DryRun = true

	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")
	assert.True(t, mockDetector.isBlacklistedCalled)
	assert.False(t, mockWriter.writeCalled)
	assert.Equal(t, 0, mockExec.CallCount())

	// State should not be set for dry run
	assert.False(t, ctx.GetStateBool(StateNouveauBlacklisted))
}

func TestNouveauBlacklistStep_Execute_MissingExecutor(t *testing.T) {
	ctx := install.NewContext(
		install.WithDistroInfo(newDebianDistro()),
	)

	step := NewNouveauBlacklistStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "executor")
}

func TestNouveauBlacklistStep_Execute_WriteError(t *testing.T) {
	ctx, _ := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()
	mockWriter.SetWriteError(errors.New("permission denied"))

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		WithSkipInitramfs(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to create blacklist file")
	assert.Error(t, result.Error)
	assert.True(t, mockWriter.writeCalled)

	// State should not be set on failure
	assert.False(t, ctx.GetStateBool(StateNouveauBlacklisted))
}

func TestNouveauBlacklistStep_Execute_InitramfsError(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	// Set initramfs command to fail
	mockExec.SetResponse("update-initramfs", exec.FailureResult(1, "initramfs update failed"))

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		// skipInitramfs defaults to false
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to regenerate initramfs")
	assert.Error(t, result.Error)
	assert.True(t, mockWriter.writeCalled)

	// Should have tried to rollback the blacklist file
	assert.True(t, mockWriter.removeCalled)

	// State should not be set on failure
	assert.False(t, ctx.GetStateBool(StateNouveauBlacklisted))
}

func TestNouveauBlacklistStep_Execute_Cancelled(t *testing.T) {
	ctx, mockExec := newTestContext()
	ctx.Cancel() // Cancel immediately

	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	assert.True(t, errors.Is(result.Error, context.Canceled))
	assert.False(t, mockDetector.isBlacklistedCalled)
	assert.False(t, mockWriter.writeCalled)
	assert.Equal(t, 0, mockExec.CallCount())
}

func TestNouveauBlacklistStep_Execute_CancelledAfterWrite(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newDebianDistro()),
	)

	mockDetector := NewMockNouveauDetector()

	// Create a writer that cancels the context after writing
	cancellingWriter := &CancellingMockFileWriter{
		MockFileWriter: NewMockFileWriter(),
		installCtx:     ctx,
	}

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(cancellingWriter),
		// skipInitramfs defaults to false, so cancellation check will happen
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	assert.True(t, cancellingWriter.writeCalled)

	// Should have tried to rollback
	assert.True(t, cancellingWriter.removeCalled)
}

// CancellingMockFileWriter cancels the context after WriteFile is called.
type CancellingMockFileWriter struct {
	*MockFileWriter
	installCtx *install.Context
}

func (m *CancellingMockFileWriter) WriteFile(ctx context.Context, path, content string, executor interface{}) error {
	m.writeCalled = true
	m.lastWritePath = path
	m.lastWriteContent = content
	// Cancel the context after writing
	if m.installCtx != nil {
		m.installCtx.Cancel()
	}
	return m.writeErr
}

func TestNouveauBlacklistStep_Execute_DebianInitramfs(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newDebianDistro()),
	)

	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalledWith("update-initramfs", "-u"))
}

func TestNouveauBlacklistStep_Execute_FedoraInitramfs(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newFedoraDistro()),
	)

	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalledWith("dracut", "--force"))
}

func TestNouveauBlacklistStep_Execute_ArchInitramfs(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newArchDistro()),
	)

	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalledWith("mkinitcpio", "-P"))
}

func TestNouveauBlacklistStep_Execute_SUSEInitramfs(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	suseDistro := &distro.Distribution{
		ID:         "opensuse-leap",
		Name:       "openSUSE Leap",
		VersionID:  "15.5",
		PrettyName: "openSUSE Leap 15.5",
		Family:     constants.FamilySUSE,
	}

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(suseDistro),
	)

	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalledWith("dracut", "--force"))
}

func TestNouveauBlacklistStep_Execute_UnknownDistroInitramfs(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	unknownDistro := &distro.Distribution{
		ID:         "unknown",
		Name:       "Unknown Linux",
		PrettyName: "Unknown Linux",
		Family:     constants.FamilyUnknown,
	}

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(unknownDistro),
	)

	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	// Should fall back to Debian-style command
	assert.True(t, mockExec.WasCalledWith("update-initramfs", "-u"))
}

func TestNouveauBlacklistStep_Execute_NoDistroInfo(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		// No distro info
	)

	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	// Should fall back to Debian-style command when no distro info
	assert.True(t, mockExec.WasCalledWith("update-initramfs", "-u"))
}

func TestNouveauBlacklistStep_Execute_Duration(t *testing.T) {
	ctx, _ := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		WithSkipInitramfs(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

func TestNouveauBlacklistStep_Execute_BlacklistCheckError(t *testing.T) {
	ctx, _ := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockDetector.SetIsBlacklistedError(errors.New("failed to check blacklist"))
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		WithSkipInitramfs(true),
	)

	// Should proceed even if blacklist check fails
	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockDetector.isBlacklistedCalled)
	assert.True(t, mockWriter.writeCalled)
}

func TestNouveauBlacklistStep_Execute_CustomBlacklistPath(t *testing.T) {
	ctx, _ := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()
	customPath := "/tmp/test-blacklist-nouveau.conf"

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		WithBlacklistPath(customPath),
		WithSkipInitramfs(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Equal(t, customPath, mockWriter.lastWritePath)
	assert.Equal(t, customPath, ctx.GetStateString(StateNouveauBlacklistFile))
}

func TestNouveauBlacklistStep_Execute_WithRealExecutor(t *testing.T) {
	// Test using real executor mock (not file writer mock)
	ctx, mockExec := newTestContext()
	mockDetector := NewMockNouveauDetector()

	// Set response for tee command (used for file writing)
	mockExec.SetResponse("tee", exec.SuccessResult(""))

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithSkipInitramfs(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalled("tee"))
}

func TestNouveauBlacklistStep_Execute_TeeCommandFails(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockDetector := NewMockNouveauDetector()

	// Set tee command to fail
	mockExec.SetResponse("tee", exec.FailureResult(1, "permission denied"))

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithSkipInitramfs(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to create blacklist file")
}

// =============================================================================
// NouveauBlacklistStep Rollback Tests
// =============================================================================

func TestNouveauBlacklistStep_Rollback_Success(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		WithSkipInitramfs(true),
	)

	// First execute the step
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Reset mock tracking
	mockWriter.removeCalled = false
	mockWriter.lastRemovePath = ""

	// Now rollback
	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.True(t, mockWriter.removeCalled)
	assert.Equal(t, DefaultBlacklistPath, mockWriter.lastRemovePath)

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateNouveauBlacklisted))
	assert.Empty(t, ctx.GetStateString(StateNouveauBlacklistFile))

	_ = mockExec // Acknowledge mockExec
}

func TestNouveauBlacklistStep_Rollback_WithInitramfs(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		// skipInitramfs defaults to false
	)

	// First execute the step
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Reset mock tracking
	mockExec.Reset()

	// Now rollback
	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.True(t, mockWriter.removeCalled)

	// Should have regenerated initramfs
	assert.True(t, mockExec.WasCalled("update-initramfs"))
}

func TestNouveauBlacklistStep_Rollback_NoBlacklistCreated(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithFileWriter(mockWriter),
	)

	// Don't execute, just rollback directly
	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.False(t, mockWriter.removeCalled)
	assert.Equal(t, 0, mockExec.CallCount())
}

func TestNouveauBlacklistStep_Rollback_NoBlacklistPath(t *testing.T) {
	ctx, _ := newTestContext()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithFileWriter(mockWriter),
	)

	// Set blacklisted but not the path
	ctx.SetState(StateNouveauBlacklisted, true)

	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.False(t, mockWriter.removeCalled)
}

func TestNouveauBlacklistStep_Rollback_RemoveError(t *testing.T) {
	ctx, _ := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		WithSkipInitramfs(true),
	)

	// First execute the step
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Set error for rollback
	mockWriter.SetRemoveError(errors.New("permission denied"))

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove blacklist file")
}

func TestNouveauBlacklistStep_Rollback_InitramfsError(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		// skipInitramfs defaults to false
	)

	// First execute the step
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Set initramfs to fail during rollback
	mockExec.SetResponse("update-initramfs", exec.FailureResult(1, "initramfs failed"))

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to regenerate initramfs")
}

func TestNouveauBlacklistStep_Rollback_NilExecutor(t *testing.T) {
	ctx := install.NewContext()
	ctx.SetState(StateNouveauBlacklisted, true)
	ctx.SetState(StateNouveauBlacklistFile, "/etc/modprobe.d/test.conf")

	step := NewNouveauBlacklistStep()

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor not available")
}

func TestNouveauBlacklistStep_Rollback_CustomPath(t *testing.T) {
	ctx, _ := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()
	customPath := "/custom/path/blacklist.conf"

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		WithBlacklistPath(customPath),
		WithSkipInitramfs(true),
	)

	// First execute the step
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Reset mock tracking
	mockWriter.removeCalled = false
	mockWriter.lastRemovePath = ""

	// Now rollback
	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.True(t, mockWriter.removeCalled)
	assert.Equal(t, customPath, mockWriter.lastRemovePath)
}

// =============================================================================
// NouveauBlacklistStep Validate Tests
// =============================================================================

func TestNouveauBlacklistStep_Validate_Success(t *testing.T) {
	ctx, _ := newTestContext()

	step := NewNouveauBlacklistStep()

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

func TestNouveauBlacklistStep_Validate_MissingExecutor(t *testing.T) {
	ctx := install.NewContext()

	step := NewNouveauBlacklistStep()

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor is required")
}

// =============================================================================
// NouveauBlacklistStep CanRollback Tests
// =============================================================================

func TestNouveauBlacklistStep_CanRollback(t *testing.T) {
	step := NewNouveauBlacklistStep()

	assert.True(t, step.CanRollback())
}

// =============================================================================
// NouveauBlacklistStep Options Tests
// =============================================================================

func TestNouveauBlacklistStep_Options(t *testing.T) {
	t.Run("WithBlacklistPath sets custom path", func(t *testing.T) {
		customPath := "/custom/path.conf"
		step := NewNouveauBlacklistStep(WithBlacklistPath(customPath))
		assert.Equal(t, customPath, step.blacklistPath)
	})

	t.Run("WithNouveauDetector sets custom detector", func(t *testing.T) {
		mockDetector := NewMockNouveauDetector()
		step := NewNouveauBlacklistStep(WithNouveauDetector(mockDetector))
		assert.Equal(t, mockDetector, step.detector)
	})

	t.Run("WithSkipInitramfs sets skipInitramfs to true", func(t *testing.T) {
		step := NewNouveauBlacklistStep(WithSkipInitramfs(true))
		assert.True(t, step.skipInitramfs)
	})

	t.Run("WithSkipInitramfs sets skipInitramfs to false", func(t *testing.T) {
		step := NewNouveauBlacklistStep(WithSkipInitramfs(false))
		assert.False(t, step.skipInitramfs)
	})

	t.Run("WithFileWriter sets custom file writer", func(t *testing.T) {
		mockWriter := NewMockFileWriter()
		step := NewNouveauBlacklistStep(WithFileWriter(mockWriter))
		assert.Equal(t, mockWriter, step.fileWriter)
	})

	t.Run("default blacklistPath is DefaultBlacklistPath", func(t *testing.T) {
		step := NewNouveauBlacklistStep()
		assert.Equal(t, DefaultBlacklistPath, step.blacklistPath)
	})

	t.Run("default skipInitramfs is false", func(t *testing.T) {
		step := NewNouveauBlacklistStep()
		assert.False(t, step.skipInitramfs)
	})

	t.Run("default detector is nil (uses real)", func(t *testing.T) {
		step := NewNouveauBlacklistStep()
		assert.Nil(t, step.detector)
	})
}

// =============================================================================
// NouveauBlacklistStep Interface Compliance Tests
// =============================================================================

func TestNouveauBlacklistStep_InterfaceCompliance(t *testing.T) {
	var _ install.Step = (*NouveauBlacklistStep)(nil)
}

// =============================================================================
// NouveauBlacklistStep State Keys Tests
// =============================================================================

func TestNouveauBlacklistStep_StateKeys(t *testing.T) {
	assert.Equal(t, "nouveau_blacklisted", StateNouveauBlacklisted)
	assert.Equal(t, "nouveau_blacklist_file", StateNouveauBlacklistFile)
}

// =============================================================================
// NouveauBlacklistStep Blacklist Content Tests
// =============================================================================

func TestNouveauBlacklistStep_BlacklistContent(t *testing.T) {
	// Verify blacklist content contains required directives
	assert.Contains(t, blacklistContent, "blacklist nouveau")
	assert.Contains(t, blacklistContent, "options nouveau modeset=0")
	assert.Contains(t, blacklistContent, "Generated by Igor")
}

// =============================================================================
// NouveauBlacklistStep Full Workflow Tests
// =============================================================================

func TestNouveauBlacklistStep_FullWorkflow_ExecuteAndRollback(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		WithSkipInitramfs(true),
	)

	// Execute
	result := step.Execute(ctx)
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateNouveauBlacklisted))

	// Get the path that was stored
	blacklistPath := ctx.GetStateString(StateNouveauBlacklistFile)
	assert.NotEmpty(t, blacklistPath)

	// Reset mock tracking
	mockWriter.removeCalled = false
	mockWriter.lastRemovePath = ""

	// Rollback
	err := step.Rollback(ctx)
	assert.NoError(t, err)
	assert.True(t, mockWriter.removeCalled)
	assert.Equal(t, blacklistPath, mockWriter.lastRemovePath)

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateNouveauBlacklisted))
	assert.Empty(t, ctx.GetStateString(StateNouveauBlacklistFile))

	_ = mockExec // Acknowledge mockExec
}

func TestNouveauBlacklistStep_FullWorkflow_WithInitramfs(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		// skipInitramfs defaults to false
	)

	// Execute
	result := step.Execute(ctx)
	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Verify initramfs was updated
	assert.True(t, mockExec.WasCalled("update-initramfs"))

	// Reset mock tracking
	mockExec.Reset()

	// Rollback
	err := step.Rollback(ctx)
	assert.NoError(t, err)

	// Verify initramfs was updated again during rollback
	assert.True(t, mockExec.WasCalled("update-initramfs"))
}

// =============================================================================
// NouveauBlacklistStep getInitramfsCommand Tests
// =============================================================================

func TestNouveauBlacklistStep_getInitramfsCommand(t *testing.T) {
	step := NewNouveauBlacklistStep()

	tests := []struct {
		name     string
		family   constants.DistroFamily
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "Debian family",
			family:   constants.FamilyDebian,
			wantCmd:  "update-initramfs",
			wantArgs: []string{"-u"},
		},
		{
			name:     "RHEL family",
			family:   constants.FamilyRHEL,
			wantCmd:  "dracut",
			wantArgs: []string{"--force"},
		},
		{
			name:     "SUSE family",
			family:   constants.FamilySUSE,
			wantCmd:  "dracut",
			wantArgs: []string{"--force"},
		},
		{
			name:     "Arch family",
			family:   constants.FamilyArch,
			wantCmd:  "mkinitcpio",
			wantArgs: []string{"-P"},
		},
		{
			name:     "Unknown family (fallback)",
			family:   constants.FamilyUnknown,
			wantCmd:  "update-initramfs",
			wantArgs: []string{"-u"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := step.getInitramfsCommand(tt.family)
			assert.Equal(t, tt.wantCmd, cmd)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}

// =============================================================================
// NouveauBlacklistStep getDistroFamily Tests
// =============================================================================

func TestNouveauBlacklistStep_getDistroFamily(t *testing.T) {
	step := NewNouveauBlacklistStep()

	t.Run("returns family from DistroInfo", func(t *testing.T) {
		ctx := install.NewContext(
			install.WithDistroInfo(newDebianDistro()),
		)
		family := step.getDistroFamily(ctx)
		assert.Equal(t, constants.FamilyDebian, family)
	})

	t.Run("returns unknown when DistroInfo is nil", func(t *testing.T) {
		ctx := install.NewContext()
		family := step.getDistroFamily(ctx)
		assert.Equal(t, constants.FamilyUnknown, family)
	})
}

// =============================================================================
// NouveauBlacklistStep getDetector Tests
// =============================================================================

func TestNouveauBlacklistStep_getDetector(t *testing.T) {
	t.Run("returns injected detector when set", func(t *testing.T) {
		mockDetector := NewMockNouveauDetector()
		step := NewNouveauBlacklistStep(WithNouveauDetector(mockDetector))

		detector := step.getDetector()
		assert.Equal(t, mockDetector, detector)
	})

	t.Run("returns new detector when none injected", func(t *testing.T) {
		step := NewNouveauBlacklistStep()

		detector := step.getDetector()
		assert.NotNil(t, detector)
		// Should be a real detector implementation
		_, ok := detector.(*nouveau.DetectorImpl)
		assert.True(t, ok)
	})
}

// =============================================================================
// NouveauBlacklistStep removeBlacklistFileAtPath Tests (with real executor)
// =============================================================================

func TestNouveauBlacklistStep_removeBlacklistFileAtPath_Success(t *testing.T) {
	ctx, mockExec := newTestContext()

	step := NewNouveauBlacklistStep()

	err := step.removeBlacklistFileAtPath(ctx, "/tmp/test-blacklist.conf")

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("rm", "-f", "/tmp/test-blacklist.conf"))
}

func TestNouveauBlacklistStep_removeBlacklistFileAtPath_Error(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockExec.SetResponse("rm", exec.FailureResult(1, "permission denied"))

	step := NewNouveauBlacklistStep()

	err := step.removeBlacklistFileAtPath(ctx, "/tmp/test-blacklist.conf")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestNouveauBlacklistStep_removeBlacklistFileAtPath_EmptyError(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockExec.SetResponse("rm", exec.FailureResult(1, ""))

	step := NewNouveauBlacklistStep()

	err := step.removeBlacklistFileAtPath(ctx, "/tmp/test-blacklist.conf")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown error")
}

// =============================================================================
// NouveauBlacklistStep writeBlacklistFile Tests (with real executor)
// =============================================================================

func TestNouveauBlacklistStep_writeBlacklistFile_Success(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockExec.SetResponse("tee", exec.SuccessResult(""))

	step := NewNouveauBlacklistStep(
		WithBlacklistPath("/tmp/test-blacklist.conf"),
	)

	err := step.writeBlacklistFile(ctx)

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalled("tee"))

	// Verify the input was the blacklist content
	calls := mockExec.Calls()
	for _, call := range calls {
		if call.Command == "tee" {
			assert.Contains(t, string(call.Input), "blacklist nouveau")
		}
	}
}

func TestNouveauBlacklistStep_writeBlacklistFile_Error(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockExec.SetResponse("tee", exec.FailureResult(1, "permission denied"))

	step := NewNouveauBlacklistStep()

	err := step.writeBlacklistFile(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestNouveauBlacklistStep_writeBlacklistFile_EmptyError(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockExec.SetResponse("tee", exec.FailureResult(1, ""))

	step := NewNouveauBlacklistStep()

	err := step.writeBlacklistFile(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown error")
}

// =============================================================================
// NouveauBlacklistStep regenerateInitramfs Tests
// =============================================================================

func TestNouveauBlacklistStep_regenerateInitramfs_Success(t *testing.T) {
	ctx, mockExec := newTestContext()

	step := NewNouveauBlacklistStep()

	err := step.regenerateInitramfs(ctx)

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("update-initramfs", "-u"))
}

func TestNouveauBlacklistStep_regenerateInitramfs_Error(t *testing.T) {
	ctx, mockExec := newTestContext()
	mockExec.SetResponse("update-initramfs", exec.FailureResult(1, "update-initramfs failed"))

	step := NewNouveauBlacklistStep()

	err := step.regenerateInitramfs(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update-initramfs failed")
}

func TestNouveauBlacklistStep_regenerateInitramfs_StdoutAsError(t *testing.T) {
	// Test case where stderr is empty but stdout has the error message
	ctx, mockExec := newTestContext()
	mockExec.SetResponse("update-initramfs", &exec.Result{
		ExitCode: 1,
		Stdout:   []byte("error in stdout"),
		Stderr:   []byte(""),
	})

	step := NewNouveauBlacklistStep()

	err := step.regenerateInitramfs(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error in stdout")
}

func TestNouveauBlacklistStep_regenerateInitramfs_UnknownError(t *testing.T) {
	// Test case where both stderr and stdout are empty
	ctx, mockExec := newTestContext()
	mockExec.SetResponse("update-initramfs", &exec.Result{
		ExitCode: 1,
		Stdout:   []byte(""),
		Stderr:   []byte(""),
	})

	step := NewNouveauBlacklistStep()

	err := step.regenerateInitramfs(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown error")
}

// =============================================================================
// NouveauBlacklistStep Rollback with Real Executor Tests
// =============================================================================

func TestNouveauBlacklistStep_Rollback_WithRealExecutor(t *testing.T) {
	ctx, mockExec := newTestContext()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(NewMockNouveauDetector()),
		WithSkipInitramfs(true),
	)

	// Set state as if Execute had been called
	ctx.SetState(StateNouveauBlacklisted, true)
	ctx.SetState(StateNouveauBlacklistFile, DefaultBlacklistPath)

	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("rm", "-f", DefaultBlacklistPath))
}

func TestNouveauBlacklistStep_Rollback_WithRealExecutorAndInitramfs(t *testing.T) {
	ctx, mockExec := newTestContext()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(NewMockNouveauDetector()),
		// skipInitramfs defaults to false
	)

	// Set state as if Execute had been called
	ctx.SetState(StateNouveauBlacklisted, true)
	ctx.SetState(StateNouveauBlacklistFile, DefaultBlacklistPath)

	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("rm", "-f", DefaultBlacklistPath))
	assert.True(t, mockExec.WasCalledWith("update-initramfs", "-u"))
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkNouveauBlacklistStep_Execute(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	mockDetector := NewMockNouveauDetector()
	mockWriter := NewMockFileWriter()

	step := NewNouveauBlacklistStep(
		WithNouveauDetector(mockDetector),
		WithFileWriter(mockWriter),
		WithSkipInitramfs(true),
	)

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newDebianDistro()),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear state between iterations
		ctx.DeleteState(StateNouveauBlacklisted)
		ctx.DeleteState(StateNouveauBlacklistFile)
		mockDetector.isBlacklistedCalled = false
		mockWriter.writeCalled = false

		step.Execute(ctx)
	}
}

func BenchmarkNouveauBlacklistStep_Validate(b *testing.B) {
	mockExec := exec.NewMockExecutor()

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newDebianDistro()),
	)

	step := NewNouveauBlacklistStep()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = step.Validate(ctx)
	}
}
