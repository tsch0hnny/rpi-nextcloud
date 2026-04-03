package steps

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
	"github.com/tsch0hnny/rpi-nextcloud/internal/exec"
	"github.com/tsch0hnny/rpi-nextcloud/internal/ui"
)

type pfPhase int

const (
	pfInputDomain pfPhase = iota
	pfConfirm
	pfUpdatingConfig
	pfDone
	pfError
)

type PortForwardStep struct {
	phase        pfPhase
	complete     bool
	input        ui.InputModel
	confirm      ui.ConfirmModel
	spinner      ui.SpinnerModel
	completedSub []string
	errMsg       string
}

func NewPortForwardStep() *PortForwardStep {
	return &PortForwardStep{}
}

func (s *PortForwardStep) ID() string       { return "port-forward" }
func (s *PortForwardStep) Title() string     { return "Port Forwarding" }
func (s *PortForwardStep) IsOptional() bool  { return true }
func (s *PortForwardStep) IsComplete() bool  { return s.complete }

func (s *PortForwardStep) Init(state *State) tea.Cmd {
	s.phase = pfInputDomain
	s.input = ui.NewInput("External Domain / Public IP", "nextcloud.example.com", state.ExternalDomain,
		"Enter the domain name or public IP you'll use to access Nextcloud externally.\nThis will be added to Nextcloud's trusted domains.")
	return s.input.Init()
}

func (s *PortForwardStep) Update(msg tea.Msg, state *State) (Step, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch s.phase {
		case pfInputDomain:
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			return s, cmd
		case pfConfirm:
			if key.Matches(msg, style.Keys.Escape) {
				return s, func() tea.Msg { return StepSkipMsg{} }
			}
			var cmd tea.Cmd
			s.confirm, cmd = s.confirm.Update(msg)
			return s, cmd
		case pfDone:
			if key.Matches(msg, style.Keys.Enter) {
				s.complete = true
				return s, func() tea.Msg { return StepCompleteMsg{} }
			}
		case pfError:
			if key.Matches(msg, style.Keys.Enter) {
				s.phase = pfConfirm
				s.confirm = ui.NewConfirm("Retry?", true)
				return s, nil
			}
		}

	case ui.InputResult:
		if s.phase == pfInputDomain {
			state.ExternalDomain = msg.Value
			s.phase = pfConfirm
			s.confirm = ui.NewConfirm(
				fmt.Sprintf("Add '%s' to trusted domains?", state.ExternalDomain), true)
			return s, nil
		}

	case ui.ConfirmResult:
		if s.phase == pfConfirm {
			if !msg.Confirmed {
				return s, func() tea.Msg { return StepSkipMsg{} }
			}
			s.phase = pfUpdatingConfig
			s.spinner = ui.NewSpinner("Updating trusted domains...")
			s.completedSub = nil
			// Use PHP to add the trusted domain properly
			phpCmd := fmt.Sprintf(
				`php -r "
				\$f = '/var/www/nextcloud/config/config.php';
				\$c = include \$f;
				\$d = '%s';
				\$found = false;
				foreach (\$c['trusted_domains'] as \$v) { if (\$v === \$d) \$found = true; }
				if (!\$found) {
					\$c['trusted_domains'][] = \$d;
					file_put_contents(\$f, '<?php' . PHP_EOL . '\$CONFIG = ' . var_export(\$c, true) . ';' . PHP_EOL);
				}
				echo 'done';
				"`, state.ExternalDomain)
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("update-trusted", phpCmd))
		}

	case exec.CmdResult:
		if msg.Tag == "update-trusted" {
			s.completedSub = append(s.completedSub,
				fmt.Sprintf("Added '%s' to trusted domains", state.ExternalDomain))
			if msg.Err != nil {
				s.phase = pfError
				s.errMsg = "Failed to update trusted domains: " + msg.Err.Error()
				return s, nil
			}
			s.phase = pfDone
			return s, nil
		}
	}

	if s.phase == pfUpdatingConfig {
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	}

	return s, nil
}

func (s *PortForwardStep) View(state *State) string {
	var sections []string

	desc := style.DescriptionStyle.Render(
		"Configure Nextcloud for external access. You'll need to forward\nports 80 (HTTP) and 443 (HTTPS) on your router to your Pi's IP.")
	sections = append(sections, "", desc, "")

	switch s.phase {
	case pfInputDomain:
		sections = append(sections, s.input.View())

	case pfConfirm:
		sections = append(sections,
			style.SubtitleStyle.Render("Port Forwarding Checklist:"),
			"",
			style.TextStyle.Render("  On your router, forward these ports to ")+
				style.CodeBlockStyle.Render(state.IPAddress)+style.TextStyle.Render(":"),
			"",
			style.TextStyle.Render("  "+style.BoldStyle.Render("Port 80")+"  (TCP) → HTTP traffic"),
			style.TextStyle.Render("  "+style.BoldStyle.Render("Port 443")+" (TCP) → HTTPS traffic"),
			"",
			style.DescriptionStyle.Render("  Tip: Consider setting up a dynamic DNS service if your ISP\n  assigns a dynamic public IP address."),
			"",
			s.confirm.View(),
		)

	case pfUpdatingConfig:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		sections = append(sections, "", s.spinner.View())

	case pfDone:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		w := state.Width - 8
		if w > 70 {
			w = 70
		}
		sections = append(sections, "", ui.SuccessBox("Trusted domain added!", w), "")

		sections = append(sections,
			ui.WarningBox("Remember to configure port forwarding on your router:\n  Port 80  (TCP) → "+state.IPAddress+"\n  Port 443 (TCP) → "+state.IPAddress, w),
			"",
			style.KeyHintStyle.Render("Press ENTER to continue →"),
		)

	case pfError:
		w := state.Width - 8
		if w > 70 {
			w = 70
		}
		sections = append(sections, "", ui.WarningBox(s.errMsg, w))
		sections = append(sections, "", style.KeyHintStyle.Render("Press ENTER to retry"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
