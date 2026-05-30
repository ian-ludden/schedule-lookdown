package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const registrationURL = "https://prodwebxe-hv.rose-hulman.edu/regweb-cgi/reg-sched.pl"
const emailDomain = "@rose-hulman.edu"
const authTimeout = 5 * time.Minute

const pollingIntervalMS = 500
const remoteTimeoutMS = 500

// browserInstallHint returns arch-appropriate install instructions.
// Google Chrome only ships an official Linux amd64 deb; arm64 and other
// architectures must use non-snap Chromium.
func browserInstallHint() string {
	if runtime.GOARCH == "amd64" {
		return "    curl -fsSL https://dl.google.com/linux/linux_signing_key.pub | sudo gpg --dearmor -o /etc/apt/keyrings/google-chrome.gpg\n" +
			"    echo 'deb [arch=amd64 signed-by=/etc/apt/keyrings/google-chrome.gpg] https://dl.google.com/linux/chrome/deb/ stable main' | sudo tee /etc/apt/sources.list.d/google-chrome.list\n" +
			"    sudo apt-get update && sudo apt-get install google-chrome-stable"
	}
	return "    sudo add-apt-repository ppa:xtradeb/apps\n" +
		"    sudo apt install chromium\n" +
		"    (Google Chrome has no official Linux build for " + runtime.GOARCH + ")"
}

// findBrowser returns the path to a Linux Chrome/Chromium binary.
// The env var SCHEDULE_LOOKDOWN_BROWSER overrides auto-detection.
func findBrowser() (string, error) {
	if p := os.Getenv("SCHEDULE_LOOKDOWN_BROWSER"); p != "" {
		return p, nil
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium-browser", "chromium"} {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		// Snap-packaged browsers wrap the real binary in a shell script with
		// filesystem confinement that blocks chromedp's temp user-data-dir.
		if resolved, err := filepath.EvalSymlinks(path); err == nil && strings.Contains(resolved, "/snap/") {
			return "", fmt.Errorf(
				"%s is a snap package and does not work with chromedp\n"+
					"  install a non-snap browser instead:\n"+
					"%s\n"+
					"  or set SCHEDULE_LOOKDOWN_BROWSER to a non-snap Chrome/Chromium binary",
				name, browserInstallHint(),
			)
		}
		return path, nil
	}
	return "", fmt.Errorf(
		"no Chrome/Chromium browser found in PATH\n"+
			"  install:\n%s\n"+
			"  or set: SCHEDULE_LOOKDOWN_BROWSER=/path/to/chrome",
		browserInstallHint(),
	)
}

// IsWSL2 reports whether the process is running inside WSL2.
func IsWSL2() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}

// findFreePort returns an available TCP port on localhost.
func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

// waitForDebugger polls Chrome's DevTools HTTP endpoint on localhost and
// returns the browser-level WebSocket URL once Chrome is ready.
func waitForDebugger(ctx context.Context, port int) (string, error) {
	endpoint := fmt.Sprintf("http://localhost:%d/json/version", port)
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(endpoint)
		if err == nil {
			var v struct {
				WSURL string `json:"webSocketDebuggerUrl"`
			}
			if json.NewDecoder(resp.Body).Decode(&v) == nil && v.WSURL != "" {
				resp.Body.Close()
				return v.WSURL, nil
			}
			resp.Body.Close()
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(remoteTimeoutMS * time.Millisecond):
		}
	}
	return "", fmt.Errorf("Chrome DevTools not ready after 15s on port %d", port)
}

// rawCookiesToHTTP converts cdproto network cookies to stdlib http.Cookie.
func rawCookiesToHTTP(raw []*network.Cookie) []*http.Cookie {
	out := make([]*http.Cookie, 0, len(raw))
	for _, c := range raw {
		out = append(out, &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		})
	}
	return out
}

