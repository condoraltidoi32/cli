package authtoken

import (
	"errors"
	"fmt"
)

// ErrNotFound indicates that no authentication token was found for the given host in the keyring.
var ErrNotFound = errors.New("token not found in keyring")

// Keyring abstracts secret storage retrieval (system keyring, keychain, secret service).
type Keyring interface {
	Get(service, key string) (string, error)
}

// KeyringOpError represents an operational failure interacting with the system keyring.
type KeyringOpError struct {
	Host string
	Err  error
}

func (e *KeyringOpError) Error() string {
	return fmt.Sprintf("keyring access failed for host %s: %v", e.Host, e.Err)
}

func (e *KeyringOpError) Unwrap() error {
	return e.Err
}

// IsKeyringOpError returns true if err is an operational keyring failure.
func IsKeyringOpError(err error) bool {
	var opErr *KeyringOpError
	return errors.As(err, &opErr)
}

// TokenFromKeyring queries the keyring for a host token.
// Returns ErrNotFound if the token does not exist, or wraps operational failures in KeyringOpError.
func TokenFromKeyring(host string, kr Keyring) (string, error) {
	if kr == nil {
		return "", ErrNotFound
	}
	token, err := kr.Get("gh:"+host, "")
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", ErrNotFound
		}
		return "", &KeyringOpError{
			Host: host,
			Err:  err,
		}
	}
	if token == "" {
		return "", ErrNotFound
	}
	return token, nil
}
