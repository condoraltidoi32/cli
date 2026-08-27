package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// fakeStore lets tests inject each keyring path.
type fakeStore struct {
	token string
	err   error
}

func (f fakeStore) Lookup() (string, error) {
	return f.token, f.err
}

func TestResolveToken_Success(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	store := fakeStore{token: "tok-123", err: nil}
	token, err := resolveToken(store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "tok-123" {
		t.Errorf("expected tok-123, got %q", token)
	}
}

func TestResolveToken_EnvWins(t *testing.T) {
	t.Setenv("GH_TOKEN", "env-tok")
	// Even if the keyring would fail, env takes priority.
	store := fakeStore{err: errors.New("should not be called")}
	token, err := resolveToken(store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "env-tok" {
		t.Errorf("expected env-tok, got %q", token)
	}
}

func TestResolveToken_NotFoundAllowsUnauthenticated(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	store := fakeStore{err: ErrTokenNotFound}
	token, err := resolveToken(store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestApi_KeyringFailureAborts(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	store := fakeStore{err: fmt.Errorf("%w: dbus connection refused", ErrKeyringFailed)}

	// api() must return an error (not silently proceed unauthenticated).
	if err := api(store, []string{"repos/owner/repo"}); err == nil {
		t.Fatal("expected error when keyring access fails, got nil")
	} else if !errors.Is(err, ErrKeyringFailed) {
		t.Errorf("expected ErrKeyringFailed in chain, got %v", err)
	} else if !strings.Contains(err.Error(), "gh auth login") {
		t.Errorf("expected guidance in error message, got %q", err)
	}
}

func TestApi_UnauthenticatedWarns(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	store := fakeStore{err: ErrTokenNotFound}

	if err := api(store, []string{"repos/owner/repo"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
