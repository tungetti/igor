package app

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tungetti/igor/internal/config"
	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/logging"
	"github.com/tungetti/igor/internal/privilege"
)

// =============================================================================
// Container Tests
// =============================================================================

func TestNewContainer(t *testing.T) {
	c := NewContainer()
	assert.NotNil(t, c)
	assert.Nil(t, c.Config)
	assert.Nil(t, c.Logger)
	assert.Nil(t, c.Executor)
	assert.Nil(t, c.Privilege)
}

func TestContainer_SetGetConfig(t *testing.T) {
	c := NewContainer()
	cfg := &config.Config{LogLevel: "debug"}

	c.SetConfig(cfg)
	got := c.GetConfig()

	assert.Equal(t, cfg, got)
	assert.Equal(t, "debug", got.LogLevel)
}

func TestContainer_SetGetLogger(t *testing.T) {
	c := NewContainer()
	logger := logging.NewNop()

	c.SetLogger(logger)
	got := c.GetLogger()

	assert.Equal(t, logger, got)
}

func TestContainer_SetGetExecutor(t *testing.T) {
	c := NewContainer()
	executor := exec.NewMockExecutor()

	c.SetExecutor(executor)
	got := c.GetExecutor()

	assert.Equal(t, executor, got)
}

func TestContainer_SetGetPrivilege(t *testing.T) {
	c := NewContainer()
	priv := privilege.NewManager()

	c.SetPrivilege(priv)
	got := c.GetPrivilege()

	assert.Equal(t, priv, got)
}

func TestContainer_Validate_Success(t *testing.T) {
	c := NewContainer()
	c.SetConfig(&config.Config{})
	c.SetLogger(logging.NewNop())

	err := c.Validate()

	assert.NoError(t, err)
}

func TestContainer_Validate_MissingConfig(t *testing.T) {
	c := NewContainer()
	c.SetLogger(logging.NewNop())

	err := c.Validate()

	assert.Error(t, err)
	assert.True(t, errors.IsCode(err, errors.Configuration))
	assert.Contains(t, err.Error(), "config not initialized")
}

func TestContainer_Validate_MissingLogger(t *testing.T) {
	c := NewContainer()
	c.SetConfig(&config.Config{})

	err := c.Validate()

	assert.Error(t, err)
	assert.True(t, errors.IsCode(err, errors.Configuration))
	assert.Contains(t, err.Error(), "logger not initialized")
}

func TestContainer_Validate_OptionalExecutorAndPrivilege(t *testing.T) {
	// Executor and Privilege are optional at startup
	c := NewContainer()
	c.SetConfig(&config.Config{})
	c.SetLogger(logging.NewNop())

	err := c.Validate()

	assert.NoError(t, err)
	assert.Nil(t, c.GetExecutor())
	assert.Nil(t, c.GetPrivilege())
}

func TestContainer_ConcurrentAccess(t *testing.T) {
	c := NewContainer()
	cfg := &config.Config{}
	logger := logging.NewNop()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.SetConfig(cfg)
			c.SetLogger(logger)
		}()
		go func() {
			defer wg.Done()
			_ = c.GetConfig()
			_ = c.GetLogger()
		}()
	}
	wg.Wait()
}

// =============================================================================
// Lifecycle Tests
// =============================================================================

func TestNewLifecycle(t *testing.T) {
	timeout := 5 * time.Second
	l := NewLifecycle(timeout)

	assert.NotNil(t, l)
	assert.Equal(t, timeout, l.Timeout())
	assert.False(t, l.IsShuttingDown())
}

func TestLifecycle_OnShutdown(t *testing.T) {
	l := NewLifecycle(time.Second)
	var callOrder []int

	l.OnShutdown(func(ctx context.Context) error {
		callOrder = append(callOrder, 1)
		return nil
	})
	l.OnShutdown(func(ctx context.Context) error {
		callOrder = append(callOrder, 2)
		return nil
	})
	l.OnShutdown(func(ctx context.Context) error {
		callOrder = append(callOrder, 3)
		return nil
	})

	err := l.Shutdown()

	assert.NoError(t, err)
	// Shutdown functions should be called in reverse order (LIFO)
	assert.Equal(t, []int{3, 2, 1}, callOrder)
}

