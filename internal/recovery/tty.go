// Package recovery provides TTY-based recovery mode for Igor.
// This file implements the TTY UI for emergency uninstallation when X.org fails.
package recovery

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// ASCII symbols for TTY compatibility (no Unicode - TTY fonts are limited).
const (
	SymbolSuccess = "[OK]"
	SymbolFailed  = "[FAIL]"
	SymbolPending = "[..]"
	SymbolWarning = "[!]"
	SymbolInfo    = "[i]"
)

// TTYUI provides a minimal text-based UI for TTY environments.
// It uses simple ASCII characters only for maximum compatibility with
// virtual console fonts that may not support Unicode.
type TTYUI struct {
	reader    *bufio.Reader
	writer    io.Writer
	width     int
	rawReader io.Reader
}

// TTYUIOption is a functional option for TTYUI.
type TTYUIOption func(*TTYUI)

// NewTTYUI creates a new TTY UI instance with the given options.
// By default, it reads from stdin and writes to stdout with 80-character width.
func NewTTYUI(opts ...TTYUIOption) *TTYUI {
	u := &TTYUI{
		rawReader: os.Stdin,
		writer:    os.Stdout,
		width:     80,
	}

	for _, opt := range opts {
		opt(u)
	}

	// Create buffered reader from raw reader
	u.reader = bufio.NewReader(u.rawReader)

	return u
}

// WithTTYReader sets the input reader for the TTY UI.
func WithTTYReader(r io.Reader) TTYUIOption {
	return func(u *TTYUI) {
		u.rawReader = r
	}
}

// WithTTYWriter sets the output writer for the TTY UI.
func WithTTYWriter(w io.Writer) TTYUIOption {
	return func(u *TTYUI) {
		u.writer = w
	}
}

// WithTTYWidth sets the terminal width for the TTY UI.
// Default is 80 characters.
func WithTTYWidth(width int) TTYUIOption {
	return func(u *TTYUI) {
		if width > 0 {
			u.width = width
		}
	}
}

// Header prints a section header with emphasis.
// The header is displayed with a separator line above and below.
func (u *TTYUI) Header(text string) {
	u.Separator()
	// Center the text if it fits
	padding := (u.width - len(text)) / 2
	if padding > 0 {
		fmt.Fprintf(u.writer, "%s%s\n", strings.Repeat(" ", padding), text)
	} else {
		fmt.Fprintln(u.writer, text)
	}
	u.Separator()
}

// Status prints a status line with the given symbol.
func (u *TTYUI) Status(symbol, text string) {
	// Truncate text if too long (leave room for symbol and space)
	maxTextLen := u.width - len(symbol) - 1
	if len(text) > maxTextLen && maxTextLen > 3 {
		text = text[:maxTextLen-3] + "..."
	}
	fmt.Fprintf(u.writer, "%s %s\n", symbol, text)
}

// Success prints a success message with [OK] symbol.
func (u *TTYUI) Success(text string) {
	u.Status(SymbolSuccess, text)
}

// Error prints an error message with [FAIL] symbol.
func (u *TTYUI) Error(text string) {
	u.Status(SymbolFailed, text)
}

// Warning prints a warning message with [!] symbol.
func (u *TTYUI) Warning(text string) {
	u.Status(SymbolWarning, text)
}

// Info prints an info message with [i] symbol.
func (u *TTYUI) Info(text string) {
	u.Status(SymbolInfo, text)
}

// Pending prints a pending/in-progress message with [..] symbol.
func (u *TTYUI) Pending(text string) {
	u.Status(SymbolPending, text)
}

// Progress prints a progress update that can overwrite the current line.
// Uses \r to return to beginning of line for in-place updates.
// After the final update, caller should print a newline.
func (u *TTYUI) Progress(current, total int, text string) {
	if total <= 0 {
		total = 1
	}
	percent := (current * 100) / total

	// Create progress bar
	barWidth := 20
	filled := (current * barWidth) / total
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)

	// Truncate text to fit
	maxTextLen := u.width - 30 // Leave room for [XXX%] [##--] and some padding
	if len(text) > maxTextLen && maxTextLen > 3 {
		text = text[:maxTextLen-3] + "..."
	}

	// Use \r to overwrite line (works in TTY)
	fmt.Fprintf(u.writer, "\r[%3d%%] [%s] %s", percent, bar, text)

	// Pad with spaces to clear any previous longer text
	clearLen := u.width - 30 - len(text)
	if clearLen > 0 {
		fmt.Fprintf(u.writer, "%s", strings.Repeat(" ", clearLen))
	}
}

// ProgressComplete finishes a progress line by printing a newline.
func (u *TTYUI) ProgressComplete() {
	fmt.Fprintln(u.writer)
}

// Separator prints a horizontal line made of dashes.
func (u *TTYUI) Separator() {
	fmt.Fprintln(u.writer, strings.Repeat("-", u.width))
}

// Blank prints an empty line.
func (u *TTYUI) Blank() {
	fmt.Fprintln(u.writer)
}

