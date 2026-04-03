package steps

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
	"github.com/tsch0hnny/rpi-nextcloud/internal/ui"
)

type WelcomeStep struct {
	complete bool
}

func NewWelcomeStep() *WelcomeStep {
	return &WelcomeStep{}
}

func (s *WelcomeStep) ID() string       { return "welcome" }
func (s *WelcomeStep) Title() string     { return "Welcome" }
func (s *WelcomeStep) IsOptional() bool  { return false }
func (s *WelcomeStep) IsComplete() bool  { return s.complete }

func (s *WelcomeStep) Init(state *State) tea.Cmd {
	return nil
}

func (s *WelcomeStep) Update(msg tea.Msg, state *State) (Step, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, style.Keys.Enter) {
			s.complete = true
			return s, func() tea.Msg { return StepCompleteMsg{} }
		}
		if msg.String() == "q" {
			return s, func() tea.Msg { return QuitConfirmMsg{} }
		}
	}
	return s, nil
}

func (s *WelcomeStep) View(state *State) string {
	width := state.Width - 4
	if width > 116 {
		width = 116
	}

	logo := ui.Logo()

	subtitle := lipgloss.NewStyle().
		Foreground(style.ColorSubtle).
		Render("This installer will guide you through setting up your own Nextcloud\npersonal cloud server on your Raspberry Pi.")

	features := lipgloss.NewStyle().
		Foreground(style.ColorText).
		Render(
			style.SuccessStyle.Render("  ✓ ") + "Apache2 + PHP 8.4 web server\n" +
				style.SuccessStyle.Render("  ✓ ") + "MariaDB database\n" +
				style.SuccessStyle.Render("  ✓ ") + "Nextcloud latest release\n" +
				style.SuccessStyle.Render("  ✓ ") + "SSL/HTTPS (optional)\n" +
				style.SuccessStyle.Render("  ✓ ") + "Custom upload limits\n" +
				style.SuccessStyle.Render("  ✓ ") + "Secure data directory",
		)

	sysInfo := ""
	if state.IPAddress != "" {
		sysInfo = style.SubtitleStyle.Render("\n  Detected System:") + "\n" +
			ui.StatusLine("IP Address", state.IPAddress, style.ColorAccent) + "\n" +
			ui.StatusLine("Hostname", state.Hostname, style.ColorAccent)
	}

	prompt := lipgloss.NewStyle().
		Foreground(style.ColorPrimary).
		Bold(true).
		Render("\n  Press ENTER to begin →")

	_ = width

	return lipgloss.JoinVertical(lipgloss.Center,
		"",
		logo,
		"",
		subtitle,
		"",
		features,
		sysInfo,
		"",
		prompt,
	)
}
