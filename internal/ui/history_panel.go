package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	panelInnerWidth    = 28
	panelTotalWidth    = panelInnerWidth + 2 // NormalBorder adds 1 char each side
	panelTimestampWidth = 6
	// Row layout: "> "/"  " (2) + label + " " + timestamp = panelInnerWidth
	panelLabelWidth = panelInnerWidth - 2 - 1 - panelTimestampWidth // 19
)

var (
	panelBorderStyle = lipgloss.NewStyle().
				Width(panelInnerWidth).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color(ROSE_SILVER))

	panelFocusedBorderStyle = lipgloss.NewStyle().
				Width(panelInnerWidth).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color(ROSE_RED))

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ROSE_RED))

	panelNormalItemStyle = normalStyle
	// lipgloss.NewStyle().Foreground(lipgloss.Color("#A9B1D6"))

	panelSelectedItemStyle = selectedStyle
	// lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true)
)

type historyPanelModel struct {
	entries []HistoryEntry
	cursor  int
	focused bool
	height  int
}

func newHistoryPanelModel() historyPanelModel {
	return historyPanelModel{}
}

func (m historyPanelModel) Init() tea.Cmd { return nil }

func (m historyPanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.entries) > 0 && m.cursor < len(m.entries) {
				e := m.entries[m.cursor]
				return m, func() tea.Msg {
					return historyEntrySelectedMsg{
						queryType: e.QueryType,
						params:    e.Params,
						result:    e.Result,
					}
				}
			}
		case "x":
			return m, func() tea.Msg { return historyClearedMsg{} }
		case "esc":
			m.focused = false
			return m, nil
		}
	}
	return m, nil
}

func (m historyPanelModel) View() string {
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("Recent Queries"))
	b.WriteByte('\n')

	cursor := m.cursor
	if cursor >= len(m.entries) && len(m.entries) > 0 {
		cursor = len(m.entries) - 1
	}

	contentLines := 1 // title
	if len(m.entries) == 0 {
		b.WriteString(panelNormalItemStyle.Render("  (no history)"))
		b.WriteByte('\n')
		contentLines++
	} else {
		now := time.Now()
		for i, e := range m.entries {
			label := historyEntryLabel(e)
			ts := formatTimestamp(e.FetchedAt, now)
			row := fmt.Sprintf("%-*s %6s", panelLabelWidth, label, ts)
			if i == cursor && m.focused {
				b.WriteString(panelSelectedItemStyle.Render("> " + row))
			} else {
				b.WriteString(panelNormalItemStyle.Render("  " + row))
			}
			b.WriteByte('\n')
			contentLines++
		}
	}

	// Fill remaining vertical space so the panel reaches full height.
	if m.height > 0 {
		helpLines := 1
		if m.focused {
			helpLines = 2
		}
		remaining := m.height - 2 - contentLines - helpLines // -2 for border
		for range remaining {
			b.WriteByte('\n')
		}
	}

	if m.focused {
		b.WriteString(helpStyle.Render("↑↓ nav • enter select"))
		b.WriteByte('\n')
		b.WriteString(helpStyle.Render("x clear • esc unfocus"))
	} else {
		b.WriteString(helpStyle.Render("ctrl+h: history"))
	}

	style := panelBorderStyle
	if m.focused {
		style = panelFocusedBorderStyle
	}
	if m.height > 0 {
		style = style.Height(m.height - 2)
	}
	return style.Render(b.String())
}

var _ tea.Model = historyPanelModel{}
