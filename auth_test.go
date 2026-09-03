package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestResolveEnvWinsEvenIfKeyringFails(t *testing.T) {
	setEnv(t, "GH_TOKEN", "env-token")
	setKeyring(t, func(_, _ string) (string, error) {
		return "", errors.New("dbus: connection closed")
	})

	token, err := resolveToken("github.com")
	if err != nil {
		t.Fatalf("expected env token, got error %v", err)
	}
	if token != "env-token" {
		t.Fatalf("expected env-token, got %q", token)
	}
}

func TestResolveKeyringToken(t *testing.T) {
	unsetEnv(t, "GH_TOKEN", "GITHUB_TOKEN")
	setKeyring(t, func(_, _ string) (string, error) {
		return "keyring-token", nil
	})

	token, err := resolveToken("github.com")
	if err != nil || token != "keyring-token" {
		t.Fatalf("expected keyring-token, got %q, %v", token, err)
	}
}

func TestResolveKeyringNotFoundFallsBackToConfigFile(t *testing.T) {
	unsetEnv(t, "GH_TOKEN", "GITHUB_TOKEN")
	setKeyring(t, func(_, _ string) (string, error) {
		return "", keyring.ErrNotFound
	})
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.yml")
	content := "github.com:\n    user: me\n    oauth_token: config-token\n    git_protocol: https\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	setConfigPath(t, path)

	token, err := resolveToken("github.com")
	if err != nil || token != "config-token" {
		t.Fatalf("expected config-token, got %q, %v", token, err)
	}
}

func TestResolveKeyringOperationalErrorHalts(t *testing.T) {
	unsetEnv(t, "GH_TOKEN", "GITHUB_TOKEN")
	setKeyring(t, func(_, _ string) (string, error) {
		return "", errors.New("dbus: connection closed")
	})

	_, err := resolveToken("github.com")
	if err == nil {
		t.Fatal("expected operational keyring error to halt resolution")
	}
	if !strings.Contains(err.Error(), "dbus: connection closed") {
		t.Fatalf("expected underlying keyring error to be surfaced, got %v", err)
	}
}

func TestResolveNothingConfigured(t *testing.T) {
	unsetEnv(t, "GH_TOKEN", "GITHUB_TOKEN")
	setKeyring(t, func(_, _ string) (string, error) {
		return "", keyring.ErrNotFound
	})
	setConfigPath(t, filepath.Join(t.TempDir(), "missing.yml"))

	token, err := resolveToken("github.com")
	if err != ErrTokenNotFound {
		t.Fatalf("expected ErrTokenNotFound, got %q, %v", token, err)
	}
	if token != "" {
		t.Fatalf("expected empty token, got %q", token)
	}
}

func TestParseHostsToken(t *testing.T) {
	content := "github.com:\n    oauth_token: abc\nother.example.com:\n    oauth_token: def\n"
	token, err := parseHostsToken(content, "github.com")
	if err != nil || token != "abc" {
		t.Fatalf("expected abc, got %q, %v", token, err)
	}
	if _, err := parseHostsToken(content, "unknown.com"); err != ErrTokenNotFound {
		t.Fatalf("expected ErrTokenNotFound for unknown host, got %v", err)
	}
}
