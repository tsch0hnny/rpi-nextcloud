package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
)

// SpinnerModel wraps a spinner with a status message.
type SpinnerModel struct {
	spinner spinner.Model
	Message string
	Done    bool
	Err     error
}

func NewSpinner(message string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = style.SpinnerStyle
	return SpinnerModel{
		spinner: s,
		Message: message,
	}
}

func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m SpinnerModel) Update(msg tea.Msg) (SpinnerModel, tea.Cmd) {
	if m.Done {
		return m, nil
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m SpinnerModel) View() string {
	if m.Done {
		if m.Err != nil {
			return style.ErrorStyle.Render("✗ " + m.Message + ": " + m.Err.Error())
		}
		return style.SuccessStyle.Render("✓ " + m.Message)
	}
	return lipgloss.JoinHorizontal(lipgloss.Center,
		m.spinner.View(),
		" ",
		style.TextStyle.Render(m.Message),
	)
}

func (m *SpinnerModel) Finish(err error) {
	m.Done = true
	m.Err = err
}
