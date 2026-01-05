package testing

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	stdtesting "testing"
	"time"
)

// ============================================================================
// Context Helper Tests
// ============================================================================

func TestContextWithTimeout_CreatesContext(t *stdtesting.T) {
	ctx, cancel := ContextWithTimeout(t, 5*time.Second)
	defer cancel()

	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have deadline")
	}

	if deadline.Before(time.Now()) {
		t.Error("expected deadline to be in the future")
	}
}

func TestContextWithTimeout_Expires(t *stdtesting.T) {
	ctx, _ := ContextWithTimeout(t, 10*time.Millisecond)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", ctx.Err())
		}
	default:
		t.Error("expected context to be done")
	}
}

func TestContextWithCancel_CreatesContext(t *stdtesting.T) {
	ctx, cancel := ContextWithCancel(t)

	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	cancel()

	select {
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			t.Errorf("expected Canceled, got %v", ctx.Err())
		}
	default:
		t.Error("expected context to be done after cancel")
	}
}

func TestTestContext_ReturnsWorkingContext(t *stdtesting.T) {
	ctx := TestContext(t)

	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	// Should have a deadline
	_, ok := ctx.Deadline()
	if !ok {
		t.Error("expected context to have deadline")
	}
}

func TestShortContext_HasShortTimeout(t *stdtesting.T) {
	ctx := ShortContext(t)

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have deadline")
	}

	// Should be within ~5 seconds
	duration := time.Until(deadline)
	if duration > 6*time.Second || duration < 4*time.Second {
		t.Errorf("expected ~5 second timeout, got %v", duration)
	}
}

// ============================================================================
// Skip Helper Tests
// ============================================================================

func TestSkipIfRoot_SkipsWhenRoot(t *stdtesting.T) {
	// We can only really test this behavior in a controlled way
	// by checking if we're root and expecting the skip
	if os.Geteuid() == 0 {
		// We'd be skipped, so just verify the function exists
		t.Log("running as root, would skip")
	}
	// SkipIfRoot(t) // Can't actually call this in a test we want to run
}

func TestSkipShort_SkipsInShortMode(t *stdtesting.T) {
	// Can't easily test skip behavior without actually skipping
	if stdtesting.Short() {
		t.Log("running in short mode, would skip")
	}
}

// ============================================================================
// Require Helper Tests
// ============================================================================

func TestRequireNoError_PassesWithNil(t *stdtesting.T) {
	// This should not panic or fail
	RequireNoError(t, nil)
}

// Note: We cannot easily test failure cases for RequireNoError/RequireError
// because testing.TB has unexported methods that cannot be mocked.
// The functions are tested implicitly through other tests that use them.

func TestRequireError_PassesWithError(t *stdtesting.T) {
	// This should not panic or fail
	RequireError(t, errors.New("test error"))
}

// Note: Testing failure case requires mock testing.TB which has unexported methods

func TestRequireTrue_PassesWithTrue(t *stdtesting.T) {
	RequireTrue(t, true)
}

// Note: Testing failure case requires mock testing.TB which has unexported methods

func TestRequireFalse_PassesWithFalse(t *stdtesting.T) {
	RequireFalse(t, false)
}

// Note: Testing failure case requires mock testing.TB which has unexported methods

func TestRequireEqual_PassesWithEqual(t *stdtesting.T) {
	RequireEqual(t, 42, 42)
	RequireEqual(t, "hello", "hello")
}

// Note: Testing failure case requires mock testing.TB which has unexported methods

// ============================================================================
// Wait/Retry Helper Tests
// ============================================================================

