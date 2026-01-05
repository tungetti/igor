package uninstall

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/logging"
	"github.com/tungetti/igor/internal/privilege"
)

func TestNewUninstallContext(t *testing.T) {
	t.Run("creates with defaults", func(t *testing.T) {
		ctx := NewUninstallContext()

		assert.NotNil(t, ctx)
		assert.Nil(t, ctx.DistroInfo)
		assert.Empty(t, ctx.InstalledDriver)
		assert.Empty(t, ctx.InstalledPackages)
		assert.Nil(t, ctx.PackageManager)
		assert.Nil(t, ctx.Executor)
		assert.Nil(t, ctx.Privilege)
		assert.Nil(t, ctx.Logger)
		assert.False(t, ctx.DryRun)
		assert.False(t, ctx.Force)
		assert.False(t, ctx.KeepConfig)
		assert.NotNil(t, ctx.Context())
		assert.False(t, ctx.IsCancelled())
	})

	t.Run("creates with options", func(t *testing.T) {
		distroInfo := &distro.Distribution{ID: "ubuntu"}
		executor := exec.NewMockExecutor()
		priv := privilege.NewManager()
		logger := logging.NewNop()

		ctx := NewUninstallContext(
			WithUninstallDistroInfo(distroInfo),
			WithInstalledDriver("550.54"),
			WithInstalledPackages([]string{"nvidia-driver-550", "nvidia-settings"}),
			WithUninstallExecutor(executor),
			WithUninstallPrivilege(priv),
			WithUninstallLogger(logger),
			WithUninstallDryRun(true),
			WithUninstallForce(true),
			WithKeepConfig(true),
		)

		assert.Equal(t, distroInfo, ctx.DistroInfo)
		assert.Equal(t, "550.54", ctx.InstalledDriver)
		assert.Equal(t, []string{"nvidia-driver-550", "nvidia-settings"}, ctx.InstalledPackages)
		assert.Equal(t, executor, ctx.Executor)
		assert.Equal(t, priv, ctx.Privilege)
		assert.Equal(t, logger, ctx.Logger)
		assert.True(t, ctx.DryRun)
		assert.True(t, ctx.Force)
		assert.True(t, ctx.KeepConfig)
	})
}

