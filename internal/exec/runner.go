package exec

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// CmdResult is sent when a command finishes.
type CmdResult struct {
	Output string
	Err    error
	Tag    string // identifier for which command completed
}

// CmdOutputLine is sent for each line of streaming output.
type CmdOutputLine struct {
	Line string
	Tag  string
}

var (
	sudoPassword   string
	sudoPasswordMu sync.RWMutex
)

// SetSudoPassword stores the sudo password for the session.
func SetSudoPassword(pw string) {
	sudoPasswordMu.Lock()
	defer sudoPasswordMu.Unlock()
	sudoPassword = pw
}

// GetSudoPassword retrieves the stored sudo password.
func GetSudoPassword() string {
	sudoPasswordMu.RLock()
	defer sudoPasswordMu.RUnlock()
	return sudoPassword
}

// HasSudoPassword returns true if a sudo password has been set.
func HasSudoPassword() bool {
	return GetSudoPassword() != ""
}

// RunCommand executes a shell command and returns the result as a tea.Cmd.
func RunCommand(tag, command string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("bash", "-c", command)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		output := stdout.String()
		if stderr.Len() > 0 {
			if output != "" {
				output += "\n"
			}
			output += stderr.String()
		}

		if err != nil {
			return CmdResult{
				Output: output,
				Err:    fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String())),
				Tag:    tag,
			}
		}

		return CmdResult{
			Output: strings.TrimSpace(output),
			Tag:    tag,
		}
	}
}

// RunSudoCommand executes a command with sudo using the stored password.
func RunSudoCommand(tag, command string) tea.Cmd {
	return func() tea.Msg {
		pw := GetSudoPassword()
		var cmd *exec.Cmd
		if pw != "" {
			// Use sudo -S to read password from stdin
			cmd = exec.Command("sudo", "-S", "bash", "-c", command)
			cmd.Stdin = strings.NewReader(pw + "\n")
		} else {
			// Try without password (may work if NOPASSWD or cached)
			cmd = exec.Command("sudo", "bash", "-c", command)
		}

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		output := stdout.String()

		// Filter out the [sudo] password prompt from stderr
		stderrStr := stderr.String()
		filteredLines := []string{}
		for _, line := range strings.Split(stderrStr, "\n") {
			if !strings.Contains(line, "[sudo]") && strings.TrimSpace(line) != "" {
				filteredLines = append(filteredLines, line)
			}
		}
		filteredStderr := strings.Join(filteredLines, "\n")

		if filteredStderr != "" {
			if output != "" {
				output += "\n"
			}
			output += filteredStderr
		}

		if err != nil {
			errMsg := strings.TrimSpace(filteredStderr)
			if errMsg == "" {
				errMsg = err.Error()
			}
			return CmdResult{
				Output: output,
				Err:    fmt.Errorf("%s", errMsg),
				Tag:    tag,
			}
		}

		return CmdResult{
			Output: strings.TrimSpace(output),
			Tag:    tag,
		}
	}
}

// RunSudoMySQL executes a MySQL command as root.
func RunSudoMySQL(tag, sql string) tea.Cmd {
	// Escape single quotes in SQL
	escaped := strings.ReplaceAll(sql, "'", "'\\''")
	command := fmt.Sprintf("mysql -u root -e '%s'", escaped)
	return RunSudoCommand(tag, command)
}

// TestSudo tests if sudo works with the given password.
func TestSudo() tea.Cmd {
	return RunSudoCommand("test-sudo", "echo ok")
}

// CheckSudoNopass checks if sudo works without a password.
func CheckSudoNopass() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("sudo", "-n", "true")
		err := cmd.Run()
		return CmdResult{
			Tag: "check-sudo-nopass",
			Err: err,
		}
	}
}
