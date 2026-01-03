package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestParser() *Parser {
	return NewParser("igor", "1.0.0", "2024-01-01T00:00:00Z", "abc1234")
}

// ============================================================================
// Command Parsing Tests
// ============================================================================

func TestParseNoArgs(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{})

	require.NoError(t, err)
	assert.True(t, result.ShowHelp)
	assert.Equal(t, CommandNone, result.Command)
}

func TestParseInstallCommand(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"install"})

	require.NoError(t, err)
	assert.Equal(t, CommandInstall, result.Command)
	assert.False(t, result.ShowHelp)
}

func TestParseUninstallCommand(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"uninstall"})

	require.NoError(t, err)
	assert.Equal(t, CommandUninstall, result.Command)
}

func TestParseDetectCommand(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"detect"})

	require.NoError(t, err)
	assert.Equal(t, CommandDetect, result.Command)
}

func TestParseListCommand(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"list"})

	require.NoError(t, err)
	assert.Equal(t, CommandList, result.Command)
}

func TestParseVersionCommand(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"version"})

	require.NoError(t, err)
	assert.Equal(t, CommandVersion, result.Command)
}

func TestParseHelpCommand(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"help"})

	require.NoError(t, err)
	assert.Equal(t, CommandHelp, result.Command)
	assert.True(t, result.ShowHelp)
}

func TestParseHelpWithCommand(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"help", "install"})

	require.NoError(t, err)
	assert.Equal(t, CommandHelp, result.Command)
	assert.True(t, result.ShowHelp)
	assert.Equal(t, "install", result.HelpCommand)
}

func TestParseUnknownCommand(t *testing.T) {
	p := newTestParser()
	_, err := p.Parse([]string{"unknown"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

// ============================================================================
// Command Alias Tests
// ============================================================================

func TestParseCommandAliases(t *testing.T) {
	tests := []struct {
		alias    string
		expected Command
	}{
		{"i", CommandInstall},
		{"install", CommandInstall},
		{"u", CommandUninstall},
		{"remove", CommandUninstall},
		{"uninstall", CommandUninstall},
		{"d", CommandDetect},
		{"detect", CommandDetect},
		{"l", CommandList},
		{"ls", CommandList},
		{"list", CommandList},
		{"v", CommandVersion},
		{"version", CommandVersion},
		{"h", CommandHelp},
		{"help", CommandHelp},
	}

	p := newTestParser()
	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			result, err := p.Parse([]string{tt.alias})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Command)
		})
	}
}

// ============================================================================
// Global Flags Tests
// ============================================================================

func TestParseGlobalVerboseFlag(t *testing.T) {
	tests := []struct {
		args    []string
		verbose bool
	}{
		{[]string{"--verbose", "install"}, true},
		{[]string{"-v", "install"}, true},
		{[]string{"install"}, false},
	}

	p := newTestParser()
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			result, err := p.Parse(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.verbose, result.GlobalFlags.Verbose)
		})
	}
}

func TestParseGlobalQuietFlag(t *testing.T) {
	tests := []struct {
		args  []string
		quiet bool
	}{
		{[]string{"--quiet", "install"}, true},
		{[]string{"-q", "install"}, true},
		{[]string{"install"}, false},
	}

	p := newTestParser()
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			result, err := p.Parse(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.quiet, result.GlobalFlags.Quiet)
		})
	}
}

func TestParseGlobalDryRunFlag(t *testing.T) {
	tests := []struct {
		args   []string
		dryRun bool
	}{
		{[]string{"--dry-run", "install"}, true},
		{[]string{"-n", "install"}, true},
		{[]string{"install"}, false},
	}

	p := newTestParser()
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			result, err := p.Parse(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.dryRun, result.GlobalFlags.DryRun)
		})
	}
}

