// Package testing provides centralized test infrastructure for the Igor project.
// It includes mock implementations, fixtures, helpers, and custom assertions
// that can be used across all test packages.
package testing

import (
	"context"
	"strings"
	"sync"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/logging"
	"github.com/tungetti/igor/internal/pkg"
)

// ============================================================================
// MockLogger - Implements logging.Logger for testing
// ============================================================================

// LogMessage represents a recorded log message.
type LogMessage struct {
	Level   logging.Level
	Message string
	Fields  []interface{}
}

// MockLogger implements logging.Logger for testing purposes.
// It records all log messages for later inspection.
type MockLogger struct {
	mu       sync.Mutex
	messages []LogMessage
	level    logging.Level
	prefix   string
	fields   []interface{}
}

// NewMockLogger creates a new MockLogger with default settings.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		messages: make([]LogMessage, 0),
		level:    logging.LevelDebug,
	}
}

// Debug logs a debug message.
func (m *MockLogger) Debug(msg string, keyvals ...interface{}) {
	m.record(logging.LevelDebug, msg, keyvals)
}

// Info logs an info message.
func (m *MockLogger) Info(msg string, keyvals ...interface{}) {
	m.record(logging.LevelInfo, msg, keyvals)
}

// Warn logs a warning message.
func (m *MockLogger) Warn(msg string, keyvals ...interface{}) {
	m.record(logging.LevelWarn, msg, keyvals)
}

// Error logs an error message.
func (m *MockLogger) Error(msg string, keyvals ...interface{}) {
	m.record(logging.LevelError, msg, keyvals)
}

// WithPrefix returns a new Logger with the given prefix.
// The new logger shares the message storage with the parent.
func (m *MockLogger) WithPrefix(prefix string) logging.Logger {
	m.mu.Lock()
	defer m.mu.Unlock()

	return &childMockLogger{
		parent: m,
		prefix: prefix,
		fields: append([]interface{}{}, m.fields...),
	}
}

// WithFields returns a new Logger with the given fields added to all messages.
// The new logger shares the message storage with the parent.
func (m *MockLogger) WithFields(keyvals ...interface{}) logging.Logger {
	m.mu.Lock()
	defer m.mu.Unlock()

	newFields := make([]interface{}, len(m.fields)+len(keyvals))
	copy(newFields, m.fields)
	copy(newFields[len(m.fields):], keyvals)

	return &childMockLogger{
		parent: m,
		prefix: m.prefix,
		fields: newFields,
	}
}

// childMockLogger is a logger that shares message storage with a parent MockLogger.
type childMockLogger struct {
	parent *MockLogger
	prefix string
	fields []interface{}
}

func (c *childMockLogger) Debug(msg string, keyvals ...interface{}) {
	c.parent.recordFromChild(logging.LevelDebug, c.prefix, msg, c.fields, keyvals)
}

func (c *childMockLogger) Info(msg string, keyvals ...interface{}) {
	c.parent.recordFromChild(logging.LevelInfo, c.prefix, msg, c.fields, keyvals)
}

func (c *childMockLogger) Warn(msg string, keyvals ...interface{}) {
	c.parent.recordFromChild(logging.LevelWarn, c.prefix, msg, c.fields, keyvals)
}

func (c *childMockLogger) Error(msg string, keyvals ...interface{}) {
	c.parent.recordFromChild(logging.LevelError, c.prefix, msg, c.fields, keyvals)
}

func (c *childMockLogger) WithPrefix(prefix string) logging.Logger {
	return &childMockLogger{
		parent: c.parent,
		prefix: prefix,
		fields: append([]interface{}{}, c.fields...),
	}
}

func (c *childMockLogger) WithFields(keyvals ...interface{}) logging.Logger {
	newFields := make([]interface{}, len(c.fields)+len(keyvals))
	copy(newFields, c.fields)
	copy(newFields[len(c.fields):], keyvals)

	return &childMockLogger{
		parent: c.parent,
		prefix: c.prefix,
		fields: newFields,
	}
}

func (c *childMockLogger) SetLevel(level logging.Level) {
	c.parent.SetLevel(level)
}

func (c *childMockLogger) GetLevel() logging.Level {
	return c.parent.GetLevel()
}

