package testing

import (
	"context"
	"errors"
	"sync"
	stdtesting "testing"
	"time"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/logging"
	"github.com/tungetti/igor/internal/pkg"
)

// ============================================================================
// MockLogger Tests
// ============================================================================

func TestMockLogger_BasicLogging(t *stdtesting.T) {
	logger := NewMockLogger()

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	messages := logger.Messages()
	if len(messages) != 4 {
		t.Errorf("expected 4 messages, got %d", len(messages))
	}

	// Verify message levels
	if messages[0].Level != logging.LevelDebug {
		t.Errorf("expected first message at debug level, got %s", messages[0].Level)
	}
	if messages[1].Level != logging.LevelInfo {
		t.Errorf("expected second message at info level, got %s", messages[1].Level)
	}
	if messages[2].Level != logging.LevelWarn {
		t.Errorf("expected third message at warn level, got %s", messages[2].Level)
	}
	if messages[3].Level != logging.LevelError {
		t.Errorf("expected fourth message at error level, got %s", messages[3].Level)
	}
}

func TestMockLogger_WithKeyValues(t *stdtesting.T) {
	logger := NewMockLogger()

	logger.Info("test message", "key1", "value1", "key2", 42)

	messages := logger.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if len(messages[0].Fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(messages[0].Fields))
	}
}

func TestMockLogger_SetLevel(t *stdtesting.T) {
	logger := NewMockLogger()
	logger.SetLevel(logging.LevelWarn)

	if logger.GetLevel() != logging.LevelWarn {
		t.Errorf("expected level %s, got %s", logging.LevelWarn, logger.GetLevel())
	}

	logger.Debug("debug - should be filtered")
	logger.Info("info - should be filtered")
	logger.Warn("warn - should appear")
	logger.Error("error - should appear")

	messages := logger.Messages()
	if len(messages) != 2 {
		t.Errorf("expected 2 messages (warn and error), got %d", len(messages))
	}
}

func TestMockLogger_MessagesAtLevel(t *stdtesting.T) {
	logger := NewMockLogger()

	logger.Debug("debug 1")
	logger.Info("info 1")
	logger.Debug("debug 2")
	logger.Warn("warn 1")

	debugMessages := logger.MessagesAtLevel(logging.LevelDebug)
	if len(debugMessages) != 2 {
		t.Errorf("expected 2 debug messages, got %d", len(debugMessages))
	}

	infoMessages := logger.MessagesAtLevel(logging.LevelInfo)
	if len(infoMessages) != 1 {
		t.Errorf("expected 1 info message, got %d", len(infoMessages))
	}

	warnMessages := logger.MessagesAtLevel(logging.LevelWarn)
	if len(warnMessages) != 1 {
		t.Errorf("expected 1 warn message, got %d", len(warnMessages))
	}
}

func TestMockLogger_ContainsMessage(t *stdtesting.T) {
	logger := NewMockLogger()

	logger.Info("hello world")
	logger.Error("error occurred: something went wrong")

	if !logger.ContainsMessage("hello") {
		t.Error("expected to find 'hello' in messages")
	}

	if !logger.ContainsMessage("something went wrong") {
		t.Error("expected to find 'something went wrong' in messages")
	}

	if logger.ContainsMessage("not found") {
		t.Error("expected NOT to find 'not found' in messages")
	}
}

func TestMockLogger_ContainsMessageAtLevel(t *stdtesting.T) {
	logger := NewMockLogger()

	logger.Info("info message")
	logger.Error("error message")

	if !logger.ContainsMessageAtLevel(logging.LevelInfo, "info") {
		t.Error("expected to find 'info' at info level")
	}

	if logger.ContainsMessageAtLevel(logging.LevelError, "info") {
		t.Error("expected NOT to find 'info' at error level")
	}

	if logger.ContainsMessageAtLevel(logging.LevelInfo, "error") {
		t.Error("expected NOT to find 'error' at info level")
	}
}

func TestMockLogger_Clear(t *stdtesting.T) {
	logger := NewMockLogger()

	logger.Info("message 1")
	logger.Info("message 2")

	if logger.MessageCount() != 2 {
		t.Errorf("expected 2 messages before clear, got %d", logger.MessageCount())
	}

	logger.Clear()

	if logger.MessageCount() != 0 {
		t.Errorf("expected 0 messages after clear, got %d", logger.MessageCount())
	}
}

