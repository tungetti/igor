package app

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/tungetti/igor/internal/config"
	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/logging"
	"github.com/tungetti/igor/internal/privilege"
)

// App represents the main application with its dependencies and lifecycle.
type App struct {
	container *Container
	lifecycle *Lifecycle
	version   string
	buildTime string
	gitCommit string
}

// Options configures the application.
type Options struct {
	Version         string
	BuildTime       string
	GitCommit       string
	ConfigPath      string
	ShutdownTimeout time.Duration
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		Version:         "unknown",
		BuildTime:       "unknown",
		GitCommit:       "unknown",
		ShutdownTimeout: 30 * time.Second,
	}
}

// New creates a new application with the given options.
func New(opts Options) *App {
	return &App{
		container: NewContainer(),
		lifecycle: NewLifecycle(opts.ShutdownTimeout),
		version:   opts.Version,
		buildTime: opts.BuildTime,
		gitCommit: opts.GitCommit,
	}
}

// Initialize sets up all application components in the correct order.
// The initialization order is:
// 1. Configuration
// 2. Logger
// 3. Privilege manager
// 4. Command executor
func (a *App) Initialize(ctx context.Context, configPath string) error {
	// 1. Load configuration
	cfg, err := a.loadConfig(configPath)
	if err != nil {
		return errors.Wrap(errors.Configuration, "failed to load config", err)
	}
	a.container.SetConfig(cfg)

	// 2. Initialize logger
	logger, err := a.initLogger(cfg)
	if err != nil {
		return errors.Wrap(errors.Configuration, "failed to initialize logger", err)
	}
	a.container.SetLogger(logger)

	logger.Info("starting application",
		"version", a.version,
		"build_time", a.buildTime,
		"git_commit", a.gitCommit,
	)

	// 3. Initialize privilege manager
	priv := privilege.NewManager()
	a.container.SetPrivilege(priv)

	if priv.IsRoot() {
		logger.Debug("running as root")
	} else if priv.CurrentUser() != nil {
		logger.Debug("running as user", "user", priv.CurrentUser().Username)
	}

	// 4. Initialize executor
	execOpts := exec.DefaultOptions()
	if cfg.CommandTimeout > 0 {
		execOpts.Timeout = cfg.CommandTimeout
	}
	executor := exec.NewExecutor(execOpts, priv)
	a.container.SetExecutor(executor)

	// 5. Validate container
	if err := a.container.Validate(); err != nil {
		return err
	}

	logger.Info("application initialized successfully")
	return nil
}

// Run executes the main application logic with panic recovery.
func (a *App) Run(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = a.handlePanic(r)
		}
	}()

	logger := a.container.GetLogger()

	// Register shutdown handler
	a.lifecycle.OnShutdown(func(ctx context.Context) error {
		if logger != nil {
			logger.Info("shutting down...")
		}
		return nil
	})

	// Main application logic would go here
	// For now, just log that we're ready
	if logger != nil {
		logger.Info("application ready")
	}

	return nil
}

// Shutdown gracefully shuts down the application.
func (a *App) Shutdown() error {
	return a.lifecycle.Shutdown()
}

// Container returns the dependency container.
func (a *App) Container() *Container {
	return a.container
}

// Lifecycle returns the lifecycle manager.
func (a *App) Lifecycle() *Lifecycle {
	return a.lifecycle
}

// Version returns the application version.
func (a *App) Version() string {
	return a.version
}

// BuildTime returns the application build time.
func (a *App) BuildTime() string {
	return a.buildTime
}

// GitCommit returns the application git commit.
func (a *App) GitCommit() string {
	return a.gitCommit
}

func (a *App) loadConfig(path string) (*config.Config, error) {
	loader := config.NewLoader(path)
	return loader.Load()
}

func (a *App) initLogger(cfg *config.Config) (logging.Logger, error) {
	level := logging.ParseLevel(cfg.LogLevel)

	if cfg.LogFile != "" {
		return logging.NewFileLogger(cfg.LogFile, level)
	}

	opts := logging.DefaultOptions()
	opts.Level = level
	return logging.New(opts), nil
}

// handlePanic handles a recovered panic and returns an error.
// It logs the panic with a stack trace if a logger is available.
func (a *App) handlePanic(r interface{}) error {
	stack := debug.Stack()
	logger := a.container.GetLogger()

	if logger != nil {
		logger.Error("panic recovered",
			"panic", fmt.Sprintf("%v", r),
			"stack", string(stack),
		)
	} else {
		fmt.Fprintf(os.Stderr, "PANIC: %v\n%s\n", r, stack)
	}

	return errors.Newf(errors.Unknown, "panic: %v", r)
}

// RecoverPanic is a helper function that can be deferred to recover from panics.
// It logs the panic with a stack trace.
func (a *App) RecoverPanic() {
	if r := recover(); r != nil {
		_ = a.handlePanic(r)
	}
}

// RunWithLifecycle runs the application and waits for shutdown signal.
// This is a convenience method that combines Run, WaitForSignal, and Shutdown.
func (a *App) RunWithLifecycle(ctx context.Context) error {
	if err := a.Run(ctx); err != nil {
		return err
	}

	// Wait for shutdown signal
	sig := a.lifecycle.WaitForSignal()
	if sig != nil {
		logger := a.container.GetLogger()
		if logger != nil {
			logger.Info("received signal", "signal", sig.String())
		}
	}

	return a.Shutdown()
}