// recordFromChild records a message from a child logger.
func (m *MockLogger) recordFromChild(level logging.Level, prefix, msg string, persistentFields, keyvals []interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if level < m.level {
		return
	}

	allFields := append([]interface{}{}, persistentFields...)
	allFields = append(allFields, keyvals...)

	fullMsg := msg
	if prefix != "" {
		fullMsg = prefix + ": " + msg
	}

	m.messages = append(m.messages, LogMessage{
		Level:   level,
		Message: fullMsg,
		Fields:  allFields,
	})
}

// SetLevel sets the minimum log level.
func (m *MockLogger) SetLevel(level logging.Level) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.level = level
}

// GetLevel returns the current log level.
func (m *MockLogger) GetLevel() logging.Level {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.level
}

// record stores a log message.
func (m *MockLogger) record(level logging.Level, msg string, keyvals []interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if level < m.level {
		return
	}

	allFields := append([]interface{}{}, m.fields...)
	allFields = append(allFields, keyvals...)

	fullMsg := msg
	if m.prefix != "" {
		fullMsg = m.prefix + ": " + msg
	}

	m.messages = append(m.messages, LogMessage{
		Level:   level,
		Message: fullMsg,
		Fields:  allFields,
	})
}

// Messages returns all recorded log messages.
func (m *MockLogger) Messages() []LogMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]LogMessage{}, m.messages...)
}

