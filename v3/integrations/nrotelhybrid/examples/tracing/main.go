package main

import (
	"context"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	// for pretty print to console
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return err
	}

	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))
	shutdown := func(ctx context.Context) error {
		err := tp.Shutdown(ctx)
		return err
	}
	defer shutdown(context.Background())

	otel.SetTracerProvider(tp)

	srv := &http.Server{
		Addr:         ":8080",
		BaseContext:  func(net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHttpHandler(),
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

func newHttpHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/routeone/", routeOne)
	mux.HandleFunc("routetwo/", routeTwo)

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

func leafCall(ctx context.Context) {
	_, span := otel.Tracer("nrotel-example").Start(ctx, "leaf-call")
	defer span.End()
	n := rand.IntN(500)
	//simulate some work
	time.Sleep(time.Duration(n) * time.Millisecond)
}
