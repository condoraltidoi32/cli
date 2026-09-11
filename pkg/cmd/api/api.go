package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/condoraltidoi32/cli/pkg/authtoken"
	"github.com/condoraltidoi32/cli/pkg/cmd/factory"
)

// ApiOptions contains options for executing an API command.
type ApiOptions struct {
	Factory    *factory.Factory
	Host       string
	Endpoint   string
	Method     string
	Body       io.Reader
	Headers    []string
	HTTPClient *http.Client
	StdOut     io.Writer
	StdErr     io.Writer
}

// RunApi executes the API command against the target host.
func RunApi(opts *ApiOptions) error {
	if opts.Host == "" {
		opts.Host = "github.com"
	}
	if opts.Method == "" {
		opts.Method = "GET"
	}

	stdout := opts.StdOut
	if stdout == nil && opts.Factory != nil {
		stdout = opts.Factory.StdOut
	}
	stderr := opts.StdErr
	if stderr == nil && opts.Factory != nil {
		stderr = opts.Factory.StdErr
	}

	cfg, err := opts.Factory.ConfigFunc()
	if err != nil {
		return fmt.Errorf("could not load configuration: %w", err)
	}

	// Resolve authentication token
	token, source, authErr := cfg.AuthTokenForHost(opts.Host)
	if authErr != nil {
		if authtoken.IsKeyringOpError(authErr) {
			// Operational keyring error: abort immediately with actionable troubleshooting output
			if stderr != nil {
				fmt.Fprintf(stderr, "error: keyring access failed for host '%s': %v\n", opts.Host, authErr)
				fmt.Fprintln(stderr, "Troubleshooting:")
				fmt.Fprintln(stderr, "- Verify that your system keyring daemon (e.g. D-Bus, Secret Service, macOS Keychain) is running and unlocked.")
				fmt.Fprintln(stderr, "- To bypass the keyring, export the GH_TOKEN or GITHUB_TOKEN environment variable.")
			}
			return fmt.Errorf("keyring access failed: %w", authErr)
		}

		if !errors.Is(authErr, authtoken.ErrNotFound) {
			if stderr != nil {
				fmt.Fprintf(stderr, "error: credential resolution failed: %v\n", authErr)
			}
			return authErr
		}
		// If ErrNotFound, token remains empty and we proceed unauthenticated
	}

	// Prepare URL
	var baseURL string
	if strings.EqualFold(opts.Host, "github.com") {
		baseURL = "https://api.github.com"
	} else {
		baseURL = fmt.Sprintf("https://%s/api/v3", opts.Host)
	}

	reqURL := fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), strings.TrimLeft(opts.Endpoint, "/"))
	req, err := http.NewRequest(opts.Method, reqURL, opts.Body)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	for _, h := range opts.Headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	client := opts.HTTPClient
	if client == nil {
		var clientErr error
		client, clientErr = opts.Factory.HttpClientFunc()
		if clientErr != nil {
			return clientErr
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		if stderr != nil {
			fmt.Fprintf(stderr, "HTTP %d: %s\n", resp.StatusCode, string(bodyBytes))
			if source == "none" {
				fmt.Fprintln(stderr, "Note: request was sent unauthenticated because no credentials were configured.")
			}
		}
		return fmt.Errorf("HTTP %d error", resp.StatusCode)
	}

	if stdout != nil {
		stdout.Write(bodyBytes)
		stdout.Write([]byte("\n"))
	}

	return nil
}
