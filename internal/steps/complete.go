package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
	"github.com/tsch0hnny/rpi-nextcloud/internal/ui"
)

const filesImageURL = "https://pimylifeup.com/wp-content/uploads/2017/06/03-Files-Screen.jpg"

type CompleteStep struct {
	complete bool
	vp       viewport.Model
	vpReady  bool
}

func NewCompleteStep() *CompleteStep {
	return &CompleteStep{}
}

func (s *CompleteStep) ID() string       { return "complete" }
func (s *CompleteStep) Title() string     { return "Complete" }
func (s *CompleteStep) IsOptional() bool  { return false }
func (s *CompleteStep) IsComplete() bool  { return s.complete }

func (s *CompleteStep) Init(state *State) tea.Cmd {
	// Start loading image asynchronously
	return ui.LoadImageAsync(filesImageURL, state.Width-8)
}

func (s *CompleteStep) Update(msg tea.Msg, state *State) (Step, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h := msg.Height - 10 // leave room for header/footer
		if h < 5 {
			h = 5
		}
		if !s.vpReady {
			s.vp = viewport.New(msg.Width-4, h)
			s.vp.SetContent(s.buildContent(state))
			s.vpReady = true
		} else {
			s.vp.Width = msg.Width - 4
			s.vp.Height = h
		}
		return s, nil

	case tea.KeyMsg:
		if key.Matches(msg, style.Keys.Enter) && s.vp.AtBottom() {
			s.complete = true
			return s, func() tea.Msg { return StepCompleteMsg{} }
		}
		if msg.String() == "q" {
			s.complete = true
			return s, func() tea.Msg { return StepCompleteMsg{} }
		}

		// Forward to viewport for scrolling (j/k, ctrl+d/u, pgup/pgdn)
		if s.vpReady {
			var cmd tea.Cmd
			s.vp, cmd = s.vp.Update(msg)
			return s, cmd
		}

	case ui.ImageLoadedMsg:
		// Re-render content now that image is available
		if s.vpReady {
			s.vp.SetContent(s.buildContent(state))
		}
		return s, nil
	}

	return s, nil
}

func (s *CompleteStep) buildContent(state *State) string {
	width := state.Width - 8
	if width > 80 {
		width = 80
	}

	var sections []string

	banner := lipgloss.NewStyle().
		Foreground(style.ColorSuccess).
		Bold(true).
		Render(`
    ╔═══════════════════════════════════════════╗
    ║                                           ║
    ║      Installation Complete!               ║
    ║                                           ║
    ╚═══════════════════════════════════════════╝`)

	sections = append(sections, banner, "")
	sections = append(sections, style.SubtitleStyle.Render("  Configuration Summary:"), "")

	httpURL := fmt.Sprintf("http://%s/nextcloud", state.IPAddress)
	httpsURL := fmt.Sprintf("https://%s/nextcloud", state.IPAddress)
	if state.ApacheMode == "domain" {
		httpURL = fmt.Sprintf("http://%s", state.DomainName)
		httpsURL = fmt.Sprintf("https://%s", state.DomainName)
	}

	sections = append(sections, ui.StatusLine("Local URL", httpURL, style.ColorAccent))
	if state.SSLEnabled {
		sections = append(sections, ui.StatusLine("HTTPS URL", httpsURL, style.ColorAccent))
	}
	sections = append(sections, "")

	sections = append(sections,
		ui.StatusLine("Database", state.DBName, style.ColorAccent),
		ui.StatusLine("DB User", state.DBUser, style.ColorAccent),
	)

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
	sections = append(sections, ui.ImageWithCaption(filesImageURL, "Your Nextcloud dashboard", width))

	sections = append(sections, "")

	nextSteps := ui.InfoBox(
		style.SubtitleStyle.Render("Next Steps:")+"\n\n"+
			style.TextStyle.Render("  - Install the Nextcloud desktop/mobile apps\n"+
				"  - Set up additional users\n"+
				"  - Configure external storage\n"+
				"  - Enable server-side encryption\n"+
				"  - Set up automatic backups"),
		width,
	)
	sections = append(sections, nextSteps, "")

	sections = append(sections,
		style.KeyHintStyle.Render("  Scroll with j/k. Press ENTER or q to exit."),
	)

	return strings.Join(sections, "\n")
}

func (s *CompleteStep) View(state *State) string {
	if !s.vpReady {
		// Before viewport is sized, render directly
		return s.buildContent(state)
	}

	scrollHint := ""
	if !s.vp.AtBottom() {
		scrollHint = style.DescriptionStyle.Render("  ↓ scroll down for more")
	}

	return s.vp.View() + "\n" + scrollHint
}
