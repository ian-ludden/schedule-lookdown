package ui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ROSE_RED)).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("26")). // "#A9B1D6"
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7AA2F7")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A9B1D6"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("160")). // "#F7768E"
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565F89"))
)

// lipgloss.Color codes, approximating
// Rose-Hulman official colors
const (
	ROSE_RED    = "1"   // exact match!
	ROSE_SILVER = "249" // should be b3b2b1, is b2b2b2
	ROSE_WHITE  = "15"  // just plain ol' ffffff
)