func TestParseGlobalConfigFlag(t *testing.T) {
	tests := []struct {
		args   []string
		config string
	}{
		{[]string{"--config", "/path/to/config.yaml", "install"}, "/path/to/config.yaml"},
		{[]string{"-c", "/custom/config.yaml", "install"}, "/custom/config.yaml"},
		{[]string{"install"}, ""},
	}

	p := newTestParser()
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			result, err := p.Parse(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.config, result.GlobalFlags.ConfigFile)
		})
	}
}

func TestParseGlobalLogFileFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"--log-file", "/var/log/igor.log", "install"})

	require.NoError(t, err)
	assert.Equal(t, "/var/log/igor.log", result.GlobalFlags.LogFile)
}

func TestParseGlobalLogLevelFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"--log-level", "debug", "install"})

	require.NoError(t, err)
	assert.Equal(t, "debug", result.GlobalFlags.LogLevel)
}

func TestParseGlobalNoColorFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"--no-color", "install"})

	require.NoError(t, err)
	assert.True(t, result.GlobalFlags.NoColor)
}

func TestParseMultipleGlobalFlags(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"-v", "--dry-run", "-c", "/config.yaml", "install"})

	require.NoError(t, err)
	assert.True(t, result.GlobalFlags.Verbose)
	assert.True(t, result.GlobalFlags.DryRun)
	assert.Equal(t, "/config.yaml", result.GlobalFlags.ConfigFile)
	assert.Equal(t, CommandInstall, result.Command)
}

func TestParseConflictingVerboseQuiet(t *testing.T) {
	p := newTestParser()
	_, err := p.Parse([]string{"-v", "-q", "install"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "verbose")
	assert.Contains(t, err.Error(), "quiet")
}

// ============================================================================
// Install Command Flags Tests
// ============================================================================

func TestParseInstallDriverFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"install", "--driver", "535.104"})

	require.NoError(t, err)
	assert.Equal(t, "535.104", result.InstallFlags.DriverVersion)
}

func TestParseInstallCUDAFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"install", "--cuda", "12.0"})

	require.NoError(t, err)
	assert.Equal(t, "12.0", result.InstallFlags.CUDAVersion)
}

func TestParseInstallWithCUDAFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"install", "--with-cuda"})

	require.NoError(t, err)
	assert.True(t, result.InstallFlags.InstallCUDA)
}

func TestParseInstallForceFlag(t *testing.T) {
	tests := []struct {
		args  []string
		force bool
	}{
		{[]string{"install", "--force"}, true},
		{[]string{"install", "-f"}, true},
		{[]string{"install"}, false},
	}

	p := newTestParser()
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			result, err := p.Parse(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.force, result.InstallFlags.Force)
		})
	}
}

func TestParseInstallSkipRebootFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"install", "--skip-reboot"})

	require.NoError(t, err)
	assert.True(t, result.InstallFlags.SkipReboot)
}

func TestParseInstallAllFlags(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{
		"install",
		"--driver", "535.104",
		"--cuda", "12.0",
		"--with-cuda",
		"--force",
		"--skip-reboot",
	})

	require.NoError(t, err)
	assert.Equal(t, "535.104", result.InstallFlags.DriverVersion)
	assert.Equal(t, "12.0", result.InstallFlags.CUDAVersion)
	assert.True(t, result.InstallFlags.InstallCUDA)
	assert.True(t, result.InstallFlags.Force)
	assert.True(t, result.InstallFlags.SkipReboot)
}

// ============================================================================
// Uninstall Command Flags Tests
// ============================================================================

func TestParseUninstallPurgeFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"uninstall", "--purge"})

	require.NoError(t, err)
	assert.True(t, result.UninstallFlags.Purge)
}

func TestParseUninstallKeepConfigFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"uninstall", "--keep-config"})

	require.NoError(t, err)
	assert.True(t, result.UninstallFlags.KeepConfig)
}

// ============================================================================
// Detect Command Flags Tests
// ============================================================================

func TestParseDetectJSONFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"detect", "--json"})

	require.NoError(t, err)
	assert.True(t, result.DetectFlags.JSON)
}

