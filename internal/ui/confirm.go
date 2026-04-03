package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
)

// ConfirmModel is a Yes/No confirmation widget with vim-style h/l focus.
type ConfirmModel struct {
	Prompt      string
	YesLabel    string
	NoLabel     string
	focused     int // 0=yes, 1=no
	defaultYes  bool
	Confirmed   bool
	Done        bool
}

type ConfirmResult struct {
	Confirmed bool
}

func NewConfirm(prompt string, defaultYes bool) ConfirmModel {
	focused := 1
	if defaultYes {
		focused = 0
	}
	return ConfirmModel{
		Prompt:     prompt,
		YesLabel:   "Yes",
		NoLabel:    "No",
		focused:    focused,
		defaultYes: defaultYes,
	}
}

func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

func (m ConfirmModel) Update(msg tea.Msg) (ConfirmModel, tea.Cmd) {
	if m.Done {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, style.Keys.Left):
			m.focused = 0
		case key.Matches(msg, style.Keys.Right):
			m.focused = 1
		case key.Matches(msg, style.Keys.Tab):
			m.focused = (m.focused + 1) % 2
		case key.Matches(msg, style.Keys.ShiftTab):
			m.focused = (m.focused + 1) % 2
		case msg.String() == "y":
			m.Confirmed = true
			m.Done = true
			return m, func() tea.Msg { return ConfirmResult{Confirmed: true} }
		case msg.String() == "n":
			m.Confirmed = false
			m.Done = true
			return m, func() tea.Msg { return ConfirmResult{Confirmed: false} }
		case key.Matches(msg, style.Keys.Enter):
			m.Confirmed = m.focused == 0
			m.Done = true
			return m, func() tea.Msg { return ConfirmResult{Confirmed: m.Confirmed} }
		}
	}

	return m, nil
}

func (m ConfirmModel) View() string {
	prompt := style.TextStyle.Bold(true).Render(m.Prompt)

	yesStyle := style.ButtonInactive
	noStyle := style.ButtonInactive
	if m.focused == 0 {
		yesStyle = style.ButtonActive
	} else {
		noStyle = style.ButtonActive
	}

	yes := yesStyle.Render(m.YesLabel)
	no := noStyle.Render(m.NoLabel)

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yes, "  ", no)

	hint := style.KeyHintStyle.Render("h/l: switch  enter: confirm  y/n: quick select")

	return lipgloss.JoinVertical(lipgloss.Left,
		prompt,
		"",
		buttons,
		"",
		hint,
	)
}