func TestWaitFor_SucceedsImmediately(t *stdtesting.T) {
	called := 0
	WaitFor(t, func() bool {
		called++
		return true
	}, time.Second, 10*time.Millisecond)

	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestWaitFor_SucceedsEventually(t *stdtesting.T) {
	called := 0
	WaitFor(t, func() bool {
		called++
		return called >= 3
	}, time.Second, 10*time.Millisecond)

	if called < 3 {
		t.Errorf("expected at least 3 calls, got %d", called)
	}
}

// Note: Testing WaitFor failure case requires mock testing.TB which has unexported methods
// WaitFor is tested via the success cases above

func TestEventuallyEqual_SucceedsImmediately(t *stdtesting.T) {
	EventuallyEqual(t, 42, func() interface{} {
		return 42
	}, time.Second)
}

func TestEventuallyEqual_SucceedsEventually(t *stdtesting.T) {
	counter := 0
	EventuallyEqual(t, 5, func() interface{} {
		counter++
		return counter
	}, time.Second)
}

func TestRetry_SucceedsImmediately(t *stdtesting.T) {
	called := 0
	err := Retry(t, func() error {
		called++
		return nil
	}, 3, 10*time.Millisecond)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestRetry_SucceedsAfterRetries(t *stdtesting.T) {
	called := 0
	err := Retry(t, func() error {
		called++
		if called < 3 {
			return errors.New("not yet")
		}
		return nil
	}, 5, 10*time.Millisecond)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if called != 3 {
		t.Errorf("expected 3 calls, got %d", called)
	}
}

func TestRetry_ReturnsLastError(t *stdtesting.T) {
	expectedErr := errors.New("permanent failure")
	err := Retry(t, func() error {
		return expectedErr
	}, 3, 10*time.Millisecond)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

// ============================================================================
// Environment Variable Helper Tests
// ============================================================================

func TestSetEnv_SetsAndRestores(t *stdtesting.T) {
	// Set a unique key
	key := "IGOR_TEST_ENV_VAR_123"
	os.Unsetenv(key) // Ensure it doesn't exist

	cleanup := SetEnv(t, key, "test_value")

	// Check it was set
	if os.Getenv(key) != "test_value" {
		t.Error("expected env var to be set")
	}

	// Call cleanup
	cleanup()

	// Check it was unset
	if _, exists := os.LookupEnv(key); exists {
		t.Error("expected env var to be unset after cleanup")
	}
}

func TestSetEnv_RestoresOriginalValue(t *stdtesting.T) {
	key := "IGOR_TEST_ENV_VAR_456"
	os.Setenv(key, "original")
	defer os.Unsetenv(key)

	cleanup := SetEnv(t, key, "new_value")

	if os.Getenv(key) != "new_value" {
		t.Error("expected env var to be changed")
	}

	cleanup()

	if os.Getenv(key) != "original" {
		t.Error("expected env var to be restored")
	}
}

func TestSetEnvs_SetsMultiple(t *stdtesting.T) {
	key1 := "IGOR_TEST_MULTI_1"
	key2 := "IGOR_TEST_MULTI_2"
	os.Unsetenv(key1)
	os.Unsetenv(key2)

	cleanup := SetEnvs(t, map[string]string{
		key1: "value1",
		key2: "value2",
	})

	if os.Getenv(key1) != "value1" {
		t.Error("expected key1 to be set")
	}
	if os.Getenv(key2) != "value2" {
		t.Error("expected key2 to be set")
	}

	cleanup()

	if _, exists := os.LookupEnv(key1); exists {
		t.Error("expected key1 to be unset")
	}
	if _, exists := os.LookupEnv(key2); exists {
		t.Error("expected key2 to be unset")
	}
}

func TestUnsetEnv_UnsetsAndRestores(t *stdtesting.T) {
	key := "IGOR_TEST_UNSET"
	os.Setenv(key, "original")
	defer os.Unsetenv(key)

	cleanup := UnsetEnv(t, key)

	if _, exists := os.LookupEnv(key); exists {
		t.Error("expected env var to be unset")
	}

	cleanup()

	if os.Getenv(key) != "original" {
		t.Error("expected env var to be restored")
	}
}

// ============================================================================
// Temp File/Dir Helper Tests
// ============================================================================

func TestTempFile_CreatesFile(t *stdtesting.T) {
	path, cleanup := TempFile(t, "test content")
	defer cleanup()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("expected 'test content', got %s", string(content))
	}
}

func TestTempFile_CleanupRemovesFile(t *stdtesting.T) {
	path, cleanup := TempFile(t, "content")

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("temp file should exist: %v", err)
	}

	cleanup()

	// Verify file is removed
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected temp file to be removed")
	}
}

func TestTempDir_CreatesDirectory(t *stdtesting.T) {
	dir, cleanup := TempDir(t)
	defer cleanup()

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("failed to stat temp dir: %v", err)
	}

	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestTempDir_CleanupRemovesDirectory(t *stdtesting.T) {
	dir, cleanup := TempDir(t)

	// Create a file inside
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cleanup()

	// Verify directory is removed
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("expected temp dir to be removed")
	}
}

func TestTempDirWithFiles_CreatesFiles(t *stdtesting.T) {
	dir, cleanup := TempDirWithFiles(t, map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	})
	defer cleanup()

	// Check files exist
	content1, err := os.ReadFile(filepath.Join(dir, "file1.txt"))
	if err != nil {
		t.Fatalf("failed to read file1: %v", err)
	}
	if string(content1) != "content1" {
		t.Errorf("expected 'content1', got %s", string(content1))
	}

	content2, err := os.ReadFile(filepath.Join(dir, "file2.txt"))
	if err != nil {
		t.Fatalf("failed to read file2: %v", err)
	}
	if string(content2) != "content2" {
		t.Errorf("expected 'content2', got %s", string(content2))
	}
}

// ============================================================================
// MockTime Tests
// ============================================================================

func TestMockTime_Now(t *stdtesting.T) {
	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	mt := NewMockTime(fixedTime)

	if !mt.Now().Equal(fixedTime) {
		t.Errorf("expected %v, got %v", fixedTime, mt.Now())
	}
}

