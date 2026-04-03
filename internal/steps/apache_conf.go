package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/exec"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
	"github.com/tsch0hnny/rpi-nextcloud/internal/ui"
)

type acPhase int

const (
	acCheckExisting acPhase = iota
	acSelectMode
	acInputDomain
	acConfirm
	acWritingConfig
	acEnablingSite
	acReloadingApache
	acDone
	acError
)

type ApacheConfStep struct {
	phase            acPhase
	complete         bool
	selector         ui.SelectorModel
	input            ui.InputModel
	confirm          ui.ConfirmModel
	spinner          ui.SpinnerModel
	completedSub     []string
	errMsg           string
	configText       string
	alreadyInstalled bool
}

func NewApacheConfStep() *ApacheConfStep {
	return &ApacheConfStep{}
}

func (s *ApacheConfStep) ID() string       { return "apache-conf" }
func (s *ApacheConfStep) Title() string     { return "Apache Configuration" }
func (s *ApacheConfStep) IsOptional() bool  { return false }
func (s *ApacheConfStep) IsComplete() bool  { return s.complete }

func (s *ApacheConfStep) Init(state *State) tea.Cmd {
	s.phase = acCheckExisting
	// Check if nextcloud.conf is already enabled
	return exec.RunCommand("check-apache-conf",
		"test -f /etc/apache2/sites-enabled/nextcloud.conf && echo 'exists'")
}

func (s *ApacheConfStep) initSelector() {
	s.phase = acSelectMode
	s.selector = ui.NewSelector("How should Nextcloud be accessed?", []ui.SelectorItem{
		{
			Label:       "Directory (/nextcloud)",
			Description: "Access via http://<IP>/nextcloud — simplest option",
			Value:       "directory",
		},
		{
			Label:       "Own Domain / Subdomain",
			Description: "Access via a custom domain like nextcloud.example.com",
			Value:       "domain",
		},
	})
}

