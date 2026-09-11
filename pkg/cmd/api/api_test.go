package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/condoraltidoi32/cli/internal/config"
	"github.com/condoraltidoi32/cli/pkg/authtoken"
	"github.com/condoraltidoi32/cli/pkg/cmd/factory"
)

type mockKeyring struct {
	tokens map[string]string
	errs   map[string]error
}

func (m *mockKeyring) Get(service, key string) (string, error) {
	if err, ok := m.errs[service]; ok {
		return "", err
	}
	if tok, ok := m.tokens[service]; ok {
		return tok, nil
	}
	return "", authtoken.ErrNotFound
}

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func newTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestRunApi(t *testing.T) {
	t.Run("operational keyring failure aborts immediately and prints actionable error to stderr", func(t *testing.T) {
		cfg := &config.Config{
			Env: func(k string) string { return "" },
			Keyring: &mockKeyring{
				errs: map[string]error{"gh:github.com": errors.New("org.freedesktop.DBus.Error.NoReply")},
			},
		}

		f := factory.NewFactory(cfg)
		var stdout, stderr bytes.Buffer

		opts := &ApiOptions{
			Factory:  f,
			Host:     "github.com",
			Endpoint: "user",
			StdOut:   &stdout,
			StdErr:   &stderr,
		}

		err := RunApi(opts)
		if err == nil {
			t.Fatal("expected RunApi to fail with non-zero error, got nil")
		}

		stderrStr := stderr.String()
		if !strings.Contains(stderrStr, "keyring access failed") {
			t.Errorf("expected stderr to contain 'keyring access failed', got:\n%s", stderrStr)
		}
		if !strings.Contains(stderrStr, "GH_TOKEN or GITHUB_TOKEN") {
			t.Errorf("expected stderr to contain troubleshooting tip about GH_TOKEN, got:\n%s", stderrStr)
		}
		if !strings.Contains(stderrStr, "org.freedesktop.DBus.Error.NoReply") {
			t.Errorf("expected stderr to contain root cause error message, got:\n%s", stderrStr)
		}
	})

	t.Run("unauthenticated request succeeds when no credentials configured and endpoint is public", func(t *testing.T) {
		cfg := &config.Config{
			Env:     func(k string) string { return "" },
			Keyring: &mockKeyring{tokens: map[string]string{}},
		}

		f := factory.NewFactory(cfg)
		var stdout, stderr bytes.Buffer

		var sentAuthHeader string
		client := newTestClient(func(req *http.Request) *http.Response {
			sentAuthHeader = req.Header.Get("Authorization")
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
				Header:     make(http.Header),
			}
		})

		opts := &ApiOptions{
			Factory:    f,
			Host:       "github.com",
			Endpoint:   "zen",
			HTTPClient: client,
			StdOut:     &stdout,
			StdErr:     &stderr,
		}

		err := RunApi(opts)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if sentAuthHeader != "" {
			t.Errorf("expected empty Authorization header for unauthenticated request, got '%s'", sentAuthHeader)
		}
		if !strings.Contains(stdout.String(), `{"status":"ok"}`) {
			t.Errorf("expected stdout to contain response, got: %s", stdout.String())
		}
	})

	t.Run("authenticated request includes token header", func(t *testing.T) {
		cfg := &config.Config{
			Env: func(k string) string {
				if k == "GH_TOKEN" {
					return "ghp_validtoken"
				}
				return ""
			},
			Keyring: &mockKeyring{tokens: map[string]string{}},
		}

		f := factory.NewFactory(cfg)
		var stdout, stderr bytes.Buffer

		var sentAuthHeader string
		client := newTestClient(func(req *http.Request) *http.Response {
			sentAuthHeader = req.Header.Get("Authorization")
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"login":"eve"}`)),
				Header:     make(http.Header),
			}
		})

		opts := &ApiOptions{
			Factory:    f,
			Host:       "github.com",
			Endpoint:   "user",
			HTTPClient: client,
			StdOut:     &stdout,
			StdErr:     &stderr,
		}

		err := RunApi(opts)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if sentAuthHeader != "token ghp_validtoken" {
			t.Errorf("expected 'token ghp_validtoken', got '%s'", sentAuthHeader)
		}
	})
}
