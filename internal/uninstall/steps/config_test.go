package steps

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/install"
)

// =============================================================================
// Mock File Checker for Testing
// =============================================================================

// MockFileChecker implements FileChecker for testing.
type MockFileChecker struct {
	existingFiles map[string]bool
}

// NewMockFileChecker creates a new mock file checker.
func NewMockFileChecker() *MockFileChecker {
	return &MockFileChecker{
		existingFiles: make(map[string]bool),
	}
}

// SetFileExists configures a file to exist.
func (m *MockFileChecker) SetFileExists(path string) {
	m.existingFiles[path] = true
}

// SetFilesExist configures multiple files to exist.
func (m *MockFileChecker) SetFilesExist(paths []string) {
	for _, path := range paths {
		m.existingFiles[path] = true
	}
}

// FileExists implements FileChecker.
func (m *MockFileChecker) FileExists(path string) bool {
	return m.existingFiles[path]
}

// Ensure MockFileChecker implements FileChecker.
var _ FileChecker = (*MockFileChecker)(nil)

// =============================================================================
// Test Helpers
// =============================================================================

// newConfigTestContext creates a basic test context with executor for config cleanup tests.
func newConfigTestContext() (*install.Context, *exec.MockExecutor) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	return ctx, mockExec
}

// =============================================================================
// ConfigCleanupStep Constructor Tests
// =============================================================================

func TestNewConfigCleanupStep_DefaultOptions(t *testing.T) {
	step := NewConfigCleanupStep()

	assert.Equal(t, "config_cleanup", step.Name())
	assert.Equal(t, "Remove NVIDIA configuration files", step.Description())
	assert.True(t, step.CanRollback())
	assert.Empty(t, step.configPaths)
	assert.Equal(t, DefaultBackupDir, step.backupDir)
	assert.True(t, step.createBackup)
	assert.True(t, step.removeBlacklist)
	assert.True(t, step.removeXorgConf)
	assert.True(t, step.removeModprobe)
	assert.True(t, step.removePersistence)
	assert.Nil(t, step.fileChecker)
}

func TestNewConfigCleanupStep_WithAllOptions(t *testing.T) {
	mockChecker := NewMockFileChecker()
	customPaths := []string{"/etc/custom/nvidia.conf"}

	step := NewConfigCleanupStep(
		WithConfigPaths(customPaths),
		WithBackupDir("/custom/backup"),
		WithCreateBackup(false),
		WithRemoveBlacklist(false),
		WithRemoveXorgConf(false),
		WithRemoveModprobe(false),
		WithRemovePersistence(false),
		WithFileChecker(mockChecker),
	)

	assert.Equal(t, customPaths, step.configPaths)
	assert.Equal(t, "/custom/backup", step.backupDir)
	assert.False(t, step.createBackup)
	assert.False(t, step.removeBlacklist)
	assert.False(t, step.removeXorgConf)
	assert.False(t, step.removeModprobe)
	assert.False(t, step.removePersistence)
	assert.NotNil(t, step.fileChecker)
}

func TestNewConfigCleanupStep_WithConfigPaths_AppendsMultipleCalls(t *testing.T) {
	step := NewConfigCleanupStep(
		WithConfigPaths([]string{"/etc/path1.conf"}),
		WithConfigPaths([]string{"/etc/path2.conf", "/etc/path3.conf"}),
	)

	assert.Equal(t, []string{"/etc/path1.conf", "/etc/path2.conf", "/etc/path3.conf"}, step.configPaths)
}

func TestConfigCleanupStep_Name(t *testing.T) {
	step := NewConfigCleanupStep()
	assert.Equal(t, "config_cleanup", step.Name())
}

func TestConfigCleanupStep_Description(t *testing.T) {
	step := NewConfigCleanupStep()
	assert.Equal(t, "Remove NVIDIA configuration files", step.Description())
}

func TestConfigCleanupStep_CanRollback_WithBackup(t *testing.T) {
	step := NewConfigCleanupStep(WithCreateBackup(true))
	assert.True(t, step.CanRollback())
}

