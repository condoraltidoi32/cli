package config

import (
	"errors"
	"testing"

	"github.com/condoraltidoi32/cli/pkg/authtoken"
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
	return "", authtoken.ErrNotFound
}

func TestAuthTokenForHost(t *testing.T) {
	t.Run("prefers GH_TOKEN env var over keyring and hosts", func(t *testing.T) {
		cfg := &Config{
			Env: func(k string) string {
				if k == "GH_TOKEN" {
					return "env_token_gh"
				}
				return ""
			},
			Keyring: &mockKeyring{
				tokens: map[string]string{"gh:github.com": "keyring_token"},
			},
			Hosts: map[string]HostConfig{
				"github.com": {OAuthToken: "host_token"},
			},
		}

		tok, src, err := cfg.AuthTokenForHost("github.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok != "env_token_gh" || src != "GH_TOKEN" {
			t.Fatalf("expected env_token_gh from GH_TOKEN, got %s from %s", tok, src)
		}
	})

	t.Run("resolves from keyring when env is empty", func(t *testing.T) {
		cfg := &Config{
			Env: func(k string) string { return "" },
			Keyring: &mockKeyring{
				tokens: map[string]string{"gh:github.com": "keyring_token_123"},
			},
			Hosts: map[string]HostConfig{
				"github.com": {OAuthToken: "host_token"},
			},
		}

		tok, src, err := cfg.AuthTokenForHost("github.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok != "keyring_token_123" || src != "keyring" {
			t.Fatalf("expected keyring_token_123 from keyring, got %s from %s", tok, src)
		}
	})

	t.Run("propagates operational keyring error and halts fallback chain", func(t *testing.T) {
		cfg := &Config{
			Env: func(k string) string { return "" },
			Keyring: &mockKeyring{
				errs: map[string]error{"gh:github.com": errors.New("dbus: connection closed by remote peer")},
			},
			Hosts: map[string]HostConfig{
				"github.com": {OAuthToken: "host_fallback_token_should_not_be_used"},
			},
		}

		tok, src, err := cfg.AuthTokenForHost("github.com")
		if err == nil {
			t.Fatalf("expected operational error, got nil")
		}
		if tok != "" {
			t.Fatalf("expected empty token on operational error, got %s", tok)
		}
		if src != "keyring" {
			t.Fatalf("expected source 'keyring', got %s", src)
		}
		if !authtoken.IsKeyringOpError(err) {
			t.Fatalf("expected KeyringOpError, got %T: %v", err, err)
		}
	})

	t.Run("falls back to hosts file when keyring returns ErrNotFound cleanly", func(t *testing.T) {
		cfg := &Config{
			Env: func(k string) string { return "" },
			Keyring: &mockKeyring{
				tokens: map[string]string{}, // empty -> ErrNotFound
			},
			Hosts: map[string]HostConfig{
				"github.com": {OAuthToken: "host_token_file"},
			},
		}

		tok, src, err := cfg.AuthTokenForHost("github.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok != "host_token_file" || src != "hosts.yml" {
			t.Fatalf("expected host_token_file from hosts.yml, got %s from %s", tok, src)
		}
	})

	t.Run("returns ErrNotFound when all providers are absent", func(t *testing.T) {
		cfg := &Config{
			Env: func(k string) string { return "" },
			Keyring: &mockKeyring{
				tokens: map[string]string{},
			},
			Hosts: map[string]HostConfig{},
		}

		tok, src, err := cfg.AuthTokenForHost("github.com")
		if !errors.Is(err, authtoken.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		if tok != "" || src != "none" {
			t.Fatalf("expected empty token and 'none' source, got %s from %s", tok, src)
		}
	})
}