func TestParseDetectBriefFlag(t *testing.T) {
	tests := []struct {
		args  []string
		brief bool
	}{
		{[]string{"detect", "--brief"}, true},
		{[]string{"detect", "-b"}, true},
		{[]string{"detect"}, false},
	}

	p := newTestParser()
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			result, err := p.Parse(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.brief, result.DetectFlags.Brief)
		})
	}
}

// ============================================================================
// List Command Flags Tests
// ============================================================================

func TestParseListInstalledFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"list", "--installed"})

	require.NoError(t, err)
	assert.True(t, result.ListFlags.Installed)
}

func TestParseListAvailableFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"list", "--available"})

	require.NoError(t, err)
	assert.True(t, result.ListFlags.Available)
}

func TestParseListJSONFlag(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"list", "--json"})

	require.NoError(t, err)
	assert.True(t, result.ListFlags.JSON)
}

// ============================================================================
// Global + Command Flags Combined Tests
// ============================================================================

func TestParseGlobalAndCommandFlags(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"-v", "--dry-run", "install", "--driver", "535.104", "--force"})

	require.NoError(t, err)
	assert.True(t, result.GlobalFlags.Verbose)
	assert.True(t, result.GlobalFlags.DryRun)
	assert.Equal(t, CommandInstall, result.Command)
	assert.Equal(t, "535.104", result.InstallFlags.DriverVersion)
	assert.True(t, result.InstallFlags.Force)
}

// ============================================================================
// Command Type Tests
// ============================================================================

func TestCommandString(t *testing.T) {
	tests := []struct {
		cmd      Command
		expected string
	}{
		{CommandNone, ""},
		{CommandInstall, "install"},
		{CommandUninstall, "uninstall"},
		{CommandDetect, "detect"},
		{CommandList, "list"},
		{CommandVersion, "version"},
		{CommandHelp, "help"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cmd.String())
		})
	}
}

func TestCommandIsValid(t *testing.T) {
	tests := []struct {
		cmd   Command
		valid bool
	}{
		{CommandNone, false},
		{CommandInstall, true},
		{CommandUninstall, true},
		{CommandDetect, true},
		{CommandList, true},
		{CommandVersion, true},
		{CommandHelp, true},
		{Command(99), false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd.String(), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.cmd.IsValid())
		})
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected Command
	}{
		{"install", CommandInstall},
		{"i", CommandInstall},
		{"uninstall", CommandUninstall},
		{"remove", CommandUninstall},
		{"u", CommandUninstall},
		{"detect", CommandDetect},
		{"d", CommandDetect},
		{"list", CommandList},
		{"l", CommandList},
		{"ls", CommandList},
		{"version", CommandVersion},
		{"v", CommandVersion},
		{"help", CommandHelp},
		{"h", CommandHelp},
		{"unknown", CommandNone},
		{"", CommandNone},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseCommand(tt.input))
		})
	}
}

// ============================================================================
// Commands() Tests
// ============================================================================

func TestCommandsReturnsAllCommands(t *testing.T) {
	cmds := Commands()

	assert.Len(t, cmds, 6)

	names := make(map[string]bool)
	for _, cmd := range cmds {
		names[cmd.Name] = true
	}

	assert.True(t, names["install"])
	assert.True(t, names["uninstall"])
	assert.True(t, names["detect"])
	assert.True(t, names["list"])
	assert.True(t, names["version"])
	assert.True(t, names["help"])
}

func TestCommandInfoHasRequiredFields(t *testing.T) {
	for _, cmd := range Commands() {
		t.Run(cmd.Name, func(t *testing.T) {
			assert.NotEmpty(t, cmd.Name)
			assert.NotEmpty(t, cmd.Description)
			assert.NotEmpty(t, cmd.Usage)
			assert.NotEmpty(t, cmd.LongDescription)
		})
	}
}

