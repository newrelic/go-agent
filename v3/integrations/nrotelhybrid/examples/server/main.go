package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "github.com/lib/pq"

	"github.com/newrelic/go-agent/v3/integrations/nrotelhybrid"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// This example exercises span kinds Server, Client, and Internal against an
// HTTP server backed by Postgres. Producer/Consumer are covered separately
// in examples/messaging.
//
// Root transactions:
//   - Server: every HTTP request, via otelhttp (no active transaction on entry).
//   - Client: a span with a fabricated remote parent context — Client spans
//     never become roots from a local parent, only from a remote one.
//
// Segments (nested within a Server-root transaction):
//   - Server: an explicit nested SpanKindServer span.
//   - Client (External): an outbound HTTP call.
//   - Client (Datastore): a Postgres query via otelsql.
//   - Internal: an explicit nested SpanKindInternal span.

type deps struct {
	db *sql.DB
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Hybrid Example - Server"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Shutdown(10 * time.Second)

	processor := nrotelhybrid.NewHybridProcessor(app)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return err
	}

	tp := trace.NewTracerProvider(trace.WithSyncer(exporter), trace.WithSpanProcessor(processor))
	defer tp.Shutdown(context.Background())

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	db, err := otelsql.Open("postgres", "host=localhost port=5432 user=postgres dbname=postgres password=docker sslmode=disable", otelsql.WithAttributes(
		semconv.DBSystemNamePostgreSQL),
		otelsql.WithDBName("secondTestDB"),
		otelsql.WithTracerProvider(otel.GetTracerProvider()),
	)
	if err != nil {
		return err
	}
	defer db.Close()

	d := &deps{db: db}

	srv := &http.Server{
		Addr:         ":8080",
		BaseContext:  func(net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHTTPHandler(d),
	}
	srvErr := make(chan error, 1)
	go func() {
		log.Println("Running HTTP server...")
		srvErr <- srv.ListenAndServe()
	}()

	select {
	case err = <-srvErr:
		return err
	case <-ctx.Done():
		stop()
	}
	err = srv.Shutdown(context.Background())
	return err
}

func newHTTPHandler(d *deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/serverroot/", serverRoot)
	mux.HandleFunc("/clientexternal/", clientExternal)
	mux.HandleFunc("/clientdatastore/", d.clientDatastore)
	mux.HandleFunc("/clientroot/", clientRootRemoteParent)

	// Add HTTP instrumentation for the whole server; this is what makes
	// each incoming request a SpanKindServer root transaction.
	return otelhttp.NewHandler(mux, "/")
}

// serverRoot is itself a Server-kind transaction root (via otelhttp).
// It also nests a Server-kind segment and an Internal-kind segment.
func serverRoot(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("nrotel-example")

	_, serverSpan := tracer.Start(r.Context(), "server-segment", oteltrace.WithSpanKind(oteltrace.SpanKindServer))
	serverSpan.End()

	_, internalSpan := tracer.Start(r.Context(), "internal-segment", oteltrace.WithSpanKind(oteltrace.SpanKindInternal))
	internalSpan.End()
}

// clientExternal is a Server-root transaction that makes an outbound HTTP
// call, producing a Client/External segment.
func clientExternal(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	req, err := http.NewRequestWithContext(r.Context(), "GET", "http://localhost:8080/serverroot/", nil)
	if err != nil {
		log.Println(err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	resp.Body.Close()
}

// clientDatastore is a Server-root transaction that queries Postgres,
// producing a Client/Datastore segment.
func (d *deps) clientDatastore(w http.ResponseWriter, r *http.Request) {
	d.db.QueryRowContext(r.Context(), "SELECT count(*) FROM pg_catalog.pg_tables")
}

// clientRootRemoteParent demonstrates that a Client-kind span only becomes
// a transaction root when it has a remote parent - locally parented Client
// spans always become segments. It fabricates a remote parent span context
// (as if this process received a trace header from an upstream caller) in a
// fresh trace, independent of the Server-root transaction otelhttp already
// started for this request.
func clientRootRemoteParent(w http.ResponseWriter, r *http.Request) {
	remoteParent := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    oteltrace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     oteltrace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})
	ctx := oteltrace.ContextWithRemoteSpanContext(context.Background(), remoteParent)

	_, span := otel.Tracer("nrotel-example").Start(ctx, "client-root", oteltrace.WithSpanKind(oteltrace.SpanKindClient))
	defer span.End()
}