func TestMockLogger_WithPrefix(t *stdtesting.T) {
	logger := NewMockLogger()

	prefixedLogger := logger.WithPrefix("myprefix")
	prefixedLogger.Info("test message")

	messages := logger.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Message != "myprefix: test message" {
		t.Errorf("expected prefixed message, got: %s", messages[0].Message)
	}
}

func TestMockLogger_WithFields(t *stdtesting.T) {
	logger := NewMockLogger()

	fieldsLogger := logger.WithFields("component", "test")
	fieldsLogger.Info("test message", "extra", "field")

	messages := logger.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	// Should have both the persistent fields and the extra fields
	if len(messages[0].Fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(messages[0].Fields))
	}
}

func TestMockLogger_ConcurrentAccess(t *stdtesting.T) {
	logger := NewMockLogger()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			logger.Info("message", "n", n)
		}(i)
	}

	wg.Wait()

	if logger.MessageCount() != 100 {
		t.Errorf("expected 100 messages, got %d", logger.MessageCount())
	}
}

// ============================================================================
// MockPackageManager Tests
// ============================================================================

func TestMockPackageManager_Install(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	err := pm.Install(ctx, pkg.DefaultInstallOptions(), "pkg1", "pkg2")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	calls := pm.InstallCalls()
	if len(calls) != 1 {
		t.Errorf("expected 1 install call, got %d", len(calls))
	}

	if len(calls[0]) != 2 {
		t.Errorf("expected 2 packages, got %d", len(calls[0]))
	}

	if !pm.WasPackageInstalled("pkg1") {
		t.Error("expected pkg1 to be installed")
	}

	if !pm.WasPackageInstalled("pkg2") {
		t.Error("expected pkg2 to be installed")
	}
}

