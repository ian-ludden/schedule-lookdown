package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luddenig/schedule-lookdown/internal/auth"
	"github.com/luddenig/schedule-lookdown/internal/ui"
)

func main() {
	fixture := flag.String("load-fixture", "", "use a local HTML file instead of authenticating and hitting the network")
	flag.Parse()

	var session *auth.Session
	initial := ui.ScreenLogin

	if *fixture != "" {
		initial = ui.ScreenMenu // skip auth entirely
	} else {
		session, _ = auth.LoadSession()
		if session != nil && session.IsValid() {
			initial = ui.ScreenMenu
		}
	}

	p := tea.NewProgram(
		ui.NewApp(session, initial, *fixture),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
