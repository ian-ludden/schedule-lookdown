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

# Debug the headless SAML auth flow directly (bypasses the TUI):
SCHEDULE_LOOKDOWN_DEBUG=1 go run ./cmd/authprobe
```

## Architecture

The app is a BubbleTea TUI that authenticates via Microsoft SAML and queries Rose-Hulman's `reg-sched.pl` Perl CGI.

**Request flow:**
1. `auth/browser.go` drives Microsoft SAML login via chromedp and captures the session cookies, stored in the OS keychain via `auth/storage.go` (go-keyring). Two paths: `Authenticate` opens a **visible** Chrome window for the user to log in manually (non-WSL2, needs a display); `AuthenticateHeadless` runs **headless** Chrome and **automates** the login with the user's stored username/password, prompting the TUI only for the SMS MFA code. WSL2 is detected via `/proc/version` and routes to the headless path.
2. `client/http.go` wraps `net/http` with a cookie jar seeded from the stored session and issues GET requests to `reg-sched.pl`.
3. `client/parser.go` parses the HTML response with goquery. The response contains a user-info table (no BORDER attr, skip it) followed by a course list table and a term-grid table (both have `BORDER=1`). `ParseSections` returns typed `[]models.Section`; `ParseTable` returns generic `[][]string` for the UI.
4. `query/` implements the `Query` interface — each type sets the correct GET params and calls the parser. Key params: `termcode` (format `YYYYTT`, e.g. `202630` for Spring 2025-26), `id` (username or course code), `view=tgrid`.
5. `ui/app.go` is the root BubbleTea model managing four screens (`ScreenLogin → ScreenMenu → ScreenSearch → ScreenResults`). Screen transitions happen via custom message types (`authSuccessMsg`, `querySelectedMsg`, `searchSubmittedMsg`, `queryResultMsg`, `backMsg`). Each sub-model (`loginModel`, `menuModel`, `searchModel`, `resultsModel`) implements `tea.Model` and lives as a field on `App`; the root delegates to the active screen via type assertions in `delegateToScreen`.

**User config (`internal/config`):** `config.Load()` reads `$XDG_CONFIG_HOME/schedule-lookdown/config.toml` (default `~/.config/...`), writing a commented default file on first run. Loaded in `main.go` and passed to `ui.NewApp`. Two settings today: `default_term` (`"current"` = computed from today, or `"latest"` = furthest-future term from reg-sched.pl) and `jump_to_roster_on_single_result` (course search with one hit jumps straight to its roster). It's deliberately structured so an in-app settings screen can be added later. Defaults preserve prior behaviour; invalid values coerce to defaults. For `"latest"`, `App` fetches the term drop-down once after auth (`fetchDefaultTermCmd` → `termsLoadedMsg`) and falls back to the computed current term if the fetch hasn't landed.

**Headless SAML automation (`AuthenticateHeadless`):**
The Microsoft login is a Knockout SPA, so the flow is a polling state machine: each loop runs `detectState` (a JS snippet that classifies the current page) and acts on the result — `aad_tile` (account picker) → `pwd_page` → `sms_tile` (MFA method picker) → `sms_code` (OTC entry) → `kmsi` ("Stay signed in?") → `done` (back on `registrationURL`, capture cookies). Hard-won conventions when editing this:
- **Use single concrete id selectors** with `chromedp.SendKeys`/`Click` (e.g. `#idTxtBx_SAOTCC_OTC`, `#idSubmit_SAOTCC_Continue`). Comma-list selectors silently fail through chromedp.
- **Detect pages structurally, not by button label** — `idSIButton9`'s value is Knockout-bound and reads empty before bindings apply.
- **Click KO-bound tiles via `chromedp.Evaluate` + `querySelector`** dispatching a `pointerdown/mousedown/mouseup/click` sequence; gate repeated clicks with a cooldown so the loop doesn't spam (and re-send SMS codes).
- **Microsoft changes this UI often.** Set `SCHEDULE_LOOKDOWN_DEBUG=1` to capture per-iteration `trace.log` + numbered HTML/PNG frames under `$TMPDIR/schedule-lookdown/run-*` (no-op when unset), and use `cmd/authprobe` to drive auth without the TUI. These frames contain an authenticated session — don't commit them.

**Key unknowns still to verify against the live site:**
- Whether a `type=` param is required for person schedule lookups (course.go and availability.go TODOs).
- Whether a bare GET (no params) to `reg-sched.pl` returns the form page containing the `<select name="termcode">` drop-down that `query.FetchAvailableTerms` / `client.ParseTermOptions` parse (used for `default_term = "latest"`). See the TODO in `query/terms.go`.

## Data model

`models.Section` mirrors the reg-sched.pl table columns exactly:
`Course | CRN | Title | Instructor | Credits | Enrolled | Capacity | Schedule | Comments | FinalExam | TermDates`

`Schedule` is a packed string: `MTRF/8:00/O159` (days/start-time/room).
