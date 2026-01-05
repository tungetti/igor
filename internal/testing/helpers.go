package testing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"
)

// ============================================================================
// Context Helpers
// ============================================================================

// ContextWithTimeout creates a context with timeout for testing.
// The context is automatically cancelled when the test completes.
func ContextWithTimeout(t testing.TB, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	return ctx, cancel
}

// ContextWithCancel creates a cancellable context for testing.
// The context is automatically cancelled when the test completes.
func ContextWithCancel(t testing.TB) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return ctx, cancel
}

// TestContext creates a context suitable for most tests.
// Uses a default timeout of 30 seconds.
func TestContext(t testing.TB) context.Context {
	t.Helper()
	ctx, _ := ContextWithTimeout(t, 30*time.Second)
	return ctx
}

// ShortContext creates a context with a short timeout (5 seconds).
func ShortContext(t testing.TB) context.Context {
	t.Helper()
	ctx, _ := ContextWithTimeout(t, 5*time.Second)
	return ctx
}

// ============================================================================
// Skip Helpers
// ============================================================================

// SkipIfRoot skips the test if running as root.
func SkipIfRoot(t testing.TB) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("skipping test when running as root")
	}
}

// SkipIfNotRoot skips the test if not running as root.
func SkipIfNotRoot(t testing.TB) {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("skipping test: requires root privileges")
	}
}

// SkipShort skips the test in short mode.
func SkipShort(t testing.TB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
}

// SkipOnCI skips the test when running in CI environment.
func SkipOnCI(t testing.TB) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("skipping test in CI environment")
	}
}

// SkipOnOS skips the test on the specified operating system.
func SkipOnOS(t testing.TB, goos string) {
	t.Helper()
	if runtime.GOOS == goos {
		t.Skipf("skipping test on %s", goos)
	}
}

// SkipUnlessLinux skips the test unless running on Linux.
func SkipUnlessLinux(t testing.TB) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("skipping test: requires Linux")
	}
}

// ============================================================================
// Require Helpers - Fail immediately if condition not met
// ============================================================================

// RequireNoError fails the test immediately if err is not nil.
func RequireNoError(t testing.TB, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Fatalf("%s: unexpected error: %v", msg, err)
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

// RequireError fails the test immediately if err is nil.
func RequireError(t testing.TB, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Fatalf("%s: expected error but got nil", msg)
		} else {
			t.Fatal("expected error but got nil")
		}
	}
}

// RequireTrue fails the test immediately if condition is false.
func RequireTrue(t testing.TB, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	if !condition {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Fatalf("%s: expected true but got false", msg)
		} else {
			t.Fatal("expected true but got false")
		}
	}
}

// RequireFalse fails the test immediately if condition is true.
func RequireFalse(t testing.TB, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	if condition {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Fatalf("%s: expected false but got true", msg)
		} else {
			t.Fatal("expected false but got true")
		}
	}
}

// RequireNil fails the test immediately if v is not nil.
func RequireNil(t testing.TB, v interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if v != nil {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Fatalf("%s: expected nil but got %v", msg, v)
		} else {
			t.Fatalf("expected nil but got %v", v)
		}
	}
}

// RequireNotNil fails the test immediately if v is nil.
func RequireNotNil(t testing.TB, v interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if v == nil {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Fatalf("%s: expected non-nil value but got nil", msg)
		} else {
			t.Fatal("expected non-nil value but got nil")
		}
	}
}

// RequireEqual fails the test immediately if expected != actual.
func RequireEqual(t testing.TB, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if expected != actual {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Fatalf("%s: expected %v but got %v", msg, expected, actual)
		} else {
			t.Fatalf("expected %v but got %v", expected, actual)
		}
	}
}

// ============================================================================
// Wait/Retry Helpers
// ============================================================================

// WaitFor waits for a condition to become true with timeout.
// If the condition does not become true within the timeout, the test fails.
func WaitFor(t testing.TB, condition func() bool, timeout, interval time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("condition not met within %v", timeout)
}

// WaitForContext waits for a condition to become true with context.
func WaitForContext(t testing.TB, ctx context.Context, condition func() bool, interval time.Duration) {
	t.Helper()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("condition not met: %v", ctx.Err())
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// EventuallyEqual waits for actual to equal expected.
func EventuallyEqual(t testing.TB, expected interface{}, actual func() interface{}, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	interval := timeout / 20
	if interval < time.Millisecond {
		interval = time.Millisecond
	}

	for time.Now().Before(deadline) {
		if expected == actual() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("expected %v but got %v after %v", expected, actual(), timeout)
}

// Retry retries a function until it succeeds or max attempts is reached.
// Returns the error from the last attempt if all attempts fail.
func Retry(t testing.TB, fn func() error, maxAttempts int, delay time.Duration) error {
	t.Helper()

	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i < maxAttempts-1 {
				time.Sleep(delay)
			}
		} else {
			return nil
		}
	}
	return lastErr
}

// ============================================================================
// Output Capture Helpers
// ============================================================================