// diagSnapshotJS returns a compact one-line summary of the current auth page
// for the trace log: URL, title, $Config.pgid, the submit button's label, and
// the visibility of the fields the state machine keys off of.
const diagSnapshotJS = `(function(){
	function vis(id){var e=document.getElementById(id);return e?(e.offsetParent!==null?'1':'0'):'-';}
	var cfg=window.$Config||{};
	var b=document.getElementById('idSIButton9');
	var bv=b?((b.value||b.innerText||'').trim()):'';
	var otc=document.querySelector('input[name="otc"],#idTxtBx_SAOTCC_OTC,#idTxtBx_SAOTCSC_OTC');
	return 'url='+location.href
		+' title='+JSON.stringify(document.title)
		+' pgid='+(cfg.pgid||'')
		+' btn9='+JSON.stringify(bv)
		+' i0116vis='+vis('i0116')
		+' i0118vis='+vis('i0118')
		+' otcvis='+(otc?(otc.offsetParent!==null?'1':'0'):'-');
})()`

// debugRecorder captures per-iteration auth diagnostics to a timestamped run
// directory. It is enabled by the SCHEDULE_LOOKDOWN_DEBUG env var; when unset,
// every method is a cheap no-op so production runs write nothing.
//
// Each loop iteration appends one line to trace.log. On every *state change* it
// also writes a numbered HTML + PNG snapshot, so the whole page sequence
// (including the MFA pages) is reconstructable from a single run.
type debugRecorder struct {
	enabled   bool
	dir       string
	traceFile *os.File
	seq       int
	lastState string
}

func newDebugRecorder(statusFn func(string)) *debugRecorder {
	if os.Getenv("SCHEDULE_LOOKDOWN_DEBUG") == "" {
		return &debugRecorder{}
	}
	dir := filepath.Join(os.TempDir(), "schedule-lookdown", "run-"+time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &debugRecorder{}
	}
	f, err := os.Create(filepath.Join(dir, "trace.log"))
	if err != nil {
		return &debugRecorder{}
	}
	statusFn("Debug capture enabled: " + dir)
	return &debugRecorder{enabled: true, dir: dir, traceFile: f}
}

// record logs one iteration and, when the state string differs from the
// previous one, writes a numbered HTML + screenshot frame.
func (d *debugRecorder) record(ctx context.Context, state string) {
	if !d.enabled {
		return
	}
	var diag string
	_ = chromedp.Run(ctx, chromedp.Evaluate(diagSnapshotJS, &diag))
	fmt.Fprintf(d.traceFile, "%s state=%q %s\n", time.Now().Format("15:04:05.000"), state, diag)

	if state == d.lastState {
		return
	}
	d.lastState = state
	d.seq++
	base := filepath.Join(d.dir, fmt.Sprintf("%03d-%s", d.seq, sanitizeState(state)))

	var html string
	if chromedp.Run(ctx, chromedp.Evaluate(`document.documentElement.outerHTML`, &html)) == nil && html != "" {
		_ = os.WriteFile(base+".html", []byte(html), 0o644)
	}
	var png []byte
	if chromedp.Run(ctx, chromedp.CaptureScreenshot(&png)) == nil && len(png) > 0 {
		_ = os.WriteFile(base+".png", png, 0o644)
	}
}

func (d *debugRecorder) close() {
	if d.traceFile != nil {
		_ = d.traceFile.Close()
	}
}

