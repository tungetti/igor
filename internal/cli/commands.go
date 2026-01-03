package cli

// Command represents a CLI command type.
type Command int

const (
	// CommandNone represents no command or an unrecognized command.
	CommandNone Command = iota

	// CommandInstall represents the install command for installing NVIDIA drivers.
	CommandInstall

	// CommandUninstall represents the uninstall command for removing NVIDIA drivers.
	CommandUninstall

	// CommandDetect represents the detect command for GPU hardware detection.
	CommandDetect

	// CommandList represents the list command for showing available/installed drivers.
	CommandList

	// CommandVersion represents the version command for displaying build information.
	CommandVersion

	// CommandHelp represents the help command for showing usage information.
	CommandHelp
)

// String returns the command name as a string.
func (c Command) String() string {
	switch c {
	case CommandInstall:
		return "install"
	case CommandUninstall:
		return "uninstall"
	case CommandDetect:
		return "detect"
	case CommandList:
		return "list"
	case CommandVersion:
		return "version"
	case CommandHelp:
		return "help"
	default:
		return ""
	}
}

// IsValid returns true if the command is a recognized command.
func (c Command) IsValid() bool {
	return c > CommandNone && c <= CommandHelp
}

// CommandInfo holds metadata about a command.
type CommandInfo struct {
	// Name is the primary command name.
	Name string

	// Aliases are alternative names for the command.
	Aliases []string

	// Description is a brief description of what the command does.
	Description string

	// Usage shows how to invoke the command.
	Usage string

	// LongDescription provides detailed help text for the command.
	LongDescription string
}

// Commands returns all available commands with their metadata.
func Commands() []CommandInfo {
	return []CommandInfo{
		{
			Name:        "install",
			Aliases:     []string{"i"},
			Description: "Install NVIDIA drivers and optionally CUDA toolkit",
			Usage:       "igor install [flags]",
			LongDescription: `Install NVIDIA drivers on your system.

By default, this command will detect your GPU and install the recommended
driver version. You can specify a particular version with --driver.

Flags:
  --driver VERSION    Install a specific driver version
  --cuda VERSION      Install CUDA toolkit with specified version
  --with-cuda         Also install CUDA toolkit (latest compatible version)
  --force             Force installation even if driver is already installed
  --skip-reboot       Don't prompt for reboot after installation

Examples:
  igor install                     Install recommended driver
  igor install --driver 535.104    Install specific driver version
  igor install --with-cuda         Install driver and CUDA toolkit`,
		},
		{
			Name:        "uninstall",
			Aliases:     []string{"u", "remove"},
			Description: "Remove NVIDIA drivers and related packages",
			Usage:       "igor uninstall [flags]",
			LongDescription: `Remove NVIDIA drivers from your system.

This command will remove the installed NVIDIA drivers and related packages.
Use --purge to also remove configuration files.

Flags:
  --purge         Remove configuration files too
  --keep-config   Keep configuration files (default behavior)

Examples:
  igor uninstall          Remove drivers, keep configuration
  igor uninstall --purge  Remove drivers and all configuration`,
		},
		{
			Name:        "detect",
			Aliases:     []string{"d"},
			Description: "Detect NVIDIA GPUs and current driver status",
			Usage:       "igor detect [flags]",
			LongDescription: `Detect NVIDIA GPUs installed in your system.

This command will scan for NVIDIA graphics cards and display information
about the hardware and currently installed drivers.

Flags:
  --json    Output detection results in JSON format
  --brief   Show condensed summary output

Examples:
  igor detect         Show detailed GPU information
  igor detect --json  Output as JSON for scripting
  igor detect --brief Show brief summary`,
		},
		{
			Name:        "list",
			Aliases:     []string{"l", "ls"},
			Description: "List available or installed driver versions",
			Usage:       "igor list [flags]",
			LongDescription: `List NVIDIA driver versions.

By default, this command shows all available driver versions for your GPU.
Use flags to filter the output.

Flags:
  --installed   Show only installed drivers
  --available   Show only available drivers (default)
  --json        Output list in JSON format

Examples:
  igor list              List all available drivers
  igor list --installed  Show currently installed driver
  igor list --json       Output as JSON for scripting`,
		},
		{
			Name:        "version",
			Aliases:     []string{"v"},
			Description: "Show version information",
			Usage:       "igor version",
			LongDescription: `Display version information about igor.

Shows the version number, build time, and git commit hash.`,
		},
		{
			Name:        "help",
			Aliases:     []string{"h"},
			Description: "Show help for a command",
			Usage:       "igor help [command]",
			LongDescription: `Display help information.

When called without arguments, shows general help and available commands.
When called with a command name, shows detailed help for that command.

Examples:
  igor help          Show general help
  igor help install  Show help for install command`,
		},
	}
}

// GetCommandInfo returns the CommandInfo for a given command.
// Returns nil if the command is not found.
func GetCommandInfo(cmd Command) *CommandInfo {
	if !cmd.IsValid() {
		return nil
	}

	cmds := Commands()
	for i := range cmds {
		if cmds[i].Name == cmd.String() {
			return &cmds[i]
		}
	}
	return nil
}

// ParseCommand parses a string into a Command.
// It recognizes both primary command names and aliases.
func ParseCommand(s string) Command {
	// Check each command's name and aliases
	for _, info := range Commands() {
		if s == info.Name {
			return commandFromName(info.Name)
		}
		for _, alias := range info.Aliases {
			if s == alias {
				return commandFromName(info.Name)
			}
		}
	}
	return CommandNone
}

// commandFromName converts a command name string to a Command type.
func commandFromName(name string) Command {
	switch name {
	case "install":
		return CommandInstall
	case "uninstall":
		return CommandUninstall
	case "detect":
		return CommandDetect
	case "list":
		return CommandList
	case "version":
		return CommandVersion
	case "help":
		return CommandHelp
	default:
		return CommandNone
	}
}
