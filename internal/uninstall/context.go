package uninstall

import (
	"context"
	"sync"

	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/logging"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/privilege"
)

// Context holds the uninstallation context and shared state.
// It provides access to system information, package management, and execution
// capabilities needed during uninstallation.
type Context struct {
	// System information
	DistroInfo *distro.Distribution

	// Installed NVIDIA components to be removed
	InstalledDriver   string   // Currently installed driver version
	InstalledPackages []string // List of NVIDIA packages to remove

	// Package manager (from pkg package)
	PackageManager pkg.Manager

	// Command executor
	Executor exec.Executor

	// Privilege handler
	Privilege *privilege.Manager

	// Logger
	Logger logging.Logger

	// Cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// State storage for steps to share data
	state   map[string]interface{}
	stateMu sync.RWMutex

	// Dry run mode - do not make actual changes
	DryRun bool

	// Force removal - skip confirmation prompts
	Force bool

	// Keep configuration files during uninstallation
	KeepConfig bool
}

// NewUninstallContext creates a new uninstallation context with the given options.
func NewUninstallContext(opts ...ContextOption) *Context {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Context{
		ctx:               ctx,
		cancel:            cancel,
		state:             make(map[string]interface{}),
		InstalledPackages: make([]string, 0),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SetState stores a value in the context state with the given key.
// This allows steps to share data with each other.
func (c *Context) SetState(key string, value interface{}) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.state[key] = value
}

// GetState retrieves a value from the context state by key.
// Returns the value and a boolean indicating whether the key was found.
func (c *Context) GetState(key string) (interface{}, bool) {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	value, ok := c.state[key]
	return value, ok
}

// GetStateString retrieves a string value from the context state.
// Returns an empty string if the key is not found or the value is not a string.
func (c *Context) GetStateString(key string) string {
	value, ok := c.GetState(key)
	if !ok {
		return ""
	}
	s, ok := value.(string)
	if !ok {
		return ""
	}
	return s
}

// GetStateInt retrieves an int value from the context state.
// Returns 0 if the key is not found or the value is not an int.
func (c *Context) GetStateInt(key string) int {
	value, ok := c.GetState(key)
	if !ok {
		return 0
	}
	i, ok := value.(int)
	if !ok {
		return 0
	}
	return i
}

// GetStateBool retrieves a bool value from the context state.
// Returns false if the key is not found or the value is not a bool.
func (c *Context) GetStateBool(key string) bool {
	value, ok := c.GetState(key)
	if !ok {
		return false
	}
	b, ok := value.(bool)
	if !ok {
		return false
	}
	return b
}

// GetStateSlice retrieves a string slice value from the context state.
// Returns nil if the key is not found or the value is not a string slice.
func (c *Context) GetStateSlice(key string) []string {
	value, ok := c.GetState(key)
	if !ok {
		return nil
	}
	s, ok := value.([]string)
	if !ok {
		return nil
	}
	return s
}

// DeleteState removes a key from the context state.
func (c *Context) DeleteState(key string) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	delete(c.state, key)
}

// ClearState removes all keys from the context state.
func (c *Context) ClearState() {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.state = make(map[string]interface{})
}

// Context returns the underlying context.Context for cancellation support.
func (c *Context) Context() context.Context {
	return c.ctx
}

// Cancel cancels the context, signaling all operations to stop.
func (c *Context) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}

// IsCancelled returns true if the context has been cancelled.
func (c *Context) IsCancelled() bool {
	if c.ctx == nil {
		return false
	}
	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

// Log logs a message at the info level if a logger is configured.
func (c *Context) Log(msg string, keyvals ...interface{}) {
	if c.Logger != nil {
		c.Logger.Info(msg, keyvals...)
	}
}

// LogDebug logs a message at the debug level if a logger is configured.
func (c *Context) LogDebug(msg string, keyvals ...interface{}) {
	if c.Logger != nil {
		c.Logger.Debug(msg, keyvals...)
	}
}

// LogWarn logs a message at the warn level if a logger is configured.
func (c *Context) LogWarn(msg string, keyvals ...interface{}) {
	if c.Logger != nil {
		c.Logger.Warn(msg, keyvals...)
	}
}

// LogError logs a message at the error level if a logger is configured.
func (c *Context) LogError(msg string, keyvals ...interface{}) {
	if c.Logger != nil {
		c.Logger.Error(msg, keyvals...)
	}
}

// ContextOption is a functional option for Context.
type ContextOption func(*Context)

// WithUninstallDistroInfo sets the distribution information in the context.
func WithUninstallDistroInfo(info *distro.Distribution) ContextOption {
	return func(c *Context) {
		c.DistroInfo = info
	}
}

// WithUninstallPackageManager sets the package manager in the context.
func WithUninstallPackageManager(pm pkg.Manager) ContextOption {
	return func(c *Context) {
		c.PackageManager = pm
	}
}

// WithUninstallExecutor sets the command executor in the context.
func WithUninstallExecutor(executor exec.Executor) ContextOption {
	return func(c *Context) {
		c.Executor = executor
	}
}

// WithUninstallPrivilege sets the privilege handler in the context.
func WithUninstallPrivilege(priv *privilege.Manager) ContextOption {
	return func(c *Context) {
		c.Privilege = priv
	}
}

// WithUninstallLogger sets the logger in the context.
func WithUninstallLogger(logger logging.Logger) ContextOption {
	return func(c *Context) {
		c.Logger = logger
	}
}

// WithUninstallDryRun sets the dry run mode in the context.
func WithUninstallDryRun(dryRun bool) ContextOption {
	return func(c *Context) {
		c.DryRun = dryRun
	}
}

// WithUninstallForce sets the force removal mode in the context.
func WithUninstallForce(force bool) ContextOption {
	return func(c *Context) {
		c.Force = force
	}
}

// WithKeepConfig sets whether to keep configuration files during uninstallation.
func WithKeepConfig(keep bool) ContextOption {
	return func(c *Context) {
		c.KeepConfig = keep
	}
}

// WithInstalledDriver sets the currently installed driver version.
func WithInstalledDriver(version string) ContextOption {
	return func(c *Context) {
		c.InstalledDriver = version
	}
}

// WithInstalledPackages sets the list of installed NVIDIA packages.
func WithInstalledPackages(packages []string) ContextOption {
	return func(c *Context) {
		c.InstalledPackages = append([]string{}, packages...)
	}
}

// WithUninstallContext sets a parent context for cancellation.
func WithUninstallContext(ctx context.Context) ContextOption {
	return func(c *Context) {
		if c.cancel != nil {
			c.cancel() // Cancel the default context
		}
		c.ctx, c.cancel = context.WithCancel(ctx)
	}
}
