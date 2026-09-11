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

	// Add HTTP instrumentation for the whole server; this is what makes
	// each incoming request a SpanKindServer root transaction.
	return otelhttp.NewHandler(mux, "/", otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
		return r.URL.Path
	}))
}

func serverRoot(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("nrotel-example")

	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	_, serverSpan := tracer.Start(ctx, "server-segment", oteltrace.WithSpanKind(oteltrace.SpanKindServer))
	serverSpan.End()

	_, internalSpan := tracer.Start(ctx, "internal-segment", oteltrace.WithSpanKind(oteltrace.SpanKindInternal))
	internalSpan.End()
}

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

func (d *deps) clientDatastore(w http.ResponseWriter, r *http.Request) {
	d.db.QueryRowContext(r.Context(), "SELECT count(*) FROM pg_catalog.pg_tables")
}
