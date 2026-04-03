package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tsch0hnny/rpi-nextcloud/internal/app"
	"github.com/tsch0hnny/rpi-nextcloud/internal/steps"
	"github.com/tsch0hnny/rpi-nextcloud/internal/ui"
)

func main() {
	imageMode := flag.String("images", "auto",
		"Image rendering mode: sixel, unicode, none, auto (default: auto)")
	flag.Parse()

	ui.SetImageMode(ui.ParseImageMode(*imageMode))

	stepList := []steps.Step{
		steps.NewWelcomeStep(),
		steps.NewApachePHPStep(),
		steps.NewMySQLStep(),
		steps.NewDownloadStep(),
		steps.NewApacheConfStep(),
		steps.NewWebSetupStep(),
		steps.NewMoveDataStep(),
		steps.NewUploadSizeStep(),
		steps.NewSSLStep(),
		steps.NewPortForwardStep(),
		steps.NewCompleteStep(),
	}

	model := app.New(stepList)

	p := tea.NewProgram(model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