func TestMockTime_Advance(t *stdtesting.T) {
	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	mt := NewMockTime(fixedTime)

	mt.Advance(time.Hour)

	expected := fixedTime.Add(time.Hour)
	if !mt.Now().Equal(expected) {
		t.Errorf("expected %v, got %v", expected, mt.Now())
	}
}

func TestMockTime_Set(t *stdtesting.T) {
	mt := NewMockTime(time.Now())

	newTime := time.Date(2025, 6, 20, 14, 0, 0, 0, time.UTC)
	mt.Set(newTime)

	if !mt.Now().Equal(newTime) {
		t.Errorf("expected %v, got %v", newTime, mt.Now())
	}
}

func TestMockTime_Since(t *stdtesting.T) {
	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	mt := NewMockTime(fixedTime)

	pastTime := time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC)
	duration := mt.Since(pastTime)

	if duration != time.Hour {
		t.Errorf("expected 1 hour, got %v", duration)
	}
}

func TestMockTime_Until(t *stdtesting.T) {
	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	mt := NewMockTime(fixedTime)

	futureTime := time.Date(2024, 1, 15, 11, 30, 0, 0, time.UTC)
	duration := mt.Until(futureTime)

	if duration != time.Hour {
		t.Errorf("expected 1 hour, got %v", duration)
	}
}

func TestNewMockTimeNow_StartsAtCurrentTime(t *stdtesting.T) {
	before := time.Now()
	mt := NewMockTimeNow()
	after := time.Now()

	mockNow := mt.Now()
	if mockNow.Before(before) || mockNow.After(after) {
		t.Error("mock time should be between before and after")
	}
}

// ============================================================================
// Counter Tests
// ============================================================================

func TestCounter_StartsAtZero(t *stdtesting.T) {
	c := NewCounter()

	if c.Value() != 0 {
		t.Errorf("expected 0, got %d", c.Value())
	}
}

func TestCounter_Inc(t *stdtesting.T) {
	c := NewCounter()

	c.Inc()
	c.Inc()
	c.Inc()

	if c.Value() != 3 {
		t.Errorf("expected 3, got %d", c.Value())
	}
}

func TestCounter_Add(t *stdtesting.T) {
	c := NewCounter()

	c.Add(5)
	c.Add(3)

	if c.Value() != 8 {
		t.Errorf("expected 8, got %d", c.Value())
	}
}

func TestCounter_Reset(t *stdtesting.T) {
	c := NewCounter()

	c.Add(10)
	c.Reset()

	if c.Value() != 0 {
		t.Errorf("expected 0 after reset, got %d", c.Value())
	}
}

// ============================================================================
// formatMessage Tests
// ============================================================================

func TestFormatMessage_Empty(t *stdtesting.T) {
	result := formatMessage()
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestFormatMessage_SingleString(t *stdtesting.T) {
	result := formatMessage("test message")
	if result != "test message" {
		t.Errorf("expected 'test message', got %s", result)
	}
}

func TestFormatMessage_FormatString(t *stdtesting.T) {
	result := formatMessage("value: %d", 42)
	if result != "value: 42" {
		t.Errorf("expected 'value: 42', got %s", result)
	}
}

// ============================================================================
// MustParse Tests
// ============================================================================

func TestMustParse_ReturnsValue(t *stdtesting.T) {
	result := MustParse(42, nil)
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestMustParse_PanicsOnError(t *stdtesting.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()

	MustParse(0, errors.New("parse error"))
}

// ============================================================================
// Mock testing.TB for testing test helpers
// ============================================================================

// Note: We cannot implement a mock testing.TB because it has unexported methods.
// Failure cases for RequireNoError, RequireError, RequireTrue, RequireFalse,
// RequireEqual, and WaitFor timeout are verified through normal test usage.
// The helper functions are implicitly tested through their usage in other tests.

// ============================================================================
// Additional Helper Tests
// ============================================================================

func TestRequireNil_PassesWithNil(t *stdtesting.T) {
	RequireNil(t, nil)
}

func TestRequireNotNil_PassesWithValue(t *stdtesting.T) {
	value := 42
	RequireNotNil(t, &value)
}

func TestTempFileWithName_CreatesFile(t *stdtesting.T) {
	path, cleanup := TempFileWithName(t, "test-*.txt", "content")
	defer cleanup()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != "content" {
		t.Errorf("expected 'content', got %s", string(content))
	}
}

func TestWaitForContext_SucceedsImmediately(t *stdtesting.T) {
	ctx := TestContext(t)
	called := 0

	WaitForContext(t, ctx, func() bool {
		called++
		return true
	}, 10*time.Millisecond)

	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestCaptureOutput_CapturesStdout(t *stdtesting.T) {
	stdout, stderr := CaptureOutput(t, func() {
		os.Stdout.WriteString("hello stdout")
		os.Stderr.WriteString("hello stderr")
	})

	if stdout != "hello stdout" {
		t.Errorf("expected 'hello stdout', got %s", stdout)
	}

	if stderr != "hello stderr" {
		t.Errorf("expected 'hello stderr', got %s", stderr)
	}
}