func TestConfigCleanupStep_CanRollback_WithoutBackup(t *testing.T) {
	step := NewConfigCleanupStep(WithCreateBackup(false))
	assert.False(t, step.CanRollback())
}

// =============================================================================
// ConfigCleanupStep Execute Tests - Successful Removal
// =============================================================================

func TestConfigCleanupStep_Execute_Success_WithExistingFiles(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFileExists("/etc/modprobe.d/blacklist-nouveau.conf")
	mockChecker.SetFileExists("/etc/X11/xorg.conf.d/20-nvidia.conf")

	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithCreateBackup(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "removed")
	assert.True(t, result.CanRollback)
	assert.True(t, ctx.GetStateBool(StateConfigsCleaned))

	// Check that cleaned configs were stored
	cleanedRaw, ok := ctx.GetState(StateCleanedConfigs)
	assert.True(t, ok)
	cleaned, ok := cleanedRaw.([]string)
	assert.True(t, ok)
	assert.Contains(t, cleaned, "/etc/modprobe.d/blacklist-nouveau.conf")
	assert.Contains(t, cleaned, "/etc/X11/xorg.conf.d/20-nvidia.conf")
}

func TestConfigCleanupStep_Execute_Success_WithSpecificPaths(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFileExists("/etc/custom/nvidia.conf")

	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithConfigPaths([]string{"/etc/custom/nvidia.conf"}),
		WithRemoveBlacklist(false),
		WithRemoveXorgConf(false),
		WithRemoveModprobe(false),
		WithRemovePersistence(false),
		WithFileChecker(mockChecker),
		WithCreateBackup(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.False(t, result.CanRollback) // No backup
	assert.True(t, ctx.GetStateBool(StateConfigsCleaned))
}

func TestConfigCleanupStep_Execute_Success_WithoutBackup(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFileExists("/etc/modprobe.d/blacklist-nouveau.conf")

	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithCreateBackup(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.False(t, result.CanRollback)
	assert.True(t, ctx.GetStateBool(StateConfigsCleaned))

	// Check that no configs were backed up
	backedUpRaw, ok := ctx.GetState(StateBackedUpConfigs)
	assert.True(t, ok)
	backedUp, ok := backedUpRaw.([]string)
	assert.True(t, ok)
	assert.Empty(t, backedUp)
}

// =============================================================================
// ConfigCleanupStep Execute Tests - Dry Run
// =============================================================================

func TestConfigCleanupStep_Execute_DryRun(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFileExists("/etc/modprobe.d/blacklist-nouveau.conf")
	mockChecker.SetFileExists("/etc/X11/xorg.conf.d/20-nvidia.conf")

	ctx, mockExec := newConfigTestContext()
	ctx.DryRun = true
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")

	// Should NOT have actually removed anything
	assert.False(t, mockExec.WasCalled("rm"))

	// State should not be set in dry run
	assert.False(t, ctx.GetStateBool(StateConfigsCleaned))
}

func TestConfigCleanupStep_Execute_DryRun_WithBackup(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFileExists("/etc/modprobe.d/blacklist-nouveau.conf")

	ctx, mockExec := newConfigTestContext()
	ctx.DryRun = true
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithCreateBackup(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")

	// Should NOT have created backup directory or removed anything
	assert.False(t, mockExec.WasCalled("mkdir"))
	assert.False(t, mockExec.WasCalled("cp"))
	assert.False(t, mockExec.WasCalled("rm"))
}

// =============================================================================
// ConfigCleanupStep Execute Tests - No Files to Remove (Skip)
// =============================================================================

func TestConfigCleanupStep_Execute_NoFilesToRemove_Skips(t *testing.T) {
	mockChecker := NewMockFileChecker()
	// No files exist

	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusSkipped, result.Status)
	assert.Contains(t, result.Message, "no NVIDIA configuration files to remove")
	assert.False(t, mockExec.WasCalled("rm"))
}

func TestConfigCleanupStep_Execute_AllFlagsDisabled_Skips(t *testing.T) {
	mockChecker := NewMockFileChecker()
	// Even if files exist, they shouldn't be collected with all flags disabled

	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithRemoveBlacklist(false),
		WithRemoveXorgConf(false),
		WithRemoveModprobe(false),
		WithRemovePersistence(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusSkipped, result.Status)
	assert.Contains(t, result.Message, "no NVIDIA configuration files to remove")
}

// =============================================================================
// ConfigCleanupStep Execute Tests - Cancellation
// =============================================================================

func TestConfigCleanupStep_Execute_Cancelled_BeforeStart(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFileExists("/etc/modprobe.d/blacklist-nouveau.conf")

	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))
	ctx.Cancel() // Cancel immediately

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	assert.True(t, errors.Is(result.Error, context.Canceled))
	assert.False(t, mockExec.WasCalled("rm"))
}

