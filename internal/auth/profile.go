package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Persistent Chrome profile for headless auth. Keeping the profile between runs
// preserves Microsoft's persistent sign-in cookie so a later launch can restore
// the session without a password or SMS code. The directory holds an
// authenticated session in effectively plaintext (--password-store=basic), so it
// is created 0700 under the XDG state dir and must be treated as a credential.

const chromeProfileName = "chrome-profile"

// profileDir returns the persistent Chrome profile directory, creating it with
// 0700 perms. It lives next to the fallback secret files and honours
// $XDG_STATE_HOME.
func profileDir() (string, error) {
	base, err := fallbackDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, chromeProfileName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	// MkdirAll leaves an existing directory's mode alone; tighten it.
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// HasChromeProfile reports whether a persistent profile with content from a
// previous run exists. It never creates the directory, so callers can use it to
// skip a pointless silent attempt on a first run.
func HasChromeProfile() bool {
	base, err := fallbackDir()
	if err != nil {
		return false
	}
	entries, err := os.ReadDir(filepath.Join(base, chromeProfileName))
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "Singleton") {
			return true
		}
	}
	return false
}

// ForgetChromeProfile deletes the persistent profile, discarding the remembered
// Microsoft sign-in. A missing profile is not an error.
func ForgetChromeProfile() error {
	base, err := fallbackDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(base, chromeProfileName))
}

// singletonFiles are the lock artifacts Chrome leaves in a profile directory.
var singletonFiles = []string{"SingletonLock", "SingletonSocket", "SingletonCookie"}

// ErrProfileInUse is returned when another live Chrome owns the profile.
var ErrProfileInUse = errors.New("the saved Chrome profile is in use by another running instance")

// clearStaleSingleton removes Chrome's Singleton* files from dir when the
// process that owns them is dead (a crashed run). SingletonLock is a symlink
// whose target is "<hostname>-<pid>". When that PID is still alive the files are
// left alone and ErrProfileInUse is returned, since deleting them would let two
// Chrome instances share one profile.
func clearStaleSingleton(dir string) error {
	target, err := os.Readlink(filepath.Join(dir, "SingletonLock"))
	if err != nil {
		return nil // no lock (or not a symlink): nothing to clear
	}
	if i := strings.LastIndex(target, "-"); i >= 0 {
		if pid, perr := strconv.Atoi(target[i+1:]); perr == nil && pid > 0 && pidAlive(pid) {
			return ErrProfileInUse
		}
	}
	for _, name := range singletonFiles {
		_ = os.Remove(filepath.Join(dir, name))
	}
	return nil
}

// pidAlive reports whether a process with the given PID exists.
func pidAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
