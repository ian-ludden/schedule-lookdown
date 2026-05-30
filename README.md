# schedule-lookdown
A Terminal User Interface for the Schedule Lookup service at Rose-Hulman Institute of Technology (because every decades-old perl script deserves a TUI)

## Requirements

- **Go 1.21+**
- **Chrome or Chromium (non-snap)** — used to drive the Microsoft SAML login (headless on WSL2, in a visible window elsewhere)
  - macOS: install [Google Chrome](https://www.google.com/chrome/)
  - Linux / WSL2 — **amd64:**
    ```bash
    curl -fsSL https://dl.google.com/linux/linux_signing_key.pub | sudo gpg --dearmor -o /etc/apt/keyrings/google-chrome.gpg
    echo 'deb [arch=amd64 signed-by=/etc/apt/keyrings/google-chrome.gpg] https://dl.google.com/linux/chrome/deb/ stable main' | sudo tee /etc/apt/sources.list.d/google-chrome.list
    sudo apt-get update && sudo apt-get install google-chrome-stable
    ```
  - Linux / WSL2 — **arm64** (no official Chrome build for this arch):
    ```bash
    sudo add-apt-repository ppa:xtradeb/apps
    sudo apt install chromium
    ```
  - > **Note:** `sudo apt install chromium-browser` on Ubuntu 22.04+ installs a **snap** package, which does not work with this app due to snap's filesystem confinement. Use the options above instead.
  - Custom path: set `SCHEDULE_LOOKDOWN_BROWSER=/path/to/chrome` in your environment

## Running

```bash
go run ./cmd/schedule-lookdown
```

## Configuration

The app reads optional settings from a local TOML file at
`$XDG_CONFIG_HOME/schedule-lookdown/config.toml` (defaulting to
`~/.config/schedule-lookdown/config.toml`). A commented file with the defaults
below is created automatically on first run; edit it by hand to change behaviour.

```toml
# Which term the search form pre-selects.
#   "current" - the term containing today's date
#   "latest"  - the furthest-future term offered by reg-sched.pl
default_term = "current"

# When a course search returns exactly one result, jump straight to that
# course's roster instead of showing a one-row table.
jump_to_roster_on_single_result = false
```

## Authentication

The app authenticates against Rose-Hulman's Microsoft SAML login. How it does so depends on your platform.

### WSL2 — automated headless login

On WSL2 the login runs in **headless** Chrome and is fully automated — **no display server is required**. On first run the TUI prompts for your Rose-Hulman username and password and drives the Microsoft login for you. Every login it then prompts for the **SMS code** texted to your phone.

Credential persistence:
- Your **username** is remembered between runs.
- Your **password** is kept in the OS keyring when one is available (and is **never written to disk**). If no keyring/Secret Service is present, you'll be prompted for it each session.
- The **SMS code** is required on every login.

### macOS / native Linux — visible browser

On platforms that aren't WSL2, the app opens a **real Chrome window** for you to complete the Microsoft login manually. This requires a display server — on a normal macOS or Linux desktop one is already running, so there's nothing to set up.

#### No display server

If the visible-browser path runs without a display (a headless/remote Linux box, or a WSL environment not auto-detected as WSL2), Chrome can't open and login fails. To provide a display:

- **Windows 11 (22H2+):** WSLg is built in — the window should appear automatically.
- **Windows 10 / older Windows 11:** Install [VcXsrv](https://sourceforge.net/projects/vcxsrv/) or [X410](https://x410.app/), then add this to your shell profile (`~/.bashrc` or `~/.zshrc`):
  ```bash
  export DISPLAY=:0
  ```

If the browser icon appears in the taskbar but no window opens, WSLg may not be running. Check with `wsl --status` in PowerShell and confirm `Default Version: 2` and that the wslg components are installed.
