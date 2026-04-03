package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tsch0hnny/rpi-nextcloud/internal/style"
)

// SectionHeader renders a styled section title with optional step number.
func SectionHeader(title string, stepNum, totalSteps int) string {
	progress := ""
	if stepNum > 0 && totalSteps > 0 {
		progress = style.StepNumberStyle.Render(fmt.Sprintf("[Step %d/%d]", stepNum, totalSteps)) + " "
	}
	return progress + style.TitleStyle.Render(title)
}

// CodeBlock renders text in a styled code block.
func CodeBlock(code string) string {
	lines := strings.Split(code, "\n")
	maxLen := 0
	for _, l := range lines {
		if len(l) > maxLen {
			maxLen = len(l)
		}
	}

	padded := make([]string, len(lines))
	for i, l := range lines {
		padded[i] = l + strings.Repeat(" ", maxLen-len(l))
	}

	return style.CodeBlockStyle.Render(strings.Join(padded, "\n"))
}

// InfoBox renders content in a bordered box.
func InfoBox(content string, width int) string {
	return style.BoxStyle.Width(width).Render(content)
}

// ActiveInfoBox renders content in a highlighted bordered box.
func ActiveInfoBox(content string, width int) string {
	return style.ActiveBoxStyle.Width(width).Render(content)
}

// KeyHint renders a key hint like: "j/k: navigate  enter: select  q: quit"
func KeyHint(hints ...string) string {
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, style.KeyHintStyle.Render(h))
	}
	return strings.Join(parts, style.DividerStyle.Render("  │  "))
}

// StatusLine renders a colored status indicator.
func StatusLine(label, value string, color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render("● ") +
		style.TextStyle.Bold(true).Render(label+": ") +
		style.TextStyle.Render(value)
}

// Logo returns the Nextcloud ASCII art logo.
func Logo() string {
	logo := `
    ╔═══════════════════════════════════════╗
    ║                                       ║
    ║     ███╗   ██╗███████╗██╗  ██╗████╗   ║
    ║     ████╗  ██║██╔════╝╚██╗██╔╝╚═██║   ║
    ║     ██╔██╗ ██║█████╗   ╚███╔╝   ██║   ║
    ║     ██║╚██╗██║██╔══╝   ██╔██╗   ██║   ║
    ║     ██║ ╚████║███████╗██╔╝ ██╗████║   ║
    ║     ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝╚═══╝  ║
    ║                                       ║
    ║        T C L O U D                    ║
    ║                                       ║
    ║   Raspberry Pi Nextcloud Installer    ║
    ║                                       ║
    ╚═══════════════════════════════════════╝`

	return lipgloss.NewStyle().Foreground(style.ColorPrimary).Bold(true).Render(logo)
}

// StepIndicator renders a progress indicator for all steps.
func StepIndicator(steps []string, current int) string {
	var parts []string
	for i, name := range steps {
		var marker string
		switch {
		case i < current:
			marker = style.SuccessStyle.Render("✓")
		case i == current:
			marker = style.StepNumberStyle.Render("▸")
		default:
			marker = style.DescriptionStyle.Render("○")
		}

		nameStyle := style.DescriptionStyle
		if i == current {
			nameStyle = style.TextStyle.Bold(true)
		}

		parts = append(parts, fmt.Sprintf(" %s %s", marker, nameStyle.Render(name)))
	}
	return strings.Join(parts, "\n")
}

// Paragraph wraps text to the given width with proper styling.
func Paragraph(text string, width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Foreground(style.ColorText).
		Render(text)
}

// WarningBox renders a warning message.
func WarningBox(msg string, width int) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.ColorWarning).
		Foreground(style.ColorWarning).
		Width(width).
		Padding(0, 1).
		Render("⚠  " + msg)
}

// SuccessBox renders a success message.
func SuccessBox(msg string, width int) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.ColorSuccess).
		Foreground(style.ColorSuccess).
		Width(width).
		Padding(0, 1).
		Render("✓  " + msg)
}
