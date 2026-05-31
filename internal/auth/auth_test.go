package auth

import (
	"testing"
	"time"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("s3cret-pw")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if !CheckPassword(hash, "s3cret-pw") {
		t.Error("expected correct password to verify")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("expected wrong password to fail")
	}
}

func TestJWTRoundTrip(t *testing.T) {
	m := NewTokenManager("test-secret", time.Hour)
	now := time.Now()
	tok, err := m.Issue("user-123", "a@b.com", now)
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}
	uid, email, err := m.Parse(tok)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if uid != "user-123" || email != "a@b.com" {
		t.Fatalf("unexpected claims: %s / %s", uid, email)
	}
}

func TestJWTExpired(t *testing.T) {
	m := NewTokenManager("test-secret", time.Hour)
	past := time.Now().Add(-2 * time.Hour)
	tok, _ := m.Issue("user-123", "a@b.com", past) // expires 1h after `past` → already expired
	if _, _, err := m.Parse(tok); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestJWTTampered(t *testing.T) {
	m := NewTokenManager("test-secret", time.Hour)
	other := NewTokenManager("different-secret", time.Hour)
	tok, _ := m.Issue("user-123", "a@b.com", time.Now())
	if _, _, err := other.Parse(tok); err == nil {
		t.Fatal("expected token signed with another secret to be rejected")
	}
}