func TestMockPackageManager_InstallError(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	expectedErr := errors.New("install failed")
	pm.SetInstallError(expectedErr)

	err := pm.Install(ctx, pkg.DefaultInstallOptions(), "pkg1")
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestMockPackageManager_Remove(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	// First install packages
	pm.SetInstalledPackages([]pkg.Package{
		{Name: "pkg1", Installed: true},
		{Name: "pkg2", Installed: true},
	})

	err := pm.Remove(ctx, pkg.DefaultRemoveOptions(), "pkg1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !pm.WasPackageRemoved("pkg1") {
		t.Error("expected pkg1 to be removed")
	}

	if pm.WasPackageRemoved("pkg2") {
		t.Error("expected pkg2 NOT to be removed")
	}
}

func TestMockPackageManager_Update(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	err := pm.Update(ctx, pkg.DefaultUpdateOptions())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if pm.UpdateCalls() != 1 {
		t.Errorf("expected 1 update call, got %d", pm.UpdateCalls())
	}
}

func TestMockPackageManager_IsInstalled(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	pm.SetInstalledPackages([]pkg.Package{
		{Name: "installed-pkg", Installed: true},
		{Name: "not-installed-pkg", Installed: false},
	})

	installed, err := pm.IsInstalled(ctx, "installed-pkg")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !installed {
		t.Error("expected installed-pkg to be installed")
	}

	installed, err = pm.IsInstalled(ctx, "not-installed-pkg")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if installed {
		t.Error("expected not-installed-pkg to NOT be installed")
	}

	installed, err = pm.IsInstalled(ctx, "unknown-pkg")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if installed {
		t.Error("expected unknown-pkg to NOT be installed")
	}
}

func TestMockPackageManager_Search(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	pm.SetAvailablePackages([]pkg.Package{
		{Name: "nvidia-driver-535"},
		{Name: "nvidia-driver-545"},
		{Name: "some-other-pkg"},
	})

	results, err := pm.Search(ctx, "nvidia", pkg.DefaultSearchOptions())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestMockPackageManager_Name(t *stdtesting.T) {
	pm := NewMockPackageManager()

	if pm.Name() != "mock" {
		t.Errorf("expected name 'mock', got %s", pm.Name())
	}

	pm.SetName("apt")
	if pm.Name() != "apt" {
		t.Errorf("expected name 'apt', got %s", pm.Name())
	}
}

func TestMockPackageManager_Family(t *stdtesting.T) {
	pm := NewMockPackageManager()

	if pm.Family() != constants.FamilyDebian {
		t.Errorf("expected family %s, got %s", constants.FamilyDebian, pm.Family())
	}

	pm.SetFamily(constants.FamilyRHEL)
	if pm.Family() != constants.FamilyRHEL {
		t.Errorf("expected family %s, got %s", constants.FamilyRHEL, pm.Family())
	}
}

func TestMockPackageManager_Reset(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	_ = pm.Install(ctx, pkg.DefaultInstallOptions(), "pkg1")
	_ = pm.Remove(ctx, pkg.DefaultRemoveOptions(), "pkg2")
	_ = pm.Update(ctx, pkg.DefaultUpdateOptions())

	pm.Reset()

	if len(pm.InstallCalls()) != 0 {
		t.Error("expected install calls to be cleared")
	}
	if len(pm.RemoveCalls()) != 0 {
		t.Error("expected remove calls to be cleared")
	}
	if pm.UpdateCalls() != 0 {
		t.Error("expected update calls to be cleared")
	}
}

func TestMockPackageManager_ResetAll(t *stdtesting.T) {
	pm := NewMockPackageManager()

	pm.SetInstallError(errors.New("error"))
	pm.SetRemoveError(errors.New("error"))
	pm.SetUpdateError(errors.New("error"))

	pm.ResetAll()

	ctx := context.Background()
	if err := pm.Install(ctx, pkg.DefaultInstallOptions(), "pkg"); err != nil {
		t.Error("expected install error to be cleared")
	}
	if err := pm.Remove(ctx, pkg.DefaultRemoveOptions(), "pkg"); err != nil {
		t.Error("expected remove error to be cleared")
	}
	if err := pm.Update(ctx, pkg.DefaultUpdateOptions()); err != nil {
		t.Error("expected update error to be cleared")
	}
}

func TestMockPackageManager_ContextCancellation(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := pm.Install(ctx, pkg.DefaultInstallOptions(), "pkg")
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestMockPackageManager_Repositories(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	repo := pkg.Repository{
		Name:    "nvidia",
		URL:     "https://nvidia.com/repo",
		Enabled: true,
	}

	err := pm.AddRepository(ctx, repo)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	repos, err := pm.ListRepositories(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("expected 1 repository, got %d", len(repos))
	}

	err = pm.DisableRepository(ctx, "nvidia")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	repos, _ = pm.ListRepositories(ctx)
	if repos[0].Enabled {
		t.Error("expected repository to be disabled")
	}

	err = pm.EnableRepository(ctx, "nvidia")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	repos, _ = pm.ListRepositories(ctx)
	if !repos[0].Enabled {
		t.Error("expected repository to be enabled")
	}

	err = pm.RemoveRepository(ctx, "nvidia")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	repos, _ = pm.ListRepositories(ctx)
	if len(repos) != 0 {
		t.Errorf("expected 0 repositories, got %d", len(repos))
	}
}

func TestMockPackageManager_ConcurrentAccess(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = pm.Install(ctx, pkg.DefaultInstallOptions(), "pkg")
		}(i)
	}

	wg.Wait()

	if len(pm.InstallCalls()) != 100 {
		t.Errorf("expected 100 install calls, got %d", len(pm.InstallCalls()))
	}
}

// ============================================================================
// MockDistroDetector Tests
// ============================================================================

func TestMockDistroDetector_Detect(t *stdtesting.T) {
	dist := UbuntuDistribution()
	detector := NewMockDistroDetector(dist)

	result, err := detector.Detect()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.ID != "ubuntu" {
		t.Errorf("expected ID 'ubuntu', got %s", result.ID)
	}
	if result.Family != constants.FamilyDebian {
		t.Errorf("expected family %s, got %s", constants.FamilyDebian, result.Family)
	}
}

func TestMockDistroDetector_SetDistribution(t *stdtesting.T) {
	detector := NewMockDistroDetector(nil)

	result, _ := detector.Detect()
	if result != nil {
		t.Error("expected nil result for nil distribution")
	}

	detector.SetDistribution(FedoraDistribution())

	result, _ = detector.Detect()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "fedora" {
		t.Errorf("expected ID 'fedora', got %s", result.ID)
	}
}

func TestMockDistroDetector_SetError(t *stdtesting.T) {
	detector := NewMockDistroDetector(UbuntuDistribution())

	expectedErr := errors.New("detection failed")
	detector.SetError(expectedErr)

	_, err := detector.Detect()
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

// ============================================================================
// ExecutorBuilder Tests
// ============================================================================

func TestExecutorBuilder_WithCommand(t *stdtesting.T) {
	executor := NewExecutorBuilder().
		WithCommand("lspci", "GPU output", "", 0).
		Build()

	result := executor.Execute(context.Background(), "lspci")

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if string(result.Stdout) != "GPU output" {
		t.Errorf("expected stdout 'GPU output', got %s", string(result.Stdout))
	}
}

func TestExecutorBuilder_WithCommandError(t *stdtesting.T) {
	expectedErr := errors.New("command failed")
	executor := NewExecutorBuilder().
		WithCommandError("lspci", expectedErr).
		Build()

	result := executor.Execute(context.Background(), "lspci")

	if result.Error == nil {
		t.Error("expected error, got nil")
	}
}

func TestExecutorBuilder_WithCommandSuccess(t *stdtesting.T) {
	executor := NewExecutorBuilder().
		WithCommandSuccess("nvidia-smi", "GPU info").
		Build()

	result := executor.Execute(context.Background(), "nvidia-smi")

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestExecutorBuilder_WithCommandFailure(t *stdtesting.T) {
	executor := NewExecutorBuilder().
		WithCommandFailure("nvidia-smi", 127, "command not found").
		Build()

	result := executor.Execute(context.Background(), "nvidia-smi")

	if result.ExitCode != 127 {
		t.Errorf("expected exit code 127, got %d", result.ExitCode)
	}
	if string(result.Stderr) != "command not found" {
		t.Errorf("expected stderr 'command not found', got %s", string(result.Stderr))
	}
}

func TestExecutorBuilder_WithDefaultSuccess(t *stdtesting.T) {
	executor := NewExecutorBuilder().
		WithDefaultSuccess().
		Build()

	result := executor.Execute(context.Background(), "any-command")

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestExecutorBuilder_WithDefaultError(t *stdtesting.T) {
	expectedErr := errors.New("default error")
	executor := NewExecutorBuilder().
		WithDefaultError(expectedErr).
		Build()

	result := executor.Execute(context.Background(), "any-command")

	if result.Error == nil {
		t.Error("expected error, got nil")
	}
}

func TestExecutorBuilder_MultipleCommands(t *stdtesting.T) {
	executor := NewExecutorBuilder().
		WithCommandSuccess("lspci", "PCI devices").
		WithCommandSuccess("nvidia-smi", "GPU info").
		WithDefaultFailure(1, "unknown command").
		Build()

	result1 := executor.Execute(context.Background(), "lspci")
	if string(result1.Stdout) != "PCI devices" {
		t.Errorf("expected 'PCI devices', got %s", string(result1.Stdout))
	}

	result2 := executor.Execute(context.Background(), "nvidia-smi")
	if string(result2.Stdout) != "GPU info" {
		t.Errorf("expected 'GPU info', got %s", string(result2.Stdout))
	}

	result3 := executor.Execute(context.Background(), "unknown")
	if result3.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result3.ExitCode)
	}
}

// ============================================================================
// MockProgressReporter Tests
// ============================================================================

func TestMockProgressReporter_ReportProgress(t *stdtesting.T) {
	reporter := NewMockProgressReporter()

	reporter.ReportProgress("step1", 0, 3, 0.0, "Starting")
	reporter.ReportProgress("step1", 0, 3, 33.3, "In progress")
	reporter.ReportProgress("step2", 1, 3, 66.6, "Almost done")

	updates := reporter.Updates()
	if len(updates) != 3 {
		t.Errorf("expected 3 updates, got %d", len(updates))
	}

	if updates[0].StepName != "step1" {
		t.Errorf("expected step name 'step1', got %s", updates[0].StepName)
	}
}

func TestMockProgressReporter_LastUpdate(t *stdtesting.T) {
	reporter := NewMockProgressReporter()

	if reporter.LastUpdate() != nil {
		t.Error("expected nil for empty reporter")
	}

	reporter.ReportProgress("step1", 0, 3, 0.0, "First")
	reporter.ReportProgress("step2", 1, 3, 50.0, "Last")

	last := reporter.LastUpdate()
	if last == nil {
		t.Fatal("expected non-nil last update")
	}
	if last.Message != "Last" {
		t.Errorf("expected message 'Last', got %s", last.Message)
	}
}

func TestMockProgressReporter_Complete(t *stdtesting.T) {
	reporter := NewMockProgressReporter()

	if reporter.IsComplete() {
		t.Error("expected not complete initially")
	}

	reporter.ReportComplete()

	if !reporter.IsComplete() {
		t.Error("expected complete after ReportComplete")
	}
}

func TestMockProgressReporter_Error(t *stdtesting.T) {
	reporter := NewMockProgressReporter()

	if reporter.Error() != nil {
		t.Error("expected no error initially")
	}

	expectedErr := errors.New("test error")
	reporter.ReportError(expectedErr)

	if reporter.Error() != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, reporter.Error())
	}
}

