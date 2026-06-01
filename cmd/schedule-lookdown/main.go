package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/auth"
	"github.com/luddenig/schedule-lookdown/internal/config"
	"github.com/luddenig/schedule-lookdown/internal/ui"
)

func main() {
	loadSamples := flag.Bool("load-samples", false, "load all sample response files from ./sample-responses/ instead of authenticating")
	debugLogArg := flag.String("debug", "", "write query debug info to this log file (e.g. debug.log)")
	flag.Parse()

	// Load user config; defaults are used (and a default file written) on first run.
	cfg, _ := config.Load()

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

	var logger = createLogger(debugLogArg)

	p := tea.NewProgram(
		ui.NewApp(session, cfg, initial, fixtures, logger),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// createLogger constructs a debugging log file and
// defers closing it to avoid resource leaks.
func createLogger(logFilename *string) *log.Logger {
	if *logFilename == "" {
		return nil
	}
	var logger *log.Logger

	f, err := os.OpenFile(*logFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot open debug log: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	logger = log.New(f, "", log.Ldate|log.Ltime|log.Lmicroseconds)
	logger.Println("=== schedule-lookdown debug log started ===")

	return logger
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

		// Determine the query type from the filename.
		var queryType string
		switch {
		case strings.Contains(name, "course"):
			queryType = "course_search"
		case strings.Contains(name, "student") || strings.Contains(name, "username"):
			queryType = "schedule_lookup"
		case strings.Contains(name, "roster") && !strings.Contains(name, "combined"):
			queryType = "roster_view"
		case strings.Contains(name, "instructor"):
			queryType = "instructor_lookup"
		case strings.Contains(name, "lastname") || strings.Contains(name, "person"):
			queryType = "person_search"
		}
		if queryType == "" {
			continue
		}

		// Files named "sample-<type>-<termcode>.html" register under a
		// compound key so term navigation can find the right fixture.
		// Files without a term code register as the generic fallback.
		parts := strings.Split(name, "-")
		last := parts[len(parts)-1]
		_, numErr := strconv.Atoi(last)
		if len(last) == 6 && numErr == nil {
			m[queryType+":"+last] = path
		} else {
			m[queryType] = path
		}
	}
	return m
}