func TestLifecycle_Shutdown_ReturnsLastError(t *testing.T) {
	l := NewLifecycle(time.Second)
	firstErr := errors.New(errors.Unknown, "first error")
	lastErr := errors.New(errors.Unknown, "last error")

	l.OnShutdown(func(ctx context.Context) error {
		return firstErr
	})
	l.OnShutdown(func(ctx context.Context) error {
		return lastErr
	})

	err := l.Shutdown()

	// Should return the first error (since we call in reverse order, firstErr is called last)
	assert.Equal(t, firstErr, err)
}

func TestLifecycle_Shutdown_ClosesChannels(t *testing.T) {
	l := NewLifecycle(time.Second)

	err := l.Shutdown()

	assert.NoError(t, err)

	// Both channels should be closed
	select {
	case <-l.ShutdownCh():
		// Good, channel is closed
	default:
		t.Error("ShutdownCh should be closed")
	}

	select {
	case <-l.Done():
		// Good, channel is closed
	default:
		t.Error("Done should be closed")
	}
}

func TestLifecycle_Shutdown_Idempotent(t *testing.T) {
	l := NewLifecycle(time.Second)
	var callCount int32

	l.OnShutdown(func(ctx context.Context) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})

	// Call shutdown multiple times
	_ = l.Shutdown()
	_ = l.Shutdown()
	_ = l.Shutdown()

	// Shutdown function should only be called once
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestLifecycle_IsShuttingDown(t *testing.T) {
	l := NewLifecycle(time.Second)

	assert.False(t, l.IsShuttingDown())

	_ = l.Shutdown()

	assert.True(t, l.IsShuttingDown())
}

func TestLifecycle_Shutdown_WithContext(t *testing.T) {
	l := NewLifecycle(time.Second)
	var receivedCtx context.Context

	l.OnShutdown(func(ctx context.Context) error {
		receivedCtx = ctx
		return nil
	})

	_ = l.Shutdown()

	assert.NotNil(t, receivedCtx)
	// Context should have a deadline
	_, ok := receivedCtx.Deadline()
	assert.True(t, ok)
}

func TestLifecycle_WaitForSignal_ShutdownChannel(t *testing.T) {
	l := NewLifecycle(time.Second)

	// Trigger shutdown in another goroutine
	go func() {
		time.Sleep(10 * time.Millisecond)
		_ = l.Shutdown()
	}()

	// WaitForSignal should return when shutdown is called
	sig := l.WaitForSignal()

	assert.Nil(t, sig) // Returns nil when shutdown channel is closed
}

func TestLifecycle_ShutdownTimeout(t *testing.T) {
	timeout := 50 * time.Millisecond
	l := NewLifecycle(timeout)

	var ctxDeadline time.Time
	l.OnShutdown(func(ctx context.Context) error {
		ctxDeadline, _ = ctx.Deadline()
		return nil
	})

	start := time.Now()
	_ = l.Shutdown()

	// Deadline should be approximately timeout from start
	assert.WithinDuration(t, start.Add(timeout), ctxDeadline, 10*time.Millisecond)
}

// =============================================================================
// App Tests
// =============================================================================

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.Equal(t, "unknown", opts.Version)
	assert.Equal(t, "unknown", opts.BuildTime)
	assert.Equal(t, "unknown", opts.GitCommit)
	assert.Equal(t, 30*time.Second, opts.ShutdownTimeout)
}

func TestNew(t *testing.T) {
	opts := Options{
		Version:         "1.0.0",
		BuildTime:       "2024-01-01",
		GitCommit:       "abc123",
		ShutdownTimeout: 10 * time.Second,
	}

	app := New(opts)

	assert.NotNil(t, app)
	assert.Equal(t, "1.0.0", app.Version())
	assert.Equal(t, "2024-01-01", app.BuildTime())
	assert.Equal(t, "abc123", app.GitCommit())
	assert.NotNil(t, app.Container())
	assert.NotNil(t, app.Lifecycle())
}

