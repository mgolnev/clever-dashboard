package auth

import (
	"testing"
	"time"
)

func TestSessions(t *testing.T) {
	if _, err := New("admin", "short"); err == nil {
		t.Fatal("weak configuration accepted")
	}
	s, err := New("admin", "test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{"wrong", "test-password-123"}, {"admin", "wrong"}} {
		if _, err := s.Login(pair[0], pair[1]); err == nil {
			t.Fatal("invalid credentials accepted")
		}
	}
	a, err := s.Login("admin", "test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Login("admin", "test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if a == b || !s.Valid(a) || s.Valid("forged") || s.Valid("") {
		t.Fatal("invalid session validation")
	}
	s.Logout(a)
	if s.Valid(a) || !s.Valid(b) {
		t.Fatal("logout must revoke only current session")
	}
	s.sessions[b] = time.Now().Add(-time.Second)
	if s.Valid(b) {
		t.Fatal("expired session accepted")
	}
}
