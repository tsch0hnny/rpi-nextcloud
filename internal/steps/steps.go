package steps

import (
	tea "github.com/charmbracelet/bubbletea"
)

// State holds shared state across all steps.
type State struct {
	// System
	IPAddress string
	Hostname  string

	// Database config
	DBName     string
	DBUser     string
	DBPassword string

	// Nextcloud admin
	AdminUser     string
	AdminPassword string

	// Apache config
	ApacheMode string // "directory" or "domain"
	DomainName string

	// Optional settings
	PHPUploadLimit string
	DataDirectory  string
	SSLEnabled     bool
	ForceHTTPS     bool

	// Port forwarding
	ExternalDomain string

	// Tracking
	CompletedSteps map[string]bool

	// Terminal
	Width  int
	Height int
}

// NewState creates a new state with sensible defaults.
func NewState() *State {
	return &State{
		DBName:         "nextclouddb",
		DBUser:         "nextclouduser",
		AdminUser:      "admin",
		ApacheMode:     "directory",
		PHPUploadLimit: "1024M",
		DataDirectory:  "/var/nextcloud/data",
		SSLEnabled:     true,
		ForceHTTPS:     true,
		CompletedSteps: make(map[string]bool),
	}
}

// Step is the interface that all wizard steps implement.
type Step interface {
	// Init initializes the step.
	Init(state *State) tea.Cmd
	// Update handles messages.
	Update(msg tea.Msg, state *State) (Step, tea.Cmd)
	// View renders the step.
	View(state *State) string
	// Title returns the step name for the progress indicator.
	Title() string
	// ID returns a unique identifier.
	ID() string
	// IsOptional returns true if the step can be skipped.
	IsOptional() bool
	// IsComplete returns true when the step is done and can advance.
	IsComplete() bool
}

// StepCompleteMsg signals that the current step is done and should advance.
type StepCompleteMsg struct{}

// StepSkipMsg signals that the user wants to skip this optional step.
type StepSkipMsg struct{}

// QuitConfirmMsg is sent when the user confirms they want to quit.
type QuitConfirmMsg struct{}
