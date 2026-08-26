package authtoken

import (
	"errors"
	"testing"
)

type mockKeyring struct {
	tokens map[string]string
	errs   map[string]error
}

func (m *mockKeyring) Get(service, key string) (string, error) {
	if err, ok := m.errs[service]; ok {
		return "", err
	}
	if tok, ok := m.tokens[service]; ok {
		return tok, nil
	}
	return "", ErrNotFound
}

func TestTokenFromKeyring(t *testing.T) {
	t.Run("token found", func(t *testing.T) {
		kr := &mockKeyring{
			tokens: map[string]string{"gh:github.com": "gho_secret123"},
		}
		tok, err := TokenFromKeyring("github.com", kr)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if tok != "gho_secret123" {
			t.Fatalf("expected token 'gho_secret123', got '%s'", tok)
		}
	})

	t.Run("token missing returns ErrNotFound", func(t *testing.T) {
		kr := &mockKeyring{
			tokens: map[string]string{},
		}
		tok, err := TokenFromKeyring("github.com", kr)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		if tok != "" {
			t.Fatalf("expected empty token, got '%s'", tok)
		}
	})

	t.Run("operational error returns KeyringOpError", func(t *testing.T) {
		kr := &mockKeyring{
			errs: map[string]error{"gh:github.com": errors.New("dbus: connection refused")},
		}
		_, err := TokenFromKeyring("github.com", kr)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errors.Is(err, ErrNotFound) {
			t.Fatal("operational error should not be ErrNotFound")
		}
		var opErr *KeyringOpError
		if !errors.As(err, &opErr) {
			t.Fatalf("expected *KeyringOpError, got %T: %v", err, err)
		}
		if opErr.Host != "github.com" {
			t.Errorf("expected host 'github.com', got '%s'", opErr.Host)
		}
		if !IsKeyringOpError(err) {
			t.Errorf("IsKeyringOpError should return true")
		}
	})
}
