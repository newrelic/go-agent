package main

import (
	"context"
	"database/sql"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/newrelic/go-agent/v3/integrations/nrotelhybrid"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

type deps struct {
	db      *sql.DB
	amqpCh  *amqp.Channel
	amqpMux sync.Mutex
}

func main() {
	// for pretty print to console
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Hybrid Example"),
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
	shutdown := func(ctx context.Context) error {
		err := tp.Shutdown(ctx)
		return err
	}
	defer shutdown(context.Background())

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	db, err := otelsql.Open("postgres", "host=localhost port=5432 user=postgres dbname=postgres password=docker sslmode=disable", otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL),
		otelsql.WithDBName("secondTestDB"),
		otelsql.WithTracerProvider(otel.GetTracerProvider()),
	)
	if err != nil {
		return err
	}
	defer db.Close()

	amqpConn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return err
	}
	defer amqpConn.Close()

	amqpCh, err := amqpConn.Channel()
	if err != nil {
		return err
	}
	defer amqpCh.Close()

	if _, err := amqpCh.QueueDeclare("route-six-queue", false, false, false, false, nil); err != nil {
		return err
	}

	d := &deps{db: db, amqpCh: amqpCh}

	srv := &http.Server{
		Addr:         ":8080",
		BaseContext:  func(net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHttpHandler(d),
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

func newHttpHandler(d *deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/routeone/", routeOne)
	mux.HandleFunc("/routetwo/", routeTwo)
	mux.HandleFunc("/routethree/", routeThree)
	mux.HandleFunc("/routefour/", routeFour)
	mux.HandleFunc("/routefive/", d.routeFive)
	mux.HandleFunc("/routesix/", d.routeSix)

	// Add HTTP instrumentation for the whole server.
	handler := otelhttp.NewHandler(mux, "/")

	return handler
}

func routeOne(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("nrotel-example")
	ctx, span := tracer.Start(r.Context(), "route-one")
	defer span.End()
	nestedRouteOne(ctx)
	leafCall(ctx)

}

func routeTwo(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("nrotel-example")
	ctx, span := tracer.Start(r.Context(), "route-two")
	defer span.End()
	nestedRouteTwo(ctx)
}

func routeThree(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("nrotel-example")
	ctx, span := tracer.Start(r.Context(), "route-three")
	defer span.End()

	nestedRouteThree(ctx)
}

func routeFour(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("nrotel-example")
	_, span := tracer.Start(r.Context(), "route-four")
	defer span.End()
	n := rand.IntN(500)
	time.Sleep(time.Duration(n) * time.Millisecond)
}

func (d *deps) routeFive(w http.ResponseWriter, r *http.Request) {
	d.db.QueryRowContext(r.Context(), "SELECT count(*) FROM pg_catalog.pg_tables")
}

func (d *deps) routeSix(w http.ResponseWriter, r *http.Request) {
	d.amqpMux.Lock()
	defer d.amqpMux.Unlock()

	err := d.amqpCh.PublishWithContext(r.Context(), "", "route-six-queue", false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("hello from routeSix"),
	})
	if err != nil {
		log.Println(err)
		return
	}
}

func nestedRouteOne(ctx context.Context) {
	_, span := otel.Tracer("nrotel-example").Start(ctx, "nested-route-two")
	defer span.End()
	leafCall(ctx)
}

func nestedRouteTwo(ctx context.Context) {
	_, span := otel.Tracer("nrotel-example").Start(ctx, "nested-route-two")
	defer span.End()
	leafCall(ctx)
}

func nestedRouteThree(ctx context.Context) {
	_, span := otel.Tracer("nrotel-example").Start(ctx, "nested-route-two")
	defer span.End()
	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/routefour/", nil)
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

func leafCall(ctx context.Context) {
	_, span := otel.Tracer("nrotel-example").Start(ctx, "leaf-call")
	defer span.End()
	n := rand.IntN(500)
	//simulate some work
	time.Sleep(time.Duration(n) * time.Millisecond)
}
