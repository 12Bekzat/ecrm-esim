package auth

import (
	"testing"
	"time"
)

func TestIssueAndParse(t *testing.T) {
	manager := NewManager("01234567890123456789012345678901", time.Minute)
	token, expires, err := manager.Issue("user-id", "admin", "Test User")
	if err != nil {
		t.Fatal(err)
	}
	if !expires.After(time.Now()) {
		t.Fatal("expiry must be in the future")
	}
	claims, err := manager.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-id" || claims.Role != "admin" || claims.Name != "Test User" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestRejectsTamperedToken(t *testing.T) {
	manager := NewManager("01234567890123456789012345678901", time.Minute)
	token, _, err := manager.Issue("user-id", "admin", "Test User")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = manager.Parse(token + "x"); err == nil {
		t.Fatal("tampered token was accepted")
	}
}

func TestRefreshTokenHash(t *testing.T) {
	plain, hash, err := RandomToken()
	if err != nil {
		t.Fatal(err)
	}
	if plain == hash || HashToken(plain) != hash {
		t.Fatal("refresh token hash mismatch")
	}
}
