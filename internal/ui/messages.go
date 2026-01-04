package ui

import tea "github.com/charmbracelet/bubbletea"

// QuitMsg signals the application should quit.
type QuitMsg struct{}

// ErrorMsg carries an error to be displayed.
type ErrorMsg struct {
	Err error
}

// NavigateMsg signals navigation to a different view.
type NavigateMsg struct {
	View ViewState
}

// WindowReadyMsg signals the window is ready with its dimensions.
type WindowReadyMsg struct {
	Width  int
	Height int
}

// TickMsg is used for animations and progress updates.
type TickMsg struct{}

// StatusMsg carries a status update message.
type StatusMsg struct {
	Message string
	IsError bool
}

// ProgressMsg carries progress information.
type ProgressMsg struct {
	Current int
	Total   int
	Message string
}

// DetectionCompleteMsg signals that system detection is complete.
type DetectionCompleteMsg struct {
	Success bool
}

// InstallationCompleteMsg signals that installation is complete.
type InstallationCompleteMsg struct {
	Success bool
	Message string
}

// Command constructors

// Navigate returns a command that navigates to the specified view.
func Navigate(view ViewState) tea.Cmd {
	return func() tea.Msg {
		return NavigateMsg{View: view}
	}
}

// ReportError returns a command that reports an error.
func ReportError(err error) tea.Cmd {
	return func() tea.Msg {
		return ErrorMsg{Err: err}
	}
}

// Quit returns a command that quits the application.
func Quit() tea.Cmd {
	return func() tea.Msg {
		return QuitMsg{}
	}
}

// SendProgress returns a command that sends a progress update.
func SendProgress(current, total int, message string) tea.Cmd {
	return func() tea.Msg {
		return ProgressMsg{
			Current: current,
			Total:   total,
			Message: message,
		}
	}
}

// SendStatus returns a command that sends a status update.
func SendStatus(message string, isError bool) tea.Cmd {
	return func() tea.Msg {
		return StatusMsg{
			Message: message,
			IsError: isError,
		}
	}
}

// SendTick returns a command that sends a tick for animations.
func SendTick() tea.Cmd {
	return func() tea.Msg {
		return TickMsg{}
	}
}

// SendWindowReady returns a command that signals the window is ready.
func SendWindowReady(width, height int) tea.Cmd {
	return func() tea.Msg {
		return WindowReadyMsg{
			Width:  width,
			Height: height,
		}
	}
}
