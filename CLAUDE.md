# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build ./...          # build all packages
go run ./cmd/schedule-lookdown  # run the TUI
go vet ./...            # lint
go test ./...           # run all tests
go test ./internal/...  # run a specific package's tests
go mod tidy             # sync go.mod/go.sum after adding/removing imports
```

## Architecture

The app is a BubbleTea TUI that authenticates via Microsoft SAML and queries Rose-Hulman's `reg-sched.pl` Perl CGI.

**Request flow:**
1. `auth/browser.go` opens a visible Chrome window (chromedp, headless=false) for SAML login, captures cookies, stores them in the OS keychain via `auth/storage.go` (go-keyring).
2. `client/http.go` wraps `net/http` with a cookie jar seeded from the stored session and issues GET requests to `reg-sched.pl`.
3. `client/parser.go` parses the HTML response with goquery. The response contains a user-info table (no BORDER attr, skip it) followed by a course list table and a term-grid table (both have `BORDER=1`). `ParseSections` returns typed `[]models.Section`; `ParseTable` returns generic `[][]string` for the UI.
4. `query/` implements the `Query` interface — each type sets the correct GET params and calls the parser. Key params: `termcode` (format `YYYYTT`, e.g. `202630` for Spring 2025-26), `id` (username or course code), `view=tgrid`.
5. `ui/app.go` is the root BubbleTea model managing four screens (`ScreenLogin → ScreenMenu → ScreenSearch → ScreenResults`). Screen transitions happen via custom message types (`authSuccessMsg`, `querySelectedMsg`, `searchSubmittedMsg`, `queryResultMsg`, `backMsg`). Each sub-model (`loginModel`, `menuModel`, `searchModel`, `resultsModel`) implements `tea.Model` and lives as a field on `App`; the root delegates to the active screen via type assertions in `delegateToScreen`.

**Key unknowns still to verify against the live site:**
- Whether a `type=` param is required for person schedule lookups (course.go and availability.go TODOs).
- The `WaitVisible` in `browser.go` needs to be replaced with a URL poll to reliably detect SAML redirect completion.

## Data model

`models.Section` mirrors the reg-sched.pl table columns exactly:
`Course | CRN | Title | Instructor | Credits | Enrolled | Capacity | Schedule | Comments | FinalExam | TermDates`

`Schedule` is a packed string: `MTRF/8:00/O159` (days/start-time/room).
