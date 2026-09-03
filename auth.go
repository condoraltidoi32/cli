package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

// ErrTokenNotFound means a credential provider holds no token for the host.
// It is not an error: resolution simply moves to the next provider.
var ErrTokenNotFound = errors.New("no token found")

// keyringGet abstracts the system credential store so tests can simulate
// both missing secrets and operational failures (locked keychain, missing
// D-Bus session, OS permission denial).
var keyringGet = keyring.Get

// hostsConfigPath locates the gh hosts config file. It is a variable so
// tests can point it at a temporary file.
var hostsConfigPath = func() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gh", "hosts.yml"), nil
}

func tokenFromEnv(host string) (string, error) {
	hostEnv := "GH_TOKEN_" + strings.ToUpper(strings.ReplaceAll(host, ".", "_"))
	for _, name := range []string{hostEnv, "GH_TOKEN", "GITHUB_TOKEN"} {
		if v := os.Getenv(name); v != "" {
			return v, nil
		}
	}
	return "", ErrTokenNotFound
}

func tokenFromKeyring(host string) (string, error) {
	secret, err := keyringGet("gh:"+host, host)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrTokenNotFound
		}
		// Any error other than ErrNotFound is an operational failure of the
		// keyring itself (e.g. "dbus: connection closed", errSecAuthFailed,
		// access denied). It must halt resolution instead of being swallowed,
		// otherwise the CLI would silently proceed unauthenticated.
		return "", err
	}
	if secret == "" {
		return "", ErrTokenNotFound
	}
	return secret, nil
}

// tokenFromConfigFile is the legacy fallback: the gh hosts.yml file.
func tokenFromConfigFile(host string) (string, error) {
	path, err := hostsConfigPath()
	if err != nil {
		return "", ErrTokenNotFound
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ErrTokenNotFound
	}
	return parseHostsToken(string(data), host)
}

// parseHostsToken extracts the oauth_token for host from a hosts.yml-style
// file without pulling in a YAML dependency.
func parseHostsToken(content, host string) (string, error) {
	want := strings.TrimSpace(host) + ":"
	inHost := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "oauth_token:"):
			if inHost {
				return strings.TrimSpace(strings.TrimPrefix(trimmed, "oauth_token:")), nil
			}
		case strings.HasPrefix(trimmed, want):
			inHost = true
		case strings.HasSuffix(trimmed, ":"):
			inHost = false
		}
	}
	return "", ErrTokenNotFound
}

// resolveToken walks the credential chain for host in order: environment,
// system keyring, config file. It returns the first token found, or
// ErrTokenNotFound when nothing is configured anywhere (meaning unauthenticated
// requests are allowed). Any other error is an operational failure that must
// abort the command rather than silently continuing without a token.
func resolveToken(host string) (string, error) {
	providers := []struct {
		name string
		fn   func(string) (string, error)
	}{
		{"environment", tokenFromEnv},
		{"keyring", tokenFromKeyring},
		{"config file", tokenFromConfigFile},
	}
	for _, provider := range providers {
		token, err := provider.fn(host)
		switch {
		case err == nil:
			return token, nil
		case errors.Is(err, ErrTokenNotFound):
			// Nothing stored here; try the next provider.
		default:
			return "", fmt.Errorf("%s: %w", provider.name, err)
		}
	}
	return "", ErrTokenNotFound
}
