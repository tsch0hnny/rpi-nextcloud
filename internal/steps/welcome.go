package steps

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/exec"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
	"github.com/tsch0hnny/rpi-nextcloud/internal/ui"
)

type WelcomeStep struct {
	complete bool
	sysInfo  exec.SystemInfo
	checked  bool
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
	if !s.checked {
		s.sysInfo = exec.DetectSystem()
		s.checked = true
	}

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
	width := state.Width - 8
	if width > 80 {
		width = 80
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
		if s.checked && s.sysInfo.DiskFreeGB > 0 {
			sysInfo += "\n" + ui.StatusLine("Disk Free", fmt.Sprintf("%.1f GB", s.sysInfo.DiskFreeGB), style.ColorAccent)
		}
	}

	// System warnings
	warnings := ""
	if s.checked {
		var warns []string
		if !s.sysInfo.IsDebian && !s.sysInfo.IsRaspberry {
			warns = append(warns, "Not running on Debian/Raspbian — some commands may differ")
		}
		if !s.sysInfo.HasApt {
			warns = append(warns, "apt package manager not found — required for installation")
		}
		if !s.sysInfo.HasSystemd {
			warns = append(warns, "systemd not found — service management may not work")
		}
		if s.sysInfo.DiskFreeGB >= 0 && s.sysInfo.DiskFreeGB < 2.0 {
			warns = append(warns, fmt.Sprintf("Low disk space (%.1f GB free) — need at least 2 GB", s.sysInfo.DiskFreeGB))
		}
		if len(warns) > 0 {
			warnText := ""
			for _, w := range warns {
				warnText += "  ! " + w + "\n"
			}
			warnings = "\n" + ui.WarningBox(warnText, width)
		}
	}

	keyhints := style.KeyHintStyle.Render("  Navigation: ") +
		style.KeyStyle.Render("h/l") + style.KeyHintStyle.Render(" switch  ") +
		style.KeyStyle.Render("j/k") + style.KeyHintStyle.Render(" scroll  ") +
		style.KeyStyle.Render("enter") + style.KeyHintStyle.Render(" confirm  ") +
		style.KeyStyle.Render("esc") + style.KeyHintStyle.Render(" back/skip")

	prompt := lipgloss.NewStyle().
		Foreground(style.ColorPrimary).
		Bold(true).
		Render("\n  Press ENTER to begin →")

	return lipgloss.JoinVertical(lipgloss.Center,
		"",
		logo,
		"",
		subtitle,
		"",
		features,
		sysInfo,
		warnings,
		"",
		keyhints,
		prompt,
	)
}
