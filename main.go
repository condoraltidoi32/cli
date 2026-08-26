package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/condoraltidoi32/cli/internal/config"
	"github.com/condoraltidoi32/cli/pkg/cmd/api"
	"github.com/condoraltidoi32/cli/pkg/cmd/factory"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: cli <command> [arguments]")
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "api":
		apiCmd := flag.NewFlagSet("api", flag.ExitOnError)
		host := apiCmd.String("hostname", "github.com", "GitHub host")
		method := apiCmd.String("method", "GET", "HTTP method")
		apiCmd.Parse(os.Args[2:])

		args := apiCmd.Args()
		endpoint := "user"
		if len(args) > 0 {
			endpoint = args[0]
		}

		cfg := config.NewConfig(nil)
		f := factory.NewFactory(cfg)
		opts := &api.ApiOptions{
			Factory:  f,
			Host:     *host,
			Method:   *method,
			Endpoint: endpoint,
			StdOut:   os.Stdout,
			StdErr:   os.Stderr,
		}

		if err := api.RunApi(opts); err != nil {
			os.Exit(1)
		}

	case "version":
		fmt.Println("cli version 1.0.0")

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}
