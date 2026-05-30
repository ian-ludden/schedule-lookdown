// Command authprobe drives the headless Microsoft SAML auth flow directly,
// printing timestamped progress to stdout and reading the SMS code from stdin.
//
// It bypasses the BubbleTea UI so the full auth sequence is visible while
// debugging. Pair it with SCHEDULE_LOOKDOWN_DEBUG=1 to capture per-iteration
// trace.log + numbered HTML/PNG frames under $TMPDIR/schedule-lookdown/run-*.
//
//	SCHEDULE_LOOKDOWN_DEBUG=1 go run ./cmd/authprobe
//
// Credentials are resolved in order: env (SCHEDULE_LOOKDOWN_USER /
// SCHEDULE_LOOKDOWN_PASS), then the OS keyring (as stored by normal app use),
// then an interactive prompt. The password prompt echoes — prefer the keyring
// or env for anything you don't want on screen.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/luddenig/schedule-lookdown/internal/auth"
)

func main() {
	in := bufio.NewReader(os.Stdin)

	username := os.Getenv("SCHEDULE_LOOKDOWN_USER")
	if username == "" {
		if u, err := auth.RetrieveUsername(); err == nil {
			username = u
		}
	}
	if username == "" {
		username = prompt(in, "Rose-Hulman username: ")
	}

	password := os.Getenv("SCHEDULE_LOOKDOWN_PASS")
	if password == "" {
		if p, err := auth.RetrievePassword(); err == nil {
			password = p
		}
	}
	if password == "" {
		fmt.Fprintln(os.Stderr, "warning: password not in env or keyring; the prompt below will echo")
		password = prompt(in, "Microsoft password: ")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logf := func(s string) { fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), s) }
	logf("Authenticating as " + username + " (headless)")

	cookies, err := auth.AuthenticateHeadless(ctx, username, password, logf, func() string {
		return prompt(in, "Enter SMS code: ")
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "auth failed:", err)
		os.Exit(1)
	}

	fmt.Printf("\nSUCCESS: captured %d cookies\n", len(cookies))
	for _, c := range cookies {
		fmt.Printf("  %-32s domain=%s\n", c.Name, c.Domain)
	}
}

func prompt(in *bufio.Reader, label string) string {
	fmt.Print(label)
	line, _ := in.ReadString('\n')
	return strings.TrimSpace(line)
}
