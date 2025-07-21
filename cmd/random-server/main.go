package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"alpineworks.io/ootel"
	"github.com/gorilla/mux"
	"github.com/michaelpeterswa/random-server/internal/config"
	"github.com/michaelpeterswa/random-server/internal/handlers"
	"github.com/michaelpeterswa/random-server/internal/logging"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "error"
	}

	slogLevel, err := logging.LogLevelToSlogLevel(logLevel)
	if err != nil {
		log.Fatalf("could not convert log level: %s", err)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	})))
	c, err := config.NewConfig()
	if err != nil {
		slog.Error("could not create config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx := context.Background()

	exporterType := ootel.ExporterTypePrometheus
	if c.Local {
		exporterType = ootel.ExporterTypeOTLPGRPC
	}

	ootelClient := ootel.NewOotelClient(
		ootel.WithMetricConfig(
			ootel.NewMetricConfig(
				c.MetricsEnabled,
				exporterType,
				c.MetricsPort,
			),
		),
		ootel.WithTraceConfig(
			ootel.NewTraceConfig(
				c.TracingEnabled,
				c.TracingSampleRate,
				c.TracingService,
				c.TracingVersion,
			),
		),
	)

	shutdown, err := ootelClient.Init(ctx)
	if err != nil {
		slog.Error("could not create ootel client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(5 * time.Second))
	if err != nil {
		slog.Error("could not create runtime metrics", slog.String("error", err.Error()))
		os.Exit(1)
	}

	err = host.Start()
	if err != nil {
		slog.Error("could not create host metrics", slog.String("error", err.Error()))
		os.Exit(1)
	}

	defer func() {
		_ = shutdown(ctx)
	}()

	handlersClient := handlers.NewHandlersClient(c)

	r := mux.NewRouter()
	r.Use(handlersClient.LoggingMiddleware)
	r.Use(otelmux.Middleware("random-server", otelmux.WithMetricAttributesFn(func(r *http.Request) []attribute.KeyValue {
		return []attribute.KeyValue{
			attribute.String("http.path", r.URL.Path),
		}
	})))
	r.PathPrefix("/").HandlerFunc(handlersClient.CatchAllHandler)

	_ = http.ListenAndServe(":8080", r) //nolint:gosec // Ignoring G114: Use of net/http serve function that has no support for setting timeouts.
}