func TestGetCommandInfo(t *testing.T) {
	info := GetCommandInfo(CommandInstall)
	require.NotNil(t, info)
	assert.Equal(t, "install", info.Name)

	info = GetCommandInfo(CommandNone)
	assert.Nil(t, info)
}

// ============================================================================
// Usage/Help Output Tests
// ============================================================================

func TestUsageContainsAllCommands(t *testing.T) {
	p := newTestParser()
	usage := p.Usage()

	assert.Contains(t, usage, "install")
	assert.Contains(t, usage, "uninstall")
	assert.Contains(t, usage, "detect")
	assert.Contains(t, usage, "list")
	assert.Contains(t, usage, "version")
	assert.Contains(t, usage, "help")
}

func TestUsageContainsGlobalFlags(t *testing.T) {
	p := newTestParser()
	usage := p.Usage()

	assert.Contains(t, usage, "--verbose")
	assert.Contains(t, usage, "--quiet")
	assert.Contains(t, usage, "--dry-run")
	assert.Contains(t, usage, "--config")
	assert.Contains(t, usage, "--log-file")
	assert.Contains(t, usage, "--log-level")
	assert.Contains(t, usage, "--no-color")
}

func TestUsageContainsShortFlags(t *testing.T) {
	p := newTestParser()
	usage := p.Usage()

	assert.Contains(t, usage, "-v")
	assert.Contains(t, usage, "-q")
	assert.Contains(t, usage, "-n")
	assert.Contains(t, usage, "-c")
}

func TestCommandUsageValid(t *testing.T) {
	p := newTestParser()

	usage := p.CommandUsage("install")
	assert.Contains(t, usage, "Install")
	assert.Contains(t, usage, "--driver")
}

func TestCommandUsageUnknown(t *testing.T) {
	p := newTestParser()

	usage := p.CommandUsage("unknown")
	assert.Contains(t, usage, "Unknown command")
}

// ============================================================================
// Version Output Tests
// ============================================================================

func TestVersionString(t *testing.T) {
	p := NewParser("igor", "1.2.3", "2024-01-15T10:00:00Z", "abcdef1234567890")

	version := p.VersionString()

	assert.Contains(t, version, "igor")
	assert.Contains(t, version, "1.2.3")
	assert.Contains(t, version, "2024-01-15")
	assert.Contains(t, version, "abcdef1") // Short hash
}

func TestVersionStringWithUnknown(t *testing.T) {
	p := NewParser("igor", "1.0.0", "unknown", "unknown")

	version := p.VersionString()

	assert.Contains(t, version, "igor")
	assert.Contains(t, version, "1.0.0")
	// Should not contain "unknown" in visible output for build time/commit
	assert.NotContains(t, version, "Build time: unknown")
	assert.NotContains(t, version, "Git commit: unknown")
}

func TestVersionInfo(t *testing.T) {
	p := NewParser("igor", "1.0.0", "2024-01-01", "abc123")

	info := p.VersionInfo()

	assert.Equal(t, "1.0.0", info["version"])
	assert.Equal(t, "2024-01-01", info["buildTime"])
	assert.Equal(t, "abc123", info["gitCommit"])
}

// ============================================================================
// GlobalFlags Validation Tests
// ============================================================================

