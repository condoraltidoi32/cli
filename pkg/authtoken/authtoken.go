// Package authtoken resolves API tokens for a target host, distinguishing a
// missing credential (ErrNotFound) from an operational credential-store
// failure (KeyringOpError). Callers must abort before any HTTP request when
// resolution fails, so a locked keychain can never degrade into a silent
// unauthenticated call.
package authtoken

import (
	"errors"
	"fmt"
)

// ErrNotFound reports that no token exists for the host.
var ErrNotFound = errors.New("authtoken: no token found")

// KeyringOpError wraps an operational credential-store failure
// (locked keychain, missing D-Bus session, permission denied, corrupt store).
type KeyringOpError struct {
	Host string
	Err  error
}

func (e *KeyringOpError) Error() string {
	return fmt.Sprintf("authtoken: credential store unavailable for host %q: %v", e.Host, e.Err)
}

func (e *KeyringOpError) Unwrap() error { return e.Err }

// Store abstracts the credential backend (system keyring, env, config file).
type Store interface {
	// Get returns the token, or ErrNotFound when absent. Any other error is
	// treated as an operational failure of the store itself.
	Get(host string) (string, error)
}

// Resolver resolves tokens via a Store.
type Resolver struct{ Store Store }

// Resolve returns the token for host. A missing token yields an error
// wrapping ErrNotFound; a broken store yields *KeyringOpError. Both must
// abort the command before any HTTP request is sent.
func (r Resolver) Resolve(host string) (string, error) {
	if host == "" {
		return "", fmt.Errorf("authtoken: empty host")
	}
	tok, err := r.Store.Get(host)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", fmt.Errorf("authtoken: no token for host %q (run login first): %w", host, ErrNotFound)
		}
		return "", &KeyringOpError{Host: host, Err: err}
	}
	if tok == "" {
		return "", fmt.Errorf("authtoken: no token for host %q (run login first): %w", host, ErrNotFound)
	}
	return tok, nil
}
