package steps

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
	"github.com/tsch0hnny/rpi-nextcloud/internal/exec"
	"github.com/tsch0hnny/rpi-nextcloud/internal/ui"
)

type dlPhase int

const (
	dlConfirm dlPhase = iota
	dlDownloading
	dlExtracting
	dlCreatingDataDir
	dlSettingOwnership
	dlSettingPermissions
	dlCleanup
	dlDone
	dlError
)

type DownloadStep struct {
	phase        dlPhase
	complete     bool
	confirm      ui.ConfirmModel
	spinner      ui.SpinnerModel
	completedSub []string
	errMsg       string
}

func NewDownloadStep() *DownloadStep {
	return &DownloadStep{}
}

func (s *DownloadStep) ID() string       { return "download" }
func (s *DownloadStep) Title() string     { return "Download Nextcloud" }
func (s *DownloadStep) IsOptional() bool  { return false }
func (s *DownloadStep) IsComplete() bool  { return s.complete }

func (s *DownloadStep) Init(state *State) tea.Cmd {
	s.phase = dlConfirm
	s.confirm = ui.NewConfirm("Download and install the latest Nextcloud release?", true)
	return nil
}

func (s *DownloadStep) Update(msg tea.Msg, state *State) (Step, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch s.phase {
		case dlConfirm:
			var cmd tea.Cmd
			s.confirm, cmd = s.confirm.Update(msg)
			return s, cmd
		case dlDone:
			if key.Matches(msg, style.Keys.Enter) {
				s.complete = true
				return s, func() tea.Msg { return StepCompleteMsg{} }
			}
		case dlError:
			if key.Matches(msg, style.Keys.Enter) {
				s.phase = dlConfirm
				s.confirm = ui.NewConfirm("Retry download?", true)
				s.errMsg = ""
				return s, nil
			}
		}

	case ui.ConfirmResult:
		if s.phase == dlConfirm && msg.Confirmed {
			s.phase = dlDownloading
			s.spinner = ui.NewSpinner("Downloading Nextcloud (this may take a few minutes)...")
			s.completedSub = nil
			cmd := exec.RunSudoCommand("download",
				"cd /var/www && wget -q https://download.nextcloud.com/server/releases/latest.tar.bz2 -O latest.tar.bz2")
			return s, tea.Batch(s.spinner.Init(), cmd)
		}

	case exec.CmdResult:
		switch msg.Tag {
		case "download":
			s.completedSub = append(s.completedSub, "Nextcloud archive downloaded")
			if msg.Err != nil {
				s.phase = dlError
				s.errMsg = "Download failed: " + msg.Err.Error()
				return s, nil
			}
			s.phase = dlExtracting
			s.spinner = ui.NewSpinner("Extracting archive...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("extract", "cd /var/www && tar -xf latest.tar.bz2"))

		case "extract":
			s.completedSub = append(s.completedSub, "Archive extracted to /var/www/nextcloud")
			if msg.Err != nil {
				s.phase = dlError
				s.errMsg = "Extraction failed: " + msg.Err.Error()
				return s, nil
			}
			s.phase = dlCreatingDataDir
			s.spinner = ui.NewSpinner("Creating data directory...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("mkdir-data", "mkdir -p /var/www/nextcloud/data"))

		case "mkdir-data":
			s.completedSub = append(s.completedSub, "Data directory created")
			if msg.Err != nil {
				s.phase = dlError
				s.errMsg = "Failed to create data directory: " + msg.Err.Error()
				return s, nil
			}
			s.phase = dlSettingOwnership
			s.spinner = ui.NewSpinner("Setting file ownership...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("chown", "chown -R www-data:www-data /var/www/nextcloud/"))

		case "chown":
			s.completedSub = append(s.completedSub, "Ownership set to www-data")
			if msg.Err != nil {
				s.phase = dlError
				s.errMsg = "Failed to set ownership: " + msg.Err.Error()
				return s, nil
			}
			s.phase = dlSettingPermissions
			s.spinner = ui.NewSpinner("Setting permissions...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("chmod", "chmod 750 /var/www/nextcloud/data"))

		case "chmod":
			s.completedSub = append(s.completedSub, "Permissions configured")
			if msg.Err != nil {
				s.phase = dlError
				s.errMsg = "Failed to set permissions: " + msg.Err.Error()
				return s, nil
			}
			s.phase = dlCleanup
			s.spinner = ui.NewSpinner("Cleaning up...")
			return s, tea.Batch(s.spinner.Init(),
				exec.RunSudoCommand("cleanup", "rm -f /var/www/latest.tar.bz2"))

		case "cleanup":
			s.completedSub = append(s.completedSub, "Archive cleaned up")
			s.phase = dlDone
			return s, nil
		}
	}

	if s.phase >= dlDownloading && s.phase <= dlCleanup {
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	}

	return s, nil
}

func (s *DownloadStep) View(state *State) string {
	var sections []string

	desc := style.DescriptionStyle.Render(
		"Download the latest Nextcloud release and set up the directory structure.")
	sections = append(sections, "", desc, "")

	switch s.phase {
	case dlConfirm:
		code := ui.CodeBlock("wget https://download.nextcloud.com/server/releases/latest.tar.bz2\ntar -xf latest.tar.bz2\nmkdir -p /var/www/nextcloud/data\nchown -R www-data:www-data /var/www/nextcloud/\nchmod 750 /var/www/nextcloud/data")
		sections = append(sections,
			style.SubtitleStyle.Render("Commands to execute:"),
			"", code, "",
			s.confirm.View(),
		)

	case dlDownloading, dlExtracting, dlCreatingDataDir, dlSettingOwnership, dlSettingPermissions, dlCleanup:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		sections = append(sections, "", s.spinner.View())

	case dlDone:
		for _, sub := range s.completedSub {
			sections = append(sections, style.SuccessStyle.Render("  ✓ "+sub))
		}
		w := state.Width - 8
		if w > 70 {
			w = 70
		}
		sections = append(sections, "", ui.SuccessBox("Nextcloud downloaded and extracted!", w))
		sections = append(sections, "", style.KeyHintStyle.Render("Press ENTER to continue →"))

	case dlError:
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
