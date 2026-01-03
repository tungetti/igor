package main

import (
	"fmt"
	"os"

	"github.com/tungetti/igor/internal/cli"
	"github.com/tungetti/igor/internal/config"
	"github.com/tungetti/igor/internal/constants"
)

// CLI encapsulates the command-line interface for Igor.
type CLI struct {
	parser *cli.Parser
	config *config.Config
}

// NewCLI creates a new CLI instance.
func NewCLI() *CLI {
	return &CLI{
		parser: cli.NewParser(constants.AppName, Version, BuildTime, GitCommit),
	}
}

// Run parses arguments and executes the appropriate command.
// It returns an exit code suitable for os.Exit().
func (c *CLI) Run(args []string) int {
	result, err := c.parser.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run '%s help' for usage.\n", constants.AppName)
		return constants.ExitValidation.Int()
	}

	// Load configuration
	if err := c.loadConfig(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return constants.ExitError.Int()
	}

	// Apply global flags to config
	c.applyGlobalFlags(result.GlobalFlags)

	// Show help if requested
	if result.ShowHelp {
		return c.showHelp(result)
	}

	// Execute command
	return c.executeCommand(result)
}

// loadConfig loads configuration from file and environment.
func (c *CLI) loadConfig(result *cli.ParseResult) error {
	configPath := result.GlobalFlags.ConfigFile
	if configPath == "" {
		// Use default config path
		configPath = config.DefaultConfig().ConfigPath()
	}

	loader := config.NewLoader(configPath)
	cfg, err := loader.Load()
	if err != nil {
		return err
	}

	c.config = cfg
	return nil
}

// applyGlobalFlags applies CLI global flags to the configuration.
// CLI flags take precedence over config file values.
func (c *CLI) applyGlobalFlags(flags cli.GlobalFlags) {
	if flags.Verbose {
		c.config.Verbose = true
	}
	if flags.Quiet {
		c.config.Quiet = true
	}
	if flags.DryRun {
		c.config.DryRun = true
	}
	if flags.LogFile != "" {
		c.config.LogFile = flags.LogFile
	}
	if flags.LogLevel != "" {
		c.config.LogLevel = flags.LogLevel
	}
}

// showHelp displays help information and returns an exit code.
func (c *CLI) showHelp(result *cli.ParseResult) int {
	if result.HelpCommand != "" {
		fmt.Print(c.parser.CommandUsage(result.HelpCommand))
	} else {
		fmt.Print(c.parser.Usage())
	}
	return constants.ExitSuccess.Int()
}

// executeCommand runs the appropriate command handler.
func (c *CLI) executeCommand(result *cli.ParseResult) int {
	switch result.Command {
	case cli.CommandVersion:
		return c.cmdVersion()
	case cli.CommandInstall:
		return c.cmdInstall(result)
	case cli.CommandUninstall:
		return c.cmdUninstall(result)
	case cli.CommandDetect:
		return c.cmdDetect(result)
	case cli.CommandList:
		return c.cmdList(result)
	default:
		fmt.Print(c.parser.Usage())
		return constants.ExitSuccess.Int()
	}
}

// cmdVersion displays version information.
func (c *CLI) cmdVersion() int {
	fmt.Print(c.parser.VersionString())
	return constants.ExitSuccess.Int()
}

// cmdInstall handles the install command.
// TODO: Implement actual installation logic in future sprints.
func (c *CLI) cmdInstall(result *cli.ParseResult) int {
	if c.config.IsVerbose() {
		fmt.Println("Install command called")
		fmt.Printf("  Driver version: %s\n", result.InstallFlags.DriverVersion)
		fmt.Printf("  CUDA version: %s\n", result.InstallFlags.CUDAVersion)
		fmt.Printf("  Install CUDA: %v\n", result.InstallFlags.InstallCUDA)
		fmt.Printf("  Force: %v\n", result.InstallFlags.Force)
		fmt.Printf("  Skip reboot: %v\n", result.InstallFlags.SkipReboot)
		fmt.Printf("  Dry run: %v\n", c.config.DryRun)
	}

	if c.config.DryRun {
		fmt.Println("[dry-run] Would install NVIDIA drivers")
		return constants.ExitSuccess.Int()
	}

	// Placeholder for actual implementation
	fmt.Println("Install command not yet implemented")
	return constants.ExitSuccess.Int()
}

// cmdUninstall handles the uninstall command.
// TODO: Implement actual uninstallation logic in future sprints.
func (c *CLI) cmdUninstall(result *cli.ParseResult) int {
	if c.config.IsVerbose() {
		fmt.Println("Uninstall command called")
		fmt.Printf("  Purge: %v\n", result.UninstallFlags.Purge)
		fmt.Printf("  Keep config: %v\n", result.UninstallFlags.KeepConfig)
		fmt.Printf("  Dry run: %v\n", c.config.DryRun)
	}

	if c.config.DryRun {
		fmt.Println("[dry-run] Would uninstall NVIDIA drivers")
		return constants.ExitSuccess.Int()
	}

	// Placeholder for actual implementation
	fmt.Println("Uninstall command not yet implemented")
	return constants.ExitSuccess.Int()
}

// cmdDetect handles the detect command.
// TODO: Implement actual GPU detection logic in future sprints.
func (c *CLI) cmdDetect(result *cli.ParseResult) int {
	if c.config.IsVerbose() {
		fmt.Println("Detect command called")
		fmt.Printf("  JSON output: %v\n", result.DetectFlags.JSON)
		fmt.Printf("  Brief: %v\n", result.DetectFlags.Brief)
	}

	// Placeholder for actual implementation
	fmt.Println("Detect command not yet implemented")
	return constants.ExitSuccess.Int()
}

// cmdList handles the list command.
// TODO: Implement actual driver listing logic in future sprints.
func (c *CLI) cmdList(result *cli.ParseResult) int {
	if c.config.IsVerbose() {
		fmt.Println("List command called")
		fmt.Printf("  Installed: %v\n", result.ListFlags.Installed)
		fmt.Printf("  Available: %v\n", result.ListFlags.Available)
		fmt.Printf("  JSON output: %v\n", result.ListFlags.JSON)
	}

	// Placeholder for actual implementation
	fmt.Println("List command not yet implemented")
	return constants.ExitSuccess.Int()
}
