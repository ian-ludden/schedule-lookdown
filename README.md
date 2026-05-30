# schedule-lookdown
A Terminal User Interface for the Schedule Lookup service at Rose-Hulman Institute of Technology (because every decades-old perl script deserves a TUI)

## Requirements

- **Go 1.21+**
- **Chrome or Chromium (non-snap)** — used to complete the Microsoft SAML login in a visible browser window
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

### WSL2 note

The login window opens in a real browser, so a display server is required.

- **Windows 11 (22H2+):** WSLg is built in — the window should appear automatically.
- **Windows 10 / older Windows 11:** Install [VcXsrv](https://sourceforge.net/projects/vcxsrv/) or [X410](https://x410.app/), then add this to your shell profile (`~/.bashrc` or `~/.zshrc`):
  ```bash
  export DISPLAY=:0
  ```

If the browser icon appears in the taskbar but no window opens, WSLg may not be running. Check with `wsl --status` in PowerShell and confirm `Default Version: 2` and that the wslg components are installed.
