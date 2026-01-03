package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

// ParseResult holds the result of parsing command line arguments.
type ParseResult struct {
	// Command is the parsed command.
	Command Command

	// GlobalFlags contains the global flag values.
	GlobalFlags GlobalFlags

	// InstallFlags contains install command flag values.
	InstallFlags InstallFlags

	// UninstallFlags contains uninstall command flag values.
	UninstallFlags UninstallFlags

	// DetectFlags contains detect command flag values.
	DetectFlags DetectFlags

	// ListFlags contains list command flag values.
	ListFlags ListFlags

	// Args contains any remaining positional arguments.
	Args []string

	// ShowHelp indicates that help should be displayed.
	ShowHelp bool

	// HelpCommand is the command to show help for (when using "help <command>").
	HelpCommand string
}

// Parser handles command line argument parsing.
type Parser struct {
	programName string
	version     string
	buildTime   string
	gitCommit   string

	// output is where usage/help text is written (defaults to stderr equivalent behavior)
	output io.Writer
}

// NewParser creates a new CLI parser with build information.
func NewParser(programName, version, buildTime, gitCommit string) *Parser {
	return &Parser{
		programName: programName,
		version:     version,
		buildTime:   buildTime,
		gitCommit:   gitCommit,
	}
}

// SetOutput sets the output writer for usage and help messages.
func (p *Parser) SetOutput(w io.Writer) {
	p.output = w
}

// Parse parses command line arguments and returns a ParseResult.
// The args parameter should not include the program name (typically os.Args[1:]).
func (p *Parser) Parse(args []string) (*ParseResult, error) {
	result := &ParseResult{}

	if len(args) == 0 {
		result.ShowHelp = true
		return result, nil
	}

	// Check for help flags first before parsing
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "-help" {
			result.ShowHelp = true
			return result, nil
		}
	}

	// Parse global flags - the flag package will stop at the first non-flag argument
	globalFs := p.createGlobalFlagSet(&result.GlobalFlags)
	globalFs.SetOutput(io.Discard) // Suppress default error output

	if err := globalFs.Parse(args); err != nil {
		return nil, fmt.Errorf("invalid global flags: %w", err)
	}

	// Get remaining args after global flags
	remaining := globalFs.Args()

	if len(remaining) == 0 {
		// No command specified, show help
		result.ShowHelp = true
		return result, nil
	}

	// Validate global flags
	if err := result.GlobalFlags.Validate(); err != nil {
		return nil, err
	}

	// Get command (first remaining argument)
	cmdStr := remaining[0]
	result.Command = ParseCommand(cmdStr)

	if result.Command == CommandNone {
		return nil, fmt.Errorf("unknown command: %s", cmdStr)
	}

	// Parse command-specific flags
	cmdArgs := remaining[1:]
	if err := p.parseCommandFlags(result, cmdArgs); err != nil {
		return nil, err
	}

	return result, nil
}

// createGlobalFlagSet creates a FlagSet with global flag definitions.
func (p *Parser) createGlobalFlagSet(flags *GlobalFlags) *flag.FlagSet {
	fs := flag.NewFlagSet("global", flag.ContinueOnError)

	// Verbose flags
	fs.BoolVar(&flags.Verbose, "verbose", false, "Enable verbose output")
	fs.BoolVar(&flags.Verbose, "v", false, "Enable verbose output (shorthand)")

	// Quiet flags
	fs.BoolVar(&flags.Quiet, "quiet", false, "Suppress non-essential output")
	fs.BoolVar(&flags.Quiet, "q", false, "Suppress non-essential output (shorthand)")

	// Dry-run flags
	fs.BoolVar(&flags.DryRun, "dry-run", false, "Show what would be done without making changes")
	fs.BoolVar(&flags.DryRun, "n", false, "Show what would be done (shorthand)")

	// Config file flags
	fs.StringVar(&flags.ConfigFile, "config", "", "Path to config file")
	fs.StringVar(&flags.ConfigFile, "c", "", "Path to config file (shorthand)")

	// Log file
	fs.StringVar(&flags.LogFile, "log-file", "", "Path to log file")

	// Log level
	fs.StringVar(&flags.LogLevel, "log-level", "", "Log level (debug, info, warn, error)")

	// No color
	fs.BoolVar(&flags.NoColor, "no-color", false, "Disable colored output")

	return fs
}

// parseCommandFlags parses flags specific to each command.
func (p *Parser) parseCommandFlags(result *ParseResult, args []string) error {
	switch result.Command {
	case CommandInstall:
		return p.parseInstallFlags(result, args)
	case CommandUninstall:
		return p.parseUninstallFlags(result, args)
	case CommandDetect:
		return p.parseDetectFlags(result, args)
	case CommandList:
		return p.parseListFlags(result, args)
	case CommandHelp:
		return p.parseHelpFlags(result, args)
	case CommandVersion:
		// Version command has no flags
		result.Args = args
		return nil
	}
	return nil
}

