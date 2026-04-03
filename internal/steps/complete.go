package steps

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
	"github.com/tsch0hnny/rpi-nextcloud/internal/ui"
)

type CompleteStep struct {
	complete bool
}

func NewCompleteStep() *CompleteStep {
	return &CompleteStep{}
}

func (s *CompleteStep) ID() string       { return "complete" }
func (s *CompleteStep) Title() string     { return "Complete" }
func (s *CompleteStep) IsOptional() bool  { return false }
func (s *CompleteStep) IsComplete() bool  { return s.complete }

func (s *CompleteStep) Init(state *State) tea.Cmd {
	return nil
}

func (s *CompleteStep) Update(msg tea.Msg, state *State) (Step, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, style.Keys.Enter) || msg.String() == "q" {
			s.complete = true
			return s, func() tea.Msg { return StepCompleteMsg{} }
		}
	}
	return s, nil
}

func (s *CompleteStep) View(state *State) string {
	width := state.Width - 8
	if width > 80 {
		width = 80
	}

	banner := lipgloss.NewStyle().
		Foreground(style.ColorSuccess).
		Bold(true).
		Render(`
    ╔═══════════════════════════════════════════╗
    ║                                           ║
    ║   🎉  Installation Complete!  🎉          ║
    ║                                           ║
    ╚═══════════════════════════════════════════╝`)

	var sections []string
	sections = append(sections, banner, "")

	// Show what was configured
	sections = append(sections, style.SubtitleStyle.Render("  Configuration Summary:"), "")

	// URLs
	httpURL := fmt.Sprintf("http://%s/nextcloud", state.IPAddress)
	httpsURL := fmt.Sprintf("https://%s/nextcloud", state.IPAddress)
	if state.ApacheMode == "domain" {
		httpURL = fmt.Sprintf("http://%s", state.DomainName)
		httpsURL = fmt.Sprintf("https://%s", state.DomainName)
	}

	sections = append(sections,
		ui.StatusLine("Local URL", httpURL, style.ColorAccent),
	)
	if state.SSLEnabled {
		sections = append(sections,
			ui.StatusLine("HTTPS URL", httpsURL, style.ColorAccent),
		)
	}
	sections = append(sections, "")

	// Database
	sections = append(sections,
		ui.StatusLine("Database", state.DBName, style.ColorAccent),
		ui.StatusLine("DB User", state.DBUser, style.ColorAccent),
	)

	// Completed steps
	sections = append(sections, "",
		style.SubtitleStyle.Render("  Completed Steps:"),
		"",
	)

	stepChecks := []struct {
		name string
		done bool
	}{
		{"Apache2 + PHP 8.4", state.CompletedSteps["apache-php"]},
		{"MariaDB Database", state.CompletedSteps["mysql"]},
		{"Nextcloud Downloaded", state.CompletedSteps["download"]},
		{"Apache Configured", state.CompletedSteps["apache-conf"]},
		{"Web Setup", state.CompletedSteps["web-setup"]},
		{"Data Directory Moved", state.CompletedSteps["move-data"]},
		{"Upload Limit Increased", state.CompletedSteps["upload-size"]},
		{"SSL / HTTPS", state.CompletedSteps["ssl"]},
		{"Port Forwarding", state.CompletedSteps["port-forward"]},
	}

	for _, sc := range stepChecks {
		if sc.done {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sc.name))
		} else {
			sections = append(sections, style.DescriptionStyle.Render("  ○ "+sc.name+" (skipped)"))
		}
	}

	// Image from tutorial
	sections = append(sections, "")
	imgURL := "https://pimylifeup.com/wp-content/uploads/2017/06/03-Files-Screen.jpg"
	sections = append(sections,
		ui.ImageWithCaption(imgURL, "Your Nextcloud dashboard", width),
	)

	sections = append(sections, "")

	nextSteps := ui.InfoBox(
		style.SubtitleStyle.Render("Next Steps:")+"\n\n"+
			style.TextStyle.Render("  • Install the Nextcloud desktop/mobile apps\n"+
				"  • Set up additional users\n"+
				"  • Configure external storage\n"+
				"  • Enable server-side encryption\n"+
				"  • Set up automatic backups"),
		width,
	)
	sections = append(sections, nextSteps, "")

	sections = append(sections,
		style.KeyHintStyle.Render("  Press ENTER or q to exit. Enjoy your cloud! ☁"),
	)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
