package auth

import (
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

// File-based fallback for secret storage. The OS keyring is the primary store,
// but on some environments it is unavailable — notably WSL2, where the
// gnome-keyring default collection is locked and there's no GUI to unlock it, so
// every keyring operation fails with "failed to unlock correct collection". When
// that happens we fall back to a 0600 file under the user's XDG state directory
// so the session and username still persist across launches.
//
// Only non-password secrets use this fallback; the password is deliberately kept
// keyring-only and never written to disk (see storage.go).

// fallbackDir returns the directory for fallback secret files, creating it with
// 0700 perms. It honours $XDG_STATE_HOME and defaults to ~/.local/state.
func fallbackDir() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	dir := filepath.Join(base, "schedule-lookdown")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// writeFallbackFile writes data to <fallbackDir>/<key> with 0600 perms, using a
// temp-file-and-rename so a crash mid-write can't leave a truncated secret.
func writeFallbackFile(key string, data []byte) error {
	dir, err := fallbackDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, key)
	tmp, err := os.CreateTemp(dir, key+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// readFallbackFile reads <fallbackDir>/<key>.
func readFallbackFile(key string) ([]byte, error) {
	dir, err := fallbackDir()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(dir, key))
}

// setWithFallback stores value in the keyring, falling back to a file when the
// keyring is unavailable.
func setWithFallback(key, value string) error {
	if err := keyring.Set(keyringService, key, value); err == nil {
		return nil
	}
	return writeFallbackFile(key, []byte(value))
}

// getWithFallback reads value from the keyring, falling back to a file when the
// keyring is unavailable or has no entry. If neither source has the value, the
// original keyring error is returned so callers' "no value → prompt" logic still
// fires.
func getWithFallback(key string) (string, error) {
	val, err := keyring.Get(keyringService, key)
	if err == nil {
		return val, nil
	}
	if data, ferr := readFallbackFile(key); ferr == nil {
		return string(data), nil
	}
	return "", err
}