func TestGlobalFlagsValidate(t *testing.T) {
	tests := []struct {
		name    string
		flags   GlobalFlags
		wantErr bool
	}{
		{
			name:    "empty flags",
			flags:   GlobalFlags{},
			wantErr: false,
		},
		{
			name:    "verbose only",
			flags:   GlobalFlags{Verbose: true},
			wantErr: false,
		},
		{
			name:    "quiet only",
			flags:   GlobalFlags{Quiet: true},
			wantErr: false,
		},
		{
			name:    "verbose and quiet",
			flags:   GlobalFlags{Verbose: true, Quiet: true},
			wantErr: true,
		},
		{
			name:    "all non-conflicting",
			flags:   GlobalFlags{Verbose: true, DryRun: true, NoColor: true},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.flags.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// FlagError Tests
// ============================================================================

func TestFlagErrorError(t *testing.T) {
	err := &FlagError{
		Flag:    "verbose",
		Message: "test error message",
	}

	assert.Equal(t, "flag error: verbose: test error message", err.Error())
}

// ============================================================================
// Edge Cases and Error Handling Tests
// ============================================================================

func TestParseOnlyFlags(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"-v", "--dry-run"})

	require.NoError(t, err)
	assert.True(t, result.ShowHelp)
	assert.True(t, result.GlobalFlags.Verbose)
	assert.True(t, result.GlobalFlags.DryRun)
}

func TestParseInvalidGlobalFlag(t *testing.T) {
	p := newTestParser()
	_, err := p.Parse([]string{"--invalid-flag", "install"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid global flags")
}

func TestParseInvalidInstallFlag(t *testing.T) {
	p := newTestParser()
	_, err := p.Parse([]string{"install", "--invalid-flag"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid install flags")
}

func TestParseInvalidUninstallFlag(t *testing.T) {
	p := newTestParser()
	_, err := p.Parse([]string{"uninstall", "--invalid-flag"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid uninstall flags")
}

func TestParseInvalidDetectFlag(t *testing.T) {
	p := newTestParser()
	_, err := p.Parse([]string{"detect", "--invalid-flag"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid detect flags")
}

func TestParseInvalidListFlag(t *testing.T) {
	p := newTestParser()
	_, err := p.Parse([]string{"list", "--invalid-flag"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid list flags")
}

func TestParseVersionWithExtraArgs(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"version", "extra", "args"})

	require.NoError(t, err)
	assert.Equal(t, CommandVersion, result.Command)
	assert.Equal(t, []string{"extra", "args"}, result.Args)
}

func TestParsePositionalArgs(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{"install", "--driver", "535.104", "extra-arg"})

	require.NoError(t, err)
	assert.Equal(t, "535.104", result.InstallFlags.DriverVersion)
	assert.Equal(t, []string{"extra-arg"}, result.Args)
}

// ============================================================================
// Comprehensive Integration Tests
// ============================================================================

func TestFullInstallWorkflow(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{
		"-v",
		"--dry-run",
		"-c", "/etc/igor/config.yaml",
		"--log-level", "debug",
		"install",
		"--driver", "535.104",
		"--with-cuda",
		"--force",
	})

	require.NoError(t, err)

	// Global flags
	assert.True(t, result.GlobalFlags.Verbose)
	assert.True(t, result.GlobalFlags.DryRun)
	assert.Equal(t, "/etc/igor/config.yaml", result.GlobalFlags.ConfigFile)
	assert.Equal(t, "debug", result.GlobalFlags.LogLevel)

	// Command
	assert.Equal(t, CommandInstall, result.Command)

	// Install flags
	assert.Equal(t, "535.104", result.InstallFlags.DriverVersion)
	assert.True(t, result.InstallFlags.InstallCUDA)
	assert.True(t, result.InstallFlags.Force)
}

func TestFullDetectWorkflow(t *testing.T) {
	p := newTestParser()
	result, err := p.Parse([]string{
		"--no-color",
		"-q",
		"detect",
		"--json",
	})

	require.NoError(t, err)

	assert.True(t, result.GlobalFlags.NoColor)
	assert.True(t, result.GlobalFlags.Quiet)
	assert.Equal(t, CommandDetect, result.Command)
	assert.True(t, result.DetectFlags.JSON)
}

// ============================================================================
// Help Flag Tests
// ============================================================================

func TestParseHelpFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		showHelp bool
	}{
		{"--help flag", []string{"--help"}, true},
		{"-h flag", []string{"-h"}, true},
		{"-help flag", []string{"-help"}, true},
		{"--help with command", []string{"--help", "install"}, true},
		{"command then --help", []string{"install", "--help"}, true},
		{"no help flag", []string{"install"}, false},
	}

	p := newTestParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Parse(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.showHelp, result.ShowHelp)
		})
	}
}