// sanitizeState makes a state string safe for use in a filename.
func sanitizeState(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

// AuthenticateHeadless runs a headless Chrome in WSL2, automates the
// Microsoft SAML login with the given credentials, and returns session cookies.
// statusFn, if non-nil, is called with user-facing progress messages.
// mfaCodeFn, if non-nil, is called when an SMS code is needed; it should
// block until the user provides the code and return it.
func AuthenticateHeadless(ctx context.Context, username, password string, statusFn func(string), mfaCodeFn func() string) ([]*http.Cookie, error) {
	if statusFn == nil {
		statusFn = func(string) {}
	}
	if mfaCodeFn == nil {
		mfaCodeFn = func() string { return "" }
	}

	browserPath, err := findBrowser()
	if err != nil {
		return nil, err
	}

	port, err := findFreePort()
	if err != nil {
		return nil, fmt.Errorf("find free port: %w", err)
	}

	// Isolated profile dir prevents a singleton conflict with any running
	// Chrome and keeps this instance away from the user's normal profile.
	tmpDir, err := os.MkdirTemp("", "schedule-lookdown-chrome-")
	if err != nil {
		return nil, fmt.Errorf("create chrome temp dir: %w", err)
	}

	cmd := exec.CommandContext(ctx, browserPath,
		"--headless=new",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--no-first-run",
		"--no-default-browser-check",
		fmt.Sprintf("--remote-debugging-port=%d", port),
		fmt.Sprintf("--user-data-dir=%s", tmpDir),
		// Prevent Chrome from trying to unlock the GNOME keyring (or any
		// other libsecret backend). On Ubuntu/WSL2 with gnome-keyring
		// installed, Chrome 130+ attempts to unlock it on startup and
		// blocks on a GUI dialog until it's dismissed.
		"--password-store=basic",
		// Force a Linux Chrome UA. Without this, WSL2's kernel fingerprint can
		// make Microsoft identify the browser as Windows Edge and attempt Desktop
		// SSO (DSSO/Kerberos), which fails headlessly and breaks the redirect chain.
		"--user-agent=Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36",
	)
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("start headless Chrome: %w", err)
	}
	defer func() {
		cmd.Process.Kill()
		os.RemoveAll(tmpDir)
	}()

	// Both Chrome and chromedp are in WSL2 — plain localhost works.
	wsURL, err := waitForDebugger(ctx, port)
	if err != nil {
		return nil, err
	}

	allocCtx, cancel := chromedp.NewRemoteAllocator(ctx, wsURL)
	defer cancel()

	chromeCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	email := username + emailDomain
	statusFn("Signing in as " + email + "...")

	// Step 1: navigate and enter email address.
	if err := chromedp.Run(chromeCtx,
		chromedp.Navigate(registrationURL),
		chromedp.WaitVisible(`#i0116`),
		chromedp.SendKeys(`#i0116`, email),
		chromedp.Click(`#idSIButton9`),
	); err != nil {
		return nil, fmt.Errorf("email entry: %w", err)
	}

	// Step 2: wait for either a direct password field, a "use password instead"
	// link (common when Windows Hello is the default on this account), or an
	// error that the account doesn't exist.
	const awaitPwdPage = `
		document.querySelector('#i0118') !== null ||
		document.querySelector('#idA_PWD_SwitchToPassword') !== null ||
		(document.querySelector('#usernameError') !== null &&
		 document.querySelector('#usernameError').offsetParent !== null)`
	if err := chromedp.Run(chromeCtx,
		chromedp.Poll(awaitPwdPage, nil, chromedp.WithPollingTimeout(30*time.Second)),
	); err != nil {
		return nil, fmt.Errorf("waiting for password page: %w", err)
	}

	var hasPwdField, hasUsernameError bool
	if err := chromedp.Run(chromeCtx,
		chromedp.Evaluate(`!!document.querySelector('#i0118')`, &hasPwdField),
		chromedp.Evaluate(`!!(document.querySelector('#usernameError')?.offsetParent)`, &hasUsernameError),
	); err != nil {
		return nil, err
	}

	if hasUsernameError {
		var errText string
		_ = chromedp.Run(chromeCtx, chromedp.Evaluate(
			`document.getElementById('usernameError').textContent.trim()`, &errText))
		return nil, fmt.Errorf("account not found: %s — is %s correct?", errText, email)
	}

	if !hasPwdField {
		// "Use your password instead" link appeared — click it.
		if err := chromedp.Run(chromeCtx,
			chromedp.Click(`#idA_PWD_SwitchToPassword`),
			chromedp.WaitVisible(`#i0118`),
		); err != nil {
			return nil, fmt.Errorf("switching to password sign-in: %w", err)
		}
	}

	// Step 3: enter password and wait for the page to leave the password step.
	statusFn("Entering credentials...")
	if err := chromedp.Run(chromeCtx,
		chromedp.SendKeys(`#i0118`, password),
		chromedp.Click(`#idSIButton9`),
	); err != nil {
		return nil, fmt.Errorf("password entry: %w", err)
	}

	// Poll until #i0118 disappears (submit was accepted and page changed) or
	// passwordError appears (wrong password). Doing this before step 4 ensures
	// the HTML dump captures the MFA page, not the password page.
	statusFn("Verifying credentials...")
	if err := chromedp.Run(chromeCtx,
		chromedp.Poll(`(function(){
			if (document.getElementById('i0118') === null) return true;
			var pe = document.getElementById('passwordError');
			if (pe && pe.offsetParent !== null && pe.textContent.trim())
				throw new Error(pe.textContent.trim());
			return false;
		})()`, nil,
			chromedp.WithPollingInterval(pollingIntervalMS*time.Millisecond),
			chromedp.WithPollingTimeout(30*time.Second)),
	); err != nil {
		return nil, fmt.Errorf("signing in: %w", err)
	}

	// Step 4: MFA + completion loop.
	// detectState always returns a non-empty string so we can use Evaluate
	// (single-shot) instead of Poll, enabling status updates between iterations.
	//
	// Return values:
	//   "done"                   — back on registrationURL; auth complete
	//   "kmsi"                   — "Stay signed in?" auto-clicked; keep looping
	//   "pick_method"            — "Use a different verification method" link visible
	//   "sms_tile"               — SMS tile directly clickable
	//   "sms_code"               — OTC input field present; need user's code
	//   "error:<msg>"            — fatal error (wrong password, ConvergedError, etc.)
	//   "waiting:<pgid>:<title>" — unrecognised page; loop continues
	detectState := `(function(){
		try {
			if (document.getElementById('KmsiCheckboxField')) {
				var b = document.getElementById('idSIButton9');
				if (b) b.click();
				return 'kmsi';
			}
			var pe = document.getElementById('passwordError');
			if (pe && pe.offsetParent !== null && pe.textContent.trim())
				return 'error:' + pe.textContent.trim();
			var cfg = window.$Config;
			if (cfg && cfg.pgid === 'ConvergedError' && cfg.iErrorCode)
				return 'error:Microsoft error ' + cfg.iErrorCode + ': ' + (cfg.strServiceExceptionMessage || cfg.strMainMessage || '');
			// OTC entry field: the code page uses idTxtBx_SAOTCC_OTC (Code
			// Challenge); accept the older SAOTCSC id and a generic name="otc"
			// too. Require it visible so a hidden remnant never matches.
			var otc = document.querySelector('input[name="otc"],#idTxtBx_SAOTCC_OTC,#idTxtBx_SAOTCSC_OTC');
			if (otc !== null && otc.offsetParent !== null) return 'sms_code';
			var saw = document.getElementById('signInAnotherWay');
			if (saw !== null && saw.offsetParent !== null) return 'pick_method';
			// Proof picker (idDiv_SAOTCS_Proofs): the SMS tile is a KO-bound div.
			// Require it visible so we stop firing once the page transitions to
			// OTC entry and the proofs list is hidden.
			var smsTile = document.querySelector('[data-value="OneWaySMS"],[data-identity="OneWaySMS"]');
			if (smsTile !== null && smsTile.offsetParent !== null) return 'sms_tile';
			if (document.location.href.startsWith('` + registrationURL + `')) return 'done';
			var aadTile = document.getElementById('aadTile');
			if (aadTile !== null && aadTile.offsetParent !== null) return 'aad_tile';
			var btn9 = document.getElementById('idSIButton9');
			var btn9val = btn9 ? (btn9.value || btn9.innerText || '').trim() : '';
			// Visible password field + a visible, enabled submit button means
			// credential entry is required. Detect this structurally rather than
			// by the button's label: idSIButton9's value is Knockout-bound, so it
			// reads back empty before bindings apply and an exact 'Sign in' match
			// silently misses, leaving the loop spinning on the password page.
			var i0118 = document.getElementById('i0118');
			if (i0118 !== null && i0118.offsetParent !== null &&
					btn9 && !btn9.disabled && btn9.offsetParent !== null) return 'pwd_page';
			// Post-password intermediate step: signal Go to click a visible "Next"
			// button.  offsetParent === null means display:none — skip hidden buttons
			// and let the SPA's autoSubmit binding handle form submission instead.
			if (btn9 && !btn9.disabled && btn9.offsetParent !== null &&
					(i0118 === null || i0118.offsetParent === null) && btn9val === 'Next') {
				return 'next_button';
			}
			return 'waiting:' + (cfg && cfg.pgid ? cfg.pgid : '') + ':' + document.title + ':' + btn9val;
		} catch(e) {
			return 'waiting:::' + e.message;
		}
	})()`

	recorder := newDebugRecorder(statusFn)
	defer recorder.close()

	loopStart := time.Now()
	lastStatus := loopStart
	lastNextClick := time.Time{}
	lastPwdEntry := time.Time{}
	lastSmsClick := time.Time{}
	deadline := loopStart.Add(authTimeout)

	var rawCookies []*network.Cookie
	for time.Now().Before(deadline) {
		var stateStr string
		if err := chromedp.Run(chromeCtx, chromedp.Evaluate(detectState, &stateStr)); err != nil {
			return nil, fmt.Errorf("auth poll: %w", err)
		}
		recorder.record(chromeCtx, stateStr)

		switch {
		case strings.HasPrefix(stateStr, "error:"):
			return nil, fmt.Errorf("%s", strings.TrimPrefix(stateStr, "error:"))

		case strings.HasPrefix(stateStr, "waiting:"):
			// Update status every 5 seconds; dump page HTML once at that point
			// (5 s gives the post-password AJAX time to complete and render the MFA UI).
			if time.Since(lastStatus) >= 5*time.Second {
				// "waiting:pgid:title:btn9value"
				parts := strings.SplitN(stateStr, ":", 4)
				info := ""
				if len(parts) >= 3 && parts[1] != "" {
					info = parts[1]
				}
				if len(parts) >= 3 && parts[2] != "" {
					if info != "" {
						info += " / "
					}
					info += parts[2]
				}
				if len(parts) == 4 && parts[3] != "" {
					info += " [btn: " + parts[3] + "]"
				}
				if info != "" {
					statusFn("Waiting (" + info + ")")
				}
				lastStatus = time.Now()
			}
			time.Sleep(pollingIntervalMS * time.Millisecond)

		case stateStr == "done":
			if err := chromedp.Run(chromeCtx, chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				rawCookies, err = network.GetCookies().Do(ctx)
				return err
			})); err != nil {
				return nil, err
			}
			return rawCookiesToHTTP(rawCookies), nil

		case stateStr == "kmsi":
			// "Stay signed in?" was auto-clicked; next iteration detects the redirect.
			time.Sleep(pollingIntervalMS * time.Millisecond)

		case stateStr == "next_button":
			// Dispatch the full mouse-event sequence via JS rather than chromedp.Click.
			// chromedp.Click blocks on NodeReady, which hangs when the SPA replaces the
			// element during the query.  JS event dispatch returns immediately.
			// Throttle to once every 3 s; wrap in a short timeout as a safety net.
			now := time.Now()
			if now.After(lastNextClick.Add(3 * time.Second)) {
				statusFn("Clicking Next...")
				clickCtx, cancel := context.WithTimeout(chromeCtx, 2*time.Second)
				_ = chromedp.Run(clickCtx, chromedp.Evaluate(`(function(){
					var btn = document.querySelector('#idSIButton9:not([disabled])');
					if (!btn) return;
					['pointerdown','mousedown','mouseup','click'].forEach(function(t){
						btn.dispatchEvent(new MouseEvent(t,{bubbles:true,cancelable:true,view:window}));
					});
				})()`, nil))
				cancel()
				lastNextClick = now
			}
			time.Sleep(pollingIntervalMS * time.Millisecond)

		case stateStr == "aad_tile":
			statusFn("Selecting work/school account...")
			_ = chromedp.Run(chromeCtx, chromedp.Click(`#aadTile`))
			// Wait for the password field to appear and Knockout to bind before
			// the loop re-evaluates; without this pwd_page fires before the
			// framework has initialized its observables.
			waitCtx, cancel := context.WithTimeout(chromeCtx, 10*time.Second)
			_ = chromedp.Run(waitCtx, chromedp.WaitVisible(`#i0118`))
			cancel()
			time.Sleep(500 * time.Millisecond)

		case stateStr == "pwd_page":
			// Cooldown prevents a double-submit if pwd_page re-fires while the
			// page is mid-transition after the first submit.
			if time.Since(lastPwdEntry) < 10*time.Second {
				time.Sleep(pollingIntervalMS * time.Millisecond)
				continue
			}
			statusFn("Entering credentials...")
			// chromedp.SendKeys dispatches key events to the currently focused
			// element, not to the selector.  In step 3 Knockout auto-focuses #i0118
			// on page load; here it doesn't, so we must focus it explicitly first.
			typeCtx, cancel := context.WithTimeout(chromeCtx, 5*time.Second)
			_ = chromedp.Run(typeCtx,
				chromedp.Focus(`#i0118`),
				// Clear any existing value before typing. SendKeys appends, so on a
				// wrong-password retry an un-cleared field would submit a doubled
				// password. Dispatch 'input' so Knockout's observable clears too.
				chromedp.Evaluate(`(function(){
					var f=document.getElementById('i0118');
					if(f){f.value='';f.dispatchEvent(new Event('input',{bubbles:true}));}
				})()`, nil),
				chromedp.SendKeys(`#i0118`, password),
			)
			cancel()
			time.Sleep(200 * time.Millisecond)
			clickCtx, cancel2 := context.WithTimeout(chromeCtx, 3*time.Second)
			_ = chromedp.Run(clickCtx, chromedp.Click(`#idSIButton9`))
			cancel2()
			lastPwdEntry = time.Now()
			time.Sleep(pollingIntervalMS * time.Millisecond)

		case stateStr == "pick_method":
			// "Sign in another way" link: open the method list. Guarded so we
			// don't re-open it every iteration while the list renders.
			if time.Since(lastSmsClick) < 8*time.Second {
				time.Sleep(pollingIntervalMS * time.Millisecond)
				continue
			}
			statusFn("Choosing another verification method...")
			_ = chromedp.Run(chromeCtx, chromedp.Click(`#signInAnotherWay`))
			lastSmsClick = time.Now()
			time.Sleep(pollingIntervalMS * time.Millisecond)

		case stateStr == "sms_tile":
			// The proof tile is a Knockout-bound <div> (click: proof_onClick).
			// Click it ONCE, then wait: clicking every iteration re-triggers the
			// picker and never lets the OTC page settle (and re-sends SMS codes).
			// Dispatch the mouse-event sequence via JS so KO's binding fires
			// without chromedp.Click blocking on NodeReady during the SPA swap.
			if time.Since(lastSmsClick) < 8*time.Second {
				time.Sleep(pollingIntervalMS * time.Millisecond)
				continue
			}
			statusFn("Requesting SMS code...")
			clickCtx, cancel := context.WithTimeout(chromeCtx, 3*time.Second)
			_ = chromedp.Run(clickCtx, chromedp.Evaluate(`(function(){
				var t=document.querySelector('[data-value="OneWaySMS"],[data-identity="OneWaySMS"]');
				if(!t) return;
				['pointerdown','mousedown','mouseup','click'].forEach(function(e){
					t.dispatchEvent(new MouseEvent(e,{bubbles:true,cancelable:true,view:window}));
				});
			})()`, nil))
			cancel()
			lastSmsClick = time.Now()
			time.Sleep(pollingIntervalMS * time.Millisecond)

		case stateStr == "sms_code":
			statusFn("Check your phone for an SMS code")
			code := mfaCodeFn()
			statusFn("Submitting code...")
			// The OTC input is a custom Knockout textbox component and the
			// Continue button is enable-bound, so it stays disabled until KO sees
			// the value. Mirror the proven password pattern: focus a concrete id,
			// clear, type real keystrokes (which fire the events KO listens for),
			// then click the concrete submit id. Comma-list selectors don't
			// resolve reliably through chromedp, which is why the prior attempt
			// left the button disabled and re-prompted.
			typeCtx, cancel := context.WithTimeout(chromeCtx, 5*time.Second)
			_ = chromedp.Run(typeCtx,
				chromedp.Focus(`#idTxtBx_SAOTCC_OTC`),
				chromedp.Evaluate(`(function(){
					var f=document.getElementById('idTxtBx_SAOTCC_OTC');
					if(f){f.value='';f.dispatchEvent(new Event('input',{bubbles:true}));}
				})()`, nil),
				chromedp.SendKeys(`#idTxtBx_SAOTCC_OTC`, code),
			)
			cancel()
			time.Sleep(300 * time.Millisecond)
			clickCtx, cancel2 := context.WithTimeout(chromeCtx, 3*time.Second)
			_ = chromedp.Run(clickCtx, chromedp.Click(`#idSubmit_SAOTCC_Continue`))
			cancel2()
			// Wait for the OTC field to leave (accepted) or an error to surface
			// (wrong/expired code) before looping, so a slow transition can't
			// trigger a duplicate code prompt.
			waitCtx, cancel3 := context.WithTimeout(chromeCtx, 20*time.Second)
			_ = chromedp.Run(waitCtx, chromedp.Poll(`(function(){
				var f=document.getElementById('idTxtBx_SAOTCC_OTC');
				if(!f || f.offsetParent===null) return true;
				var e=document.querySelector('[id^="idDiv_SAOTCC_Error"],[id^="idSpan_SAOTCC_Error"]');
				if(e && e.offsetParent!==null && e.textContent.trim()) return true;
				return false;
			})()`, nil, chromedp.WithPollingTimeout(20*time.Second)))
			cancel3()
		}
	}
	return nil, fmt.Errorf("authentication timed out after %v", authTimeout)
}

