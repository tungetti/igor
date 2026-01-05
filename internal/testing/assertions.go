package testing

import (
	"os"
	"strings"
	"testing"

	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/install"
	"github.com/tungetti/igor/internal/logging"
)

// ============================================================================
// Error Assertions
// ============================================================================

// AssertErrorCode checks if an error has a specific error code.
func AssertErrorCode(t testing.TB, err error, expectedCode errors.Code) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error with code %s, but got nil", expectedCode)
		return
	}

	actualCode := errors.GetCode(err)
	if actualCode != expectedCode {
		t.Errorf("expected error code %s, but got %s (error: %v)", expectedCode, actualCode, err)
	}
}

// AssertErrorContains checks if error message contains a substring.
func AssertErrorContains(t testing.TB, err error, substring string) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error containing %q, but got nil", substring)
		return
	}

	if !strings.Contains(err.Error(), substring) {
		t.Errorf("expected error to contain %q, but got: %v", substring, err)
	}
}

// AssertNoError asserts no error occurred (with better messages than testify).
func AssertNoError(t testing.TB, err error, msgAndArgs ...interface{}) {
	t.Helper()

	if err != nil {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: unexpected error: %v", msg, err)
		} else {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

// AssertError asserts an error occurred.
func AssertError(t testing.TB, err error, msgAndArgs ...interface{}) {
	t.Helper()

	if err == nil {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected error but got nil", msg)
		} else {
			t.Errorf("expected error but got nil")
		}
	}
}

// AssertErrorIs checks if error matches target using errors.Is.
func AssertErrorIs(t testing.TB, err, target error) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error matching %v, but got nil", target)
		return
	}

	if !errors.IsCode(err, errors.GetCode(target)) {
		t.Errorf("expected error matching %v, but got: %v", target, err)
	}
}

// ============================================================================
// Step Result Assertions
// ============================================================================

// AssertStepSuccess asserts a step result is successful.
func AssertStepSuccess(t testing.TB, result install.StepResult) {
	t.Helper()

	if result.Status != install.StepStatusCompleted {
		t.Errorf("expected step status %s, got %s (message: %s, error: %v)",
			install.StepStatusCompleted, result.Status, result.Message, result.Error)
	}
}

// AssertStepFailed asserts a step result is failed.
func AssertStepFailed(t testing.TB, result install.StepResult) {
	t.Helper()

	if result.Status != install.StepStatusFailed {
		t.Errorf("expected step status %s, got %s (message: %s)",
			install.StepStatusFailed, result.Status, result.Message)
	}
}

// AssertStepSkipped asserts a step result is skipped.
func AssertStepSkipped(t testing.TB, result install.StepResult) {
	t.Helper()

	if result.Status != install.StepStatusSkipped {
		t.Errorf("expected step status %s, got %s (message: %s)",
			install.StepStatusSkipped, result.Status, result.Message)
	}
}

// AssertStepStatus asserts a step result has a specific status.
func AssertStepStatus(t testing.TB, result install.StepResult, expected install.StepStatus) {
	t.Helper()

	if result.Status != expected {
		t.Errorf("expected step status %s, got %s (message: %s, error: %v)",
			expected, result.Status, result.Message, result.Error)
	}
}

// AssertStepMessage checks if the step result message contains a substring.
func AssertStepMessage(t testing.TB, result install.StepResult, substring string) {
	t.Helper()

	if !strings.Contains(result.Message, substring) {
		t.Errorf("expected step message to contain %q, got: %s", substring, result.Message)
	}
}

// ============================================================================
// Package Manager Assertions
// ============================================================================

// AssertPackageInstalled checks if a package was installed via the mock.
func AssertPackageInstalled(t testing.TB, pm *MockPackageManager, pkgName string) {
	t.Helper()

	if !pm.WasPackageInstalled(pkgName) {
		calls := pm.InstallCalls()
		t.Errorf("expected package %q to be installed, but it wasn't (install calls: %v)", pkgName, calls)
	}
}

