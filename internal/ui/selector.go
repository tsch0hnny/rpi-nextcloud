package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
)

// SelectorItem represents a choice in the selector.
type SelectorItem struct {
	Label       string
	Description string
	Value       string
}

// SelectorModel is a single-select list with vim-style j/k navigation.
type SelectorModel struct {
	Title    string
	Items    []SelectorItem
	cursor   int
	Done     bool
	Selected int
}

type SelectorResult struct {
	Index int
	Value string
}

func NewSelector(title string, items []SelectorItem) SelectorModel {
	return SelectorModel{
		Title: title,
		Items: items,
	}
}

func (m SelectorModel) Init() tea.Cmd {
	return nil
}

func (m SelectorModel) Update(msg tea.Msg) (SelectorModel, tea.Cmd) {
	if m.Done {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, style.Keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, style.Keys.Down):
			if m.cursor < len(m.Items)-1 {
				m.cursor++
			}
		case key.Matches(msg, style.Keys.Enter):
			m.Selected = m.cursor
			m.Done = true
			return m, func() tea.Msg {
				return SelectorResult{Index: m.cursor, Value: m.Items[m.cursor].Value}
			}
		}
	}

	return m, nil
}

func (m SelectorModel) View() string {
	title := style.SubtitleStyle.Render(m.Title)

	items := ""
	for i, item := range m.Items {
		cursor := "  "
		labelStyle := style.TextStyle
		descStyle := style.DescriptionStyle

		if i == m.cursor {
			cursor = style.SuccessStyle.Render("▸ ")
			labelStyle = style.TextStyle.Bold(true).Foreground(style.ColorWhite)
			descStyle = style.DescriptionStyle.Foreground(style.ColorText)
		}

		line := fmt.Sprintf("%s%s", cursor, labelStyle.Render(item.Label))
		if item.Description != "" {
			line += "\n    " + descStyle.Render(item.Description)
		}
		items += line + "\n"
	}

	hint := style.KeyHintStyle.Render("j/k: navigate  enter: select")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", items, hint,
	)
}
