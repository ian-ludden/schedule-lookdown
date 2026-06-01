// Package config loads and persists the user's local settings from a TOML file.
//
// The file lives at $XDG_CONFIG_HOME/schedule-lookdown/config.toml (falling back
// to ~/.config/schedule-lookdown/config.toml). It is created with commented
// defaults on first run and is meant to be edited by hand; an in-app settings
// screen may be layered on top of this package later.
//
// Unlike auth secrets (which live under XDG_STATE_HOME, see internal/auth), this
// is user-editable configuration, so it follows the XDG_CONFIG_HOME convention.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Term-default modes for DefaultTerm.
const (
	// DefaultTermCurrent computes the term from today's date (models.CurrentTerm).
	DefaultTermCurrent = "current"
	// DefaultTermLatest uses the furthest-future term offered by reg-sched.pl's
	// term drop-down.
	DefaultTermLatest = "latest"
)

// Config holds the user's persisted default settings.
type Config struct {
	// DefaultTerm controls which term the search form pre-selects:
	// "current" (computed from today) or "latest" (furthest-future available).
	DefaultTerm string `toml:"default_term"`
	// JumpToRosterOnSingleResult, when true, skips the results table and jumps
	// straight to a course's roster view when a course search returns exactly
	// one result.
	JumpToRosterOnSingleResult bool `toml:"jump_to_roster_on_single_result"`
	// DownloadDir is where downloaded roster CSVs are saved. A leading "~" is
	// expanded to the user's home directory. When empty, downloads go to the
	// first usable of $XDG_DOWNLOAD_DIR, ~/Downloads, or the app's data dir.
	DownloadDir string `toml:"download_dir"`
}

// Default returns the configuration used when no file exists or a value is
// missing/invalid. The defaults preserve the app's prior behaviour.
func Default() Config {
	return Config{
		DefaultTerm:                DefaultTermCurrent,
		JumpToRosterOnSingleResult: false,
	}
}

// defaultFileContents is written verbatim on first run so users have a
// self-documenting starting point.
const defaultFileContents = `# schedule-lookdown configuration

# Which term the search form pre-selects.
#   "current" - the term containing today's date
#   "latest"  - the furthest-future term offered by reg-sched.pl
default_term = "current"

# When a course search returns exactly one result, jump straight to that
# course's roster instead of showing a one-row table.
jump_to_roster_on_single_result = false

# Where downloaded roster CSVs are saved. A leading "~" expands to your home
# directory. When unset, downloads go to the first usable of $XDG_DOWNLOAD_DIR,
# ~/Downloads, or the app's own data directory.
# download_dir = "~/Downloads"
`

// configPath returns the path to config.toml, creating its parent directory with
// 0700 perms. It honours $XDG_CONFIG_HOME and defaults to ~/.config.
func configPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "schedule-lookdown")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// Load reads the config file, returning defaults (and writing a commented
// default file) when none exists. Missing or invalid individual values are
// coerced to their defaults so a partial or hand-edited file never breaks
// startup.
func Load() (Config, error) {
	cfg := Default()

	path, err := configPath()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Best-effort: write a self-documenting default file for next time.
		_ = os.WriteFile(path, []byte(defaultFileContents), 0o600)
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}

	if _, err := toml.Decode(string(data), &cfg); err != nil {
		// A malformed file shouldn't block the app; fall back to defaults.
		return Default(), err
	}

	cfg.normalize()
	return cfg, nil
}

// normalize coerces unrecognised values back to defaults.
func (c *Config) normalize() {
	if c.DefaultTerm != DefaultTermCurrent && c.DefaultTerm != DefaultTermLatest {
		c.DefaultTerm = DefaultTermCurrent
	}
}

// ResolvedDownloadDir returns the directory where downloaded rosters should be
// written, creating it (0700) if needed.
//
// An explicit DownloadDir (with a leading "~" expanded) is respected strictly:
// if it can't be used, that's an error. When DownloadDir is unset, sensible
// defaults are tried in order — $XDG_DOWNLOAD_DIR, ~/Downloads, then the app's
// own data dir — and the first usable one wins. This lets a broken default
// (e.g. a dangling or self-referential ~/Downloads symlink, as some WSL setups
// have) degrade gracefully instead of being fatal.
func (c Config) ResolvedDownloadDir() (string, error) {
	if c.DownloadDir != "" {
		dir, err := expandHome(c.DownloadDir)
		if err != nil {
			return "", err
		}
		if err := ensureDir(dir); err != nil {
			return "", err
		}
		return dir, nil
	}

	var candidates []string
	if xdg := os.Getenv("XDG_DOWNLOAD_DIR"); xdg != "" {
		candidates = append(candidates, xdg)
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, "Downloads"))
	}
	// Last resort: the app's own data dir, which we can reliably create.
	candidates = append(candidates, dataDownloadDir())

	var lastErr error
	for _, dir := range candidates {
		if err := ensureDir(dir); err != nil {
			lastErr = err
			continue
		}
		return dir, nil
	}
	return "", lastErr
}

// ensureDir returns nil if dir already resolves to a directory, otherwise it
// attempts to create it (0700). Following the symlink chain with os.Stat means
// an already-usable symlinked directory is accepted as-is rather than being
// passed to MkdirAll (which fails with "file exists" on a looping symlink).
func ensureDir(dir string) error {
	if info, err := os.Stat(dir); err == nil {
		if info.IsDir() {
			return nil
		}
		return fmt.Errorf("%s exists but is not a directory", dir)
	}
	return os.MkdirAll(dir, 0o700)
}

// dataDownloadDir is the fallback download location under the app's XDG data
// dir, used when no usable Downloads directory is available.
func dataDownloadDir() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, ".local", "share")
		}
	}
	return filepath.Join(base, "schedule-lookdown", "downloads")
}

// expandHome replaces a leading "~" or "~/" in path with the user's home dir.
func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~"), "/")), nil
}
