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

type mysqlPhase int

const (
	myCheckExisting mysqlPhase = iota
	myInputDBName
	myInputDBUser
	myInputDBPassword
	myConfirmInstall
	myInstallingMariaDB
	myCreatingDB
	myCreatingUser
	myGranting
	myFlushing
	myDone
	myError
)

type MySQLStep struct {
	phase            mysqlPhase
	complete         bool
	input            ui.InputModel
	passwordInput    ui.PasswordModel
	confirm          ui.ConfirmModel
	spinner          ui.SpinnerModel
	completedSub     []string
	errMsg           string
	alreadyInstalled bool
	existingDBName   string
}

func NewMySQLStep() *MySQLStep {
	return &MySQLStep{}
}

func (s *MySQLStep) ID() string       { return "mysql" }
func (s *MySQLStep) Title() string     { return "Database Setup" }
func (s *MySQLStep) IsOptional() bool  { return false }
func (s *MySQLStep) IsComplete() bool  { return s.complete }

func (s *MySQLStep) Init(state *State) tea.Cmd {
	s.phase = myCheckExisting
	// Check if MariaDB is installed and the default database already exists
	return exec.RunCommand("check-mysql",
		fmt.Sprintf("dpkg -l mariadb-server 2>/dev/null | grep -q '^ii' && sudo mysql -u root -e \"SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='%s'\" 2>/dev/null | grep -q '%s' && echo 'exists'",
			state.DBName, state.DBName))
}

func (s *MySQLStep) Update(msg tea.Msg, state *State) (Step, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch s.phase {
		case myInputDBName:
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			return s, cmd
		case myInputDBUser:
			if key.Matches(msg, style.Keys.Escape) {
				s.phase = myInputDBName
				s.input = ui.NewInputWithValidation("Database Name", "nextclouddb", state.DBName,
					"The name of the MySQL database for Nextcloud.", ui.ValidateDBName)
				return s, s.input.Init()
			}
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			return s, cmd
		case myInputDBPassword:
			if key.Matches(msg, style.Keys.Escape) {
				s.phase = myInputDBUser
				s.input = ui.NewInputWithValidation("Database User", "nextclouduser", state.DBUser,
					"The MySQL user that Nextcloud will use to connect.", ui.ValidateDBUser)
				return s, s.input.Init()
			}
			var cmd tea.Cmd
			s.passwordInput, cmd = s.passwordInput.Update(msg)
			return s, cmd
		case myConfirmInstall:
			if key.Matches(msg, style.Keys.Escape) {
				s.phase = myInputDBPassword
				s.passwordInput = ui.NewPassword("Database Password",
					"Choose a strong password for the database user.")
				return s, s.passwordInput.Init()
			}
			var cmd tea.Cmd
			s.confirm, cmd = s.confirm.Update(msg)
			return s, cmd
		case myDone:
			if key.Matches(msg, style.Keys.Enter) {
				s.complete = true
				return s, func() tea.Msg { return StepCompleteMsg{} }
			}
		case myError:
			if key.Matches(msg, style.Keys.Enter) {
				s.phase = myConfirmInstall
				s.confirm = ui.NewConfirm("Retry database setup?", true)
				s.errMsg = ""
				return s, nil
			}
		}

	case ui.InputResult:
		switch s.phase {
		case myInputDBName:
			state.DBName = msg.Value
			s.phase = myInputDBUser
			s.input = ui.NewInputWithValidation("Database User", "nextclouduser", state.DBUser,
				"The MySQL user that Nextcloud will use to connect.", ui.ValidateDBUser)
			return s, s.input.Init()
		case myInputDBUser:
			state.DBUser = msg.Value
			s.phase = myInputDBPassword
			s.passwordInput = ui.NewPassword("Database Password",
				"Choose a strong password for the database user.")
			return s, s.passwordInput.Init()
		}

	case ui.PasswordResult:
		if s.phase == myInputDBPassword {
			state.DBPassword = msg.Value
			s.phase = myConfirmInstall
			s.confirm = ui.NewConfirm("Install MariaDB and create the database?", true)
			return s, nil
		}

	case ui.ConfirmResult:
		if s.phase == myConfirmInstall {
			if msg.Confirmed {
				s.phase = myInstallingMariaDB
				s.spinner = ui.NewSpinner("Installing MariaDB server...")
				s.completedSub = nil
				return s, tea.Batch(
					s.spinner.Init(),
					exec.RunSudoCommand("install-mariadb", "DEBIAN_FRONTEND=noninteractive apt install -y mariadb-server"),
				)
			}
			s.confirm = ui.NewConfirm("A database is required. Proceed with installation?", true)
			return s, nil
		}

	case exec.CmdResult:
		switch msg.Tag {
		case "check-mysql":
			if msg.Err == nil && strings.TrimSpace(msg.Output) == "exists" {
				s.alreadyInstalled = true
				s.existingDBName = state.DBName
				s.phase = myDone
				s.completedSub = []string{
					"MariaDB already installed",
					fmt.Sprintf("Database '%s' already exists", state.DBName),
				}
				return s, nil
			}
			// Not installed or DB doesn't exist — proceed with setup
			s.phase = myInputDBName
			s.input = ui.NewInputWithValidation("Database Name", "nextclouddb", state.DBName,
				"The name of the MySQL database for Nextcloud.", ui.ValidateDBName)
			return s, s.input.Init()

		case "install-mariadb":
			s.completedSub = append(s.completedSub, "MariaDB server installed")
			if msg.Err != nil {
				s.phase = myError
				s.errMsg = "Failed to install MariaDB: " + msg.Err.Error()
				return s, nil
			}
			s.phase = myCreatingDB
			s.spinner = ui.NewSpinner(fmt.Sprintf("Creating database '%s'...", state.DBName))
			sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", state.DBName)
			return s, tea.Batch(s.spinner.Init(), exec.RunSudoMySQL("create-db", sql))

		case "create-db":
			s.completedSub = append(s.completedSub, fmt.Sprintf("Database '%s' created", state.DBName))
			if msg.Err != nil {
				s.phase = myError
				s.errMsg = "Failed to create database: " + msg.Err.Error()
				return s, nil
			}
			s.phase = myCreatingUser
			s.spinner = ui.NewSpinner(fmt.Sprintf("Creating user '%s'...", state.DBUser))
			sql := fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s';",
				state.DBUser, state.DBPassword)
			return s, tea.Batch(s.spinner.Init(), exec.RunSudoMySQL("create-user", sql))

		case "create-user":
			s.completedSub = append(s.completedSub, fmt.Sprintf("User '%s' created", state.DBUser))
			if msg.Err != nil {
				s.phase = myError
				s.errMsg = "Failed to create user: " + msg.Err.Error()
				return s, nil
			}
			s.phase = myGranting
			s.spinner = ui.NewSpinner("Granting privileges...")
			sql := fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost';",
				state.DBName, state.DBUser)
			return s, tea.Batch(s.spinner.Init(), exec.RunSudoMySQL("grant", sql))

		case "grant":
			s.completedSub = append(s.completedSub, "Privileges granted")
			if msg.Err != nil {
				s.phase = myError
				s.errMsg = "Failed to grant privileges: " + msg.Err.Error()
				return s, nil
			}
			s.phase = myFlushing
			s.spinner = ui.NewSpinner("Flushing privileges...")
			return s, tea.Batch(s.spinner.Init(), exec.RunSudoMySQL("flush", "FLUSH PRIVILEGES;"))

		case "flush":
			s.completedSub = append(s.completedSub, "Privileges flushed")
			if msg.Err != nil {
				s.phase = myError
				s.errMsg = "Failed to flush privileges: " + msg.Err.Error()
				return s, nil
			}
			s.phase = myDone
			return s, nil
		}
	}

	// Update spinner
	if s.phase >= myInstallingMariaDB && s.phase <= myFlushing {
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	}

	return s, nil
}