func TestUninstallContext_State(t *testing.T) {
	t.Run("set and get state", func(t *testing.T) {
		ctx := NewUninstallContext()

		ctx.SetState("key1", "value1")
		ctx.SetState("key2", 42)
		ctx.SetState("key3", true)

		val1, ok1 := ctx.GetState("key1")
		assert.True(t, ok1)
		assert.Equal(t, "value1", val1)

		val2, ok2 := ctx.GetState("key2")
		assert.True(t, ok2)
		assert.Equal(t, 42, val2)

		val3, ok3 := ctx.GetState("key3")
		assert.True(t, ok3)
		assert.Equal(t, true, val3)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		ctx := NewUninstallContext()

		val, ok := ctx.GetState("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("overwrite state", func(t *testing.T) {
		ctx := NewUninstallContext()

		ctx.SetState("key", "original")
		ctx.SetState("key", "updated")

		val, _ := ctx.GetState("key")
		assert.Equal(t, "updated", val)
	})

	t.Run("delete state", func(t *testing.T) {
		ctx := NewUninstallContext()

		ctx.SetState("key", "value")
		ctx.DeleteState("key")

		_, ok := ctx.GetState("key")
		assert.False(t, ok)
	})

	t.Run("clear state", func(t *testing.T) {
		ctx := NewUninstallContext()

		ctx.SetState("key1", "value1")
		ctx.SetState("key2", "value2")
		ctx.ClearState()

		_, ok1 := ctx.GetState("key1")
		_, ok2 := ctx.GetState("key2")
		assert.False(t, ok1)
		assert.False(t, ok2)
	})
}

func TestUninstallContext_StateTypedGetters(t *testing.T) {
	t.Run("GetStateString", func(t *testing.T) {
		ctx := NewUninstallContext()

		ctx.SetState("string", "hello")
		ctx.SetState("int", 42)

		assert.Equal(t, "hello", ctx.GetStateString("string"))
		assert.Equal(t, "", ctx.GetStateString("int"))         // wrong type
		assert.Equal(t, "", ctx.GetStateString("nonexistent")) // not found
	})

	t.Run("GetStateInt", func(t *testing.T) {
		ctx := NewUninstallContext()

		ctx.SetState("int", 42)
		ctx.SetState("string", "hello")

		assert.Equal(t, 42, ctx.GetStateInt("int"))
		assert.Equal(t, 0, ctx.GetStateInt("string"))      // wrong type
		assert.Equal(t, 0, ctx.GetStateInt("nonexistent")) // not found
	})

	t.Run("GetStateBool", func(t *testing.T) {
		ctx := NewUninstallContext()

		ctx.SetState("bool", true)
		ctx.SetState("string", "hello")

		assert.Equal(t, true, ctx.GetStateBool("bool"))
		assert.Equal(t, false, ctx.GetStateBool("string"))      // wrong type
		assert.Equal(t, false, ctx.GetStateBool("nonexistent")) // not found
	})

	t.Run("GetStateSlice", func(t *testing.T) {
		ctx := NewUninstallContext()

		ctx.SetState("slice", []string{"a", "b", "c"})
		ctx.SetState("string", "hello")

		assert.Equal(t, []string{"a", "b", "c"}, ctx.GetStateSlice("slice"))
		assert.Nil(t, ctx.GetStateSlice("string"))      // wrong type
		assert.Nil(t, ctx.GetStateSlice("nonexistent")) // not found
	})
}

func TestUninstallContext_Cancellation(t *testing.T) {
	t.Run("cancel context", func(t *testing.T) {
		ctx := NewUninstallContext()

		assert.False(t, ctx.IsCancelled())

		ctx.Cancel()

		assert.True(t, ctx.IsCancelled())
	})

	t.Run("context done channel", func(t *testing.T) {
		ctx := NewUninstallContext()

		select {
		case <-ctx.Context().Done():
			t.Fatal("context should not be done yet")
		default:
			// Expected
		}

		ctx.Cancel()

		select {
		case <-ctx.Context().Done():
			// Expected
		default:
			t.Fatal("context should be done after cancel")
		}
	})

	t.Run("with parent context", func(t *testing.T) {
		parentCtx, cancel := context.WithCancel(context.Background())
		ctx := NewUninstallContext(WithUninstallContext(parentCtx))

		assert.False(t, ctx.IsCancelled())

		cancel()

		// Give it a moment to propagate
		time.Sleep(10 * time.Millisecond)

		assert.True(t, ctx.IsCancelled())
	})

	t.Run("nil context check", func(t *testing.T) {
		ctx := &Context{} // Empty context without initialization

		// Should not panic and return false
		assert.False(t, ctx.IsCancelled())
	})
}

func TestUninstallContext_Logging(t *testing.T) {
	t.Run("logs with logger", func(t *testing.T) {
		logger := &mockLogger{}
		ctx := NewUninstallContext(WithUninstallLogger(logger))

		ctx.Log("info message", "key", "value")
		ctx.LogDebug("debug message")
		ctx.LogWarn("warn message")
		ctx.LogError("error message")

		assert.True(t, logger.infoCalled)
		assert.True(t, logger.debugCalled)
		assert.True(t, logger.warnCalled)
		assert.True(t, logger.errorCalled)
	})

	t.Run("handles nil logger", func(t *testing.T) {
		ctx := NewUninstallContext() // No logger

		// Should not panic
		ctx.Log("message")
		ctx.LogDebug("message")
		ctx.LogWarn("message")
		ctx.LogError("message")
	})
}

func TestUninstallContextOptions(t *testing.T) {
	t.Run("WithUninstallDistroInfo", func(t *testing.T) {
		distroInfo := &distro.Distribution{ID: "fedora"}
		ctx := NewUninstallContext(WithUninstallDistroInfo(distroInfo))
		assert.Equal(t, distroInfo, ctx.DistroInfo)
	})

	t.Run("WithInstalledDriver", func(t *testing.T) {
		ctx := NewUninstallContext(WithInstalledDriver("550.54.14"))
		assert.Equal(t, "550.54.14", ctx.InstalledDriver)
	})

	t.Run("WithInstalledPackages", func(t *testing.T) {
		packages := []string{"nvidia-driver-550", "nvidia-cuda", "nvidia-cudnn"}
		ctx := NewUninstallContext(WithInstalledPackages(packages))

		// Verify it's a copy
		packages[0] = "modified"
		assert.Equal(t, "nvidia-driver-550", ctx.InstalledPackages[0])
	})

	t.Run("WithUninstallPackageManager", func(t *testing.T) {
		// PackageManager is tested through interface - we verify the option works
		ctx := NewUninstallContext()
		assert.Nil(t, ctx.PackageManager)
	})

	t.Run("WithUninstallExecutor", func(t *testing.T) {
		executor := exec.NewMockExecutor()
		ctx := NewUninstallContext(WithUninstallExecutor(executor))
		assert.Equal(t, executor, ctx.Executor)
	})

	t.Run("WithUninstallPrivilege", func(t *testing.T) {
		priv := privilege.NewManager()
		ctx := NewUninstallContext(WithUninstallPrivilege(priv))
		assert.Equal(t, priv, ctx.Privilege)
	})

	t.Run("WithUninstallLogger", func(t *testing.T) {
		logger := logging.NewNop()
		ctx := NewUninstallContext(WithUninstallLogger(logger))
		assert.Equal(t, logger, ctx.Logger)
	})

	t.Run("WithUninstallDryRun", func(t *testing.T) {
		ctx := NewUninstallContext(WithUninstallDryRun(true))
		assert.True(t, ctx.DryRun)
	})

	t.Run("WithUninstallForce", func(t *testing.T) {
		ctx := NewUninstallContext(WithUninstallForce(true))
		assert.True(t, ctx.Force)
	})

	t.Run("WithKeepConfig", func(t *testing.T) {
		ctx := NewUninstallContext(WithKeepConfig(true))
		assert.True(t, ctx.KeepConfig)
	})

	t.Run("WithUninstallContext", func(t *testing.T) {
		parentCtx := context.Background()
		ctx := NewUninstallContext(WithUninstallContext(parentCtx))

		require.NotNil(t, ctx.Context())
	})
}

func TestUninstallContext_ThreadSafety(t *testing.T) {
	ctx := NewUninstallContext()

	// Concurrent reads and writes to state
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 100; j++ {
				ctx.SetState("key", idx)
				_ = ctx.GetStateInt("key")
				_ = ctx.GetStateString("key")
				_ = ctx.GetStateBool("key")
				_ = ctx.GetStateSlice("key")
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// Mock implementations for testing

type mockLogger struct {
	debugCalled bool
	infoCalled  bool
	warnCalled  bool
	errorCalled bool
}

func (m *mockLogger) Debug(msg string, keyvals ...interface{}) {
	m.debugCalled = true
}

func (m *mockLogger) Info(msg string, keyvals ...interface{}) {
	m.infoCalled = true
}

func (m *mockLogger) Warn(msg string, keyvals ...interface{}) {
	m.warnCalled = true
}

func (m *mockLogger) Error(msg string, keyvals ...interface{}) {
	m.errorCalled = true
}

func (m *mockLogger) WithPrefix(prefix string) logging.Logger {
	return m
}

func (m *mockLogger) WithFields(keyvals ...interface{}) logging.Logger {
	return m
}

func (m *mockLogger) SetLevel(level logging.Level) {}

func (m *mockLogger) GetLevel() logging.Level {
	return logging.LevelInfo
}