func TestApp_Initialize_Success(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("log_level: debug\n"), 0644)
	require.NoError(t, err)

	app := New(DefaultOptions())
	ctx := context.Background()

	err = app.Initialize(ctx, configPath)

	assert.NoError(t, err)
	assert.NotNil(t, app.Container().GetConfig())
	assert.NotNil(t, app.Container().GetLogger())
	assert.NotNil(t, app.Container().GetExecutor())
	assert.NotNil(t, app.Container().GetPrivilege())
}

func TestApp_Initialize_NoConfigFile(t *testing.T) {
	// Initialize without a config file (should use defaults)
	app := New(DefaultOptions())
	ctx := context.Background()

	err := app.Initialize(ctx, "")

	assert.NoError(t, err)
	assert.NotNil(t, app.Container().GetConfig())
}

func TestApp_Initialize_InvalidConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: [yaml: content\n"), 0644)
	require.NoError(t, err)

	app := New(DefaultOptions())
	ctx := context.Background()

	err = app.Initialize(ctx, configPath)

	assert.Error(t, err)
	assert.True(t, errors.IsCode(err, errors.Configuration))
}

func TestApp_Initialize_WithLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	logPath := filepath.Join(tmpDir, "app.log")
	configContent := "log_level: debug\nlog_file: " + logPath + "\n"
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	app := New(DefaultOptions())
	ctx := context.Background()

	err = app.Initialize(ctx, configPath)

	assert.NoError(t, err)
	// Log file should be created
	_, err = os.Stat(logPath)
	assert.NoError(t, err)
}

func TestApp_Run_Success(t *testing.T) {
	app := New(DefaultOptions())
	app.container.SetConfig(&config.Config{})
	app.container.SetLogger(logging.NewNop())

	ctx := context.Background()
	err := app.Run(ctx)

	assert.NoError(t, err)
}

func TestApp_Run_PanicRecovery(t *testing.T) {
	app := New(DefaultOptions())
	app.container.SetLogger(logging.NewNop())

	// Test panic recovery through the handlePanic method
	err := app.handlePanic("test panic")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic: test panic")
}

func TestApp_Shutdown(t *testing.T) {
	app := New(DefaultOptions())
	app.container.SetLogger(logging.NewNop())

	var shutdownCalled bool
	app.Lifecycle().OnShutdown(func(ctx context.Context) error {
		shutdownCalled = true
		return nil
	})

	err := app.Shutdown()

	assert.NoError(t, err)
	assert.True(t, shutdownCalled)
}

func TestApp_RecoverPanic(t *testing.T) {
	app := New(DefaultOptions())
	app.container.SetLogger(logging.NewNop())

	// This should not panic
	func() {
		defer app.RecoverPanic()
		panic("test panic")
	}()

	// If we get here, the panic was recovered
}

func TestApp_Accessors(t *testing.T) {
	opts := Options{
		Version:         "1.0.0",
		BuildTime:       "2024-01-01",
		GitCommit:       "abc123",
		ShutdownTimeout: 10 * time.Second,
	}

	app := New(opts)

	assert.Equal(t, "1.0.0", app.Version())
	assert.Equal(t, "2024-01-01", app.BuildTime())
	assert.Equal(t, "abc123", app.GitCommit())
	assert.NotNil(t, app.Container())
	assert.NotNil(t, app.Lifecycle())
}

func TestApp_Initialize_WithCommandTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := "command_timeout: 5m\n"
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	app := New(DefaultOptions())
	ctx := context.Background()

	err = app.Initialize(ctx, configPath)

	assert.NoError(t, err)
	assert.Equal(t, 5*time.Minute, app.Container().GetConfig().CommandTimeout)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestApp_FullLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("log_level: info\n"), 0644)
	require.NoError(t, err)

	opts := Options{
		Version:         "1.0.0",
		BuildTime:       "2024-01-01",
		GitCommit:       "abc123",
		ShutdownTimeout: time.Second,
	}
	app := New(opts)
	ctx := context.Background()

	// Initialize
	err = app.Initialize(ctx, configPath)
	require.NoError(t, err)

	// Run
	err = app.Run(ctx)
	require.NoError(t, err)

	// Register a shutdown handler
	var cleanupCalled bool
	app.Lifecycle().OnShutdown(func(ctx context.Context) error {
		cleanupCalled = true
		return nil
	})

	// Shutdown
	err = app.Shutdown()
	require.NoError(t, err)

	assert.True(t, cleanupCalled)
	assert.True(t, app.Lifecycle().IsShuttingDown())
}