// MessagesAtLevel returns all messages at a specific log level.
func (m *MockLogger) MessagesAtLevel(level logging.Level) []LogMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []LogMessage
	for _, msg := range m.messages {
		if msg.Level == level {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// Clear removes all recorded messages.
func (m *MockLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = m.messages[:0]
}

// ContainsMessage checks if any recorded message contains the given substring.
func (m *MockLogger) ContainsMessage(substring string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, msg := range m.messages {
		if strings.Contains(msg.Message, substring) {
			return true
		}
	}
	return false
}

// ContainsMessageAtLevel checks if any message at the given level contains the substring.
func (m *MockLogger) ContainsMessageAtLevel(level logging.Level, substring string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, msg := range m.messages {
		if msg.Level == level && strings.Contains(msg.Message, substring) {
			return true
		}
	}
	return false
}

// MessageCount returns the total number of recorded messages.
func (m *MockLogger) MessageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

// Ensure MockLogger implements logging.Logger.
var _ logging.Logger = (*MockLogger)(nil)

// ============================================================================
// MockPackageManager - Implements pkg.Manager for testing
// ============================================================================

// MockPackageManager implements pkg.Manager for testing purposes.
type MockPackageManager struct {
	mu            sync.Mutex
	installedPkgs []pkg.Package
	availablePkgs []pkg.Package
	repositories  []pkg.Repository
	installError  error
	removeError   error
	updateError   error
	upgradeError  error
	searchError   error
	installCalls  [][]string
	removeCalls   [][]string
	updateCalls   int
	upgradeCalls  [][]string
	name          string
	family        constants.DistroFamily
	isAvailable   bool
}

// NewMockPackageManager creates a new MockPackageManager with default settings.
func NewMockPackageManager() *MockPackageManager {
	return &MockPackageManager{
		installedPkgs: make([]pkg.Package, 0),
		availablePkgs: make([]pkg.Package, 0),
		repositories:  make([]pkg.Repository, 0),
		installCalls:  make([][]string, 0),
		removeCalls:   make([][]string, 0),
		upgradeCalls:  make([][]string, 0),
		name:          "mock",
		family:        constants.FamilyDebian,
		isAvailable:   true,
	}
}

// Install installs one or more packages.
func (m *MockPackageManager) Install(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.installCalls = append(m.installCalls, packages)

	if m.installError != nil {
		return m.installError
	}

	// Mark packages as installed
	for _, name := range packages {
		found := false
		for i := range m.installedPkgs {
			if m.installedPkgs[i].Name == name {
				m.installedPkgs[i].Installed = true
				found = true
				break
			}
		}
		if !found {
			m.installedPkgs = append(m.installedPkgs, pkg.Package{
				Name:      name,
				Installed: true,
			})
		}
	}

	return nil
}

// Remove removes one or more packages.
func (m *MockPackageManager) Remove(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.removeCalls = append(m.removeCalls, packages)

	if m.removeError != nil {
		return m.removeError
	}

	// Mark packages as not installed
	for _, name := range packages {
		for i := range m.installedPkgs {
			if m.installedPkgs[i].Name == name {
				m.installedPkgs[i].Installed = false
				break
			}
		}
	}

	return nil
}

// Update updates the package database.
func (m *MockPackageManager) Update(ctx context.Context, opts pkg.UpdateOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.updateCalls++

	if m.updateError != nil {
		return m.updateError
	}

	return nil
}

// Upgrade upgrades installed packages.
func (m *MockPackageManager) Upgrade(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.upgradeCalls = append(m.upgradeCalls, packages)

	if m.upgradeError != nil {
		return m.upgradeError
	}

	return nil
}

// IsInstalled checks if a package is currently installed.
func (m *MockPackageManager) IsInstalled(ctx context.Context, pkgName string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	for _, p := range m.installedPkgs {
		if p.Name == pkgName && p.Installed {
			return true, nil
		}
	}
	return false, nil
}

// Search searches for packages matching the query.
func (m *MockPackageManager) Search(ctx context.Context, query string, opts pkg.SearchOptions) ([]pkg.Package, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if m.searchError != nil {
		return nil, m.searchError
	}

	var results []pkg.Package
	for _, p := range m.availablePkgs {
		if strings.Contains(p.Name, query) {
			results = append(results, p)
		}
	}
	return results, nil
}

// Info returns detailed information about a specific package.
func (m *MockPackageManager) Info(ctx context.Context, pkgName string) (*pkg.Package, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check installed first
	for _, p := range m.installedPkgs {
		if p.Name == pkgName {
			return &p, nil
		}
	}

	// Check available
	for _, p := range m.availablePkgs {
		if p.Name == pkgName {
			return &p, nil
		}
	}

	return nil, pkg.ErrPackageNotFound
}

// ListInstalled returns a list of all installed packages.
func (m *MockPackageManager) ListInstalled(ctx context.Context) ([]pkg.Package, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var installed []pkg.Package
	for _, p := range m.installedPkgs {
		if p.Installed {
			installed = append(installed, p)
		}
	}
	return installed, nil
}

// ListUpgradable returns a list of packages that can be upgraded.
func (m *MockPackageManager) ListUpgradable(ctx context.Context) ([]pkg.Package, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Return empty slice for mock
	return []pkg.Package{}, nil
}

// AddRepository adds a new package repository.
func (m *MockPackageManager) AddRepository(ctx context.Context, repo pkg.Repository) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.repositories = append(m.repositories, repo)
	return nil
}

// RemoveRepository removes a package repository.
func (m *MockPackageManager) RemoveRepository(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	for i, r := range m.repositories {
		if r.Name == name {
			m.repositories = append(m.repositories[:i], m.repositories[i+1:]...)
			return nil
		}
	}
	return pkg.ErrRepositoryNotFound
}

// ListRepositories returns a list of configured repositories.
func (m *MockPackageManager) ListRepositories(ctx context.Context) ([]pkg.Repository, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return append([]pkg.Repository{}, m.repositories...), nil
}

// EnableRepository enables a disabled repository.
func (m *MockPackageManager) EnableRepository(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	for i := range m.repositories {
		if m.repositories[i].Name == name {
			m.repositories[i].Enabled = true
			return nil
		}
	}
	return pkg.ErrRepositoryNotFound
}

// DisableRepository disables an enabled repository.
func (m *MockPackageManager) DisableRepository(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	for i := range m.repositories {
		if m.repositories[i].Name == name {
			m.repositories[i].Enabled = false
			return nil
		}
	}
	return pkg.ErrRepositoryNotFound
}

// RefreshRepositories refreshes all repository metadata.
func (m *MockPackageManager) RefreshRepositories(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return nil
}

// Clean removes cached package files.
func (m *MockPackageManager) Clean(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return nil
}

// AutoRemove removes automatically installed packages that are no longer needed.
func (m *MockPackageManager) AutoRemove(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return nil
}

// Verify verifies the integrity of an installed package.
func (m *MockPackageManager) Verify(ctx context.Context, pkgName string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	for _, p := range m.installedPkgs {
		if p.Name == pkgName && p.Installed {
			return true, nil
		}
	}
	return false, pkg.ErrPackageNotInstalled
}

// Name returns the package manager name.
func (m *MockPackageManager) Name() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.name
}

// Family returns the distribution family.
func (m *MockPackageManager) Family() constants.DistroFamily {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.family
}

// IsAvailable checks if this package manager is available.
func (m *MockPackageManager) IsAvailable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isAvailable
}

