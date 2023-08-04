package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jonas27/ramp-up-k8s-operator/frontend/pkg"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/errgroup"
)

const (
	exitFail             = 1
	serverTimeoutSeconds = 3
	tickerSeconds        = 100
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{AddSource: true}))

	if err := run(os.Args, log); err != nil {
		log.Error(err.Error())
		os.Exit(exitFail)
	}
}

func run(args []string, log *slog.Logger) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	addr := flags.String("addr", ":8080", "The addr with colon")
	grpcAddr := flags.String("grpc-addr", ":8000", "The server addr with colon")
	debug := flags.Bool("debug", false, "Start the server in debug mode")
	templatePath := flags.String("templates-path", "templates/", "The path for html templates")

	if err := flags.Parse(args[1:]); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	server := pkg.Server{
		Debug:    *debug,
		GRPCAddr: *grpcAddr,
		Log:      log,
		Server: &http.Server{
			Addr:              *addr,
			ReadHeaderTimeout: serverTimeoutSeconds * time.Second,
		},
		TemplatePath: *templatePath,
	}
	server.Routes()

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	errWg, errCtx := errgroup.WithContext(ctx)
	log.Info("grpcAddr for dialing", "address", *grpcAddr)

	errWg.Go(func() error {
		log.Info("Server running", "address", *addr)
		if err := server.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("the server failed with error: %w", err)
		}
		return nil
	})

	errWg.Go(func() error {
		<-errCtx.Done()
		if err := server.Server.Shutdown(errCtx); err != nil {
			return fmt.Errorf("could not shutdown server gracefully: %w", err)
		}
		return nil
	})

	err := errWg.Wait()
	if !errors.Is(err, context.Canceled) && err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	server.Log.Info("server quit gracefully")
	return nil
}