// Authenticate opens a visible browser for the user to complete Microsoft SAML
// auth and returns session cookies. Used on non-WSL2 platforms where a display
// server is available. On WSL2, call AuthenticateHeadless instead.
func Authenticate(ctx context.Context) ([]*http.Cookie, error) {
	browserPath, err := findBrowser()
	if err != nil {
		return nil, err
	}

	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return nil, fmt.Errorf(
			"no display server found ($DISPLAY and $WAYLAND_DISPLAY are unset)\n" +
				"  on WSL2 (Windows 11): WSLg should work automatically — check that wslg is not disabled\n" +
				"  on WSL2 (Windows 10): install VcXsrv or X410 and add 'export DISPLAY=:0' to your shell profile",
		)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(browserPath),
		chromedp.Flag("headless", false),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("ozone-platform", "x11"),
		chromedp.Flag("start-maximized", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	chromeCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var rawCookies []*network.Cookie
	err = chromedp.Run(chromeCtx,
		chromedp.Navigate(registrationURL),
		chromedp.Poll(
			`document.location.href.startsWith("`+registrationURL+`")`,
			nil,
			chromedp.WithPollingInterval(pollingIntervalMS*time.Millisecond),
			chromedp.WithPollingTimeout(authTimeout),
		),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			rawCookies, err = network.GetCookies().Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, err
	}
	return rawCookiesToHTTP(rawCookies), nil
}
