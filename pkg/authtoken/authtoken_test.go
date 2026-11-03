package authtoken

import (
	"errors"
	"strings"
	"testing"
)

type fakeStore struct {
	tok string
	err error
}

func (f fakeStore) Get(host string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.tok, nil
}

func TestResolve(t *testing.T) {
	opErr := errors.New("keychain locked")
	cases := []struct {
		name    string
		host    string
		store   fakeStore
		wantTok string
		wantNot bool // want ErrNotFound
		wantOp  bool // want *KeyringOpError
	}{
		{"success", "api.github.com", fakeStore{tok: "abc"}, "abc", false, false},
		{"missing token", "api.github.com", fakeStore{err: ErrNotFound}, "", true, false},
		{"empty token treated as missing", "api.github.com", fakeStore{tok: ""}, "", true, false},
		{"operational failure", "api.github.com", fakeStore{err: opErr}, "", false, true},
		{"empty host", "", fakeStore{tok: "abc"}, "", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Resolver{Store: c.store}.Resolve(c.host)
			if c.name == "empty host" {
				if err == nil || got != "" {
					t.Fatalf("expected error, got %q, %v", got, err)
				}
				return
			}
			if c.wantOp {
				var op *KeyringOpError
				if !errors.As(err, &op) {
					t.Fatalf("expected *KeyringOpError, got %v", err)
				}
				if !strings.Contains(err.Error(), c.host) {
					t.Fatalf("error should name the host: %v", err)
				}
				return
			}
			if c.wantNot {
				if !errors.Is(err, ErrNotFound) {
					t.Fatalf("expected ErrNotFound, got %v", err)
				}
				return
			}
			if err != nil || got != c.wantTok {
				t.Fatalf("got %q, %v; want %q, nil", got, err, c.wantTok)
			}
		})
	}
}