// AssertPackageRemoved checks if a package was removed via the mock.
func AssertPackageRemoved(t testing.TB, pm *MockPackageManager, pkgName string) {
	t.Helper()

	if !pm.WasPackageRemoved(pkgName) {
		calls := pm.RemoveCalls()
		t.Errorf("expected package %q to be removed, but it wasn't (remove calls: %v)", pkgName, calls)
	}
}

// AssertPackageNotInstalled checks if a package was NOT installed.
func AssertPackageNotInstalled(t testing.TB, pm *MockPackageManager, pkgName string) {
	t.Helper()

	if pm.WasPackageInstalled(pkgName) {
		t.Errorf("expected package %q to NOT be installed, but it was", pkgName)
	}
}

// AssertPackageNotRemoved checks if a package was NOT removed.
func AssertPackageNotRemoved(t testing.TB, pm *MockPackageManager, pkgName string) {
	t.Helper()

	if pm.WasPackageRemoved(pkgName) {
		t.Errorf("expected package %q to NOT be removed, but it was", pkgName)
	}
}

// AssertInstallCallCount checks the number of install calls.
func AssertInstallCallCount(t testing.TB, pm *MockPackageManager, expected int) {
	t.Helper()

	actual := len(pm.InstallCalls())
	if actual != expected {
		t.Errorf("expected %d install calls, got %d", expected, actual)
	}
}

// AssertRemoveCallCount checks the number of remove calls.
func AssertRemoveCallCount(t testing.TB, pm *MockPackageManager, expected int) {
	t.Helper()

	actual := len(pm.RemoveCalls())
	if actual != expected {
		t.Errorf("expected %d remove calls, got %d", expected, actual)
	}
}

// ============================================================================
// Executor Assertions
// ============================================================================

// AssertCommandCalled checks if a command was called on the executor.
func AssertCommandCalled(t testing.TB, executor *exec.MockExecutor, cmd string) {
	t.Helper()

	if !executor.WasCalled(cmd) {
		calls := executor.Calls()
		var calledCmds []string
		for _, call := range calls {
			calledCmds = append(calledCmds, call.Command)
		}
		t.Errorf("expected command %q to be called, but it wasn't (called: %v)", cmd, calledCmds)
	}
}

// AssertCommandNotCalled checks that a command was NOT called.
func AssertCommandNotCalled(t testing.TB, executor *exec.MockExecutor, cmd string) {
	t.Helper()

	if executor.WasCalled(cmd) {
		t.Errorf("expected command %q to NOT be called, but it was", cmd)
	}
}

// AssertCommandCalledWith checks if a command was called with specific args.
func AssertCommandCalledWith(t testing.TB, executor *exec.MockExecutor, cmd string, args ...string) {
	t.Helper()

	if !executor.WasCalledWith(cmd, args...) {
		calls := executor.Calls()
		var callDetails []string
		for _, call := range calls {
			callDetails = append(callDetails, call.Command+" "+strings.Join(call.Args, " "))
		}
		t.Errorf("expected command %q with args %v to be called, but it wasn't (calls: %v)",
			cmd, args, callDetails)
	}
}

// AssertCallCount checks the number of calls to the executor.
func AssertCallCount(t testing.TB, executor *exec.MockExecutor, expected int) {
	t.Helper()

	actual := executor.CallCount()
	if actual != expected {
		t.Errorf("expected %d executor calls, got %d", expected, actual)
	}
}

// ============================================================================
// Logger Assertions
// ============================================================================

// AssertLogContains checks if the mock logger contains a message.
func AssertLogContains(t testing.TB, logger *MockLogger, substring string) {
	t.Helper()

	if !logger.ContainsMessage(substring) {
		messages := logger.Messages()
		var msgs []string
		for _, m := range messages {
			msgs = append(msgs, m.Message)
		}
		t.Errorf("expected log to contain %q, but it doesn't (messages: %v)", substring, msgs)
	}
}

