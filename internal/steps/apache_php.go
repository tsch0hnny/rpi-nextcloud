package steps

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/exec"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
	"github.com/tsch0hnny/rpi-nextcloud/internal/ui"
)

type apachePhase int

const (
	apCheckExisting apachePhase = iota
	apSudoCheck
	apSudoPassword
	apConfirmInstall
	apUpdating
	apUpgrading
	apAddingPHPRepo
	apInstallingApache
	apInstallingPHP
	apEnablingModules
	apRestartingApache
	apVerifying
	apDone
	apError
)

type ApachePHPStep struct {
	phase            apachePhase
	complete         bool
	confirm          ui.ConfirmModel
	passwordInput    ui.PasswordModel
	spinner          ui.SpinnerModel
	output           string
	errMsg           string
	completedSub     []string
	alreadyInstalled bool
}

func NewApachePHPStep() *ApachePHPStep {
	return &ApachePHPStep{}
}

func (s *ApachePHPStep) ID() string      { return "apache-php" }
func (s *ApachePHPStep) Title() string    { return "Apache & PHP" }
func (s *ApachePHPStep) IsOptional() bool { return false }
func (s *ApachePHPStep) IsComplete() bool { return s.complete }

func (s *ApachePHPStep) Init(state *State) tea.Cmd {
	s.phase = apCheckExisting
	// Check if Apache, PHP 8.4, and the Apache PHP module are all properly set up
	return exec.RunCommand("check-apache-php",
		"dpkg -l apache2 2>/dev/null | grep -q '^ii' && "+
			"php8.4 --version >/dev/null 2>&1 && "+
			"apache2ctl -M 2>/dev/null | grep -q php && "+
			"echo 'fully_installed'")
}

