package auth

import (
	"encoding/json"
	"net/http"
	"time"
)

const sessionDuration = 8 * time.Hour

type Session struct {
	Cookies   []*http.Cookie `json:"cookies"`
	ExpiresAt time.Time      `json:"expires_at"`
}

func (s *Session) IsValid() bool {
	return s != nil && len(s.Cookies) > 0 && time.Now().Before(s.ExpiresAt)
}

func NewSession(cookies []*http.Cookie) *Session {
	return &Session{
		Cookies:   cookies,
		ExpiresAt: time.Now().Add(sessionDuration),
	}
}

func LoadSession() (*Session, error) {
	data, err := retrieveSession()
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func SaveSession(s *Session) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return storeSession(data)
}