// AssertLogNotContains checks if the mock logger does NOT contain a message.
func AssertLogNotContains(t testing.TB, logger *MockLogger, substring string) {
	t.Helper()

	if logger.ContainsMessage(substring) {
		t.Errorf("expected log to NOT contain %q, but it does", substring)
	}
}

// AssertLogLevel checks if a message was logged at a specific level.
func AssertLogLevel(t testing.TB, logger *MockLogger, level logging.Level, substring string) {
	t.Helper()

	if !logger.ContainsMessageAtLevel(level, substring) {
		messages := logger.MessagesAtLevel(level)
		var msgs []string
		for _, m := range messages {
			msgs = append(msgs, m.Message)
		}
		t.Errorf("expected log at level %s to contain %q, but it doesn't (messages: %v)",
			level, substring, msgs)
	}
}

// AssertLogEmpty checks if the mock logger has no messages.
func AssertLogEmpty(t testing.TB, logger *MockLogger) {
	t.Helper()

	if logger.MessageCount() > 0 {
		messages := logger.Messages()
		var msgs []string
		for _, m := range messages {
			msgs = append(msgs, m.Message)
		}
		t.Errorf("expected log to be empty, but it has %d messages: %v",
			logger.MessageCount(), msgs)
	}
}

// AssertLogCount checks the number of log messages.
func AssertLogCount(t testing.TB, logger *MockLogger, expected int) {
	t.Helper()

	actual := logger.MessageCount()
	if actual != expected {
		t.Errorf("expected %d log messages, got %d", expected, actual)
	}
}

// ============================================================================
// File System Assertions
// ============================================================================

// AssertFileExists checks if a file exists.
func AssertFileExists(t testing.TB, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file %q to exist, but it doesn't", path)
	}
}

// AssertFileNotExists checks if a file does NOT exist.
func AssertFileNotExists(t testing.TB, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected file %q to NOT exist, but it does", path)
	} else if !os.IsNotExist(err) {
		t.Errorf("unexpected error checking file %q: %v", path, err)
	}
}

// AssertFileContains checks if a file contains a substring.
func AssertFileContains(t testing.TB, path, substring string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("failed to read file %q: %v", path, err)
		return
	}

	if !strings.Contains(string(content), substring) {
		t.Errorf("expected file %q to contain %q, but it doesn't", path, substring)
	}
}

// AssertFileNotContains checks if a file does NOT contain a substring.
func AssertFileNotContains(t testing.TB, path, substring string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("failed to read file %q: %v", path, err)
		return
	}

	if strings.Contains(string(content), substring) {
		t.Errorf("expected file %q to NOT contain %q, but it does", path, substring)
	}
}

// AssertFileEquals checks if a file has exact content.
func AssertFileEquals(t testing.TB, path, expected string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("failed to read file %q: %v", path, err)
		return
	}

	if string(content) != expected {
		t.Errorf("file %q content mismatch:\nexpected:\n%s\ngot:\n%s", path, expected, string(content))
	}
}

// AssertDirExists checks if a directory exists.
func AssertDirExists(t testing.TB, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("expected directory %q to exist, but it doesn't", path)
		return
	}
	if err != nil {
		t.Errorf("error checking directory %q: %v", path, err)
		return
	}
	if !info.IsDir() {
		t.Errorf("expected %q to be a directory, but it's a file", path)
	}
}

// AssertDirNotExists checks if a directory does NOT exist.
func AssertDirNotExists(t testing.TB, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected directory %q to NOT exist, but it does", path)
	} else if !os.IsNotExist(err) {
		t.Errorf("unexpected error checking directory %q: %v", path, err)
	}
}

// ============================================================================
// Value Assertions
// ============================================================================

// AssertEqual checks if two values are equal.
func AssertEqual(t testing.TB, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	if expected != actual {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected %v, got %v", msg, expected, actual)
		} else {
			t.Errorf("expected %v, got %v", expected, actual)
		}
	}
}

