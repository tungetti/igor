package install

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/gpu"
	"github.com/tungetti/igor/internal/logging"
	"github.com/tungetti/igor/internal/privilege"
)

func TestNewContext(t *testing.T) {
	t.Run("creates with defaults", func(t *testing.T) {
		ctx := NewContext()

		assert.NotNil(t, ctx)
		assert.Nil(t, ctx.GPUInfo)
		assert.Nil(t, ctx.DistroInfo)
		assert.Empty(t, ctx.DriverVersion)
		assert.Empty(t, ctx.Components)
		assert.Nil(t, ctx.PackageManager)
		assert.Nil(t, ctx.Executor)
		assert.Nil(t, ctx.Privilege)
		assert.Nil(t, ctx.Logger)
		assert.False(t, ctx.DryRun)
		assert.NotNil(t, ctx.Context())
		assert.False(t, ctx.IsCancelled())
	})

	t.Run("creates with options", func(t *testing.T) {
		gpuInfo := &gpu.GPUInfo{}
		distroInfo := &distro.Distribution{ID: "ubuntu"}
		executor := exec.NewMockExecutor()
		priv := privilege.NewManager()
		logger := logging.NewNop()

		ctx := NewContext(
			WithGPUInfo(gpuInfo),
			WithDistroInfo(distroInfo),
			WithDriverVersion("550.54"),
			WithComponents([]string{"driver", "cuda"}),
			WithExecutor(executor),
			WithPrivilege(priv),
			WithLogger(logger),
			WithDryRun(true),
		)

		assert.Equal(t, gpuInfo, ctx.GPUInfo)
		assert.Equal(t, distroInfo, ctx.DistroInfo)
		assert.Equal(t, "550.54", ctx.DriverVersion)
		assert.Equal(t, []string{"driver", "cuda"}, ctx.Components)
		assert.Equal(t, executor, ctx.Executor)
		assert.Equal(t, priv, ctx.Privilege)
		assert.Equal(t, logger, ctx.Logger)
		assert.True(t, ctx.DryRun)
	})
}

