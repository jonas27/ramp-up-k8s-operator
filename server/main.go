// see https://github.com/grpc-ecosystem/go-grpc-middleware/blob/main/examples/server/main.go#L30

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	pb "github.com/jonas27/ramp-up-k8s-operator/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const (
	exitFail             = 1
	serverTimeoutSeconds = 3
	component            = "grpc-example"
)

// interceptorLogger adapts slog logger to interceptor logger.
// This code is simple enough to be copied and not imported.
// see https://github.com/grpc-ecosystem/go-grpc-middleware/blob/main/interceptors/logging/examples/slog/example_test.go
func interceptorLogger(l *slog.Logger) logging.Logger { //nolint:ireturn
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{AddSource: true}))

	if err := run(os.Args, log); err != nil {
		log.Error(err.Error())
		os.Exit(exitFail)
	}
}

type server struct {
	pb.UnimplementedCharacterCounterServer
}

func (s *server) CountCharacters(_ context.Context, req *pb.CountCharactersRequest) (*pb.CountCharactersResponse, error) { //nolint:lll
	chars := uint64(len(req.Text))
	return &pb.CountCharactersResponse{Characters: chars}, nil
}

func run(args []string, log *slog.Logger) error { //nolint:funlen,cyclop
	// setup logging
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	logTraceID := func(ctx context.Context) logging.Fields {
		if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
			return logging.Fields{"traceID", span.TraceID().String()}
		}
		return nil
	}

	// setup flags
	grpcAddr := flags.String("grpc-addr", ":8000", "The server addr with colon")
	httpAddr := flags.String("http-addr", ":8001", "The server addr with colon")
	if err := flags.Parse(args[1:]); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	srvMetrics, reg, exemplarFromContext := setupMetrics()

	exporter, err := setupOLTPTracing()
	if err != nil {
		return err
	}
	defer func() { _ = exporter.Shutdown(context.Background()) }()

	// Setup metric for panic recoveries.
	panicsTotal := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "grpc_req_panics_recovered_total",
		Help: "Total number of gRPC requests recovered from internal panic.",
	})
	grpcPanicRecoveryHandler := func(p any) error {
		panicsTotal.Inc()
		log.Warn("recovered from panic", "panic", p, "stack", debug.Stack())
		return status.Errorf(codes.Internal, "%s", p)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	errWg, errCtx := errgroup.WithContext(ctx)

	// setup grpc server
	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// Order matters e.g. tracing interceptor have to create span first for the later exemplars to work.
			otelgrpc.UnaryServerInterceptor(),
			srvMetrics.UnaryServerInterceptor(grpcprom.WithExemplarFromContext(exemplarFromContext)),
			logging.UnaryServerInterceptor(interceptorLogger(log), logging.WithFieldsFromContext(logTraceID)),
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
		),
	)

	errWg.Go(func() error {
		defer stop()
		// run servers
		lis, err := net.Listen("tcp", *grpcAddr)
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
		pb.RegisterCharacterCounterServer(grpcSrv, &server{})
		// Register reflection service on gRPC server.
		reflection.Register(grpcSrv)
		if err := grpcSrv.Serve(lis); err != nil {
			return fmt.Errorf("failed to serve: %w", err)
		}
		return nil
	})

	errWg.Go(func() error {
		<-errCtx.Done()
		log.Info("grpc attempting graceful shutdown")
		grpcSrv.GracefulStop()
		return nil
	})

	// setup http server
	httpSrv := &http.Server{Addr: *httpAddr, ReadTimeout: serverTimeoutSeconds}

	errWg.Go(func() error {
		defer stop()

		log.Info("Server running", "address", *httpAddr)
		m := http.NewServeMux()
		// Create HTTP handler for Prometheus metrics.
		m.Handle("/metrics", promhttp.HandlerFor(
			reg,
			promhttp.HandlerOpts{
				// Opt into OpenMetrics e.g. to support exemplars.
				EnableOpenMetrics: true,
			},
		))
		httpSrv.Handler = m
		log.Info("starting HTTP server", "addr", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("the server failed with error: %w", err)
		}
		return nil
	})

	errWg.Go(func() error {
		<-errCtx.Done()
		log.Info("http attempting graceful shutdown")
		// https://gist.github.com/s8508235/bc248d046d5001d5cae46cc39066cdf5?permalink_comment_id=4360249#gistcomment-4360249
		if err := httpSrv.Shutdown(context.Background()); err != nil { //nolint:contextcheck
			return fmt.Errorf("could not shutdown server gracefully: %w", err)
		}
		return nil
	})

	err = errWg.Wait()
	if !errors.Is(err, context.Canceled) && err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func setupOLTPTracing() (*stdout.Exporter, error) {
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("can not create new exporter, error: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return exporter, nil
}

func setupMetrics() (*grpcprom.ServerMetrics, *prometheus.Registry, func(ctx context.Context) prometheus.Labels) {
	// Setup metrics.
	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)
	reg := prometheus.NewRegistry()
	reg.MustRegister(srvMetrics)
	return srvMetrics, reg, func(ctx context.Context) prometheus.Labels {
		if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
			return prometheus.Labels{"traceID": span.TraceID().String()}
		}
		return nil
	}
}