func (s *MySQLStep) View(state *State) string {
	var sections []string

	desc := style.DescriptionStyle.Render(
		"Set up MariaDB with a dedicated database and user for Nextcloud.")
	sections = append(sections, "", desc, "")

	switch s.phase {
	case myCheckExisting:
		sections = append(sections, style.DescriptionStyle.Render("Checking for existing database..."))

	case myInputDBName:
		sections = append(sections, s.input.View())

	case myInputDBUser:
		sections = append(sections,
			style.SuccessStyle.Render("  ✓ Database: ")+style.TextStyle.Render(state.DBName),
			"", s.input.View(),
		)

	case myInputDBPassword:
		sections = append(sections,
			style.SuccessStyle.Render("  ✓ Database: ")+style.TextStyle.Render(state.DBName),
			style.SuccessStyle.Render("  ✓ User: ")+style.TextStyle.Render(state.DBUser),
			"", s.passwordInput.View(),
		)

	case myConfirmInstall:
		sections = append(sections,
			style.SubtitleStyle.Render("Configuration Summary:"),
			"",
			ui.StatusLine("Database", state.DBName, style.ColorAccent),
			ui.StatusLine("User", state.DBUser, style.ColorAccent),
			ui.StatusLine("Password", "••••••••", style.ColorAccent),
			"",
			s.confirm.View(),
		)

	case myInstallingMariaDB, myCreatingDB, myCreatingUser, myGranting, myFlushing:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		sections = append(sections, "", s.spinner.View())

	case myDone:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		w := state.Width - 8
		if w > 70 {
			w = 70
		}
		doneMsg := "Database configured successfully!"
		if s.alreadyInstalled {
			doneMsg = fmt.Sprintf("MariaDB and database '%s' already exist — skipping.", s.existingDBName)
		}
		sections = append(sections, "", ui.SuccessBox(doneMsg, w))
		sections = append(sections, "", style.KeyHintStyle.Render("Press ENTER to continue →"))

	case myError:
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
