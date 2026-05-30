package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWritesDefaultWhenMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg != Default() {
		t.Errorf("Load() = %+v, want defaults %+v", cfg, Default())
	}

	// A commented default file should have been written for next time.
	path := filepath.Join(dir, "schedule-lookdown", "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected default file written: %v", err)
	}
	if len(data) == 0 {
		t.Error("default config file is empty")
	}

	// Loading again reads the written file and yields the same defaults.
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if cfg2 != Default() {
		t.Errorf("second Load() = %+v, want %+v", cfg2, Default())
	}
}

func TestLoadReadsValues(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	contents := "default_term = \"latest\"\njump_to_roster_on_single_result = true\n"
	writeConfig(t, dir, contents)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultTerm != DefaultTermLatest {
		t.Errorf("DefaultTerm = %q, want %q", cfg.DefaultTerm, DefaultTermLatest)
	}
	if !cfg.JumpToRosterOnSingleResult {
		t.Error("JumpToRosterOnSingleResult = false, want true")
	}
}

func TestLoadCoercesInvalidDefaultTerm(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	writeConfig(t, dir, "default_term = \"bogus\"\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultTerm != DefaultTermCurrent {
		t.Errorf("invalid default_term coerced to %q, want %q", cfg.DefaultTerm, DefaultTermCurrent)
	}
}

// writeConfig writes contents to the config path under dir.
func writeConfig(t *testing.T, dir, contents string) {
	t.Helper()
	cfgDir := filepath.Join(dir, "schedule-lookdown")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
