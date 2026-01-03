// Package privilege provides privilege elevation handling for running commands
// that require root access. It supports sudo, pkexec, and doas with automatic
// detection and fallback, along with environment sanitization for security.
package privilege

import (
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/tungetti/igor/internal/errors"
)

// ElevationMethod represents how to elevate privileges.
type ElevationMethod int

const (
	// MethodNone indicates no elevation method is available.
	MethodNone ElevationMethod = iota
	// MethodSudo uses sudo for privilege elevation.
	MethodSudo
	// MethodPkexec uses pkexec (PolicyKit) for privilege elevation.
	MethodPkexec
	// MethodDoas uses doas (OpenBSD-style) for privilege elevation.
	MethodDoas
)

// String returns the string representation of the elevation method.
func (m ElevationMethod) String() string {
	switch m {
	case MethodSudo:
		return "sudo"
	case MethodPkexec:
		return "pkexec"
	case MethodDoas:
		return "doas"
	default:
		return "none"
	}
}

// Manager handles privilege elevation operations.
type Manager struct {
	method      ElevationMethod
	sudoPath    string
	pkexecPath  string
	doasPath    string
	isRoot      bool
	currentUser *user.User
}

// NewManager creates a new privilege manager that automatically detects
// the current user's privileges and available elevation methods.
func NewManager() *Manager {
	m := &Manager{}
	m.detectCurrentUser()
	m.detectElevationMethods()
	return m
}

// IsRoot returns true if the current process is running as root.
func (m *Manager) IsRoot() bool {
	return m.isRoot
}

// CurrentUser returns the current user information.
func (m *Manager) CurrentUser() *user.User {
	return m.currentUser
}

// CanElevate returns true if privilege elevation is available.
// This is true if already running as root or if an elevation method is available.
func (m *Manager) CanElevate() bool {
	return m.method != MethodNone || m.isRoot
}

// ElevationMethod returns the detected elevation method.
func (m *Manager) ElevationMethod() ElevationMethod {
	return m.method
}

// RequireRoot checks if the process can run as root.
// Returns nil if already root or if elevation is available.
// Returns an error if root is required but no elevation method is available.
func (m *Manager) RequireRoot() error {
	if m.isRoot {
		return nil
	}
	if !m.CanElevate() {
		return errors.New(errors.Permission, "root privileges required but no elevation method available (install sudo or pkexec)")
	}
	return nil
}

// ElevatedCommand returns a command and arguments with privilege elevation.
// If already running as root, returns the command unchanged.
// Otherwise, prepends the appropriate elevation command (sudo, pkexec, or doas).
func (m *Manager) ElevatedCommand(cmd string, args ...string) (string, []string) {
	if m.isRoot {
		return cmd, args
	}

	switch m.method {
	case MethodSudo:
		return m.sudoPath, append([]string{"-n", cmd}, args...)
	case MethodPkexec:
		return m.pkexecPath, append([]string{cmd}, args...)
	case MethodDoas:
		return m.doasPath, append([]string{cmd}, args...)
	default:
		return cmd, args
	}
}

// dangerousEnvVars lists environment variables that should be removed for security.
var dangerousEnvVars = map[string]bool{
	"LD_PRELOAD":       true,
	"LD_LIBRARY_PATH":  true,
	"LD_AUDIT":         true,
	"LD_ORIGIN_PATH":   true,
	"LD_PROFILE":       true,
	"LD_USE_LOAD_BIAS": true,
	"LD_DEBUG":         true,
	"LD_DEBUG_OUTPUT":  true,
	"LD_DYNAMIC_WEAK":  true,
	"LD_SHOW_AUXV":     true,
	"LD_BIND_NOW":      true,
	"LD_BIND_NOT":      true,
	"GCONV_PATH":       true,
	"GETCONF_DIR":      true,
	"HOSTALIASES":      true,
	"LOCALDOMAIN":      true,
	"LOCPATH":          true,
	"MALLOC_TRACE":     true,
	"NIS_PATH":         true,
	"NLSPATH":          true,
	"RESOLV_HOST_CONF": true,
	"RES_OPTIONS":      true,
	"TMPDIR":           true,
	"TZDIR":            true,
}

// safePath is a secure PATH used in sanitized environments.
const safePath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

// SanitizedEnv returns an environment with dangerous variables removed.
// This helps prevent privilege escalation attacks through environment manipulation.
func (m *Manager) SanitizedEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		// Skip dangerous variables
		if dangerousEnvVars[name] {
			continue
		}
		// Replace PATH with safe PATH
		if name == "PATH" {
			continue
		}
		env = append(env, e)
	}

	// Add safe PATH
	env = append(env, "PATH="+safePath)

	return env
}

// detectCurrentUser detects if the current process is running as root
// and retrieves the current user information.
func (m *Manager) detectCurrentUser() {
	m.isRoot = os.Geteuid() == 0
	m.currentUser, _ = user.Current()
}

// detectElevationMethods detects available privilege elevation methods.
// Checks in order: sudo, pkexec, doas. Uses the first available method.
func (m *Manager) detectElevationMethods() {
	// Check sudo
	if path, err := exec.LookPath("sudo"); err == nil {
		m.sudoPath = path
		m.method = MethodSudo
		return
	}

	// Check pkexec
	if path, err := exec.LookPath("pkexec"); err == nil {
		m.pkexecPath = path
		m.method = MethodPkexec
		return
	}

	// Check doas (OpenBSD-style)
	if path, err := exec.LookPath("doas"); err == nil {
		m.doasPath = path
		m.method = MethodDoas
		return
	}

	m.method = MethodNone
}

// SetMethod allows setting the elevation method for testing purposes.
// This should not be used in production code.
func (m *Manager) SetMethod(method ElevationMethod) {
	m.method = method
}

// SetRoot allows setting the root status for testing purposes.
// This should not be used in production code.
func (m *Manager) SetRoot(isRoot bool) {
	m.isRoot = isRoot
}

// SetPaths allows setting the elevation tool paths for testing purposes.
// This should not be used in production code.
func (m *Manager) SetPaths(sudo, pkexec, doas string) {
	m.sudoPath = sudo
	m.pkexecPath = pkexec
	m.doasPath = doas
}