// Print prints text without any formatting.
func (u *TTYUI) Print(text string) {
	fmt.Fprintln(u.writer, text)
}

// Printf prints formatted text.
func (u *TTYUI) Printf(format string, args ...interface{}) {
	fmt.Fprintf(u.writer, format, args...)
}

// Confirm asks for yes/no confirmation from the user.
// Returns true for yes, false for no.
// defaultYes determines what happens when the user presses Enter without input.
func (u *TTYUI) Confirm(prompt string, defaultYes bool) bool {
	defaultHint := "[y/N]"
	if defaultYes {
		defaultHint = "[Y/n]"
	}

	fmt.Fprintf(u.writer, "%s %s: ", prompt, defaultHint)

	input, err := u.reader.ReadString('\n')
	if err != nil {
		// EOF or error - return default
		fmt.Fprintln(u.writer)
		return defaultYes
	}

	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	case "":
		return defaultYes
	default:
		// Unrecognized input - ask again
		u.Warning("Please enter 'y' or 'n'")
		return u.Confirm(prompt, defaultYes)
	}
}

// ConfirmList shows a list of items and asks for confirmation.
// Returns true if the user confirms, false otherwise.
func (u *TTYUI) ConfirmList(header string, items []string, prompt string, defaultYes bool) bool {
	if header != "" {
		u.Print(header)
	}

	// Show items with bullet points
	for _, item := range items {
		fmt.Fprintf(u.writer, "  * %s\n", item)
	}

	u.Blank()

	return u.Confirm(prompt, defaultYes)
}

// ShowPackages displays a list of discovered packages grouped by type.
func (u *TTYUI) ShowPackages(packages []string) {
	if len(packages) == 0 {
		u.Warning("No packages found")
		return
	}

	u.Info(fmt.Sprintf("Found %d package(s):", len(packages)))
	u.Blank()

	// Display packages in columns if there are many
	const columnsThreshold = 5
	const columnWidth = 35

	if len(packages) <= columnsThreshold {
		// Simple list for few packages
		for _, pkg := range packages {
			fmt.Fprintf(u.writer, "  - %s\n", pkg)
		}
	} else {
		// Two-column layout for many packages
		numCols := 2
		if u.width < columnWidth*2 {
			numCols = 1
		}

		for i := 0; i < len(packages); i += numCols {
			for j := 0; j < numCols && i+j < len(packages); j++ {
				pkg := packages[i+j]
				// Truncate if needed
				if len(pkg) > columnWidth-3 {
					pkg = pkg[:columnWidth-6] + "..."
				}
				if j == 0 {
					fmt.Fprintf(u.writer, "  - %-*s", columnWidth-4, pkg)
				} else {
					fmt.Fprintf(u.writer, "  - %s", pkg)
				}
			}
			fmt.Fprintln(u.writer)
		}
	}

	u.Blank()
}

// ShowResult displays the final result of an operation.
// success indicates whether the operation succeeded.
// message is the main result message.
// details provides additional information lines.
func (u *TTYUI) ShowResult(success bool, message string, details []string) {
	u.Separator()

	if success {
		u.Success(message)
	} else {
		u.Error(message)
	}

	if len(details) > 0 {
		u.Blank()
		for _, detail := range details {
			fmt.Fprintf(u.writer, "  %s\n", detail)
		}
	}

	u.Separator()
}

// ShowError displays an error with optional recovery suggestions.
func (u *TTYUI) ShowError(message string, suggestions []string) {
	u.Separator()
	u.Error("ERROR: " + message)

	if len(suggestions) > 0 {
		u.Blank()
		u.Print("Suggestions:")
		for _, s := range suggestions {
			fmt.Fprintf(u.writer, "  * %s\n", s)
		}
	}

	u.Separator()
}

// ShowWarning displays a warning box.
func (u *TTYUI) ShowWarning(message string) {
	u.Separator()
	u.Warning("WARNING: " + message)
	u.Separator()
}

// ShowStep displays a step being executed with its number.
func (u *TTYUI) ShowStep(stepNum, totalSteps int, description string) {
	fmt.Fprintf(u.writer, "[%d/%d] %s\n", stepNum, totalSteps, description)
}

// StepSuccess marks a step as successful.
func (u *TTYUI) StepSuccess(stepNum, totalSteps int, description string) {
	fmt.Fprintf(u.writer, "[%d/%d] %s %s\n", stepNum, totalSteps, SymbolSuccess, description)
}

// StepFailed marks a step as failed.
func (u *TTYUI) StepFailed(stepNum, totalSteps int, description string) {
	fmt.Fprintf(u.writer, "[%d/%d] %s %s\n", stepNum, totalSteps, SymbolFailed, description)
}

// ReadLine reads a line of input from the user.
// Returns the input string (trimmed) and any error.
func (u *TTYUI) ReadLine(prompt string) (string, error) {
	if prompt != "" {
		fmt.Fprint(u.writer, prompt)
	}

	input, err := u.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(input), nil
}

// Width returns the configured terminal width.
func (u *TTYUI) Width() int {
	return u.width
}