// AssertNotEqual checks if two values are not equal.
func AssertNotEqual(t testing.TB, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	if expected == actual {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected values to be different, but both are %v", msg, expected)
		} else {
			t.Errorf("expected values to be different, but both are %v", expected)
		}
	}
}

// AssertTrue checks if condition is true.
func AssertTrue(t testing.TB, condition bool, msgAndArgs ...interface{}) {
	t.Helper()

	if !condition {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected true, got false", msg)
		} else {
			t.Errorf("expected true, got false")
		}
	}
}

// AssertFalse checks if condition is false.
func AssertFalse(t testing.TB, condition bool, msgAndArgs ...interface{}) {
	t.Helper()

	if condition {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected false, got true", msg)
		} else {
			t.Errorf("expected false, got true")
		}
	}
}

// AssertNil checks if value is nil.
func AssertNil(t testing.TB, v interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	if v != nil {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected nil, got %v", msg, v)
		} else {
			t.Errorf("expected nil, got %v", v)
		}
	}
}

// AssertNotNil checks if value is not nil.
func AssertNotNil(t testing.TB, v interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	if v == nil {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected non-nil value, got nil", msg)
		} else {
			t.Errorf("expected non-nil value, got nil")
		}
	}
}

// AssertLen checks if the length of a slice/string/map matches expected.
func AssertLen(t testing.TB, object interface{}, length int, msgAndArgs ...interface{}) {
	t.Helper()

	var actualLen int
	switch v := object.(type) {
	case string:
		actualLen = len(v)
	case []interface{}:
		actualLen = len(v)
	case []string:
		actualLen = len(v)
	case []int:
		actualLen = len(v)
	case map[string]interface{}:
		actualLen = len(v)
	case map[string]string:
		actualLen = len(v)
	default:
		t.Errorf("AssertLen: unsupported type %T", object)
		return
	}

	if actualLen != length {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected length %d, got %d", msg, length, actualLen)
		} else {
			t.Errorf("expected length %d, got %d", length, actualLen)
		}
	}
}

// AssertContains checks if a string contains a substring.
func AssertContains(t testing.TB, s, substring string, msgAndArgs ...interface{}) {
	t.Helper()

	if !strings.Contains(s, substring) {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected %q to contain %q", msg, s, substring)
		} else {
			t.Errorf("expected %q to contain %q", s, substring)
		}
	}
}

// AssertNotContains checks if a string does not contain a substring.
func AssertNotContains(t testing.TB, s, substring string, msgAndArgs ...interface{}) {
	t.Helper()

	if strings.Contains(s, substring) {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected %q to NOT contain %q", msg, s, substring)
		} else {
			t.Errorf("expected %q to NOT contain %q", s, substring)
		}
	}
}

// AssertEmpty checks if a string/slice/map is empty.
func AssertEmpty(t testing.TB, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	isEmpty := false
	switch v := object.(type) {
	case string:
		isEmpty = v == ""
	case []interface{}:
		isEmpty = len(v) == 0
	case []string:
		isEmpty = len(v) == 0
	case map[string]interface{}:
		isEmpty = len(v) == 0
	case nil:
		isEmpty = true
	default:
		t.Errorf("AssertEmpty: unsupported type %T", object)
		return
	}

	if !isEmpty {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected empty, got %v", msg, object)
		} else {
			t.Errorf("expected empty, got %v", object)
		}
	}
}

// AssertNotEmpty checks if a string/slice/map is not empty.
func AssertNotEmpty(t testing.TB, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	isEmpty := false
	switch v := object.(type) {
	case string:
		isEmpty = v == ""
	case []interface{}:
		isEmpty = len(v) == 0
	case []string:
		isEmpty = len(v) == 0
	case map[string]interface{}:
		isEmpty = len(v) == 0
	case nil:
		isEmpty = true
	default:
		t.Errorf("AssertNotEmpty: unsupported type %T", object)
		return
	}

	if isEmpty {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected non-empty, got empty", msg)
		} else {
			t.Errorf("expected non-empty, got empty")
		}
	}
}
