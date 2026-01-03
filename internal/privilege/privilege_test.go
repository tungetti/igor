package privilege

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestElevationMethodString(t *testing.T) {
	tests := []struct {
		name     string
		method   ElevationMethod
		expected string
	}{
		{
			name:     "MethodNone",
			method:   MethodNone,
			expected: "none",
		},
		{
			name:     "MethodSudo",
			method:   MethodSudo,
			expected: "sudo",
		},
		{
			name:     "MethodPkexec",
			method:   MethodPkexec,
			expected: "pkexec",
		},
		{
			name:     "MethodDoas",
			method:   MethodDoas,
			expected: "doas",
		},
		{
			name:     "UnknownMethod",
			method:   ElevationMethod(99),
			expected: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.method.String())
		})
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	require.NotNil(t, m)

	// Should detect current user
	assert.NotNil(t, m.CurrentUser())

	// IsRoot should match expected value based on euid
	expectedRoot := os.Geteuid() == 0
	assert.Equal(t, expectedRoot, m.IsRoot())
}

func TestManagerIsRoot(t *testing.T) {
	m := &Manager{}

	// Test when not root
	m.isRoot = false
	assert.False(t, m.IsRoot())

	// Test when root
	m.isRoot = true
	assert.True(t, m.IsRoot())
}

func TestManagerCanElevate(t *testing.T) {
	tests := []struct {
		name     string
		isRoot   bool
		method   ElevationMethod
		expected bool
	}{
		{
			name:     "IsRoot",
			isRoot:   true,
			method:   MethodNone,
			expected: true,
		},
		{
			name:     "HasSudo",
			isRoot:   false,
			method:   MethodSudo,
			expected: true,
		},
		{
			name:     "HasPkexec",
			isRoot:   false,
			method:   MethodPkexec,
			expected: true,
		},
		{
			name:     "HasDoas",
			isRoot:   false,
			method:   MethodDoas,
			expected: true,
		},
		{
			name:     "NoMethod",
			isRoot:   false,
			method:   MethodNone,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				isRoot: tt.isRoot,
				method: tt.method,
			}
			assert.Equal(t, tt.expected, m.CanElevate())
		})
	}
}

func TestManagerElevationMethod(t *testing.T) {
	m := &Manager{method: MethodPkexec}
	assert.Equal(t, MethodPkexec, m.ElevationMethod())
}