// CaptureOutput captures stdout and stderr during a function call.
func CaptureOutput(t testing.TB, fn func()) (stdout, stderr string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	rOut, wOut, err := os.Pipe()
	RequireNoError(t, err, "failed to create stdout pipe")

	rErr, wErr, err := os.Pipe()
	RequireNoError(t, err, "failed to create stderr pipe")

	os.Stdout = wOut
	os.Stderr = wErr

	outC := make(chan string)
	errC := make(chan string)

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rOut)
		outC <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rErr)
		errC <- buf.String()
	}()

	fn()

	wOut.Close()
	wErr.Close()

	stdout = <-outC
	stderr = <-errC

	return stdout, stderr
}

// ============================================================================
// Environment Variable Helpers
// ============================================================================

// SetEnv sets an environment variable and returns a cleanup function.
// The original value (or unset state) is restored when the cleanup function is called.
func SetEnv(t testing.TB, key, value string) func() {
	t.Helper()

	oldValue, existed := os.LookupEnv(key)

	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env var %s: %v", key, err)
	}

	return func() {
		if existed {
			os.Setenv(key, oldValue)
		} else {
			os.Unsetenv(key)
		}
	}
}

// SetEnvs sets multiple environment variables and returns a cleanup function.
func SetEnvs(t testing.TB, vars map[string]string) func() {
	t.Helper()

	cleanups := make([]func(), 0, len(vars))
	for key, value := range vars {
		cleanups = append(cleanups, SetEnv(t, key, value))
	}

	return func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}
}

// UnsetEnv unsets an environment variable and returns a cleanup function.
func UnsetEnv(t testing.TB, key string) func() {
	t.Helper()

	oldValue, existed := os.LookupEnv(key)

	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("failed to unset env var %s: %v", key, err)
	}

	return func() {
		if existed {
			os.Setenv(key, oldValue)
		}
	}
}

// ============================================================================
// Temporary File/Dir Helpers
// ============================================================================

// TempFile creates a temporary file with content.
// Returns the file path and a cleanup function.
func TempFile(t testing.TB, content string) (path string, cleanup func()) {
	t.Helper()

	f, err := os.CreateTemp("", "igor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		t.Fatalf("failed to write temp file: %v", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		t.Fatalf("failed to close temp file: %v", err)
	}

	return f.Name(), func() {
		os.Remove(f.Name())
	}
}

// TempFileWithName creates a temporary file with a specific name pattern.
func TempFileWithName(t testing.TB, pattern, content string) (path string, cleanup func()) {
	t.Helper()

	f, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		t.Fatalf("failed to write temp file: %v", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		t.Fatalf("failed to close temp file: %v", err)
	}

	return f.Name(), func() {
		os.Remove(f.Name())
	}
}

// TempDir creates a temporary directory.
// Returns the directory path and a cleanup function.
func TempDir(t testing.TB) (path string, cleanup func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "igor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	return dir, func() {
		os.RemoveAll(dir)
	}
}

// TempDirWithFiles creates a temporary directory with files.
func TempDirWithFiles(t testing.TB, files map[string]string) (path string, cleanup func()) {
	t.Helper()

	builder := NewTempDirBuilder()
	for name, content := range files {
		builder.WithFile(name, content)
	}

	return builder.Build(t)
}

// ============================================================================
// MockTime - Controllable time for testing
// ============================================================================

// MockTime provides controllable time for testing.
type MockTime struct {
	mu      sync.Mutex
	current time.Time
}

// NewMockTime creates a new MockTime starting at the given time.
func NewMockTime(t time.Time) *MockTime {
	return &MockTime{current: t}
}

// NewMockTimeNow creates a new MockTime starting at the current time.
func NewMockTimeNow() *MockTime {
	return &MockTime{current: time.Now()}
}

// Now returns the current mock time.
func (m *MockTime) Now() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

// Advance advances the mock time by the given duration.
func (m *MockTime) Advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = m.current.Add(d)
}

// Set sets the mock time to the given value.
func (m *MockTime) Set(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = t
}

// Since returns the duration since the given time.
func (m *MockTime) Since(t time.Time) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current.Sub(t)
}

// Until returns the duration until the given time.
func (m *MockTime) Until(t time.Time) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return t.Sub(m.current)
}

// ============================================================================
// Counter - Thread-safe counter for testing
// ============================================================================

// Counter is a thread-safe counter for testing.
type Counter struct {
	mu    sync.Mutex
	count int
}

// NewCounter creates a new counter starting at 0.
func NewCounter() *Counter {
	return &Counter{}
}

// Inc increments the counter by 1.
func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}

// Add adds n to the counter.
func (c *Counter) Add(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count += n
}

// Value returns the current counter value.
func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

// Reset resets the counter to 0.
func (c *Counter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count = 0
}

// ============================================================================
// Helper Functions
// ============================================================================

// formatMessage formats optional message arguments for error output.
func formatMessage(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 {
		return ""
	}

	if len(msgAndArgs) == 1 {
		if msg, ok := msgAndArgs[0].(string); ok {
			return msg
		}
		return fmt.Sprintf("%v", msgAndArgs[0])
	}

	if format, ok := msgAndArgs[0].(string); ok {
		return fmt.Sprintf(format, msgAndArgs[1:]...)
	}

	return fmt.Sprintf("%v", msgAndArgs)
}

// MustParse is a helper for parsing values in test setup.
// It panics if parsing fails, which is acceptable in test code.
func MustParse[T any](value T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("MustParse failed: %v", err))
	}
	return value
}
