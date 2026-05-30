package auth

import "github.com/zalando/go-keyring"

const (
	keyringService  = "schedule-lookdown"
	keyringUser     = "session"
	keyringUsername = "username"
	keyringHistory  = "history"
	keyringPassword = "password"
)

func StoreUsername(username string) error {
	return keyring.Set(keyringService, keyringUsername, username)
}

func RetrieveUsername() (string, error) {
	return keyring.Get(keyringService, keyringUsername)
}

func storeSession(data []byte) error {
	return keyring.Set(keyringService, keyringUser, string(data))
}

func retrieveSession() ([]byte, error) {
	val, err := keyring.Get(keyringService, keyringUser)
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
