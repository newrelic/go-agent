// Server Example
//
// This example starts a server that demonstrates SpanKind INTERNAL, SERVER, and CLIENT
// Spans.
//
// # Run a local Postgres instance in Docker:
//
//	docker run -d --name postgres -e POSTGRES_PASSWORD=<password> -p 5432:5432 postgres
//
// # Set NEW_RELIC_LICENSE_KEY then run with `go run main.go`.
//
// # Generate Traffic
//
// curl "localhost:8080/serverroot/"
// curl "localhost:8080/clientexternal/"
// curl "localhost:8080/clientdatastore/"
//
// # Generate Traffic with Remote Parent
//
// curl -H "traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" \                                                                                                                                                      ─╯
// "http://localhost:8080/serverroot/"

package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"time"

	_ "github.com/lib/pq"

	"github.com/newrelic/go-agent/v3/integrations/nrotelhybrid/examples"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type deps struct {
	db *sql.DB
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	// Initialize New Relic Application
	app, ctx, stop, cleanup, err := examples.NewHybridApp("Hybrid Example - Server")
	if err != nil {
		return err
	}
	defer app.Shutdown(10 * time.Second)
	defer cleanup()

	// Initialize DB connection (change password)
	db, err := otelsql.Open("postgres", "host=localhost port=5432 user=postgres dbname=postgres password=<password> sslmode=disable", otelsql.WithAttributes(
		semconv.DBSystemNamePostgreSQL),
		otelsql.WithDBName("postrgres"),
		otelsql.WithTracerProvider(otel.GetTracerProvider()),
	)
	if err != nil {
		return err
	}
	defer db.Close()

	d := &deps{db: db}

	// Initialize and run Server
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

	// Add HTTP instrumentation for the whole server; this is what makes
	// each incoming request a SpanKindServer root transaction.
	return otelhttp.NewHandler(mux, "/", otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
		return r.URL.Path
	}))
}

// serverRoot Extracts headers to check for a remote parent. It also
// begins two child spans with different SpanKinds
func serverRoot(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("nrotel-example")

	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	_, serverSpan := tracer.Start(ctx, "server-segment", oteltrace.WithSpanKind(oteltrace.SpanKindServer))
	serverSpan.End()

	_, internalSpan := tracer.Start(ctx, "internal-segment", oteltrace.WithSpanKind(oteltrace.SpanKindInternal))
	internalSpan.End()
}

// clientExternal makes an "external" request
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

// clientDatastore makes a simple query
func (d *deps) clientDatastore(w http.ResponseWriter, r *http.Request) {
	d.db.QueryRowContext(r.Context(), "SELECT count(*) FROM pg_catalog.pg_tables")
}
