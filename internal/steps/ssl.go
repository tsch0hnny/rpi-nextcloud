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

type sslPhase int

const (
	sslConfirmEnable sslPhase = iota
	sslCreatingDir
	sslGeneratingCert
	sslEnablingMod
	sslUpdatingSSLConf
	sslEnablingSSLSite
	sslConfirmForceHTTPS
	sslWritingRedirect
	sslEnablingRewrite
	sslRestartingApache
	sslDone
	sslError
)

type SSLStep struct {
	phase        sslPhase
	complete     bool
	confirm      ui.ConfirmModel
	spinner      ui.SpinnerModel
	completedSub []string
	errMsg       string
}

func NewSSLStep() *SSLStep {
	return &SSLStep{}
}

func (s *SSLStep) ID() string       { return "ssl" }
func (s *SSLStep) Title() string     { return "SSL / HTTPS" }
func (s *SSLStep) IsOptional() bool  { return true }
func (s *SSLStep) IsComplete() bool  { return s.complete }

func (s *SSLStep) Init(state *State) tea.Cmd {
	s.phase = sslConfirmEnable
	s.confirm = ui.NewConfirm("Set up SSL with a self-signed certificate?", true)
	return nil
}

func (s *SSLStep) Update(msg tea.Msg, state *State) (Step, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, style.Keys.Skip) && s.phase <= sslConfirmEnable {
			state.SSLEnabled = false
			return s, func() tea.Msg { return StepSkipMsg{} }
		}

		switch s.phase {
		case sslConfirmEnable, sslConfirmForceHTTPS:
			var cmd tea.Cmd
			s.confirm, cmd = s.confirm.Update(msg)
			return s, cmd
		case sslDone:
			if key.Matches(msg, style.Keys.Enter) {
				s.complete = true
				return s, func() tea.Msg { return StepCompleteMsg{} }
			}
		case sslError:
			if key.Matches(msg, style.Keys.Enter) {
				s.phase = sslConfirmEnable
				s.confirm = ui.NewConfirm("Retry SSL setup?", true)
				return s, nil
			}
		}

	case ui.ConfirmResult:
		switch s.phase {
		case sslConfirmEnable:
			if !msg.Confirmed {
				state.SSLEnabled = false
				return s, func() tea.Msg { return StepSkipMsg{} }
			}
			state.SSLEnabled = true
			s.phase = sslCreatingDir
			s.spinner = ui.NewSpinner("Creating SSL directory...")
			s.completedSub = nil
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("mkdir-ssl", "mkdir -p /etc/apache2/ssl"))

		case sslConfirmForceHTTPS:
			if msg.Confirmed {
				state.ForceHTTPS = true
				s.phase = sslWritingRedirect
				s.spinner = ui.NewSpinner("Writing HTTP redirect config...")
				redirectConf := `<VirtualHost *:80>
   ServerAdmin webmaster@localhost
   RewriteEngine On
   RewriteCond %{HTTPS} off
   RewriteRule ^(.*)$ https://%{HTTP_HOST}$1 [R=301,L]
</VirtualHost>`
				return s, tea.Batch(s.spinner.Init(),
					exec.RunSudoCommand("write-redirect",
						fmt.Sprintf("cat > /etc/apache2/sites-available/000-default.conf << 'CONFEOF'\n%s\nCONFEOF", redirectConf)))
			}
			state.ForceHTTPS = false
			s.phase = sslRestartingApache
			s.spinner = ui.NewSpinner("Restarting Apache...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("restart-apache-final", "service apache2 restart"))
		}

	case exec.CmdResult:
		switch msg.Tag {
		case "mkdir-ssl":
			s.completedSub = append(s.completedSub, "SSL directory created")
			if msg.Err != nil {
				s.phase = sslError
				s.errMsg = "Failed to create SSL directory: " + msg.Err.Error()
				return s, nil
			}
			s.phase = sslGeneratingCert
			s.spinner = ui.NewSpinner("Generating self-signed certificate (RSA 4096)...")
			certCmd := `openssl req -x509 -nodes -days 365 -newkey rsa:4096 ` +
				`-keyout /etc/apache2/ssl/apache.key -out /etc/apache2/ssl/apache.crt ` +
				`-subj "/C=US/ST=State/L=City/O=Nextcloud/CN=localhost"`
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("gen-cert", certCmd))

		case "gen-cert":
			s.completedSub = append(s.completedSub, "Self-signed certificate generated (365 days)")
			if msg.Err != nil {
				s.phase = sslError
				s.errMsg = "Failed to generate certificate: " + msg.Err.Error()
				return s, nil
			}
			s.phase = sslEnablingMod
			s.spinner = ui.NewSpinner("Enabling SSL module...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("enable-ssl-mod", "a2enmod ssl"))

		case "enable-ssl-mod":
			s.completedSub = append(s.completedSub, "SSL module enabled")
			if msg.Err != nil {
				s.phase = sslError
				s.errMsg = "Failed to enable SSL module: " + msg.Err.Error()
				return s, nil
			}
			s.phase = sslUpdatingSSLConf
			s.spinner = ui.NewSpinner("Updating SSL configuration...")
			sedCmd := `sed -i 's|SSLCertificateFile.*|SSLCertificateFile /etc/apache2/ssl/apache.crt|' /etc/apache2/sites-available/default-ssl.conf && ` +
				`sed -i 's|SSLCertificateKeyFile.*|SSLCertificateKeyFile /etc/apache2/ssl/apache.key|' /etc/apache2/sites-available/default-ssl.conf`
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("update-ssl-conf", sedCmd))

		case "update-ssl-conf":
			s.completedSub = append(s.completedSub, "SSL config updated with new certificate paths")
			if msg.Err != nil {
				s.phase = sslError
				s.errMsg = "Failed to update SSL config: " + msg.Err.Error()
				return s, nil
			}
			s.phase = sslEnablingSSLSite
			s.spinner = ui.NewSpinner("Enabling SSL site...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("enable-ssl-site", "a2ensite default-ssl.conf"))

		case "enable-ssl-site":
			s.completedSub = append(s.completedSub, "SSL site enabled")
			if msg.Err != nil {
				s.phase = sslError
				s.errMsg = "Failed to enable SSL site: " + msg.Err.Error()
				return s, nil
			}
			// Ask about HTTPS redirect
			s.phase = sslConfirmForceHTTPS
			s.confirm = ui.NewConfirm("Force redirect all HTTP traffic to HTTPS?", true)
			return s, nil

		case "write-redirect":
			s.completedSub = append(s.completedSub, "HTTP → HTTPS redirect configured")
			if msg.Err != nil {
				s.phase = sslError
				s.errMsg = "Failed to write redirect config: " + msg.Err.Error()
				return s, nil
			}
			s.phase = sslEnablingRewrite
			s.spinner = ui.NewSpinner("Enabling rewrite module...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("enable-rewrite", "a2enmod rewrite"))

		case "enable-rewrite":
			s.completedSub = append(s.completedSub, "Rewrite module enabled")
			if msg.Err != nil {
				s.phase = sslError
				s.errMsg = "Failed to enable rewrite module: " + msg.Err.Error()
				return s, nil
			}
			s.phase = sslRestartingApache
			s.spinner = ui.NewSpinner("Restarting Apache...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("restart-apache-final", "service apache2 restart"))

		case "restart-apache-final":
			s.completedSub = append(s.completedSub, "Apache restarted")
			if msg.Err != nil {
				s.phase = sslError
				s.errMsg = "Failed to restart Apache: " + msg.Err.Error()
				return s, nil
			}
			s.phase = sslDone
			return s, nil
		}
	}

	if s.phase >= sslCreatingDir && s.phase <= sslRestartingApache &&
		s.phase != sslConfirmForceHTTPS {
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	}

	return s, nil
}