func TestConfigCleanupStep_Execute_Cancelled_DuringRemoval(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFileExists("/etc/modprobe.d/blacklist-nouveau.conf")
	mockChecker.SetFileExists("/etc/X11/xorg.conf.d/20-nvidia.conf")

	ctx, mockExec := newConfigTestContext()

	// Set up to cancel after first successful operation
	mockExec.SetDefaultResponse(exec.SuccessResult(""))
	// We'll cancel after first file is processed

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithCreateBackup(false),
	)

	// Custom behavior to cancel after first rm
	go func() {
		for {
			if mockExec.CallCount() >= 1 {
				ctx.Cancel()
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()

	result := step.Execute(ctx)

	// Either completed or cancelled depending on timing
	if result.Status == install.StepStatusFailed {
		assert.Contains(t, result.Message, "cancelled")
	}
	// If it completed before cancellation, that's also fine
}

// =============================================================================
// ConfigCleanupStep Execute Tests - Error Handling
// =============================================================================

func TestConfigCleanupStep_Execute_NilContext(t *testing.T) {
	step := NewConfigCleanupStep()

	// This should panic or fail gracefully
	defer func() {
		if r := recover(); r != nil {
			// Panic is acceptable for nil context
			t.Log("Recovered from panic on nil context")
		}
	}()

	result := step.Execute(nil)
	assert.Equal(t, install.StepStatusFailed, result.Status)
}

func TestConfigCleanupStep_Execute_NilExecutor(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFileExists("/etc/modprobe.d/blacklist-nouveau.conf")

	ctx := install.NewContext()
	// No executor set

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "executor is required")
}

func TestConfigCleanupStep_Execute_RemoveFails(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFileExists("/etc/modprobe.d/blacklist-nouveau.conf")

	ctx, mockExec := newConfigTestContext()
	mockExec.SetResponse("rm", exec.FailureResult(1, "Permission denied"))
	mockExec.SetResponse("mkdir", exec.SuccessResult(""))
	mockExec.SetResponse("cp", exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithCreateBackup(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to remove config file")
	assert.Error(t, result.Error)
}

func TestConfigCleanupStep_Execute_BackupDirCreationFails(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFileExists("/etc/modprobe.d/blacklist-nouveau.conf")

	ctx, mockExec := newConfigTestContext()
	mockExec.SetResponse("mkdir", exec.FailureResult(1, "Permission denied"))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithCreateBackup(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to create backup directory")
	assert.Error(t, result.Error)
}

func TestConfigCleanupStep_Execute_BackupFileFails_Continues(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFileExists("/etc/modprobe.d/blacklist-nouveau.conf")

	ctx, mockExec := newConfigTestContext()
	// mkdir succeeds, cp fails, rm succeeds
	mockExec.SetResponse("mkdir", exec.SuccessResult(""))
	mockExec.SetResponse("cp", exec.FailureResult(1, "No space left on device"))
	mockExec.SetResponse("rm", exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithCreateBackup(true),
	)

	result := step.Execute(ctx)

	// Should still complete since backup failure is not fatal
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	// But CanRollback should be false since no backups were made
	assert.False(t, result.CanRollback)
}

// =============================================================================
// ConfigCleanupStep Execute Tests - State Storage
// =============================================================================

func TestConfigCleanupStep_Execute_StoresCorrectState(t *testing.T) {
	mockChecker := NewMockFileChecker()
	// Use specific paths that we control
	specificPaths := []string{"/etc/custom1.conf", "/etc/custom2.conf"}
	mockChecker.SetFilesExist(specificPaths)

	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithConfigPaths(specificPaths),
		WithRemoveBlacklist(false), // Disable all default paths
		WithRemoveXorgConf(false),
		WithRemoveModprobe(false),
		WithRemovePersistence(false),
		WithBackupDir("/var/lib/igor/backup/configs"), // Use allowed path
		WithCreateBackup(true),
	)

	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Verify all state keys
	assert.True(t, ctx.GetStateBool(StateConfigsCleaned))
	assert.Equal(t, "/var/lib/igor/backup/configs", ctx.GetStateString(StateBackupDir))

	cleanedRaw, ok := ctx.GetState(StateCleanedConfigs)
	require.True(t, ok)
	cleaned, ok := cleanedRaw.([]string)
	require.True(t, ok)
	assert.Len(t, cleaned, 2)

	backedUpRaw, ok := ctx.GetState(StateBackedUpConfigs)
	require.True(t, ok)
	backedUp, ok := backedUpRaw.([]string)
	require.True(t, ok)
	assert.Len(t, backedUp, 2)
}

func TestConfigCleanupStep_StateKeys(t *testing.T) {
	assert.Equal(t, "configs_cleaned", StateConfigsCleaned)
	assert.Equal(t, "cleaned_configs", StateCleanedConfigs)
	assert.Equal(t, "backed_up_configs", StateBackedUpConfigs)
	assert.Equal(t, "backup_dir", StateBackupDir)
}

// =============================================================================
// ConfigCleanupStep Validate Tests
// =============================================================================

func TestConfigCleanupStep_Validate_Success(t *testing.T) {
	ctx, _ := newConfigTestContext()

	step := NewConfigCleanupStep()
	err := step.Validate(ctx)

	assert.NoError(t, err)
}

func TestConfigCleanupStep_Validate_NilContext(t *testing.T) {
	step := NewConfigCleanupStep()
	err := step.Validate(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is nil")
}

func TestConfigCleanupStep_Validate_NilExecutor(t *testing.T) {
	ctx := install.NewContext()

	step := NewConfigCleanupStep()
	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor is required")
}

func TestConfigCleanupStep_Validate_InvalidBackupDir(t *testing.T) {
	ctx, _ := newConfigTestContext()

	step := NewConfigCleanupStep(
		WithBackupDir("/some/path;rm -rf /"), // Command injection attempt
	)
	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid backup directory path")
}

func TestConfigCleanupStep_Validate_InvalidConfigPath(t *testing.T) {
	ctx, _ := newConfigTestContext()

	step := NewConfigCleanupStep(
		WithConfigPaths([]string{"/etc/valid.conf", "/etc/../../../root/.ssh/authorized_keys"}),
	)
	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config path")
}

// =============================================================================
// ConfigCleanupStep Rollback Tests
// =============================================================================

func TestConfigCleanupStep_Rollback_Success(t *testing.T) {
	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	// Set up state as if Execute had run
	ctx.SetState(StateConfigsCleaned, true)
	ctx.SetState(StateCleanedConfigs, []string{
		"/etc/modprobe.d/blacklist-nouveau.conf",
		"/etc/X11/xorg.conf.d/20-nvidia.conf",
	})
	ctx.SetState(StateBackedUpConfigs, []string{
		"/etc/modprobe.d/blacklist-nouveau.conf",
		"/etc/X11/xorg.conf.d/20-nvidia.conf",
	})
	ctx.SetState(StateBackupDir, "/var/lib/igor/backup/configs")

	step := NewConfigCleanupStep()
	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// Verify cp was called to restore files
	assert.True(t, mockExec.WasCalled("cp"))

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateConfigsCleaned))
}

func TestConfigCleanupStep_Rollback_NoConfigsCleaned(t *testing.T) {
	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	// State indicates no configs were cleaned
	ctx.SetState(StateConfigsCleaned, false)

	step := NewConfigCleanupStep()
	err := step.Rollback(ctx)

	assert.NoError(t, err)
	// Should not have called any commands
	assert.Equal(t, 0, mockExec.CallCount())
}

func TestConfigCleanupStep_Rollback_NoBackedUpConfigs(t *testing.T) {
	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	// State indicates configs were cleaned but not backed up
	ctx.SetState(StateConfigsCleaned, true)
	ctx.SetState(StateBackedUpConfigs, []string{})

	step := NewConfigCleanupStep()
	err := step.Rollback(ctx)

	assert.NoError(t, err)
	// Should not have called any commands
	assert.Equal(t, 0, mockExec.CallCount())
}

func TestConfigCleanupStep_Rollback_RestoreFails(t *testing.T) {
	ctx, mockExec := newConfigTestContext()
	mockExec.SetResponse("cp", exec.FailureResult(1, "No such file or directory"))
	mockExec.SetResponse("mkdir", exec.SuccessResult(""))

	// Set up state
	ctx.SetState(StateConfigsCleaned, true)
	ctx.SetState(StateBackedUpConfigs, []string{"/etc/modprobe.d/blacklist-nouveau.conf"})
	ctx.SetState(StateBackupDir, "/var/lib/igor/backup/configs")

	step := NewConfigCleanupStep()
	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to restore config")

	// State should still be cleared (we tried to rollback)
	assert.False(t, ctx.GetStateBool(StateConfigsCleaned))
}

func TestConfigCleanupStep_Rollback_NilExecutor(t *testing.T) {
	ctx := install.NewContext()
	// No executor set

	// Set up state
	ctx.SetState(StateConfigsCleaned, true)
	ctx.SetState(StateBackedUpConfigs, []string{"/etc/modprobe.d/blacklist-nouveau.conf"})

	step := NewConfigCleanupStep()
	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor not available")
}

func TestConfigCleanupStep_Rollback_InvalidStateType(t *testing.T) {
	ctx, _ := newConfigTestContext()

	// Set up state with invalid type
	ctx.SetState(StateConfigsCleaned, true)
	ctx.SetState(StateBackedUpConfigs, "not a slice") // Wrong type

	step := NewConfigCleanupStep()
	err := step.Rollback(ctx)

	// Should not error, just skip
	assert.NoError(t, err)
}

// =============================================================================
// Path Validation Tests
// =============================================================================

func TestIsValidPath_ValidPaths(t *testing.T) {
	validPaths := []string{
		"/etc/modprobe.d/blacklist-nouveau.conf",
		"/etc/X11/xorg.conf.d/20-nvidia.conf",
		"/usr/share/X11/xorg.conf.d/nvidia.conf",
		"/var/lib/igor/backup/configs/test.conf",
	}

	for _, path := range validPaths {
		t.Run(path, func(t *testing.T) {
			assert.True(t, isValidPath(path), "expected valid: %s", path)
		})
	}
}

func TestIsValidPath_InvalidPaths(t *testing.T) {
	invalidPaths := []string{
		"",                                 // Empty
		"relative/path.conf",               // Not absolute
		"/etc/../../../root/.ssh/config",   // Path traversal
		"/etc/nvidia.conf;rm -rf /",        // Command injection with semicolon
		"/etc/nvidia.conf|cat /etc/passwd", // Command injection with pipe
		"/etc/nvidia.conf`whoami`",         // Command injection with backticks
		"/etc/nvidia.conf$(id)",            // Command injection with $()
		"/etc/nvidia.conf\nrm -rf /",       // Newline injection
		"/home/user/.config/nvidia.conf",   // Not in allowed dirs
		"/tmp/nvidia.conf",                 // Not in allowed dirs
		"/root/.bashrc",                    // Not in allowed dirs
	}

	for _, path := range invalidPaths {
		t.Run(path, func(t *testing.T) {
			assert.False(t, isValidPath(path), "expected invalid: %s", path)
		})
	}
}

func TestGetBackupPath(t *testing.T) {
	testCases := []struct {
		srcPath   string
		backupDir string
		expected  string
	}{
		{
			srcPath:   "/etc/modprobe.d/blacklist-nouveau.conf",
			backupDir: "/var/lib/igor/backup/configs",
			expected:  "/var/lib/igor/backup/configs/etc/modprobe.d/blacklist-nouveau.conf",
		},
		{
			srcPath:   "/etc/X11/xorg.conf.d/20-nvidia.conf",
			backupDir: "/custom/backup",
			expected:  "/custom/backup/etc/X11/xorg.conf.d/20-nvidia.conf",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.srcPath, func(t *testing.T) {
			result := getBackupPath(tc.srcPath, tc.backupDir)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// collectConfigPaths Tests
// =============================================================================

func TestConfigCleanupStep_CollectConfigPaths_AllEnabled(t *testing.T) {
	step := NewConfigCleanupStep()
	paths := step.collectConfigPaths()

	// Should contain paths from all categories
	assert.Contains(t, paths, "/etc/modprobe.d/blacklist-nouveau.conf")
	assert.Contains(t, paths, "/etc/X11/xorg.conf.d/20-nvidia.conf")
	assert.Contains(t, paths, "/etc/modprobe.d/nvidia.conf")
}

func TestConfigCleanupStep_CollectConfigPaths_OnlyBlacklist(t *testing.T) {
	step := NewConfigCleanupStep(
		WithRemoveBlacklist(true),
		WithRemoveXorgConf(false),
		WithRemoveModprobe(false),
		WithRemovePersistence(false),
	)
	paths := step.collectConfigPaths()

	// Should only contain blacklist paths
	for _, path := range paths {
		found := false
		for _, blacklistPath := range NouveauBlacklistPaths {
			if path == blacklistPath {
				found = true
				break
			}
		}
		assert.True(t, found, "unexpected path: %s", path)
	}
}

func TestConfigCleanupStep_CollectConfigPaths_OnlyXorg(t *testing.T) {
	step := NewConfigCleanupStep(
		WithRemoveBlacklist(false),
		WithRemoveXorgConf(true),
		WithRemoveModprobe(false),
		WithRemovePersistence(false),
	)
	paths := step.collectConfigPaths()

	// Should only contain xorg paths
	for _, path := range paths {
		found := false
		for _, xorgPath := range XorgConfigPaths {
			if path == xorgPath {
				found = true
				break
			}
		}
		assert.True(t, found, "unexpected path: %s", path)
	}
}

func TestConfigCleanupStep_CollectConfigPaths_OnlyModprobe(t *testing.T) {
	step := NewConfigCleanupStep(
		WithRemoveBlacklist(false),
		WithRemoveXorgConf(false),
		WithRemoveModprobe(true),
		WithRemovePersistence(false),
	)
	paths := step.collectConfigPaths()

	// Should only contain modprobe paths
	for _, path := range paths {
		found := false
		for _, modprobePath := range ModprobeConfigPaths {
			if path == modprobePath {
				found = true
				break
			}
		}
		assert.True(t, found, "unexpected path: %s", path)
	}
}

func TestConfigCleanupStep_CollectConfigPaths_OnlyPersistence(t *testing.T) {
	step := NewConfigCleanupStep(
		WithRemoveBlacklist(false),
		WithRemoveXorgConf(false),
		WithRemoveModprobe(false),
		WithRemovePersistence(true),
	)
	paths := step.collectConfigPaths()

	// Should only contain persistence paths
	for _, path := range paths {
		found := false
		for _, persistPath := range PersistenceConfigPaths {
			if path == persistPath {
				found = true
				break
			}
		}
		assert.True(t, found, "unexpected path: %s", path)
	}
}

func TestConfigCleanupStep_CollectConfigPaths_CustomPathsFirst(t *testing.T) {
	customPath := "/etc/custom-nvidia.conf"
	step := NewConfigCleanupStep(
		WithConfigPaths([]string{customPath}),
		WithRemoveBlacklist(true),
	)
	paths := step.collectConfigPaths()

	// Custom path should be first
	assert.Equal(t, customPath, paths[0])
}

func TestConfigCleanupStep_CollectConfigPaths_Deduplication(t *testing.T) {
	// Add a path that's already in the defaults
	existingPath := "/etc/modprobe.d/blacklist-nouveau.conf"
	step := NewConfigCleanupStep(
		WithConfigPaths([]string{existingPath}),
		WithRemoveBlacklist(true),
	)
	paths := step.collectConfigPaths()

	// Path should only appear once
	count := 0
	for _, p := range paths {
		if p == existingPath {
			count++
		}
	}
	assert.Equal(t, 1, count, "path should appear only once")
}

func TestConfigCleanupStep_CollectConfigPaths_SkipsInvalidPaths(t *testing.T) {
	step := NewConfigCleanupStep(
		WithConfigPaths([]string{
			"/etc/valid.conf",
			"/etc/../../../root/.ssh/config", // Invalid - will be skipped
		}),
		WithRemoveBlacklist(false),
		WithRemoveXorgConf(false),
		WithRemoveModprobe(false),
		WithRemovePersistence(false),
	)
	paths := step.collectConfigPaths()

	assert.Contains(t, paths, "/etc/valid.conf")
	assert.NotContains(t, paths, "/etc/../../../root/.ssh/config")
}

// =============================================================================
// Each Remove Option Flag Tests
// =============================================================================

func TestConfigCleanupStep_Execute_OnlyRemoveBlacklist(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFilesExist(NouveauBlacklistPaths)
	mockChecker.SetFilesExist(XorgConfigPaths)
	mockChecker.SetFilesExist(ModprobeConfigPaths)

	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithRemoveBlacklist(true),
		WithRemoveXorgConf(false),
		WithRemoveModprobe(false),
		WithRemovePersistence(false),
		WithCreateBackup(false),
	)

	result := step.Execute(ctx)
	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Check only blacklist files were cleaned
	cleanedRaw, _ := ctx.GetState(StateCleanedConfigs)
	cleaned := cleanedRaw.([]string)

	for _, path := range cleaned {
		found := false
		for _, blacklistPath := range NouveauBlacklistPaths {
			if path == blacklistPath {
				found = true
				break
			}
		}
		assert.True(t, found, "unexpected path in cleaned: %s", path)
	}
}

func TestConfigCleanupStep_Execute_OnlyRemoveXorg(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFilesExist(NouveauBlacklistPaths)
	mockChecker.SetFilesExist(XorgConfigPaths)
	mockChecker.SetFilesExist(ModprobeConfigPaths)

	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithRemoveBlacklist(false),
		WithRemoveXorgConf(true),
		WithRemoveModprobe(false),
		WithRemovePersistence(false),
		WithCreateBackup(false),
	)

	result := step.Execute(ctx)
	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Check only xorg files were cleaned
	cleanedRaw, _ := ctx.GetState(StateCleanedConfigs)
	cleaned := cleanedRaw.([]string)

	for _, path := range cleaned {
		found := false
		for _, xorgPath := range XorgConfigPaths {
			if path == xorgPath {
				found = true
				break
			}
		}
		assert.True(t, found, "unexpected path in cleaned: %s", path)
	}
}

// =============================================================================
// Duration Tests
// =============================================================================

func TestConfigCleanupStep_Execute_Duration(t *testing.T) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFileExists("/etc/modprobe.d/blacklist-nouveau.conf")

	ctx, mockExec := newConfigTestContext()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithCreateBackup(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

func TestConfigCleanupStep_InterfaceCompliance(t *testing.T) {
	var _ install.Step = (*ConfigCleanupStep)(nil)
}

func TestFileChecker_InterfaceCompliance(t *testing.T) {
	var _ FileChecker = (*RealFileChecker)(nil)
	var _ FileChecker = (*MockFileChecker)(nil)
}

// =============================================================================
// Default Path Variables Tests
// =============================================================================

func TestDefaultNvidiaConfigPaths_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, DefaultNvidiaConfigPaths)
}

func TestNouveauBlacklistPaths_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, NouveauBlacklistPaths)
}

func TestXorgConfigPaths_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, XorgConfigPaths)
}

