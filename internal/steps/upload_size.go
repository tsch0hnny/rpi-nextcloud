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

type usPhase int

const (
	usInputLimit usPhase = iota
	usConfirm
	usUpdatingPHP
	usRestartingApache
	usDone
	usError
)

type UploadSizeStep struct {
	phase        usPhase
	complete     bool
	input        ui.InputModel
	confirm      ui.ConfirmModel
	spinner      ui.SpinnerModel
	completedSub []string
	errMsg       string
}

func NewUploadSizeStep() *UploadSizeStep {
	return &UploadSizeStep{}
}

func (s *UploadSizeStep) ID() string       { return "upload-size" }
func (s *UploadSizeStep) Title() string     { return "Upload Limit" }
func (s *UploadSizeStep) IsOptional() bool  { return true }
func (s *UploadSizeStep) IsComplete() bool  { return s.complete }

func (s *UploadSizeStep) Init(state *State) tea.Cmd {
	s.phase = usInputLimit
	s.input = ui.NewInputWithValidation("Max Upload Size", "1024M", state.PHPUploadLimit,
		"PHP's default 2MB limit is too low for cloud storage.\nRecommended: 1024M (1 GB). Use values like 512M, 2048M, etc.",
		ui.ValidatePHPSize)
	s.input.EscHint = "esc: skip"
	return s.input.Init()
}

func (s *UploadSizeStep) Update(msg tea.Msg, state *State) (Step, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch s.phase {
		case usInputLimit:
			if key.Matches(msg, style.Keys.Escape) {
				return s, func() tea.Msg { return StepSkipMsg{} }
			}
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			return s, cmd
		case usConfirm:
			if key.Matches(msg, style.Keys.Escape) {
				return s, func() tea.Msg { return StepSkipMsg{} }
			}
			var cmd tea.Cmd
			s.confirm, cmd = s.confirm.Update(msg)
			return s, cmd
		case usDone:
			if key.Matches(msg, style.Keys.Enter) {
				s.complete = true
				return s, func() tea.Msg { return StepCompleteMsg{} }
			}
		case usError:
			if key.Matches(msg, style.Keys.Enter) {
				s.phase = usConfirm
				s.confirm = ui.NewConfirm("Retry?", true)
				return s, nil
			}
		}

	case ui.InputResult:
		if s.phase == usInputLimit {
			state.PHPUploadLimit = msg.Value
			s.phase = usConfirm
			s.confirm = ui.NewConfirm(
				fmt.Sprintf("Set upload limit to %s?", state.PHPUploadLimit), true)
			return s, nil
		}

	case ui.ConfirmResult:
		if s.phase == usConfirm {
			if !msg.Confirmed {
				return s, func() tea.Msg { return StepSkipMsg{} }
			}
			s.phase = usUpdatingPHP
			s.spinner = ui.NewSpinner("Updating PHP configuration...")
			s.completedSub = nil
			// Update both values in php.ini
			sedCmd := fmt.Sprintf(
				`PHP_INI=$(find /etc/php -name "php.ini" -path "*/apache2/*" 2>/dev/null | head -1) && `+
					`sed -i "s/^post_max_size = .*/post_max_size = %s/" "$PHP_INI" && `+
					`sed -i "s/^upload_max_filesize = .*/upload_max_filesize = %s/" "$PHP_INI"`,
				state.PHPUploadLimit, state.PHPUploadLimit)
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("update-php", sedCmd))
		}

	case exec.CmdResult:
		switch msg.Tag {
		case "update-php":
			s.completedSub = append(s.completedSub,
				fmt.Sprintf("post_max_size = %s", state.PHPUploadLimit),
				fmt.Sprintf("upload_max_filesize = %s", state.PHPUploadLimit),
			)
			if msg.Err != nil {
				s.phase = usError
				s.errMsg = "Failed to update PHP config: " + msg.Err.Error()
				return s, nil
			}
			s.phase = usRestartingApache
			s.spinner = ui.NewSpinner("Restarting Apache...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("restart-apache", "service apache2 restart"))

		case "restart-apache":
			s.completedSub = append(s.completedSub, "Apache restarted")
			if msg.Err != nil {
				s.phase = usError
				s.errMsg = "Failed to restart Apache: " + msg.Err.Error()
				return s, nil
			}
			s.phase = usDone
			return s, nil
		}
	}

	if s.phase >= usUpdatingPHP && s.phase <= usRestartingApache {
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	}

	return s, nil
}

func (s *UploadSizeStep) View(state *State) string {
	var sections []string

	desc := style.DescriptionStyle.Render(
		"PHP's default upload limit is only 2MB — far too low for cloud storage.\nIncrease it to something practical.")
	sections = append(sections, "", desc, "")

	switch s.phase {
	case usInputLimit:
		sections = append(sections, s.input.View())
	case usConfirm:
		code := ui.CodeBlock(fmt.Sprintf("post_max_size = %s\nupload_max_filesize = %s",
			state.PHPUploadLimit, state.PHPUploadLimit))
		sections = append(sections,
			style.SubtitleStyle.Render("PHP configuration changes:"),
			"", code, "",
			s.confirm.View(),
		)
	case usUpdatingPHP, usRestartingApache:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		sections = append(sections, "", s.spinner.View())
	case usDone:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		w := state.Width - 8
		if w > 70 {
			w = 70
		}
		sections = append(sections, "", ui.SuccessBox(
			fmt.Sprintf("Upload limit increased to %s!", state.PHPUploadLimit), w))
		sections = append(sections, "", style.KeyHintStyle.Render("Press ENTER to continue →"))
	case usError:
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