func (s *ApachePHPStep) Update(msg tea.Msg, state *State) (Step, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch s.phase {
		case apSudoPassword:
			var cmd tea.Cmd
			s.passwordInput, cmd = s.passwordInput.Update(msg)
			return s, cmd
		case apConfirmInstall:
			var cmd tea.Cmd
			s.confirm, cmd = s.confirm.Update(msg)
			return s, cmd
		case apDone:
			if key.Matches(msg, style.Keys.Enter) {
				s.complete = true
				return s, func() tea.Msg { return StepCompleteMsg{} }
			}
		case apError:
			if key.Matches(msg, style.Keys.Enter) {
				s.phase = apConfirmInstall
				s.confirm = ui.NewConfirm("Retry installation?", true)
				s.errMsg = ""
				return s, nil
			}
			if msg.String() == "q" {
				return s, func() tea.Msg { return QuitConfirmMsg{} }
			}
		}

	case ui.PasswordResult:
		exec.SetSudoPassword(msg.Value)
		s.phase = apConfirmInstall
		s.confirm = ui.NewConfirm("Ready to install Apache2 and PHP 8.4?", true)
		return s, exec.TestSudo()

	case ui.ConfirmResult:
		if s.phase == apConfirmInstall {
			if msg.Confirmed {
				if s.alreadyInstalled {
					// Packages already installed — just enable modules and restart
					s.phase = apEnablingModules
					s.spinner = ui.NewSpinner("Enabling Apache modules...")
					return s, tea.Batch(
						s.spinner.Init(),
						exec.RunSudoCommand("enable-modules", "a2enmod rewrite headers env dir mime php8.4"),
					)
				}
				s.phase = apUpdating
				s.spinner = ui.NewSpinner("Updating package lists...")
				s.completedSub = nil
				return s, tea.Batch(
					s.spinner.Init(),
					exec.RunSudoCommand("apt-update", "apt update -y"),
				)
			}
			s.confirm = ui.NewConfirm("This step is required. Install Apache2 and PHP 8.4?", true)
			return s, nil
		}

	case exec.CmdResult:
		switch msg.Tag {
		case "check-apache-php":
			if msg.Err == nil && strings.TrimSpace(msg.Output) == "fully_installed" {
				// Packages installed AND PHP module loaded — truly nothing to do
				s.alreadyInstalled = true
				s.phase = apDone
				s.completedSub = []string{
					"Apache2 already installed",
					"PHP 8.4 already installed",
					"Apache PHP module already loaded",
				}
				return s, nil
			}
			// Check if packages are installed but module isn't loaded
			return s, exec.RunCommand("check-packages-only",
				"dpkg -l apache2 2>/dev/null | grep -q '^ii' && php8.4 --version >/dev/null 2>&1 && echo 'packages_ok'")

		case "check-packages-only":
			if msg.Err == nil && strings.TrimSpace(msg.Output) == "packages_ok" {
				// Packages exist but PHP module not loaded — need sudo to enable modules + restart
				s.alreadyInstalled = true
				s.completedSub = []string{
					"Apache2 already installed",
					"PHP 8.4 already installed",
				}
				s.phase = apSudoCheck
				return s, exec.CheckSudoNopass()
			}
			// Nothing installed — full installation needed
			s.phase = apSudoCheck
			return s, exec.CheckSudoNopass()

		case "check-sudo-nopass":
			confirmMsg := "Ready to install Apache2 and PHP 8.4?"
			if s.alreadyInstalled {
				confirmMsg = "Packages installed but PHP module not loaded. Enable modules and restart Apache?"
			}
			if msg.Err == nil {
				s.phase = apConfirmInstall
				s.confirm = ui.NewConfirm(confirmMsg, true)
				return s, nil
			}
			if exec.HasSudoPassword() {
				s.phase = apConfirmInstall
				s.confirm = ui.NewConfirm(confirmMsg, true)
				return s, nil
			}
			s.phase = apSudoPassword
			s.passwordInput = ui.NewPassword("Sudo Password",
				"Root privileges are required to install packages.")
			return s, s.passwordInput.Init()

		case "test-sudo":
			if msg.Err != nil {
				s.phase = apSudoPassword
				s.passwordInput = ui.NewPassword("Sudo Password",
					"Incorrect password. Please try again.")
				s.passwordInput.Reset()
				exec.SetSudoPassword("")
				return s, s.passwordInput.Init()
			}
			return s, nil

		case "apt-update":
			s.completedSub = append(s.completedSub, "Package lists updated")
			if msg.Err != nil {
				s.phase = apError
				s.errMsg = "Failed to update packages: " + msg.Err.Error()
				return s, nil
			}
			s.phase = apUpgrading
			s.spinner = ui.NewSpinner("Upgrading installed packages...")
			return s, tea.Batch(
				s.spinner.Init(),
				exec.RunSudoCommand("apt-upgrade", "DEBIAN_FRONTEND=noninteractive apt upgrade -y"),
			)

		case "apt-upgrade":
			s.completedSub = append(s.completedSub, "Packages upgraded")
			if msg.Err != nil {
				s.phase = apError
				s.errMsg = "Failed to upgrade packages: " + msg.Err.Error()
				return s, nil
			}
			s.phase = apAddingPHPRepo
			s.spinner = ui.NewSpinner("Adding PHP 8.4 repository...")
			// Add the Sury PHP repo for PHP 8.4 (required on Debian/Raspbian)
			// First check if sury repo is already configured (via debsuryorg-archive-keyring or manual setup)
			// If so, skip adding it again to avoid GPG key path conflicts
			addRepoCmd := `apt install -y lsb-release ca-certificates curl && ` +
				`if grep -rqs 'packages.sury.org/php' /etc/apt/sources.list /etc/apt/sources.list.d/ 2>/dev/null; then ` +
				`echo "Sury PHP repository already configured, skipping"; ` +
				`else ` +
				`curl -sSLo /tmp/php.gpg https://packages.sury.org/php/apt.gpg && ` +
				`gpg --dearmor < /tmp/php.gpg > /usr/share/keyrings/deb.sury.org-php.gpg 2>/dev/null && ` +
				`echo "deb [signed-by=/usr/share/keyrings/deb.sury.org-php.gpg] https://packages.sury.org/php/ $(lsb_release -sc) main" > /etc/apt/sources.list.d/sury-php.list && ` +
				`rm -f /tmp/php.gpg; ` +
				`fi && ` +
				`apt update -y`
			return s, tea.Batch(
				s.spinner.Init(),
				exec.RunSudoCommand("add-php-repo", addRepoCmd),
			)

		case "add-php-repo":
			s.completedSub = append(s.completedSub, "PHP 8.4 repository added")
			if msg.Err != nil {
				s.phase = apError
				s.errMsg = "Failed to add PHP repository: " + msg.Err.Error()
				return s, nil
			}
			s.phase = apInstallingApache
			s.spinner = ui.NewSpinner("Installing Apache2...")
			return s, tea.Batch(
				s.spinner.Init(),
				exec.RunSudoCommand("install-apache", "DEBIAN_FRONTEND=noninteractive apt install -y apache2"),
			)

		case "install-apache":
			s.completedSub = append(s.completedSub, "Apache2 installed")
			if msg.Err != nil {
				s.phase = apError
				s.errMsg = "Failed to install Apache2: " + msg.Err.Error()
				return s, nil
			}
			s.phase = apInstallingPHP
			s.spinner = ui.NewSpinner("Installing PHP 8.4 and extensions...")
			phpPkgs := "php8.4 php8.4-gd php8.4-sqlite3 php8.4-curl php8.4-zip " +
				"php8.4-xml php8.4-simplexml php8.4-mbstring php8.4-mysql php8.4-bz2 php8.4-intl " +
				"php8.4-smbclient php8.4-gmp php8.4-bcmath libapache2-mod-php8.4"
			return s, tea.Batch(
				s.spinner.Init(),
				exec.RunSudoCommand("install-php", "DEBIAN_FRONTEND=noninteractive apt install -y "+phpPkgs),
			)

		case "install-php":
			s.completedSub = append(s.completedSub, "PHP 8.4 with extensions installed")
			if msg.Err != nil {
				s.phase = apError
				s.errMsg = "Failed to install PHP: " + msg.Err.Error()
				return s, nil
			}
			s.phase = apEnablingModules
			s.spinner = ui.NewSpinner("Enabling Apache modules...")
			return s, tea.Batch(
				s.spinner.Init(),
				exec.RunSudoCommand("enable-modules", "a2enmod rewrite headers env dir mime php8.4"),
			)

		case "enable-modules":
			s.completedSub = append(s.completedSub, "Apache modules enabled (rewrite, headers, env, dir, mime)")
			if msg.Err != nil {
				s.phase = apError
				s.errMsg = "Failed to enable Apache modules: " + msg.Err.Error()
				return s, nil
			}
			s.phase = apRestartingApache
			s.spinner = ui.NewSpinner("Restarting Apache2...")
			return s, tea.Batch(
				s.spinner.Init(),
				exec.RunSudoCommand("restart-apache", "service apache2 restart"),
			)

		case "restart-apache":
			s.completedSub = append(s.completedSub, "Apache2 restarted")
			if msg.Err != nil {
				s.phase = apError
				s.errMsg = "Failed to restart Apache: " + msg.Err.Error()
				return s, nil
			}
			// Verify PHP module is actually loaded
			s.phase = apVerifying
			s.spinner = ui.NewSpinner("Verifying PHP module is loaded...")
			return s, tea.Batch(
				s.spinner.Init(),
				exec.RunCommand("verify-php-mod", "apache2ctl -M 2>/dev/null | grep -q php && echo 'ok'"),
			)

		case "verify-php-mod":
			if msg.Err != nil || strings.TrimSpace(msg.Output) != "ok" {
				s.phase = apError
				s.errMsg = "PHP module is not loaded in Apache. Try: sudo a2enmod php8.4 && sudo systemctl restart apache2"
				return s, nil
			}
			s.completedSub = append(s.completedSub, "PHP module verified in Apache")
			s.phase = apDone
			return s, nil
		}
	}

	// Update spinner animation
	if s.phase >= apUpdating && s.phase <= apVerifying {
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	}

	return s, nil
}