// ============================================================================
// MockPackageManager Helper Methods
// ============================================================================

// SetInstalledPackages sets the list of installed packages.
func (m *MockPackageManager) SetInstalledPackages(pkgs []pkg.Package) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installedPkgs = pkgs
}

// SetAvailablePackages sets the list of available packages.
func (m *MockPackageManager) SetAvailablePackages(pkgs []pkg.Package) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.availablePkgs = pkgs
}

// SetInstallError sets the error to return on install.
func (m *MockPackageManager) SetInstallError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installError = err
}

// SetRemoveError sets the error to return on remove.
func (m *MockPackageManager) SetRemoveError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeError = err
}

// SetUpdateError sets the error to return on update.
func (m *MockPackageManager) SetUpdateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateError = err
}

// SetUpgradeError sets the error to return on upgrade.
func (m *MockPackageManager) SetUpgradeError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upgradeError = err
}

// SetSearchError sets the error to return on search.
func (m *MockPackageManager) SetSearchError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchError = err
}

// SetName sets the package manager name.
func (m *MockPackageManager) SetName(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.name = name
}

// SetFamily sets the distribution family.
func (m *MockPackageManager) SetFamily(family constants.DistroFamily) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.family = family
}

// SetAvailable sets whether the package manager is available.
func (m *MockPackageManager) SetAvailable(available bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isAvailable = available
}

// InstallCalls returns all install call arguments.
func (m *MockPackageManager) InstallCalls() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([][]string{}, m.installCalls...)
}

// RemoveCalls returns all remove call arguments.
func (m *MockPackageManager) RemoveCalls() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([][]string{}, m.removeCalls...)
}

// UpdateCalls returns the number of update calls.
func (m *MockPackageManager) UpdateCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateCalls
}

// UpgradeCalls returns all upgrade call arguments.
func (m *MockPackageManager) UpgradeCalls() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([][]string{}, m.upgradeCalls...)
}

// Reset clears all recorded calls.
func (m *MockPackageManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installCalls = m.installCalls[:0]
	m.removeCalls = m.removeCalls[:0]
	m.updateCalls = 0
	m.upgradeCalls = m.upgradeCalls[:0]
}

// ResetAll clears all recorded calls and resets errors.
func (m *MockPackageManager) ResetAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installCalls = m.installCalls[:0]
	m.removeCalls = m.removeCalls[:0]
	m.updateCalls = 0
	m.upgradeCalls = m.upgradeCalls[:0]
	m.installError = nil
	m.removeError = nil
	m.updateError = nil
	m.upgradeError = nil
	m.searchError = nil
}

// WasPackageInstalled checks if a specific package was installed.
func (m *MockPackageManager) WasPackageInstalled(pkgName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, call := range m.installCalls {
		for _, name := range call {
			if name == pkgName {
				return true
			}
		}
	}
	return false
}

// WasPackageRemoved checks if a specific package was removed.
func (m *MockPackageManager) WasPackageRemoved(pkgName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, call := range m.removeCalls {
		for _, name := range call {
			if name == pkgName {
				return true
			}
		}
	}
	return false
}

// Ensure MockPackageManager implements pkg.Manager.
var _ pkg.Manager = (*MockPackageManager)(nil)

// ============================================================================
// MockDistroDetector - For testing distro detection
// ============================================================================

// MockDistroDetector provides controlled distribution detection for testing.
type MockDistroDetector struct {
	mu           sync.Mutex
	distribution *distro.Distribution
	detectError  error
}

// NewMockDistroDetector creates a new MockDistroDetector with the given distribution.
func NewMockDistroDetector(d *distro.Distribution) *MockDistroDetector {
	return &MockDistroDetector{
		distribution: d,
	}
}

// Detect returns the configured distribution or error.
func (m *MockDistroDetector) Detect() (*distro.Distribution, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.detectError != nil {
		return nil, m.detectError
	}

	if m.distribution == nil {
		return nil, nil
	}

	return m.distribution, nil
}

// SetDistribution sets the distribution to return.
func (m *MockDistroDetector) SetDistribution(d *distro.Distribution) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.distribution = d
}

// SetError sets the error to return on Detect.
func (m *MockDistroDetector) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.detectError = err
}

// ============================================================================
// ExecutorBuilder - Fluent API for building MockExecutor
// ============================================================================

// ExecutorBuilder provides a fluent API for creating configured MockExecutor instances.
type ExecutorBuilder struct {
	executor *exec.MockExecutor
}

