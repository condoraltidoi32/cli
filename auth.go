package main

import (
	"bufio"
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
		return "", fmt.Errorf("keyring error: %w", err)
	}
	if secret == "" {
		return "", ErrTokenNotFound
	}
	return secret, nil
}

func tokenFromConfig(host string) (string, error) {
	path, err := hostsConfigPath()
	if err != nil {
		return "", ErrTokenNotFound
	}
	f, err := os.Open(path)
	if err != nil {
		return "", ErrTokenNotFound
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inHost := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, host+":") || strings.HasPrefix(line, "\""+host+"\":") {
			inHost = true
			continue
		}
		if inHost && len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			inHost = false
		}
		if inHost && (strings.HasPrefix(trimmed, "oauth_token:") || strings.HasPrefix(trimmed, "user:")) {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if val != "" {
					return val, nil
				}
			}
		}
	}
	return "", ErrTokenNotFound
}

// resolveToken resolves the auth token for the specified host according to the resolution order:
// 1. GH_TOKEN / GITHUB_TOKEN environment variables
// 2. System Keyring (distinguishes ErrNotFound from operational failures)
// 3. hosts.yml configuration file
func resolveToken(host string) (string, error) {
	// 1. Environment variables
	if token, err := tokenFromEnv(host); err == nil && token != "" {
		return token, nil
	}

	// 2. System Keyring
	if token, err := tokenFromKeyring(host); err == nil && token != "" {
		return token, nil
	} else if err != nil && !errors.Is(err, ErrTokenNotFound) {
		// Keyring operational failure: abort immediately
		return "", err
	}

	// 3. Hosts config file
	if token, err := tokenFromConfig(host); err == nil && token != "" {
		return token, nil
	}

	return "", ErrTokenNotFound
}
