package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/auth"
	"github.com/luddenig/schedule-lookdown/internal/ui"
)

func main() {
	loadSamples := flag.Bool("load-samples", false, "load all sample response files from ./sample-responses/ instead of authenticating")
	flag.Parse()

	var session *auth.Session
	initial := ui.ScreenLogin

	var fixtures map[string]string
	if *loadSamples {
		fixtures = discoverSamples("sample-responses")
		initial = ui.ScreenMenu
	} else {
		session, _ = auth.LoadSession()
		if session != nil && session.IsValid() {
			initial = ui.ScreenMenu
		}
	}

	p := tea.NewProgram(
		ui.NewApp(session, initial, fixtures),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// discoverSamples scans dir for .html files and maps each to a query type
// based on filename patterns.
func discoverSamples(dir string) map[string]string {
	m := make(map[string]string)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return m
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		name := strings.ToLower(strings.TrimSuffix(e.Name(), ".html"))
		path := filepath.Join(dir, e.Name())
		switch {
		case strings.Contains(name, "course"):
			m["course_search"] = path
		case strings.Contains(name, "student") || strings.Contains(name, "username"):
			m["schedule_lookup"] = path
		case strings.Contains(name, "roster") && !strings.Contains(name, "combined"):
			m["roster_view"] = path
		case strings.Contains(name, "instructor"):
			m["instructor_lookup"] = path
		}
	}
	return m
}
