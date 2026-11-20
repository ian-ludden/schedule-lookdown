package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestProfileDirCreatesWithMode0700(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)

	dir, err := profileDir()
	if err != nil {
		t.Fatalf("profileDir: %v", err)
	}
	want := filepath.Join(state, "schedule-lookdown", "chrome-profile")
	if dir != want {
		t.Errorf("profileDir = %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Errorf("mode = %o, want 700", got)
	}
}

func TestProfileDirTightensExistingMode(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	dir := filepath.Join(state, "schedule-lookdown", "chrome-profile")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := profileDir(); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(dir)
	if got := info.Mode().Perm(); got != 0o700 {
		t.Errorf("mode = %o, want 700", got)
	}
}

func TestHasAndForgetChromeProfile(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	if HasChromeProfile() {
		t.Error("HasChromeProfile true before any profile exists")
	}
	dir, err := profileDir()
	if err != nil {
		t.Fatal(err)
	}
	if HasChromeProfile() {
		t.Error("HasChromeProfile true for an empty profile dir")
	}
	if err := os.WriteFile(filepath.Join(dir, "Local State"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !HasChromeProfile() {
		t.Error("HasChromeProfile false for a populated profile dir")
	}
	if err := ForgetChromeProfile(); err != nil {
		t.Fatalf("ForgetChromeProfile: %v", err)
	}
	if HasChromeProfile() {
		t.Error("HasChromeProfile true after ForgetChromeProfile")
	}
	if err := ForgetChromeProfile(); err != nil {
		t.Errorf("ForgetChromeProfile on missing dir: %v", err)
	}
}

func TestClearStaleSingleton(t *testing.T) {
	t.Run("dead owner is cleared", func(t *testing.T) {
		dir := t.TempDir()
		// PID 0x7fffffff is above any real pid_max, so it is never alive.
		if err := os.Symlink("host-2147483647", filepath.Join(dir, "SingletonLock")); err != nil {
			t.Fatal(err)
		}
		_ = os.Symlink("/tmp/x", filepath.Join(dir, "SingletonSocket"))
		_ = os.Symlink("123", filepath.Join(dir, "SingletonCookie"))
		if err := clearStaleSingleton(dir); err != nil {
			t.Fatalf("clearStaleSingleton: %v", err)
		}
		for _, n := range singletonFiles {
			if _, err := os.Lstat(filepath.Join(dir, n)); !os.IsNotExist(err) {
				t.Errorf("%s still present", n)
			}
		}
	})
	t.Run("live owner is kept", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Symlink(fmt.Sprintf("host-%d", os.Getpid()), filepath.Join(dir, "SingletonLock")); err != nil {
			t.Fatal(err)
		}
		if err := clearStaleSingleton(dir); !errors.Is(err, ErrProfileInUse) {
			t.Errorf("err = %v, want ErrProfileInUse", err)
		}
		if _, err := os.Lstat(filepath.Join(dir, "SingletonLock")); err != nil {
			t.Errorf("SingletonLock removed despite live owner: %v", err)
		}
	})
	t.Run("no lock is a no-op", func(t *testing.T) {
		if err := clearStaleSingleton(t.TempDir()); err != nil {
			t.Errorf("err = %v, want nil", err)
		}
	})
}
