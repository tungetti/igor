package privilege

import (
	"context"
	"os/exec"
	"time"
)

// SudoOptions configures sudo behavior.
type SudoOptions struct {
	// NonInteractive uses the -n flag to avoid prompting for password.
	NonInteractive bool
	// PreserveEnv uses the -E flag to preserve environment variables.
	PreserveEnv bool
	// User specifies a user to run the command as (uses -u flag).
	User string
	// Timeout is the maximum duration for the command.
	Timeout time.Duration
}

// DefaultSudoOptions returns sensible defaults for sudo execution.
// Non-interactive mode is enabled by default to avoid hanging on password prompts.
func DefaultSudoOptions() SudoOptions {
	return SudoOptions{
		NonInteractive: true,
		PreserveEnv:    false,
		Timeout:        2 * time.Minute,
	}
}

// buildSudoArgs builds the sudo argument list based on options.
func buildSudoArgs(cmd string, args []string, opts SudoOptions) []string {
	sudoArgs := []string{}

	if opts.NonInteractive {
		sudoArgs = append(sudoArgs, "-n")
	}
	if opts.PreserveEnv {
		sudoArgs = append(sudoArgs, "-E")
	}
	if opts.User != "" {
		sudoArgs = append(sudoArgs, "-u", opts.User)
	}

	sudoArgs = append(sudoArgs, cmd)
	sudoArgs = append(sudoArgs, args...)

	return sudoArgs
}

// SudoCommand builds a sudo command with the specified options.
// The returned *exec.Cmd can be customized further before execution.
func SudoCommand(cmd string, args []string, opts SudoOptions) *exec.Cmd {
	sudoArgs := buildSudoArgs(cmd, args, opts)
	return exec.Command("sudo", sudoArgs...)
}

// SudoCommandContext builds a sudo command with context for timeout/cancellation.
// The command will be killed if the context is cancelled or times out.
func SudoCommandContext(ctx context.Context, cmd string, args []string, opts SudoOptions) *exec.Cmd {
	sudoArgs := buildSudoArgs(cmd, args, opts)
	return exec.CommandContext(ctx, "sudo", sudoArgs...)
}

// CheckSudoAccess checks if sudo can run without a password prompt.
// This is useful to verify if the user has passwordless sudo configured
// or if their credentials are cached.
func CheckSudoAccess() bool {
	cmd := exec.Command("sudo", "-n", "true")
	return cmd.Run() == nil
}

// CheckSudoAvailable checks if sudo is available on the system.
func CheckSudoAvailable() bool {
	_, err := exec.LookPath("sudo")
	return err == nil
}

// SudoValidate checks if the sudo credentials are currently cached.
// Unlike CheckSudoAccess, this uses 'sudo -v -n' which validates
// the user's sudo credentials without running a command.
func SudoValidate() bool {
	cmd := exec.Command("sudo", "-v", "-n")
	return cmd.Run() == nil
}
