package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/condoraltidoi32/cli/pkg/authtoken"
)

// envStore resolves tokens from the GH_TOKEN environment variable.
// A production build would back this with the system keyring; the failure
// contract is identical either way: missing means ErrNotFound, anything else
// is an operational failure that must abort the command.
type envStore struct{}

func (envStore) Get(host string) (string, error) {
	if host == "" {
		return "", authtoken.ErrNotFound
	}
	if v := os.Getenv("GH_TOKEN"); v != "" {
		return v, nil
	}
	return "", authtoken.ErrNotFound
}

func main() {
	const host = "api.github.com"
	tok, err := authtoken.Resolver{Store: envStore{}}.Resolve(host)
	if err != nil {
		var opErr *authtoken.KeyringOpError
		if errors.As(err, &opErr) {
			fmt.Fprintf(os.Stderr, "gh: cannot authenticate with %s: credential store failed (%v); refusing to send an unauthenticated request\n", host, opErr.Err)
		} else {
			fmt.Fprintf(os.Stderr, "gh: cannot authenticate with %s: %v\n", host, err)
		}
		os.Exit(1)
	}
	_ = tok // authenticated; no token is ever printed
	fmt.Println("Hello, Bounty Hunter!")
}