func (p *Parser) parseInstallFlags(result *ParseResult, args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&result.InstallFlags.DriverVersion, "driver", "", "Driver version to install")
	fs.StringVar(&result.InstallFlags.CUDAVersion, "cuda", "", "CUDA version to install")
	fs.BoolVar(&result.InstallFlags.InstallCUDA, "with-cuda", false, "Also install CUDA toolkit")
	fs.BoolVar(&result.InstallFlags.Force, "force", false, "Force installation even if already installed")
	fs.BoolVar(&result.InstallFlags.Force, "f", false, "Force installation (shorthand)")
	fs.BoolVar(&result.InstallFlags.SkipReboot, "skip-reboot", false, "Don't prompt for reboot")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("invalid install flags: %w", err)
	}
	result.Args = fs.Args()
	return nil
}

func (p *Parser) parseUninstallFlags(result *ParseResult, args []string) error {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.BoolVar(&result.UninstallFlags.Purge, "purge", false, "Remove configuration files too")
	fs.BoolVar(&result.UninstallFlags.KeepConfig, "keep-config", false, "Keep configuration files")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("invalid uninstall flags: %w", err)
	}
	result.Args = fs.Args()
	return nil
}

func (p *Parser) parseDetectFlags(result *ParseResult, args []string) error {
	fs := flag.NewFlagSet("detect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.BoolVar(&result.DetectFlags.JSON, "json", false, "Output in JSON format")
	fs.BoolVar(&result.DetectFlags.Brief, "brief", false, "Show brief output")
	fs.BoolVar(&result.DetectFlags.Brief, "b", false, "Show brief output (shorthand)")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("invalid detect flags: %w", err)
	}
	result.Args = fs.Args()
	return nil
}

func (p *Parser) parseListFlags(result *ParseResult, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.BoolVar(&result.ListFlags.Installed, "installed", false, "Show only installed drivers")
	fs.BoolVar(&result.ListFlags.Available, "available", false, "Show only available drivers")
	fs.BoolVar(&result.ListFlags.JSON, "json", false, "Output in JSON format")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("invalid list flags: %w", err)
	}
	result.Args = fs.Args()
	return nil
}

func (p *Parser) parseHelpFlags(result *ParseResult, args []string) error {
	result.ShowHelp = true
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		result.HelpCommand = args[0]
	}
	return nil
}

// Usage returns the main usage string.
func (p *Parser) Usage() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("%s - NVIDIA TUI Installer for Linux\n\n", p.programName))
	b.WriteString("Usage:\n")
	b.WriteString(fmt.Sprintf("  %s [global flags] <command> [command flags]\n\n", p.programName))

	b.WriteString("Commands:\n")
	for _, cmd := range Commands() {
		b.WriteString(fmt.Sprintf("  %-12s %s\n", cmd.Name, cmd.Description))
	}

	b.WriteString("\nGlobal Flags:\n")
	b.WriteString("  -v, --verbose     Enable verbose output\n")
	b.WriteString("  -q, --quiet       Suppress non-essential output\n")
	b.WriteString("  -n, --dry-run     Show what would be done without making changes\n")
	b.WriteString("  -c, --config      Path to config file\n")
	b.WriteString("      --log-file    Path to log file\n")
	b.WriteString("      --log-level   Log level (debug, info, warn, error)\n")
	b.WriteString("      --no-color    Disable colored output\n")

	b.WriteString(fmt.Sprintf("\nUse \"%s help <command>\" for more information about a command.\n", p.programName))

	return b.String()
}

// CommandUsage returns the usage string for a specific command.
func (p *Parser) CommandUsage(cmd string) string {
	// Check if it's a valid command
	parsedCmd := ParseCommand(cmd)
	if parsedCmd == CommandNone {
		return fmt.Sprintf("Unknown command: %s\n\nRun '%s help' for usage.\n", cmd, p.programName)
	}

	info := GetCommandInfo(parsedCmd)
	if info == nil {
		return fmt.Sprintf("No help available for: %s\n", cmd)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s\n\n", info.Description))
	b.WriteString(fmt.Sprintf("Usage:\n  %s\n\n", info.Usage))

	if info.LongDescription != "" {
		b.WriteString(info.LongDescription)
		b.WriteString("\n")
	}

	return b.String()
}

// VersionString returns formatted version information.
func (p *Parser) VersionString() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("%s version %s\n", p.programName, p.version))

	if p.buildTime != "" && p.buildTime != "unknown" {
		b.WriteString(fmt.Sprintf("Build time: %s\n", p.buildTime))
	}

	if p.gitCommit != "" && p.gitCommit != "unknown" {
		commit := p.gitCommit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		b.WriteString(fmt.Sprintf("Git commit: %s\n", commit))
	}

	return b.String()
}

// VersionInfo returns version components for structured output.
func (p *Parser) VersionInfo() map[string]string {
	return map[string]string{
		"version":   p.version,
		"buildTime": p.buildTime,
		"gitCommit": p.gitCommit,
	}
}
