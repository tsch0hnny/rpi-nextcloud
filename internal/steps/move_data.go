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

type mdPhase int

const (
	mdInputDir mdPhase = iota
	mdConfirm
	mdCreatingDir
	mdMovingData
	mdUpdatingConfig
	mdFixingOwnership
	mdDone
	mdError
)

type MoveDataStep struct {
	phase        mdPhase
	complete     bool
	input        ui.InputModel
	confirm      ui.ConfirmModel
	spinner      ui.SpinnerModel
	completedSub []string
	errMsg       string
}

func NewMoveDataStep() *MoveDataStep {
	return &MoveDataStep{}
}

func (s *MoveDataStep) ID() string       { return "move-data" }
func (s *MoveDataStep) Title() string     { return "Move Data Directory" }
func (s *MoveDataStep) IsOptional() bool  { return true }
func (s *MoveDataStep) IsComplete() bool  { return s.complete }

func (s *MoveDataStep) Init(state *State) tea.Cmd {
	s.phase = mdInputDir
	s.input = ui.NewInputWithValidation("Data Directory", "/var/nextcloud/data", state.DataDirectory,
		"Moving the data directory outside of /var/www improves security.\nThis is also how you'd point to an external hard drive.",
		ui.ValidatePath)
	return s.input.Init()
}

func (s *MoveDataStep) Update(msg tea.Msg, state *State) (Step, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch s.phase {
		case mdInputDir:
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			return s, cmd
		case mdConfirm:
			if key.Matches(msg, style.Keys.Escape) {
				return s, func() tea.Msg { return StepSkipMsg{} }
			}
			var cmd tea.Cmd
			s.confirm, cmd = s.confirm.Update(msg)
			return s, cmd
		case mdDone:
			if key.Matches(msg, style.Keys.Enter) {
				s.complete = true
				return s, func() tea.Msg { return StepCompleteMsg{} }
			}
		case mdError:
			if key.Matches(msg, style.Keys.Enter) {
				s.phase = mdConfirm
				s.confirm = ui.NewConfirm("Retry moving data directory?", true)
				return s, nil
			}
		}

	case ui.InputResult:
		if s.phase == mdInputDir {
			state.DataDirectory = msg.Value
			s.phase = mdConfirm
			s.confirm = ui.NewConfirm(fmt.Sprintf("Move data to %s?", state.DataDirectory), true)
			return s, nil
		}

	case ui.ConfirmResult:
		if s.phase == mdConfirm {
			if !msg.Confirmed {
				return s, func() tea.Msg { return StepSkipMsg{} }
			}
			s.phase = mdCreatingDir
			s.spinner = ui.NewSpinner("Creating target directory...")
			s.completedSub = nil
			// Get the parent directory
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("mkdir-target",
					fmt.Sprintf("mkdir -p %s", parentDir(state.DataDirectory))))
		}

	case exec.CmdResult:
		switch msg.Tag {
		case "mkdir-target":
			s.completedSub = append(s.completedSub, "Target directory created")
			if msg.Err != nil {
				s.phase = mdError
				s.errMsg = "Failed to create directory: " + msg.Err.Error()
				return s, nil
			}
			s.phase = mdMovingData
			s.spinner = ui.NewSpinner("Moving data (this may take a while)...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("move-data",
					fmt.Sprintf("mv -v /var/www/nextcloud/data %s", state.DataDirectory)))

		case "move-data":
			s.completedSub = append(s.completedSub, fmt.Sprintf("Data moved to %s", state.DataDirectory))
			if msg.Err != nil {
				s.phase = mdError
				s.errMsg = "Failed to move data: " + msg.Err.Error()
				return s, nil
			}
			s.phase = mdUpdatingConfig
			s.spinner = ui.NewSpinner("Updating Nextcloud config...")
			sedCmd := fmt.Sprintf(
				`sed -i "s|'datadirectory' => '/var/www/nextcloud/data'|'datadirectory' => '%s'|" /var/www/nextcloud/config/config.php`,
				state.DataDirectory)
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("update-config", sedCmd))

		case "update-config":
			s.completedSub = append(s.completedSub, "Config updated")
			if msg.Err != nil {
				s.phase = mdError
				s.errMsg = "Failed to update config: " + msg.Err.Error()
				return s, nil
			}
			s.phase = mdFixingOwnership
			s.spinner = ui.NewSpinner("Fixing ownership...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("fix-ownership",
					fmt.Sprintf("chown -R www-data:www-data %s", state.DataDirectory)))

		case "fix-ownership":
			s.completedSub = append(s.completedSub, "Ownership set to www-data")
			if msg.Err != nil {
				s.phase = mdError
				s.errMsg = "Failed to fix ownership: " + msg.Err.Error()
				return s, nil
			}
			s.phase = mdDone
			return s, nil
		}
	}

	if s.phase >= mdCreatingDir && s.phase <= mdFixingOwnership {
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	}

	return s, nil
}

func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' && i != len(path)-1 {
			return path[:i]
		}
	}
	return path
}

func (s *MoveDataStep) View(state *State) string {
	var sections []string

	desc := style.DescriptionStyle.Render(
		"Move Nextcloud's data directory outside the web-accessible folder\nfor improved security. Also useful for mounting external storage.")
	sections = append(sections, "", desc, "")

	switch s.phase {
	case mdInputDir:
		sections = append(sections, s.input.View())
	case mdConfirm:
		sections = append(sections,
			ui.StatusLine("Target", state.DataDirectory, style.ColorAccent),
			"", s.confirm.View(),
		)
	case mdCreatingDir, mdMovingData, mdUpdatingConfig, mdFixingOwnership:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		sections = append(sections, "", s.spinner.View())
	case mdDone:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		w := state.Width - 8
		if w > 70 {
			w = 70
		}
		sections = append(sections, "", ui.SuccessBox("Data directory moved successfully!", w))
		sections = append(sections, "", style.KeyHintStyle.Render("Press ENTER to continue →"))
	case mdError:
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
