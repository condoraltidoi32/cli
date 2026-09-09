package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// httpClient is swappable so tests never touch the network.
var httpClient = &http.Client{Timeout: 30 * time.Second}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

const usage = `usage:
  cli api [-host HOST] <url>

performs an API request, authenticated when credentials are available:
  cli api https://api.github.com/repos/cli/cli

credentials are resolved in order: GH_TOKEN / GITHUB_TOKEN environment
variables, the system keyring, then the gh hosts config file. A request
is sent unauthenticated only when no credentials are configured at all.
`

// run is the entry point, kept separate from main so tests can assert on
// exit codes and output without spawning processes.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	host := fs.String("host", "github.com", "host whose credentials to use")
	if err := fs.Parse(args); err != nil {
		fmt.Fprint(stderr, usage)
		return 2
	}

	rest := fs.Args()
	if len(rest) == 0 || rest[0] == "help" || rest[0] == "-h" {
		fmt.Fprint(stderr, usage)
		return 2
	}
	if rest[0] != "api" {
		fmt.Fprintf(stderr, "cli: unknown command %q\n%s", rest[0], usage)
		return 2
	}
	if len(rest) < 2 {
		fmt.Fprintf(stderr, "cli: missing URL argument\n%s", usage)
		return 2
	}

	token, err := resolveToken(*host)
	if err != nil && !errors.Is(err, ErrTokenNotFound) {
		fmt.Fprintf(stderr, "cli: failed to read credentials from the system keyring for %s: %v\n", *host, err)
		fmt.Fprintln(stderr, "Troubleshooting: verify the keyring or secret service is unlocked and responding")
		fmt.Fprintln(stderr, "  (e.g. unset DBUS_SESSION_BUS_ADDRESS on Linux, unlock the login keychain on macOS).")
		fmt.Fprintln(stderr, "Alternatively set GH_TOKEN to bypass the keyring and try again.")
		return 1
	}

	req, err := http.NewRequest(http.MethodGet, rest[1], nil)
	if err != nil {
		fmt.Fprintf(stderr, "cli: invalid URL %q: %v\n", rest[1], err)
		return 2
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(stderr, "cli: request failed: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Fprintln(stdout, resp.Status)
	fmt.Fprintln(stdout, string(body))
	if resp.StatusCode >= 400 {
		return 1
	}
	return 0
}