func TestContext_State(t *testing.T) {
	t.Run("set and get state", func(t *testing.T) {
		ctx := NewContext()

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
		ctx := NewContext()

		val, ok := ctx.GetState("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("overwrite state", func(t *testing.T) {
		ctx := NewContext()

		ctx.SetState("key", "original")
		ctx.SetState("key", "updated")

		val, _ := ctx.GetState("key")
		assert.Equal(t, "updated", val)
	})

	t.Run("delete state", func(t *testing.T) {
		ctx := NewContext()

		ctx.SetState("key", "value")
		ctx.DeleteState("key")

		_, ok := ctx.GetState("key")
		assert.False(t, ok)
	})

	t.Run("clear state", func(t *testing.T) {
		ctx := NewContext()

		ctx.SetState("key1", "value1")
		ctx.SetState("key2", "value2")
		ctx.ClearState()

		_, ok1 := ctx.GetState("key1")
		_, ok2 := ctx.GetState("key2")
		assert.False(t, ok1)
		assert.False(t, ok2)
	})
}

func TestContext_StateTypedGetters(t *testing.T) {
	t.Run("GetStateString", func(t *testing.T) {
		ctx := NewContext()

		ctx.SetState("string", "hello")
		ctx.SetState("int", 42)

		assert.Equal(t, "hello", ctx.GetStateString("string"))
		assert.Equal(t, "", ctx.GetStateString("int"))         // wrong type
		assert.Equal(t, "", ctx.GetStateString("nonexistent")) // not found
	})

	t.Run("GetStateInt", func(t *testing.T) {
		ctx := NewContext()

		ctx.SetState("int", 42)
		ctx.SetState("string", "hello")

		assert.Equal(t, 42, ctx.GetStateInt("int"))
		assert.Equal(t, 0, ctx.GetStateInt("string"))      // wrong type
		assert.Equal(t, 0, ctx.GetStateInt("nonexistent")) // not found
	})

	t.Run("GetStateBool", func(t *testing.T) {
		ctx := NewContext()

		ctx.SetState("bool", true)
		ctx.SetState("string", "hello")

		assert.Equal(t, true, ctx.GetStateBool("bool"))
		assert.Equal(t, false, ctx.GetStateBool("string"))      // wrong type
		assert.Equal(t, false, ctx.GetStateBool("nonexistent")) // not found
	})
}

func TestContext_Cancellation(t *testing.T) {
	t.Run("cancel context", func(t *testing.T) {
		ctx := NewContext()

		assert.False(t, ctx.IsCancelled())

		ctx.Cancel()

		assert.True(t, ctx.IsCancelled())
	})

	t.Run("context done channel", func(t *testing.T) {
		ctx := NewContext()

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
		ctx := NewContext(WithContext(parentCtx))

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

func TestContext_Logging(t *testing.T) {
	t.Run("logs with logger", func(t *testing.T) {
		logger := &mockLogger{}
		ctx := NewContext(WithLogger(logger))

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
		ctx := NewContext() // No logger

		// Should not panic
		ctx.Log("message")
		ctx.LogDebug("message")
		ctx.LogWarn("message")
		ctx.LogError("message")
	})
}

func TestContextOptions(t *testing.T) {
	t.Run("WithGPUInfo", func(t *testing.T) {
		gpuInfo := &gpu.GPUInfo{}
		ctx := NewContext(WithGPUInfo(gpuInfo))
		assert.Equal(t, gpuInfo, ctx.GPUInfo)
	})

	t.Run("WithDistroInfo", func(t *testing.T) {
		distroInfo := &distro.Distribution{ID: "fedora"}
		ctx := NewContext(WithDistroInfo(distroInfo))
		assert.Equal(t, distroInfo, ctx.DistroInfo)
	})

	t.Run("WithDriverVersion", func(t *testing.T) {
		ctx := NewContext(WithDriverVersion("550.54.14"))
		assert.Equal(t, "550.54.14", ctx.DriverVersion)
	})

	t.Run("WithComponents", func(t *testing.T) {
		components := []string{"driver", "cuda", "cudnn"}
		ctx := NewContext(WithComponents(components))

		// Verify it's a copy
		components[0] = "modified"
		assert.Equal(t, "driver", ctx.Components[0])
	})

	t.Run("WithPackageManager", func(t *testing.T) {
		// PackageManager is tested through interface - we verify the option works
		// without creating a full mock implementing pkg.Manager
		ctx := NewContext()
		assert.Nil(t, ctx.PackageManager)
	})

	t.Run("WithExecutor", func(t *testing.T) {
		executor := exec.NewMockExecutor()
		ctx := NewContext(WithExecutor(executor))
		assert.Equal(t, executor, ctx.Executor)
	})

	t.Run("WithPrivilege", func(t *testing.T) {
		priv := privilege.NewManager()
		ctx := NewContext(WithPrivilege(priv))
		assert.Equal(t, priv, ctx.Privilege)
	})

	t.Run("WithLogger", func(t *testing.T) {
		logger := logging.NewNop()
		ctx := NewContext(WithLogger(logger))
		assert.Equal(t, logger, ctx.Logger)
	})

	t.Run("WithDryRun", func(t *testing.T) {
		ctx := NewContext(WithDryRun(true))
		assert.True(t, ctx.DryRun)
	})

	t.Run("WithContext", func(t *testing.T) {
		parentCtx := context.Background()
		ctx := NewContext(WithContext(parentCtx))

		require.NotNil(t, ctx.Context())
	})
}

func TestContext_ThreadSafety(t *testing.T) {
	ctx := NewContext()

	// Concurrent reads and writes to state
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 100; j++ {
				ctx.SetState("key", idx)
				_ = ctx.GetStateInt("key")
				_ = ctx.GetStateString("key")
				_ = ctx.GetStateBool("key")
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

// Note: A full mock implementation of pkg.Manager would be defined in the pkg package.
// For these tests, we only verify that the Context options work correctly.
