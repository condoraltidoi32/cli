package main

import (
	"errors"
	"fmt"
	"os"
)

// ErrTokenNotFound indicates the token simply does not exist in the store —
// the only case where proceeding unauthenticated is safe.
var ErrTokenNotFound = errors.New("no token found in credential store")

// ErrKeyringFailed indicates an operational failure of the credential store
// (locked keychain, missing D-Bus, OS permission denied, corrupted store).
// Unlike ErrTokenNotFound, this is a system error that must abort, never be
// silently swallowed into an unauthenticated request.
var ErrKeyringFailed = errors.New("keyring access failed")

// keyringStore abstracts the credential store so tests can inject success,
// not-found, and failure paths.
type keyringStore interface {
	// Lookup returns the stored token, ErrTokenNotFound when no token is
	// stored, or ErrKeyringFailed (wrapped) on an operational failure.
	Lookup() (string, error)
}

// defaultKeyring simulates the system keyring.
type defaultKeyring struct{}

func (defaultKeyring) Lookup() (string, error) {
	// Simulate an operational failure, e.g. dbus connection refused.
	return "", fmt.Errorf("%w: dbus connection refused", ErrKeyringFailed)
}

// resolveToken returns the auth token for the host. It checks environment
// first, then the keyring. A keyring OPERATIONAL failure aborts with a
// descriptive error instead of silently falling back to unauthenticated API
// calls; only a genuine "no token stored" (ErrTokenNotFound) permits the
// unauthenticated path.
func resolveToken(store keyringStore) (string, error) {
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token, nil
	}

	token, err := store.Lookup()
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			// No token stored — safe to proceed unauthenticated.
			return "", nil
		}
		// Operational keyring failure. Surface it with guidance; do not send
		// an unauthenticated request that the user did not intend.
		return "", fmt.Errorf(
			"cannot resolve credentials from system keyring; run 'gh auth login' or set GH_TOKEN: %w",
			err,
		)
	}
	return token, nil
}

// api is the entry point for `gh api`-style commands. It resolves the token
// and aborts on keyring failure before making any request.
func api(store keyringStore, args []string) error {
	token, err := resolveToken(store)
	if err != nil {
		return err
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "Warning: running unauthenticated (no token stored)")
	}
	// In production, build the HTTP request with the resolved token here.
	_ = args
	return nil
}

func main() {
	if err := api(defaultKeyring{}, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}
