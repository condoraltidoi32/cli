package factory

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/condoraltidoi32/cli/internal/config"
	"github.com/condoraltidoi32/cli/pkg/authtoken"
)

// Factory provides shared dependencies for CLI commands.
type Factory struct {
	ConfigFunc     func() (*config.Config, error)
	HttpClientFunc func() (*http.Client, error)
	StdOut         io.Writer
	StdErr         io.Writer
}

// NewFactory creates a standard CLI factory.
func NewFactory(cfg *config.Config) *Factory {
	f := &Factory{
		StdOut: os.Stdout,
		StdErr: os.Stderr,
	}

	f.ConfigFunc = func() (*config.Config, error) {
		if cfg == nil {
			return config.NewConfig(nil), nil
		}
		return cfg, nil
	}

	f.HttpClientFunc = func() (*http.Client, error) {
		c, err := f.ConfigFunc()
		if err != nil {
			return nil, err
		}

		// Resolve token for default host (github.com)
		_, _, authErr := c.AuthTokenForHost("github.com")
		if authErr != nil && authtoken.IsKeyringOpError(authErr) {
			return nil, fmt.Errorf("operational keyring failure: %w", authErr)
		}

		return &http.Client{}, nil
	}

	return f
}
