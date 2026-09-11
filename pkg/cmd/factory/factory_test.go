package factory

import (
	"errors"
	"testing"

	"github.com/condoraltidoi32/cli/internal/config"
	"github.com/condoraltidoi32/cli/pkg/authtoken"
)

type mockKeyring struct {
	err error
}

func (m *mockKeyring) Get(service, key string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "", authtoken.ErrNotFound
}

func TestFactoryHttpClient(t *testing.T) {
	t.Run("returns error on keyring operational failure", func(t *testing.T) {
		cfg := &config.Config{
			Env:     func(k string) string { return "" },
			Keyring: &mockKeyring{err: errors.New("keychain locked")},
		}

		f := NewFactory(cfg)
		client, err := f.HttpClientFunc()
		if err == nil {
			t.Fatal("expected error from HttpClientFunc when keyring fails operationally, got nil")
		}
		if client != nil {
			t.Fatal("expected nil client on operational error")
		}
	})

	t.Run("returns client successfully when token is clean not-found (unauthenticated fallback)", func(t *testing.T) {
		cfg := &config.Config{
			Env:     func(k string) string { return "" },
			Keyring: &mockKeyring{err: nil},
		}

		f := NewFactory(cfg)
		client, err := f.HttpClientFunc()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})
}
