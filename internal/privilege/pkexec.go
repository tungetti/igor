package privilege

import (
	"context"
	"os/exec"
)

// PkexecCommand builds a pkexec command for running with elevated privileges.
// Unlike sudo, pkexec uses PolicyKit for authentication and may display
// a graphical prompt on systems with a display server.
func PkexecCommand(cmd string, args []string) *exec.Cmd {
	pkexecArgs := append([]string{cmd}, args...)
	return exec.Command("pkexec", pkexecArgs...)
}

// PkexecCommandContext builds a pkexec command with context for timeout/cancellation.
// The command will be killed if the context is cancelled or times out.
func PkexecCommandContext(ctx context.Context, cmd string, args []string) *exec.Cmd {
	pkexecArgs := append([]string{cmd}, args...)
	return exec.CommandContext(ctx, "pkexec", pkexecArgs...)
}

// CheckPkexecAvailable checks if pkexec is available on the system.
func CheckPkexecAvailable() bool {
	_, err := exec.LookPath("pkexec")
	return err == nil
}

// GetPkexecPath returns the path to pkexec if available, empty string otherwise.
func GetPkexecPath() string {
	path, err := exec.LookPath("pkexec")
	if err != nil {
		return ""
	}
	return path
}
