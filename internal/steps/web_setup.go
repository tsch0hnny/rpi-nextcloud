package steps

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
	"github.com/tsch0hnny/rpi-nextcloud/internal/ui"
)

const setupImageURL = "https://pimylifeup.com/wp-content/uploads/2017/06/Raspberry-Pi-Nextcloud-Setup-Screen.jpg"

type wsPhase int

const (
	wsShowInstructions wsPhase = iota
	wsConfirmOpen
	wsWaitingForUser
)

type WebSetupStep struct {
	phase    wsPhase
	complete bool
	confirm  ui.ConfirmModel
	url      string
	isSSH    bool
}

func NewWebSetupStep() *WebSetupStep {
	return &WebSetupStep{}
}

func (s *WebSetupStep) ID() string       { return "web-setup" }
func (s *WebSetupStep) Title() string     { return "Web Setup" }
func (s *WebSetupStep) IsOptional() bool  { return false }
func (s *WebSetupStep) IsComplete() bool  { return s.complete }

func (s *WebSetupStep) Init(state *State) tea.Cmd {
	if state.ApacheMode == "domain" {
		s.url = fmt.Sprintf("http://%s", state.DomainName)
	} else {
		s.url = fmt.Sprintf("http://%s/nextcloud", state.IPAddress)
	}
	// Detect if running over SSH
	s.isSSH = os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != ""
	if s.isSSH {
		// Skip browser prompt, go straight to waiting
		s.phase = wsWaitingForUser
	} else {
		s.phase = wsShowInstructions
	}
	// Start loading image asynchronously
	return ui.LoadImageAsync(setupImageURL, state.Width-8)
}

func (s *WebSetupStep) Update(msg tea.Msg, state *State) (Step, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch s.phase {
		case wsShowInstructions:
			if key.Matches(msg, style.Keys.Enter) {
				s.phase = wsConfirmOpen
				s.confirm = ui.NewConfirm(
					fmt.Sprintf("Open %s in your browser?", s.url), true)
				return s, nil
			}
		case wsConfirmOpen:
			var cmd tea.Cmd
			s.confirm, cmd = s.confirm.Update(msg)
			return s, cmd
		case wsWaitingForUser:
			if key.Matches(msg, style.Keys.Enter) {
				s.complete = true
				return s, func() tea.Msg { return StepCompleteMsg{} }
			}
		}

	case ui.ConfirmResult:
		if s.phase == wsConfirmOpen {
			if msg.Confirmed {
				openBrowser(s.url)
			}
			s.phase = wsWaitingForUser
			return s, nil
		}

	case ui.ImageLoadedMsg:
		// Image loaded in background, UI will pick it up from cache
		return s, nil
	}

	return s, nil
}

func openBrowser(url string) {
	for _, cmd := range []string{"xdg-open", "open", "sensible-browser"} {
		if _, err := exec.LookPath(cmd); err == nil {
			_ = exec.Command(cmd, url).Start()
			return
		}
	}
}

func (s *WebSetupStep) View(state *State) string {
	var sections []string

	sections = append(sections, "")

	// Show setup image (from async cache)
	sections = append(sections,
		ui.ImageWithCaption(setupImageURL, "Nextcloud Setup Screen", state.Width-8),
		"",
	)

	sections = append(sections,
		style.SubtitleStyle.Render("Complete the Nextcloud setup in your web browser:"),
		"",
		style.TextStyle.Render("  1. Set your ")+style.BoldStyle.Render("admin username")+
			style.TextStyle.Render(fmt.Sprintf(" (recommended: %s)", state.AdminUser)),
		style.TextStyle.Render("  2. Set a ")+style.BoldStyle.Render("strong admin password"),
		style.TextStyle.Render("  3. Click ")+style.BoldStyle.Render("\"Storage & Database\""),
		style.TextStyle.Render("  4. Select ")+style.BoldStyle.Render("\"MySQL/MariaDB\""),
		style.TextStyle.Render("  5. Enter database details:"),
		style.TextStyle.Render(fmt.Sprintf("     - Database user: %s", style.CodeBlockStyle.Render(state.DBUser))),
		style.TextStyle.Render(fmt.Sprintf("     - Database password: %s", style.CodeBlockStyle.Render("(your password)"))),
		style.TextStyle.Render(fmt.Sprintf("     - Database name: %s", style.CodeBlockStyle.Render(state.DBName))),
		style.TextStyle.Render("  6. Click ")+style.BoldStyle.Render("\"Finish Setup\""),
		"",
	)

	urlBox := ui.ActiveInfoBox(
		style.SubtitleStyle.Render("Nextcloud URL: ")+
			lipgloss.NewStyle().Foreground(style.ColorAccent).Underline(true).Render(s.url),
		state.Width-8,
	)
	sections = append(sections, urlBox, "")

	switch s.phase {
	case wsShowInstructions:
		sections = append(sections, style.KeyHintStyle.Render("Press ENTER to open in browser →"))
	case wsConfirmOpen:
		sections = append(sections, s.confirm.View())
	case wsWaitingForUser:
		w := state.Width - 8
		if w > 70 {
			w = 70
		}
		if s.isSSH {
			sections = append(sections,
				ui.WarningBox("You're connected via SSH. Open the URL above in a browser\non your local computer, complete the setup, then press ENTER.", w),
			)
		} else {
			sections = append(sections,
				ui.WarningBox("Complete the setup in your browser, then press ENTER here to continue.", w),
			)
		}
		sections = append(sections,
			"",
			style.KeyHintStyle.Render("Press ENTER when setup is complete →"),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
