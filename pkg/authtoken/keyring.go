package authtoken

import (
	"errors"
	"fmt"
	"github.com/zalando/go-keyring"
)

// ErrKeyringFailure represents an operational failure of the keyring service
// (e.g., locked keychain, missing daemon). It is distinct from ErrNotFound.
var ErrKeyringFailure = fmt.Errorf("keyring access error")

func Get(host string) (string, error) {
	token, err := keyring.Get("github.com", host)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", keyring.ErrNotFound
		}
		return "", fmt.Errorf("%w: %w", ErrKeyringFailure, err)
	}
	return token, nil
}