func (s *ApacheConfStep) Update(msg tea.Msg, state *State) (Step, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch s.phase {
		case acSelectMode:
			var cmd tea.Cmd
			s.selector, cmd = s.selector.Update(msg)
			return s, cmd
		case acInputDomain:
			if key.Matches(msg, style.Keys.Escape) {
				s.initSelector()
				return s, nil
			}
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			return s, cmd
		case acConfirm:
			if key.Matches(msg, style.Keys.Escape) {
				s.initSelector()
				return s, nil
			}
			var cmd tea.Cmd
			s.confirm, cmd = s.confirm.Update(msg)
			return s, cmd
		case acDone:
			if key.Matches(msg, style.Keys.Enter) {
				s.complete = true
				return s, func() tea.Msg { return StepCompleteMsg{} }
			}
		case acError:
			if key.Matches(msg, style.Keys.Enter) {
				s.phase = acConfirm
				s.confirm = ui.NewConfirm("Retry Apache configuration?", true)
				return s, nil
			}
		}

	case ui.SelectorResult:
		state.ApacheMode = msg.Value
		if msg.Value == "domain" {
			s.phase = acInputDomain
			s.input = ui.NewInputWithValidation("Domain Name", "nextcloud.example.com", state.DomainName,
				"Enter the domain or subdomain for your Nextcloud instance.",
				ui.ValidateDomain)
			return s, s.input.Init()
		}
		s.configText = s.generateConfig(state)
		s.phase = acConfirm
		s.confirm = ui.NewConfirm("Write this Apache configuration?", true)
		return s, nil

	case ui.InputResult:
		if s.phase == acInputDomain {
			state.DomainName = msg.Value
			s.configText = s.generateConfig(state)
			s.phase = acConfirm
			s.confirm = ui.NewConfirm("Write this Apache configuration?", true)
			return s, nil
		}

	case ui.ConfirmResult:
		if s.phase == acConfirm && !msg.Confirmed && s.alreadyInstalled {
			// User chose not to reconfigure — skip
			s.phase = acDone
			s.completedSub = []string{"Existing Apache configuration kept"}
			return s, nil
		}
		if s.phase == acConfirm && msg.Confirmed && s.alreadyInstalled && s.configText == "" {
			// User wants to reconfigure — show selector
			s.alreadyInstalled = false
			s.initSelector()
			return s, nil
		}
		if s.phase == acConfirm && msg.Confirmed {
			s.phase = acWritingConfig
			s.spinner = ui.NewSpinner("Writing Apache configuration...")
			s.completedSub = nil
			// Write config file
			cmd := exec.RunSudoCommand("write-config",
				fmt.Sprintf("cat > /etc/apache2/sites-available/nextcloud.conf << 'CONFEOF'\n%s\nCONFEOF", s.configText))
			return s, tea.Batch(s.spinner.Init(), cmd)
		}

	case exec.CmdResult:
		switch msg.Tag {
		case "check-apache-conf":
			if msg.Err == nil && strings.TrimSpace(msg.Output) == "exists" {
				s.alreadyInstalled = true
				s.phase = acConfirm
				s.confirm = ui.NewConfirm("Apache Nextcloud config already exists. Reconfigure?", false)
				return s, nil
			}
			s.initSelector()
			return s, nil

		case "write-config":
			s.completedSub = append(s.completedSub, "Configuration file written")
			if msg.Err != nil {
				s.phase = acError
				s.errMsg = "Failed to write config: " + msg.Err.Error()
				return s, nil
			}
			s.phase = acEnablingSite
			s.spinner = ui.NewSpinner("Enabling Nextcloud site...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("enable-site", "a2ensite nextcloud.conf"))

		case "enable-site":
			s.completedSub = append(s.completedSub, "Site enabled")
			if msg.Err != nil {
				s.phase = acError
				s.errMsg = "Failed to enable site: " + msg.Err.Error()
				return s, nil
			}
			s.phase = acReloadingApache
			s.spinner = ui.NewSpinner("Reloading Apache...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("reload-apache", "systemctl reload apache2"))

		case "reload-apache":
			s.completedSub = append(s.completedSub, "Apache reloaded")
			if msg.Err != nil {
				s.phase = acError
				s.errMsg = "Failed to reload Apache: " + msg.Err.Error()
				return s, nil
			}
			s.phase = acDone
			return s, nil
		}
	}

	if s.phase >= acWritingConfig && s.phase <= acReloadingApache {
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	}

	return s, nil
}

func (s *ApacheConfStep) generateConfig(state *State) string {
	if state.ApacheMode == "domain" {
		return fmt.Sprintf(`<VirtualHost *:80>
  DocumentRoot /var/www/nextcloud/
  ServerName  %s

  <Directory /var/www/nextcloud/>
    Require all granted
    AllowOverride All
    Options FollowSymLinks MultiViews

    SetEnv HOME /var/www/nextcloud
    SetEnv HTTP_HOME /var/www/nextcloud

    <IfModule mod_dav.c>
      Dav off
    </IfModule>
  </Directory>
</VirtualHost>`, state.DomainName)
	}

	return `Alias /nextcloud "/var/www/nextcloud/"

<Directory /var/www/nextcloud/>
  Require all granted
  AllowOverride All
  Options FollowSymLinks MultiViews

  SetEnv HOME /var/www/nextcloud
  SetEnv HTTP_HOME /var/www/nextcloud

  <IfModule mod_dav.c>
    Dav off
  </IfModule>
</Directory>`
}

func (s *ApacheConfStep) View(state *State) string {
	var sections []string

	desc := style.DescriptionStyle.Render(
		"Configure Apache to serve Nextcloud. Choose between running under a\ndirectory path or a dedicated domain/subdomain.")
	sections = append(sections, "", desc, "")

	switch s.phase {
	case acCheckExisting:
		sections = append(sections, style.DescriptionStyle.Render("Checking for existing Apache configuration..."))

	case acSelectMode:
		sections = append(sections, s.selector.View())

	case acInputDomain:
		sections = append(sections,
			style.SuccessStyle.Render("  ✓ Mode: Domain-based virtual host"),
			"", s.input.View(),
		)

	case acConfirm:
		mode := "Directory (/nextcloud)"
		if state.ApacheMode == "domain" {
			mode = "Domain: " + state.DomainName
		}
		sections = append(sections,
			style.SuccessStyle.Render("  ✓ Mode: ")+style.TextStyle.Render(mode),
			"",
			style.SubtitleStyle.Render("Configuration to write:"),
			"", ui.CodeBlock(s.configText), "",
			s.confirm.View(),
		)

	case acWritingConfig, acEnablingSite, acReloadingApache:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		sections = append(sections, "", s.spinner.View())

	case acDone:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		w := state.Width - 8
		if w > 70 {
			w = 70
		}
		doneMsg := "Apache configured for Nextcloud!"
		if s.alreadyInstalled {
			doneMsg = "Existing Apache configuration kept — skipping."
		}
		sections = append(sections, "", ui.SuccessBox(doneMsg, w))
		sections = append(sections, "", style.KeyHintStyle.Render("Press ENTER to continue →"))

	case acError:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		w := state.Width - 8
		if w > 70 {
			w = 70
		}
		sections = append(sections, "", ui.WarningBox(s.errMsg, w))
		sections = append(sections, "", style.KeyHintStyle.Render("Press ENTER to retry"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
