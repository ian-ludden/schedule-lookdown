package auth

import "github.com/zalando/go-keyring"

const (
	keyringService  = "schedule-lookdown"
	keyringUser     = "session"
	keyringUsername = "username"
	keyringHistory  = "history"
	keyringPassword = "password"
)

// Username and the session use setWithFallback/getWithFallback so they persist
// even when the OS keyring is unavailable (e.g. WSL2). The password stays
// keyring-only below and is never written to disk.

func StoreUsername(username string) error {
	return setWithFallback(keyringUsername, username)
}

func RetrieveUsername() (string, error) {
	return getWithFallback(keyringUsername)
}

func storeSession(data []byte) error {
	return setWithFallback(keyringUser, string(data))
}

func retrieveSession() ([]byte, error) {
	val, err := getWithFallback(keyringUser)
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func StorePassword(password string) error {
	return keyring.Set(keyringService, keyringPassword, password)
}

func RetrievePassword() (string, error) {
	return keyring.Get(keyringService, keyringPassword)
}

func DeletePassword() error {
	return keyring.Delete(keyringService, keyringPassword)
}

func StoreHistory(data []byte) error {
	return keyring.Set(keyringService, keyringHistory, string(data))
}

func RetrieveHistory() ([]byte, error) {
	val, err := keyring.Get(keyringService, keyringHistory)
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}