func (s *SSLStep) View(state *State) string {
	var sections []string

	desc := style.DescriptionStyle.Render(
		"Set up HTTPS with a self-signed SSL certificate.\nNote: Self-signed certificates will show browser warnings but\nencrypt your traffic. Use Let's Encrypt for trusted certificates.")
	sections = append(sections, "", desc, "")

	switch s.phase {
	case sslConfirmEnable:
		code := ui.CodeBlock("openssl req -x509 -nodes -days 365 -newkey rsa:4096 ...\na2enmod ssl\na2ensite default-ssl.conf")
		sections = append(sections,
			style.SubtitleStyle.Render("This will:"),
			style.TextStyle.Render("  • Generate a self-signed RSA 4096-bit certificate"),
			style.TextStyle.Render("  • Enable Apache SSL module"),
			style.TextStyle.Render("  • Configure Apache to use the certificate"),
			"", code, "",
			s.confirm.View(),
		)

	case sslConfirmForceHTTPS:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		sections = append(sections, "",
			style.SubtitleStyle.Render("Optional: Force HTTPS Redirect"),
			style.DescriptionStyle.Render("  This will redirect all HTTP requests to HTTPS automatically."),
			"",
			s.confirm.View(),
		)

	case sslCreatingDir, sslGeneratingCert, sslEnablingMod, sslUpdatingSSLConf,
		sslEnablingSSLSite, sslWritingRedirect, sslEnablingRewrite, sslRestartingApache:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		sections = append(sections, "", s.spinner.View())

	case sslDone:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		w := state.Width - 8
		if w > 70 {
			w = 70
		}
		httpsURL := fmt.Sprintf("https://%s/nextcloud", state.IPAddress)
		if state.ApacheMode == "domain" {
			httpsURL = fmt.Sprintf("https://%s", state.DomainName)
		}
		sections = append(sections, "",
			ui.SuccessBox("SSL configured successfully!", w),
			"",
			style.TextStyle.Render("  Test URL: ")+
				lipgloss.NewStyle().Foreground(style.ColorAccent).Underline(true).Render(httpsURL),
			"",
			ui.WarningBox("Your browser will show a certificate warning — this is\nnormal for self-signed certificates. Click 'Advanced' to proceed.", w),
			"",
			style.KeyHintStyle.Render("Press ENTER to continue →"),
		)

	case sslError:
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
