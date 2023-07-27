package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jonas27/ramp-up-k8s-operator/frontend/pkg"
	"golang.org/x/exp/slog"
)

const (
	exitFail             = 1
	serverTimeoutSeconds = 3
	tickerSeconds        = 100
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{AddSource: true}))

	if err := run(os.Args, logger); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFail)
	}
}

func run(args []string, log *slog.Logger) error { //nolint:cyclop,funlen
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	addr := flags.String("addr", ":8080", "The server addr with colon")
	debug := flags.Bool("debug", false, "Start the server in debug mode")
	templatePath := flags.String("templates-path", "templates/", "The path for html templates")

	if err := flags.Parse(args[1:]); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	server := pkg.Server{
		Debug: *debug,
		Log:   log,
		Mux:   http.NewServeMux(),
		Server: &http.Server{
			Addr:              *addr,
			ReadHeaderTimeout: serverTimeoutSeconds * time.Second,
		},
		TemplatePath: *templatePath}
	return server.Serve()
}