func (s *ApachePHPStep) View(state *State) string {
	var sections []string

	desc := style.DescriptionStyle.Render(
		"We'll install Apache2 web server and PHP 8.4 with all required\nextensions for Nextcloud.")
	sections = append(sections, "", desc, "")

	switch s.phase {
	case apCheckExisting:
		sections = append(sections, style.DescriptionStyle.Render("Checking for existing installation..."))

	case apSudoCheck:
		sections = append(sections, style.DescriptionStyle.Render("Checking sudo access..."))

	case apSudoPassword:
		sections = append(sections, s.passwordInput.View())

	case apConfirmInstall:
		packages := ui.CodeBlock(
			"sudo apt update && sudo apt upgrade\n" +
				"# Add PHP 8.4 repository (packages.sury.org)\n" +
				"sudo apt install apache2\n" +
				"sudo apt install php8.4 php8.4-gd php8.4-sqlite3\n" +
				"  php8.4-curl php8.4-zip php8.4-xml php8.4-simplexml\n" +
				"  php8.4-mbstring\n" +
				"  php8.4-mysql php8.4-bz2 php8.4-intl php8.4-smbclient\n" +
				"  php8.4-gmp php8.4-bcmath libapache2-mod-php8.4\n" +
				"sudo a2enmod rewrite headers env dir mime php8.4")
		sections = append(sections,
			style.SubtitleStyle.Render("The following will be installed:"),
			"", packages, "",
			s.confirm.View(),
		)

	case apUpdating, apUpgrading, apAddingPHPRepo, apInstallingApache,
		apInstallingPHP, apEnablingModules, apRestartingApache, apVerifying:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		sections = append(sections, "", s.spinner.View())

	case apDone:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		sections = append(sections, "")
		w := state.Width - 8
		if w > 70 {
			w = 70
		}
		doneMsg := "Apache2 and PHP 8.4 installed successfully!"
		if s.alreadyInstalled {
			doneMsg = "Apache2 and PHP 8.4 are already installed — skipping."
		}
		sections = append(sections, ui.SuccessBox(doneMsg, w))
		sections = append(sections, "", style.KeyHintStyle.Render("Press ENTER to continue →"))

	case apError:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		w := state.Width - 8
		if w > 70 {
			w = 70
		}
		sections = append(sections, "", ui.WarningBox(s.errMsg, w))
		sections = append(sections, "", style.KeyHintStyle.Render("Press ENTER to retry  |  q: quit"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
