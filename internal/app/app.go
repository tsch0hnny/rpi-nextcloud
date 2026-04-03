package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/exec"
	"github.com/tsch0hnny/rpi-nextcloud/internal/steps"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
	"github.com/tsch0hnny/rpi-nextcloud/internal/ui"
)

// Model is the top-level bubbletea model.
type Model struct {
	steps       []steps.Step
	stepNames   []string
	currentStep int
	state       *steps.State
	width       int
	height      int
	quitting    bool
	confirmQuit bool
	quitConfirm ui.ConfirmModel
	initialized bool
}

// New creates a new app model with the given steps.
func New(stepList []steps.Step) Model {
	names := make([]string, len(stepList))
	for i, s := range stepList {
		names[i] = s.Title()
	}

	return Model{
		steps:     stepList,
		stepNames: names,
		state:     steps.NewState(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tea.SetWindowTitle("Nextcloud Installer"),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.state.Width = msg.Width
		m.state.Height = msg.Height

		if !m.initialized {
			m.initialized = true
			sysInfo := exec.DetectSystem()
			m.state.IPAddress = sysInfo.IPAddress
			m.state.Hostname = sysInfo.Hostname
			if len(m.steps) > 0 {
				return m, m.steps[m.currentStep].Init(m.state)
			}
		}
		return m, nil

	case tea.KeyMsg:
		// Handle quit confirmation
		if m.confirmQuit {
			var cmd tea.Cmd
			m.quitConfirm, cmd = m.quitConfirm.Update(msg)
			if m.quitConfirm.Done {
				if m.quitConfirm.Confirmed {
					m.quitting = true
					return m, tea.Quit
				}
				m.confirmQuit = false
			}
			return m, cmd
		}

		// Global quit via ctrl+c
		if key.Matches(msg, style.Keys.Quit) && msg.String() != "q" {
			m.quitting = true
			return m, tea.Quit
		}

	case steps.StepCompleteMsg:
		if m.currentStep < len(m.steps)-1 {
			m.state.CompletedSteps[m.steps[m.currentStep].ID()] = true
			m.currentStep++
			return m, m.steps[m.currentStep].Init(m.state)
		}
		m.quitting = true
		return m, tea.Quit

	case steps.StepSkipMsg:
		if m.currentStep < len(m.steps)-1 {
			m.currentStep++
			return m, m.steps[m.currentStep].Init(m.state)
		}
		m.quitting = true
		return m, tea.Quit

	case steps.QuitConfirmMsg:
		m.confirmQuit = true
		m.quitConfirm = ui.NewConfirm("Are you sure you want to quit?", false)
		return m, nil
	}

	// Forward to current step
	if m.currentStep < len(m.steps) && m.initialized {
		var cmd tea.Cmd
		m.steps[m.currentStep], cmd = m.steps[m.currentStep].Update(msg, m.state)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return "\n" + style.SuccessStyle.Render("  Thanks for using Nextcloud Installer! ") + "\n\n"
	}

	if !m.initialized {
		return "\n  Initializing..."
	}

	width := m.width
	if width == 0 {
		width = 80
	}
	if width > 120 {
		width = 120
	}

	contentWidth := width - 4

	var b strings.Builder

	header := m.renderHeader(contentWidth)
	b.WriteString(header)
	b.WriteString("\n")

	if m.confirmQuit {
		content := lipgloss.Place(contentWidth, m.height-8,
			lipgloss.Center, lipgloss.Center,
			m.quitConfirm.View(),
		)
		b.WriteString(content)
	} else if m.currentStep < len(m.steps) {
		content := m.steps[m.currentStep].View(m.state)
		b.WriteString(content)
	}

	b.WriteString("\n")
	b.WriteString(m.renderFooter(contentWidth))

	return lipgloss.NewStyle().
		Padding(0, 2).
		MaxWidth(width).
		Render(b.String())
}

func (m Model) renderHeader(width int) string {
	current := m.currentStep + 1
	total := len(m.steps)
	barWidth := width - 10
	if barWidth < 20 {
		barWidth = 20
	}

	filled := (current * barWidth) / total
	if filled > barWidth {
		filled = barWidth
	}

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	empty := ""
	for i := 0; i < barWidth-filled; i++ {
		empty += "░"
	}

	progressLine := style.ProgressBarFilled.Render(bar) +
		style.ProgressBarEmpty.Render(empty) +
		style.SubtitleStyle.Render(fmt.Sprintf(" %d/%d", current, total))

	title := ""
	if m.currentStep < len(m.steps) {
		step := m.steps[m.currentStep]
		titleText := step.Title()
		if step.IsOptional() {
			titleText += " " + style.OptionalBadge.Render("[optional]")
		}
		title = style.TitleStyle.Render(fmt.Sprintf(" Step %d: %s ", current, titleText))
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		title,
		progressLine,
		style.Divider(width),
	)
}

func (m Model) renderFooter(width int) string {
	hints := []string{
		style.KeyStyle.Render("ctrl+c") + style.KeyHintStyle.Render(": quit"),
	}

	return style.Divider(width) + "\n" + strings.Join(hints, "  ")
}
