// Package auth manages a single shared account and revocable in-memory sessions.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

const Lifetime = 12 * time.Hour

type Service struct {
	username string
	password [32]byte
	mu       sync.Mutex
	sessions map[string]time.Time
}

func New(username, password string) (*Service, error) {
	if username == "" || len(password) < 12 {
		return nil, errors.New("AUTH_USERNAME и AUTH_PASSWORD обязательны; пароль должен содержать минимум 12 байт")
	}
	return &Service{username: username, password: sha256.Sum256([]byte(password)), sessions: make(map[string]time.Time)}, nil
}

func (s *Service) Login(username, password string) (string, error) {
	u, expected := sha256.Sum256([]byte(username)), sha256.Sum256([]byte(s.username))
	p := sha256.Sum256([]byte(password))
	if subtle.ConstantTimeCompare(u[:], expected[:])&subtle.ConstantTimeCompare(p[:], s.password[:]) != 1 {
		return "", errors.New("Неверный логин или пароль")
	}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	value := hex.EncodeToString(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, expiry := range s.sessions {
		if !time.Now().Before(expiry) {
			delete(s.sessions, key)
		}
	}
	if len(s.sessions) >= 1000 {
		return "", errors.New("Слишком много активных сессий")
	}
	s.sessions[value] = time.Now().Add(Lifetime)
	return value, nil
}

func (s *Service) Valid(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, ok := s.sessions[token]
	if !ok || !time.Now().Before(expiry) {
		delete(s.sessions, token)
		return false
	}
	return true
}

func (s *Service) Logout(token string) { s.mu.Lock(); defer s.mu.Unlock(); delete(s.sessions, token) }
func (s *Service) Username() string    { return s.username }
