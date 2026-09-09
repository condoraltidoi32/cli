package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

func setKeyring(t *testing.T, fn func(string, string) (string, error)) {
	t.Helper()
	old := keyringGet
	keyringGet = fn
	t.Cleanup(func() { keyringGet = old })
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	old, existed := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		old, existed := os.LookupEnv(key)
		_ = os.Unsetenv(key)
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(key, old)
			}
		})
	}
}

func setConfigPath(t *testing.T, path string) {
	t.Helper()
	old := hostsConfigPath
	hostsConfigPath = func() (string, error) { return path, nil }
	t.Cleanup(func() { hostsConfigPath = old })
}

func TestResolveToken_EnvPriority(t *testing.T) {
	setEnv(t, "GH_TOKEN", "env-token")
	setKeyring(t, func(_, _ string) (string, error) {
		return "keyring-token", nil
	})

	token, err := resolveToken("github.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "env-token" {
		t.Fatalf("expected env-token, got %q", token)
	}
}

func TestResolveToken_KeyringOperationalErrorHalts(t *testing.T) {
	unsetEnv(t, "GH_TOKEN", "GITHUB_TOKEN", "GH_TOKEN_GITHUB_COM")
	setKeyring(t, func(_, _ string) (string, error) {
		return "", errors.New("dbus: connection closed")
	})

	_, err := resolveToken("github.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("expected operational error, got ErrTokenNotFound")
	}
}

func TestResolveToken_KeyringNotFoundFallsBack(t *testing.T) {
	unsetEnv(t, "GH_TOKEN", "GITHUB_TOKEN", "GH_TOKEN_GITHUB_COM")
	setKeyring(t, func(_, _ string) (string, error) {
		return "", keyring.ErrNotFound
	})

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "hosts.yml")
	content := "github.com:\n  oauth_token: config-token\n"
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	setConfigPath(t, configPath)

	token, err := resolveToken("github.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "config-token" {
		t.Fatalf("expected config-token, got %q", token)
	}
}
