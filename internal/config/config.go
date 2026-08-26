package config

import (
	"errors"
	"os"

	"github.com/condoraltidoi32/cli/pkg/authtoken"
)

// Config represents the application configuration.
type Config struct {
	Keyring authtoken.Keyring
	Hosts   map[string]HostConfig
	Env     func(string) string
}

// HostConfig stores settings per hostname.
type HostConfig struct {
	User      string
	OAuthToken string
}

// NewConfig creates a Config instance with default environment lookup.
func NewConfig(kr authtoken.Keyring) *Config {
	return &Config{
		Keyring: kr,
		Hosts:   make(map[string]HostConfig),
		Env:     os.Getenv,
	}
}

// AuthTokenForHost resolves the authentication token for a given host.
// It searches in order:
// 1. GH_TOKEN / GITHUB_TOKEN environment variables
// 2. System Keyring (via authtoken.TokenFromKeyring)
// 3. Hosts configuration file/store
// If the keyring encounters an operational failure, resolution aborts immediately and propagates the error.
func (c *Config) AuthTokenForHost(host string) (token string, source string, err error) {
	// 1. Environment variables
	if c.Env != nil {
		if val := c.Env("GH_TOKEN"); val != "" {
			return val, "GH_TOKEN", nil
		}
		if val := c.Env("GITHUB_TOKEN"); val != "" {
			return val, "GITHUB_TOKEN", nil
		}
	}

	// 2. System Keyring
	if c.Keyring != nil {
		tok, krErr := authtoken.TokenFromKeyring(host, c.Keyring)
		if krErr == nil && tok != "" {
			return tok, "keyring", nil
		}
		if krErr != nil && !errors.Is(krErr, authtoken.ErrNotFound) {
			// Operational error accessing keyring -> MUST NOT fall back or swallow!
			return "", "keyring", krErr
		}
	}

	// 3. Hosts config
	if c.Hosts != nil {
		if hc, ok := c.Hosts[host]; ok && hc.OAuthToken != "" {
			return hc.OAuthToken, "hosts.yml", nil
		}
	}

	return "", "none", authtoken.ErrNotFound
}

// Authentication returns an Authentication helper for the config.
func (c *Config) Authentication() *Config {
	return c
}
