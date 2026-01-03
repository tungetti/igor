// Package cli provides command-line argument parsing for the Igor application.
// It supports subcommands, global flags, and command-specific flags with both
// short and long variants. The parser integrates with the config package to
// provide a unified configuration experience.
package cli

// GlobalFlags holds flags common to all commands.
// These flags can be specified before the command name and affect
// the overall behavior of the application.
type GlobalFlags struct {
	// Verbose enables detailed output for debugging and troubleshooting.
	Verbose bool

	// Quiet suppresses non-essential output, only showing errors.
	Quiet bool

	// DryRun shows what would be done without making actual changes.
	DryRun bool

	// ConfigFile specifies a custom configuration file path.
	ConfigFile string

	// LogFile specifies the path to write log output.
	LogFile string

	// LogLevel sets the logging verbosity (debug, info, warn, error).
	LogLevel string

	// NoColor disables colored terminal output.
	NoColor bool
}

// InstallFlags holds install command specific flags.
// These flags control the driver installation process.
type InstallFlags struct {
	// DriverVersion specifies the exact driver version to install.
	DriverVersion string

	// CUDAVersion specifies the CUDA toolkit version to install.
	CUDAVersion string

	// InstallCUDA indicates whether to also install the CUDA toolkit.
	InstallCUDA bool

	// Force forces installation even if the driver is already installed.
	Force bool

	// SkipReboot skips the reboot prompt after installation.
	SkipReboot bool
}

// UninstallFlags holds uninstall command specific flags.
// These flags control the driver removal process.
type UninstallFlags struct {
	// Purge removes configuration files in addition to packages.
	Purge bool

	// KeepConfig preserves configuration files during uninstallation.
	KeepConfig bool
}

// DetectFlags holds detect command specific flags.
// These flags control the GPU detection output format.
type DetectFlags struct {
	// JSON outputs detection results in JSON format.
	JSON bool

	// Brief shows a condensed summary of detected hardware.
	Brief bool
}

// ListFlags holds list command specific flags.
// These flags filter and format the driver listing output.
type ListFlags struct {
	// Installed shows only drivers that are currently installed.
	Installed bool

	// Available shows only drivers available for installation.
	Available bool

	// JSON outputs the list in JSON format.
	JSON bool
}

// Validate checks GlobalFlags for conflicting options.
// It returns an error if incompatible flags are set together.
func (f *GlobalFlags) Validate() error {
	if f.Verbose && f.Quiet {
		return &FlagError{
			Flag:    "verbose/quiet",
			Message: "cannot use --verbose and --quiet together",
		}
	}
	return nil
}

// FlagError represents an error with a command-line flag.
type FlagError struct {
	Flag    string
	Message string
}

// Error implements the error interface.
func (e *FlagError) Error() string {
	return "flag error: " + e.Flag + ": " + e.Message
}
