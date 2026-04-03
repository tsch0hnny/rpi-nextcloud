package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
)

// InputModel wraps a text input with a label and description.
type InputModel struct {
	Label       string
	Description string
	input       textinput.Model
	Done        bool
	Value       string
}

type InputResult struct {
	Value string
}

func NewInput(label, placeholder, defaultValue, description string) InputModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(defaultValue)
	ti.CharLimit = 256
	ti.Width = 40
	ti.PromptStyle = lipgloss.NewStyle().Foreground(style.ColorPrimary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(style.ColorText)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(style.ColorAccent)
	ti.Focus()

	return InputModel{
		Label:       label,
		Description: description,
		input:       ti,
	}
}

func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	if m.Done {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.Value = m.input.Value()
			m.Done = true
			return m, func() tea.Msg { return InputResult{Value: m.Value} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m InputModel) View() string {
	label := style.InputLabelStyle.Render(m.Label)
	desc := ""
	if m.Description != "" {
		desc = style.DescriptionStyle.Render(m.Description)
	}

	parts := []string{label}
	if desc != "" {
		parts = append(parts, desc)
	}
	parts = append(parts, "", m.input.View(), "")
	parts = append(parts, style.KeyHintStyle.Render("enter: confirm"))

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *InputModel) Focus() tea.Cmd {
	return m.input.Focus()
}

func (m *InputModel) Blur() {
	m.input.Blur()
}

func (m *InputModel) SetValue(v string) {
	m.input.SetValue(v)
}
