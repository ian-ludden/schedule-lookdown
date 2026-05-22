package auth

import "github.com/zalando/go-keyring"

const (
	keyringService = "schedule-lookdown"
	keyringUser    = "session"
)

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
