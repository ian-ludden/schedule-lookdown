package ui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ROSE_RED)).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(PEWTER)).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(CREAM)).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C4C7CA"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D9635F")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(PEWTER))
)

// Palette: a muted Rose-Hulman crimson accent with cream/pewter neutrals.
// Truecolor hex; lipgloss downsamples on limited terminals.
const (
	ROSE_RED    = "124"
	ROSE_SILVER = "#B3B2B1"
	ROSE_WHITE  = "15"
	CREAM       = "#E3E4E1"
	PEWTER      = "#8B9096"
)
