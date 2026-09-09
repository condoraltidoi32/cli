package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func setHTTPClient(t *testing.T, client *http.Client) {
	t.Helper()
	old := httpClient
	httpClient = client
	t.Cleanup(func() { httpClient = old })
}

func TestRunAbortsOnKeyringOperationalFailure(t *testing.T) {
	setKeyring(t, func(_, _ string) (string, error) {
		return "", errors.New("dbus: connection closed")
	})
	unsetEnv(t, "GH_TOKEN", "GITHUB_TOKEN", "GH_TOKEN_GITHUB_COM")

	var stdout, stderr bytes.Buffer
	code := run([]string{"api", "https://api.github.com/user"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	out := stderr.String()
	if !strings.Contains(out, "keyring") || !strings.Contains(out, "dbus: connection closed") {
		t.Fatalf("expected actionable keyring error on stderr, got: %s", out)
	}
	if !strings.Contains(out, "GH_TOKEN") {
		t.Fatalf("expected troubleshooting advice mentioning GH_TOKEN, got: %s", out)
	}
}

func TestRunUnauthenticatedWhenNothingConfigured(t *testing.T) {
	setKeyring(t, func(_, _ string) (string, error) {
		return "", keyring.ErrNotFound
	})
	unsetEnv(t, "GH_TOKEN", "GITHUB_TOKEN", "GH_TOKEN_GITHUB_COM")
	setConfigPath(t, os.DevNull)

	setHTTPClient(t, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Fatalf("expected unauthenticated request, got Authorization %q", auth)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader("{\"ok\":true}")),
			Header:     make(http.Header),
		}, nil
	})})

	var stdout, stderr bytes.Buffer
	code := run([]string{"api", "https://api.github.com/octocat"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "200 OK") {
		t.Fatalf("expected response status on stdout, got: %s", stdout.String())
	}
}

func TestRunAuthenticatedWithToken(t *testing.T) {
	setKeyring(t, func(_, _ string) (string, error) {
		return "secret-token", nil
	})
	unsetEnv(t, "GH_TOKEN", "GITHUB_TOKEN", "GH_TOKEN_GITHUB_COM")
	var sawAuth string
	setHTTPClient(t, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		sawAuth = r.Header.Get("Authorization")
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})})

	var stdout, stderr bytes.Buffer
	code := run([]string{"api", "https://api.github.com/user"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}
	if sawAuth != "Bearer secret-token" {
		t.Fatalf("expected Bearer secret-token, got %q", sawAuth)
	}
}

func TestRunMissingArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"api"}, &stdout, &stderr); code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage on stderr, got: %s", stderr.String())
	}
}