func TestModprobeConfigPaths_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, ModprobeConfigPaths)
}

func TestPersistenceConfigPaths_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, PersistenceConfigPaths)
}

func TestAllowedConfigDirs_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, AllowedConfigDirs)
}

func TestAllowedConfigDirs_ContainsExpectedDirs(t *testing.T) {
	assert.Contains(t, AllowedConfigDirs, "/etc/")
	assert.Contains(t, AllowedConfigDirs, "/usr/share/")
	assert.Contains(t, AllowedConfigDirs, "/var/lib/igor/")
}

// =============================================================================
// Options Tests
// =============================================================================

func TestConfigCleanupStep_Options(t *testing.T) {
	t.Run("WithConfigPaths", func(t *testing.T) {
		step := NewConfigCleanupStep(
			WithConfigPaths([]string{"/etc/test1.conf", "/etc/test2.conf"}),
		)
		assert.Equal(t, []string{"/etc/test1.conf", "/etc/test2.conf"}, step.configPaths)
	})

	t.Run("WithBackupDir", func(t *testing.T) {
		step := NewConfigCleanupStep(WithBackupDir("/custom/backup"))
		assert.Equal(t, "/custom/backup", step.backupDir)
	})

	t.Run("WithCreateBackup true", func(t *testing.T) {
		step := NewConfigCleanupStep(WithCreateBackup(true))
		assert.True(t, step.createBackup)
	})

	t.Run("WithCreateBackup false", func(t *testing.T) {
		step := NewConfigCleanupStep(WithCreateBackup(false))
		assert.False(t, step.createBackup)
	})

	t.Run("WithRemoveBlacklist true", func(t *testing.T) {
		step := NewConfigCleanupStep(WithRemoveBlacklist(true))
		assert.True(t, step.removeBlacklist)
	})

	t.Run("WithRemoveBlacklist false", func(t *testing.T) {
		step := NewConfigCleanupStep(WithRemoveBlacklist(false))
		assert.False(t, step.removeBlacklist)
	})

	t.Run("WithRemoveXorgConf true", func(t *testing.T) {
		step := NewConfigCleanupStep(WithRemoveXorgConf(true))
		assert.True(t, step.removeXorgConf)
	})

	t.Run("WithRemoveXorgConf false", func(t *testing.T) {
		step := NewConfigCleanupStep(WithRemoveXorgConf(false))
		assert.False(t, step.removeXorgConf)
	})

	t.Run("WithRemoveModprobe true", func(t *testing.T) {
		step := NewConfigCleanupStep(WithRemoveModprobe(true))
		assert.True(t, step.removeModprobe)
	})

	t.Run("WithRemoveModprobe false", func(t *testing.T) {
		step := NewConfigCleanupStep(WithRemoveModprobe(false))
		assert.False(t, step.removeModprobe)
	})

	t.Run("WithRemovePersistence true", func(t *testing.T) {
		step := NewConfigCleanupStep(WithRemovePersistence(true))
		assert.True(t, step.removePersistence)
	})

	t.Run("WithRemovePersistence false", func(t *testing.T) {
		step := NewConfigCleanupStep(WithRemovePersistence(false))
		assert.False(t, step.removePersistence)
	})

	t.Run("WithFileChecker", func(t *testing.T) {
		checker := NewMockFileChecker()
		step := NewConfigCleanupStep(WithFileChecker(checker))
		assert.NotNil(t, step.fileChecker)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkConfigCleanupStep_Execute(b *testing.B) {
	mockChecker := NewMockFileChecker()
	mockChecker.SetFilesExist(NouveauBlacklistPaths)
	mockChecker.SetFilesExist(XorgConfigPaths)

	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewConfigCleanupStep(
		WithFileChecker(mockChecker),
		WithCreateBackup(false),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := install.NewContext(
			install.WithExecutor(mockExec),
		)
		step.Execute(ctx)
	}
}

func BenchmarkConfigCleanupStep_Validate(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	step := NewConfigCleanupStep()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = step.Validate(ctx)
	}
}

func BenchmarkConfigCleanupStep_CollectConfigPaths(b *testing.B) {
	step := NewConfigCleanupStep(
		WithConfigPaths([]string{"/etc/extra1.conf", "/etc/extra2.conf"}),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = step.collectConfigPaths()
	}
}

func BenchmarkIsValidPath(b *testing.B) {
	paths := []string{
		"/etc/modprobe.d/blacklist-nouveau.conf",
		"/etc/X11/xorg.conf.d/20-nvidia.conf",
		"/usr/share/X11/xorg.conf.d/nvidia.conf",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			_ = isValidPath(path)
		}
	}
}