func TestManagerRequireRoot(t *testing.T) {
	tests := []struct {
		name      string
		isRoot    bool
		method    ElevationMethod
		expectErr bool
	}{
		{
			name:      "AlreadyRoot",
			isRoot:    true,
			method:    MethodNone,
			expectErr: false,
		},
		{
			name:      "CanElevateWithSudo",
			isRoot:    false,
			method:    MethodSudo,
			expectErr: false,
		},
		{
			name:      "CanElevateWithPkexec",
			isRoot:    false,
			method:    MethodPkexec,
			expectErr: false,
		},
		{
			name:      "CannotElevate",
			isRoot:    false,
			method:    MethodNone,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				isRoot: tt.isRoot,
				method: tt.method,
			}
			err := m.RequireRoot()
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "root privileges required")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManagerElevatedCommand(t *testing.T) {
	tests := []struct {
		name         string
		isRoot       bool
		method       ElevationMethod
		sudoPath     string
		pkexecPath   string
		doasPath     string
		cmd          string
		args         []string
		expectedCmd  string
		expectedArgs []string
	}{
		{
			name:         "AlreadyRoot",
			isRoot:       true,
			method:       MethodNone,
			cmd:          "apt",
			args:         []string{"install", "nvidia-driver"},
			expectedCmd:  "apt",
			expectedArgs: []string{"install", "nvidia-driver"},
		},
		{
			name:         "UseSudo",
			isRoot:       false,
			method:       MethodSudo,
			sudoPath:     "/usr/bin/sudo",
			cmd:          "apt",
			args:         []string{"install", "nvidia-driver"},
			expectedCmd:  "/usr/bin/sudo",
			expectedArgs: []string{"-n", "apt", "install", "nvidia-driver"},
		},
		{
			name:         "UsePkexec",
			isRoot:       false,
			method:       MethodPkexec,
			pkexecPath:   "/usr/bin/pkexec",
			cmd:          "apt",
			args:         []string{"install", "nvidia-driver"},
			expectedCmd:  "/usr/bin/pkexec",
			expectedArgs: []string{"apt", "install", "nvidia-driver"},
		},
		{
			name:         "UseDoas",
			isRoot:       false,
			method:       MethodDoas,
			doasPath:     "/usr/bin/doas",
			cmd:          "apt",
			args:         []string{"install", "nvidia-driver"},
			expectedCmd:  "/usr/bin/doas",
			expectedArgs: []string{"apt", "install", "nvidia-driver"},
		},
		{
			name:         "NoMethodAvailable",
			isRoot:       false,
			method:       MethodNone,
			cmd:          "apt",
			args:         []string{"install", "nvidia-driver"},
			expectedCmd:  "apt",
			expectedArgs: []string{"install", "nvidia-driver"},
		},
		{
			name:         "EmptyArgs",
			isRoot:       false,
			method:       MethodSudo,
			sudoPath:     "/usr/bin/sudo",
			cmd:          "whoami",
			args:         []string{},
			expectedCmd:  "/usr/bin/sudo",
			expectedArgs: []string{"-n", "whoami"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				isRoot:     tt.isRoot,
				method:     tt.method,
				sudoPath:   tt.sudoPath,
				pkexecPath: tt.pkexecPath,
				doasPath:   tt.doasPath,
			}
			cmd, args := m.ElevatedCommand(tt.cmd, tt.args...)
			assert.Equal(t, tt.expectedCmd, cmd)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestManagerSanitizedEnv(t *testing.T) {
	// Set some dangerous environment variables for testing
	dangerous := []string{
		"LD_PRELOAD=/tmp/evil.so",
		"LD_LIBRARY_PATH=/tmp/evil",
		"LD_AUDIT=/tmp/audit.so",
		"GCONV_PATH=/tmp/gconv",
	}

	// Save original env
	origEnv := os.Environ()

	// Set dangerous vars
	for _, d := range dangerous {
		parts := strings.SplitN(d, "=", 2)
		os.Setenv(parts[0], parts[1])
	}

	// Cleanup after test
	defer func() {
		for _, d := range dangerous {
			parts := strings.SplitN(d, "=", 2)
			os.Unsetenv(parts[0])
		}
		// Restore PATH if it was changed
		for _, e := range origEnv {
			if strings.HasPrefix(e, "PATH=") {
				parts := strings.SplitN(e, "=", 2)
				os.Setenv(parts[0], parts[1])
				break
			}
		}
	}()

	m := NewManager()
	sanitized := m.SanitizedEnv()

	// Check that dangerous variables are removed
	for _, env := range sanitized {
		for _, d := range dangerous {
			parts := strings.SplitN(d, "=", 2)
			assert.False(t, strings.HasPrefix(env, parts[0]+"="),
				"dangerous variable %s should be removed", parts[0])
		}
	}

	// Check that safe PATH is set
	hasPath := false
	for _, env := range sanitized {
		if strings.HasPrefix(env, "PATH=") {
			hasPath = true
			assert.Equal(t, "PATH="+safePath, env)
			break
		}
	}
	assert.True(t, hasPath, "safe PATH should be set")
}

func TestManagerCurrentUser(t *testing.T) {
	m := NewManager()
	user := m.CurrentUser()

	// Should have a valid user
	if user != nil {
		assert.NotEmpty(t, user.Username)
	}
}

func TestManagerSetters(t *testing.T) {
	m := NewManager()

	// Test SetMethod
	m.SetMethod(MethodPkexec)
	assert.Equal(t, MethodPkexec, m.ElevationMethod())

	// Test SetRoot
	m.SetRoot(true)
	assert.True(t, m.IsRoot())
	m.SetRoot(false)
	assert.False(t, m.IsRoot())

	// Test SetPaths
	m.SetPaths("/custom/sudo", "/custom/pkexec", "/custom/doas")
	m.SetMethod(MethodSudo)
	cmd, _ := m.ElevatedCommand("test")
	assert.Equal(t, "/custom/sudo", cmd)
}

// Sudo tests

func TestDefaultSudoOptions(t *testing.T) {
	opts := DefaultSudoOptions()
	assert.True(t, opts.NonInteractive)
	assert.False(t, opts.PreserveEnv)
	assert.Empty(t, opts.User)
	assert.Equal(t, 2*time.Minute, opts.Timeout)
}

func TestSudoCommand(t *testing.T) {
	tests := []struct {
		name         string
		cmd          string
		args         []string
		opts         SudoOptions
		expectedArgs []string
	}{
		{
			name:         "DefaultOptions",
			cmd:          "apt",
			args:         []string{"update"},
			opts:         DefaultSudoOptions(),
			expectedArgs: []string{"-n", "apt", "update"},
		},
		{
			name: "WithPreserveEnv",
			cmd:  "apt",
			args: []string{"update"},
			opts: SudoOptions{
				NonInteractive: true,
				PreserveEnv:    true,
			},
			expectedArgs: []string{"-n", "-E", "apt", "update"},
		},
		{
			name: "WithUser",
			cmd:  "command",
			args: []string{"arg1"},
			opts: SudoOptions{
				NonInteractive: true,
				User:           "nobody",
			},
			expectedArgs: []string{"-n", "-u", "nobody", "command", "arg1"},
		},
		{
			name: "AllOptions",
			cmd:  "service",
			args: []string{"restart", "nginx"},
			opts: SudoOptions{
				NonInteractive: true,
				PreserveEnv:    true,
				User:           "www-data",
			},
			expectedArgs: []string{"-n", "-E", "-u", "www-data", "service", "restart", "nginx"},
		},
		{
			name: "Interactive",
			cmd:  "vim",
			args: []string{"/etc/hosts"},
			opts: SudoOptions{
				NonInteractive: false,
			},
			expectedArgs: []string{"vim", "/etc/hosts"},
		},
		{
			name:         "NoArgs",
			cmd:          "whoami",
			args:         []string{},
			opts:         DefaultSudoOptions(),
			expectedArgs: []string{"-n", "whoami"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := SudoCommand(tt.cmd, tt.args, tt.opts)
			assert.Equal(t, "sudo", cmd.Path[len(cmd.Path)-4:]) // Ends with "sudo"
			assert.Equal(t, append([]string{"sudo"}, tt.expectedArgs...), cmd.Args)
		})
	}
}

func TestSudoCommandContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := DefaultSudoOptions()
	cmd := SudoCommandContext(ctx, "echo", []string{"hello"}, opts)

	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Args, "echo")
	assert.Contains(t, cmd.Args, "hello")
	assert.Contains(t, cmd.Args, "-n")
}

func TestCheckSudoAvailable(t *testing.T) {
	// This test just verifies the function runs without panicking
	// The actual result depends on the system
	_ = CheckSudoAvailable()
}

func TestCheckSudoAccess(t *testing.T) {
	// Skip if running as root (sudo check doesn't make sense)
	if os.Geteuid() == 0 {
		t.Skip("skipping sudo access check when running as root")
	}

	// This test just verifies the function runs without panicking
	// The actual result depends on the system configuration
	_ = CheckSudoAccess()
}

func TestSudoValidate(t *testing.T) {
	// Skip if running as root
	if os.Geteuid() == 0 {
		t.Skip("skipping sudo validate check when running as root")
	}

	// This test just verifies the function runs without panicking
	_ = SudoValidate()
}

func TestBuildSudoArgs(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		args     []string
		opts     SudoOptions
		expected []string
	}{
		{
			name:     "OnlyNonInteractive",
			cmd:      "test",
			args:     []string{"arg"},
			opts:     SudoOptions{NonInteractive: true},
			expected: []string{"-n", "test", "arg"},
		},
		{
			name:     "OnlyPreserveEnv",
			cmd:      "test",
			args:     []string{},
			opts:     SudoOptions{PreserveEnv: true},
			expected: []string{"-E", "test"},
		},
		{
			name:     "OnlyUser",
			cmd:      "test",
			args:     []string{},
			opts:     SudoOptions{User: "testuser"},
			expected: []string{"-u", "testuser", "test"},
		},
		{
			name:     "Empty",
			cmd:      "test",
			args:     []string{},
			opts:     SudoOptions{},
			expected: []string{"test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSudoArgs(tt.cmd, tt.args, tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Pkexec tests

func TestPkexecCommand(t *testing.T) {
	cmd := PkexecCommand("apt", []string{"update"})
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Path, "pkexec")
	assert.Equal(t, []string{"pkexec", "apt", "update"}, cmd.Args)
}

func TestPkexecCommandContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := PkexecCommandContext(ctx, "apt", []string{"install", "vim"})
	assert.NotNil(t, cmd)
	assert.Equal(t, []string{"pkexec", "apt", "install", "vim"}, cmd.Args)
}

func TestPkexecCommandEmpty(t *testing.T) {
	cmd := PkexecCommand("whoami", []string{})
	assert.Equal(t, []string{"pkexec", "whoami"}, cmd.Args)
}

func TestCheckPkexecAvailable(t *testing.T) {
	// This test just verifies the function runs without panicking
	_ = CheckPkexecAvailable()
}

func TestGetPkexecPath(t *testing.T) {
	path := GetPkexecPath()
	if CheckPkexecAvailable() {
		assert.NotEmpty(t, path)
		assert.Contains(t, path, "pkexec")
	} else {
		assert.Empty(t, path)
	}
}

// Integration-style tests

func TestManagerElevatedCommandIntegration(t *testing.T) {
	m := NewManager()

	// Test with a simple command
	cmd, args := m.ElevatedCommand("echo", "hello", "world")

	if m.IsRoot() {
		assert.Equal(t, "echo", cmd)
		assert.Equal(t, []string{"hello", "world"}, args)
	} else if m.CanElevate() {
		// Should be prefixed with elevation command
		assert.NotEqual(t, "echo", cmd)
		assert.Contains(t, args, "echo")
	}
}

func TestSanitizedEnvDoesNotContainOriginalPath(t *testing.T) {
	// Temporarily set a custom PATH
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "/tmp/evil:/var/evil")
	defer os.Setenv("PATH", origPath)

	m := NewManager()
	sanitized := m.SanitizedEnv()

	for _, env := range sanitized {
		if strings.HasPrefix(env, "PATH=") {
			// Should not contain the evil paths
			assert.NotContains(t, env, "/tmp/evil")
			assert.NotContains(t, env, "/var/evil")
			// Should contain the safe path
			assert.Contains(t, env, "/usr/bin")
			break
		}
	}
}

func TestSanitizedEnvPreservesNormalVars(t *testing.T) {
	// Set a normal environment variable
	os.Setenv("TEST_IGOR_VAR", "test_value")
	defer os.Unsetenv("TEST_IGOR_VAR")

	m := NewManager()
	sanitized := m.SanitizedEnv()

	found := false
	for _, env := range sanitized {
		if env == "TEST_IGOR_VAR=test_value" {
			found = true
			break
		}
	}
	assert.True(t, found, "normal environment variables should be preserved")
}

func TestDangerousEnvVarsComplete(t *testing.T) {
	// Verify all expected dangerous vars are in the map
	expected := []string{
		"LD_PRELOAD",
		"LD_LIBRARY_PATH",
		"LD_AUDIT",
		"GCONV_PATH",
	}

	for _, v := range expected {
		assert.True(t, dangerousEnvVars[v], "expected %s to be in dangerousEnvVars", v)
	}
}

func TestSafePathConstant(t *testing.T) {
	// Verify safe path contains essential directories
	assert.Contains(t, safePath, "/usr/bin")
	assert.Contains(t, safePath, "/bin")
	assert.Contains(t, safePath, "/sbin")
	assert.Contains(t, safePath, "/usr/sbin")
}