func TestLifecycle_ConcurrentShutdown(t *testing.T) {
	l := NewLifecycle(time.Second)
	var callCount int32

	l.OnShutdown(func(ctx context.Context) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = l.Shutdown()
		}()
	}
	wg.Wait()

	// Despite concurrent calls, shutdown should only happen once
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestContainer_AllDependencies(t *testing.T) {
	c := NewContainer()

	// Set all dependencies
	cfg := &config.Config{LogLevel: "debug"}
	logger := logging.NewNop()
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()

	c.SetConfig(cfg)
	c.SetLogger(logger)
	c.SetExecutor(mockExec)
	c.SetPrivilege(priv)

	// Validate should pass
	err := c.Validate()
	assert.NoError(t, err)

	// All getters should return the set values
	assert.Equal(t, cfg, c.GetConfig())
	assert.Equal(t, logger, c.GetLogger())
	assert.Equal(t, mockExec, c.GetExecutor())
	assert.Equal(t, priv, c.GetPrivilege())
}

func TestApp_HandlePanic_WithoutLogger(t *testing.T) {
	app := New(DefaultOptions())
	// Don't set a logger

	err := app.handlePanic("test panic without logger")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic: test panic without logger")
}

func TestLifecycle_ShutdownFuncError(t *testing.T) {
	l := NewLifecycle(time.Second)
	expectedErr := errors.New(errors.Unknown, "shutdown error")

	l.OnShutdown(func(ctx context.Context) error {
		return expectedErr
	})

	err := l.Shutdown()

	assert.Equal(t, expectedErr, err)
}

func TestLifecycle_EmptyShutdown(t *testing.T) {
	l := NewLifecycle(time.Second)

	// No shutdown functions registered
	err := l.Shutdown()

	assert.NoError(t, err)
	assert.True(t, l.IsShuttingDown())
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestContainer_NilValues(t *testing.T) {
	c := NewContainer()

	// All getters should handle nil gracefully
	assert.Nil(t, c.GetConfig())
	assert.Nil(t, c.GetLogger())
	assert.Nil(t, c.GetExecutor())
	assert.Nil(t, c.GetPrivilege())
}

func TestApp_RunWithNilLogger(t *testing.T) {
	app := New(DefaultOptions())
	// Set config but not logger
	app.container.SetConfig(&config.Config{})

	ctx := context.Background()
	err := app.Run(ctx)

	// Should not panic, just log to stderr or skip logging
	assert.NoError(t, err)
}

func TestLifecycle_Timeout_DefaultValue(t *testing.T) {
	opts := DefaultOptions()
	app := New(opts)

	assert.Equal(t, 30*time.Second, app.Lifecycle().Timeout())
}

func TestApp_RunWithLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("log_level: info\n"), 0644)
	require.NoError(t, err)

	opts := Options{
		Version:         "1.0.0",
		BuildTime:       "2024-01-01",
		GitCommit:       "abc123",
		ShutdownTimeout: time.Second,
	}
	application := New(opts)
	ctx := context.Background()

	// Initialize
	err = application.Initialize(ctx, configPath)
	require.NoError(t, err)

	// Track if we completed
	done := make(chan error, 1)

	// Run in goroutine since RunWithLifecycle blocks
	go func() {
		done <- application.RunWithLifecycle(ctx)
	}()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Trigger shutdown programmatically
	_ = application.Shutdown()

	// Wait for completion
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("RunWithLifecycle did not complete in time")
	}
}

func TestApp_RunWithLifecycle_Error(t *testing.T) {
	application := New(DefaultOptions())
	// Don't initialize - this will cause Run to work but not have proper setup
	application.container.SetConfig(&config.Config{})
	application.container.SetLogger(logging.NewNop())

	ctx := context.Background()

	done := make(chan error, 1)

	go func() {
		done <- application.RunWithLifecycle(ctx)
	}()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Trigger shutdown
	_ = application.Shutdown()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("RunWithLifecycle did not complete in time")
	}
}