func TestMockProgressReporter_Clear(t *stdtesting.T) {
	reporter := NewMockProgressReporter()

	reporter.ReportProgress("step1", 0, 1, 100, "Done")
	reporter.ReportComplete()
	reporter.ReportError(errors.New("error"))

	reporter.Clear()

	if len(reporter.Updates()) != 0 {
		t.Error("expected updates to be cleared")
	}
	if reporter.IsComplete() {
		t.Error("expected complete to be cleared")
	}
	if reporter.Error() != nil {
		t.Error("expected error to be cleared")
	}
}

func TestMockProgressReporter_ConcurrentAccess(t *stdtesting.T) {
	reporter := NewMockProgressReporter()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			reporter.ReportProgress("step", n, 100, float64(n), "message")
		}(i)
	}

	wg.Wait()

	if len(reporter.Updates()) != 100 {
		t.Errorf("expected 100 updates, got %d", len(reporter.Updates()))
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestMockLogger_ImplementsInterface(t *stdtesting.T) {
	var _ logging.Logger = NewMockLogger()
}

func TestMockPackageManager_ImplementsInterface(t *stdtesting.T) {
	var _ pkg.Manager = NewMockPackageManager()
}

func TestMockExecutor_ImplementsInterface(t *stdtesting.T) {
	var _ exec.Executor = NewExecutorBuilder().Build()
}

func TestMocksWorkflow(t *stdtesting.T) {
	// Simulate a typical test workflow using multiple mocks
	logger := NewMockLogger()
	pm := NewMockPackageManager()
	detector := NewMockDistroDetector(UbuntuDistribution())
	executor := NewExecutorBuilder().
		WithCommandSuccess("nvidia-smi", "GPU: RTX 3080").
		Build()

	// Simulate detection
	dist, err := detector.Detect()
	if err != nil {
		t.Fatalf("detection failed: %v", err)
	}
	logger.Info("detected distribution", "id", dist.ID)

	// Simulate command execution
	result := executor.Execute(context.Background(), "nvidia-smi")
	if result.ExitCode != 0 {
		t.Fatalf("nvidia-smi failed")
	}
	logger.Debug("nvidia-smi output", "output", string(result.Stdout))

	// Simulate package installation
	ctx := context.Background()
	if err := pm.Install(ctx, pkg.DefaultInstallOptions(), "nvidia-driver-535"); err != nil {
		t.Fatalf("installation failed: %v", err)
	}
	logger.Info("installed driver", "package", "nvidia-driver-535")

	// Verify the workflow
	if !pm.WasPackageInstalled("nvidia-driver-535") {
		t.Error("expected nvidia-driver-535 to be installed")
	}
	if !logger.ContainsMessage("detected distribution") {
		t.Error("expected log message about detection")
	}
	if !logger.ContainsMessage("installed driver") {
		t.Error("expected log message about installation")
	}
	if !executor.WasCalled("nvidia-smi") {
		t.Error("expected nvidia-smi to be called")
	}
}

func TestMocksTimeout(t *stdtesting.T) {
	pm := NewMockPackageManager()

	// Create a context that will timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(2 * time.Millisecond)

	err := pm.Install(ctx, pkg.DefaultInstallOptions(), "pkg")
	if err != context.DeadlineExceeded {
		t.Errorf("expected deadline exceeded, got %v", err)
	}
}

// ============================================================================
// Additional MockPackageManager Tests
// ============================================================================

func TestMockPackageManager_Info(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	pm.SetInstalledPackages([]pkg.Package{
		{Name: "nvidia-driver-535", Version: "535.154.05", Installed: true},
	})

	// Test found package
	info, err := pm.Info(ctx, "nvidia-driver-535")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if info.Name != "nvidia-driver-535" {
		t.Errorf("expected name nvidia-driver-535, got %s", info.Name)
	}

	// Test not found package
	_, err = pm.Info(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent package")
	}
}

func TestMockPackageManager_ListInstalled(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	pm.SetInstalledPackages([]pkg.Package{
		{Name: "pkg1", Installed: true},
		{Name: "pkg2", Installed: true},
		{Name: "pkg3", Installed: false},
	})

	installed, err := pm.ListInstalled(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(installed) != 2 {
		t.Errorf("expected 2 installed packages, got %d", len(installed))
	}
}

func TestMockPackageManager_Upgrade(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	err := pm.Upgrade(ctx, pkg.DefaultInstallOptions(), "pkg1", "pkg2")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	calls := pm.UpgradeCalls()
	if len(calls) != 1 {
		t.Errorf("expected 1 upgrade call, got %d", len(calls))
	}
}

func TestMockPackageManager_UpgradeError(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	expectedErr := errors.New("upgrade failed")
	pm.SetUpgradeError(expectedErr)

	err := pm.Upgrade(ctx, pkg.DefaultInstallOptions(), "pkg1")
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestMockPackageManager_ListUpgradable(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	upgradable, err := pm.ListUpgradable(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(upgradable) != 0 {
		t.Errorf("expected 0 upgradable packages, got %d", len(upgradable))
	}
}

func TestMockPackageManager_SearchError(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	expectedErr := errors.New("search failed")
	pm.SetSearchError(expectedErr)

	_, err := pm.Search(ctx, "nvidia", pkg.DefaultSearchOptions())
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestMockPackageManager_RefreshRepositories(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	err := pm.RefreshRepositories(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockPackageManager_Clean(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	err := pm.Clean(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockPackageManager_AutoRemove(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	err := pm.AutoRemove(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockPackageManager_Verify(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	pm.SetInstalledPackages([]pkg.Package{
		{Name: "installed-pkg", Installed: true},
	})

	// Verify installed package
	ok, err := pm.Verify(ctx, "installed-pkg")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected verification to pass")
	}

	// Verify non-installed package
	_, err = pm.Verify(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for non-installed package")
	}
}

func TestMockPackageManager_IsAvailable(t *stdtesting.T) {
	pm := NewMockPackageManager()

	if !pm.IsAvailable() {
		t.Error("expected mock to be available by default")
	}

	pm.SetAvailable(false)
	if pm.IsAvailable() {
		t.Error("expected mock to be unavailable")
	}
}

func TestMockPackageManager_RepositoryNotFound(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	err := pm.RemoveRepository(ctx, "nonexistent")
	if err != pkg.ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}

	err = pm.EnableRepository(ctx, "nonexistent")
	if err != pkg.ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}

	err = pm.DisableRepository(ctx, "nonexistent")
	if err != pkg.ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}
}

// ============================================================================
// ChildMockLogger Tests
// ============================================================================

func TestChildMockLogger_AllLevels(t *stdtesting.T) {
	logger := NewMockLogger()
	child := logger.WithPrefix("child").WithFields("key", "value")

	child.Debug("debug message")
	child.Info("info message")
	child.Warn("warn message")
	child.Error("error message")

	if logger.MessageCount() != 4 {
		t.Errorf("expected 4 messages, got %d", logger.MessageCount())
	}
}

func TestChildMockLogger_SetLevel(t *stdtesting.T) {
	logger := NewMockLogger()
	child := logger.WithPrefix("child")

	child.SetLevel(logging.LevelError)

	if logger.GetLevel() != logging.LevelError {
		t.Error("expected level to be set on parent")
	}
}

func TestChildMockLogger_GetLevel(t *stdtesting.T) {
	logger := NewMockLogger()
	child := logger.WithPrefix("child")

	logger.SetLevel(logging.LevelWarn)

	if child.GetLevel() != logging.LevelWarn {
		t.Error("expected child to get parent level")
	}
}

func TestChildMockLogger_NestedPrefix(t *stdtesting.T) {
	logger := NewMockLogger()
	child1 := logger.WithPrefix("child1")
	child2 := child1.WithPrefix("child2")

	child2.Info("test message")

	messages := logger.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Message != "child2: test message" {
		t.Errorf("expected 'child2: test message', got '%s'", messages[0].Message)
	}
}

func TestChildMockLogger_NestedFields(t *stdtesting.T) {
	logger := NewMockLogger()
	child1 := logger.WithFields("key1", "value1")
	child2 := child1.WithFields("key2", "value2")

	child2.Info("test message")

	messages := logger.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if len(messages[0].Fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(messages[0].Fields))
	}
}
