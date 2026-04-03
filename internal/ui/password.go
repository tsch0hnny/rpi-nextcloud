package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
)

// PasswordModel is a masked password input field.
type PasswordModel struct {
	Label       string
	Description string
	input       textinput.Model
	Done        bool
	Value       string
}

type PasswordResult struct {
	Value string
}

func NewPassword(label, description string) PasswordModel {
	ti := textinput.New()
	ti.Placeholder = "Enter password..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.CharLimit = 128
	ti.Width = 40
	ti.PromptStyle = lipgloss.NewStyle().Foreground(style.ColorPrimary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(style.ColorText)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(style.ColorAccent)
	ti.Focus()

	return PasswordModel{
		Label:       label,
		Description: description,
		input:       ti,
	}
}

func (m PasswordModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m PasswordModel) Update(msg tea.Msg) (PasswordModel, tea.Cmd) {
	if m.Done {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.Value = m.input.Value()
			if m.Value != "" {
				m.Done = true
				return m, func() tea.Msg { return PasswordResult{Value: m.Value} }
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m PasswordModel) View() string {
	label := style.InputLabelStyle.Render(m.Label)

	parts := []string{label}
	if m.Description != "" {
		parts = append(parts, style.DescriptionStyle.Render(m.Description))
	}
	parts = append(parts, "", m.input.View(), "")
	parts = append(parts, style.KeyHintStyle.Render("enter: confirm (password must not be empty)"))

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *PasswordModel) Focus() tea.Cmd {
	return m.input.Focus()
}

func (m *PasswordModel) Blur() {
	m.input.Blur()
}

func (m *PasswordModel) Reset() {
	m.input.SetValue("")
	m.Done = false
	m.Value = ""
}