// NewExecutorBuilder creates a new ExecutorBuilder.
func NewExecutorBuilder() *ExecutorBuilder {
	return &ExecutorBuilder{
		executor: exec.NewMockExecutor(),
	}
}

// WithCommand configures a response for a specific command.
func (b *ExecutorBuilder) WithCommand(cmd string, stdout, stderr string, exitCode int) *ExecutorBuilder {
	result := &exec.Result{
		Stdout:   []byte(stdout),
		Stderr:   []byte(stderr),
		ExitCode: exitCode,
	}
	b.executor.SetResponse(cmd, result)
	return b
}

// WithCommandError configures an error response for a specific command.
func (b *ExecutorBuilder) WithCommandError(cmd string, err error) *ExecutorBuilder {
	result := exec.ErrorResult(err)
	b.executor.SetResponse(cmd, result)
	return b
}

// WithCommandSuccess configures a successful response for a specific command.
func (b *ExecutorBuilder) WithCommandSuccess(cmd string, stdout string) *ExecutorBuilder {
	result := exec.SuccessResult(stdout)
	b.executor.SetResponse(cmd, result)
	return b
}

// WithCommandFailure configures a failure response for a specific command.
func (b *ExecutorBuilder) WithCommandFailure(cmd string, exitCode int, stderr string) *ExecutorBuilder {
	result := exec.FailureResult(exitCode, stderr)
	b.executor.SetResponse(cmd, result)
	return b
}

// WithDefaultSuccess configures the default response to be successful.
func (b *ExecutorBuilder) WithDefaultSuccess() *ExecutorBuilder {
	result := exec.SuccessResult("")
	b.executor.SetDefaultResponse(result)
	return b
}

// WithDefaultError configures the default response to be an error.
func (b *ExecutorBuilder) WithDefaultError(err error) *ExecutorBuilder {
	result := exec.ErrorResult(err)
	b.executor.SetDefaultResponse(result)
	return b
}

// WithDefaultFailure configures the default response to be a failure.
func (b *ExecutorBuilder) WithDefaultFailure(exitCode int, stderr string) *ExecutorBuilder {
	result := exec.FailureResult(exitCode, stderr)
	b.executor.SetDefaultResponse(result)
	return b
}

// Build returns the configured MockExecutor.
func (b *ExecutorBuilder) Build() *exec.MockExecutor {
	return b.executor
}

// ============================================================================
// MockProgressReporter - For testing progress reporting
// ============================================================================

// ProgressUpdate represents a recorded progress update.
type ProgressUpdate struct {
	StepName   string
	StepIndex  int
	TotalSteps int
	Percent    float64
	Message    string
}

// MockProgressReporter records progress updates for testing.
type MockProgressReporter struct {
	mu       sync.Mutex
	updates  []ProgressUpdate
	complete bool
	error    error
}

// NewMockProgressReporter creates a new MockProgressReporter.
func NewMockProgressReporter() *MockProgressReporter {
	return &MockProgressReporter{
		updates: make([]ProgressUpdate, 0),
	}
}

// ReportProgress records a progress update.
func (m *MockProgressReporter) ReportProgress(stepName string, stepIndex, totalSteps int, percent float64, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updates = append(m.updates, ProgressUpdate{
		StepName:   stepName,
		StepIndex:  stepIndex,
		TotalSteps: totalSteps,
		Percent:    percent,
		Message:    message,
	})
}

// ReportComplete records completion.
func (m *MockProgressReporter) ReportComplete() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.complete = true
}

// ReportError records an error.
func (m *MockProgressReporter) ReportError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.error = err
}

// Updates returns all recorded progress updates.
func (m *MockProgressReporter) Updates() []ProgressUpdate {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ProgressUpdate{}, m.updates...)
}

// IsComplete returns whether completion was reported.
func (m *MockProgressReporter) IsComplete() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.complete
}

// Error returns the recorded error, if any.
func (m *MockProgressReporter) Error() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.error
}

// LastUpdate returns the most recent progress update.
func (m *MockProgressReporter) LastUpdate() *ProgressUpdate {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.updates) == 0 {
		return nil
	}
	return &m.updates[len(m.updates)-1]
}

// Clear resets all recorded data.
func (m *MockProgressReporter) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updates = m.updates[:0]
	m.complete = false
	m.error = nil
}
