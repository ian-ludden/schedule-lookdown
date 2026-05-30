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
	"os"
	"path/filepath"

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
