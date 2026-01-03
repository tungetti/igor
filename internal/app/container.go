// Package app provides application initialization, lifecycle management,
// and dependency injection for the Igor application.
package app

import (
	"sync"

	"github.com/tungetti/igor/internal/config"
	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/logging"
	"github.com/tungetti/igor/internal/privilege"
)

// Container holds all application dependencies.
// It provides thread-safe access to shared components and ensures
// proper initialization order during application startup.
type Container struct {
	mu        sync.RWMutex
	Config    *config.Config
	Logger    logging.Logger
	Executor  exec.Executor
	Privilege *privilege.Manager
}

// NewContainer creates a new dependency container.
func NewContainer() *Container {
	return &Container{}
}

// SetConfig sets the configuration.
func (c *Container) SetConfig(cfg *config.Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Config = cfg
}

// SetLogger sets the logger.
func (c *Container) SetLogger(l logging.Logger) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Logger = l
}

// SetExecutor sets the command executor.
func (c *Container) SetExecutor(e exec.Executor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Executor = e
}

// SetPrivilege sets the privilege manager.
func (c *Container) SetPrivilege(p *privilege.Manager) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Privilege = p
}

// GetConfig returns the configuration.
func (c *Container) GetConfig() *config.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Config
}

// GetLogger returns the logger.
func (c *Container) GetLogger() logging.Logger {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Logger
}

// GetExecutor returns the command executor.
func (c *Container) GetExecutor() exec.Executor {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Executor
}

// GetPrivilege returns the privilege manager.
func (c *Container) GetPrivilege() *privilege.Manager {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Privilege
}

// Validate checks that all required dependencies are set.
// Returns an error if any required dependency is missing.
// Note: Executor and Privilege are optional at startup.
func (c *Container) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Config == nil {
		return errors.New(errors.Configuration, "config not initialized")
	}
	if c.Logger == nil {
		return errors.New(errors.Configuration, "logger not initialized")
	}
	// Executor and Privilege are optional at startup
	return nil
}
