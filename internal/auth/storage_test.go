package auth

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestFallbackFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	want := []byte("hello-secret")
	if err := writeFallbackFile("probe", want); err != nil {
		t.Fatalf("writeFallbackFile: %v", err)
	}

	got, err := readFallbackFile("probe")
	if err != nil {
		t.Fatalf("readFallbackFile: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("round-trip = %q, want %q", got, want)
	}

	path := filepath.Join(dir, "schedule-lookdown", "probe")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 600", perm)
	}
}

func TestGetWithFallbackReadsFileWhenKeyringMisses(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	// Use a key the keyring won't have. Whether the keyring is broken (WSL2) or
	// merely empty, getWithFallback must read the value we wrote to the file.
	const key = "fallback-test-key"
	if err := writeFallbackFile(key, []byte("from-file")); err != nil {
		t.Fatalf("writeFallbackFile: %v", err)
	}

	got, err := getWithFallback(key)
	if err != nil {
		t.Fatalf("getWithFallback: %v", err)
	}
	if got != "from-file" {
		t.Errorf("getWithFallback = %q, want %q", got, "from-file")
	}
}

func TestSessionPersistsThroughFallbackFile(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	cookies := []*http.Cookie{{Name: "sid", Value: "abc123", Domain: "example.edu", Path: "/"}}
	// Write the session straight to the fallback file to simulate a launch where
	// the keyring was unavailable when SaveSession ran.
	data, err := json.Marshal(NewSession(cookies))
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	if err := writeFallbackFile(keyringUser, data); err != nil {
		t.Fatalf("writeFallbackFile: %v", err)
	}

	got, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if !got.IsValid() {
		t.Error("loaded session is not valid")
	}
	if len(got.Cookies) != 1 || got.Cookies[0].Name != "sid" || got.Cookies[0].Value != "abc123" {
		t.Errorf("loaded cookies = %+v, want sid=abc123", got.Cookies)
	}
}
