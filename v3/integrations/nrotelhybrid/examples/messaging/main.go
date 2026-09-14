package main

import (
	"context"
	"log"
	"net"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/newrelic/go-agent/v3/integrations/nrotelhybrid/examples"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	requestQueue = "hybrid-example-requests"
	replyQueue   = "hybrid-example-replies"
)

type deps struct {
	amqpConn *amqp.Connection
	amqpCh   *amqp.Channel
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	app, ctx, stop, cleanup, err := examples.NewHybridApp("Hybrid Example - Messaging")
	if err != nil {
		return err
	}
	defer app.Shutdown(10 * time.Second)
	defer stop()
	defer cleanup()

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

	for _, q := range []string{requestQueue, replyQueue} {
		if _, err := amqpCh.QueueDeclare(q, false, false, false, false, nil); err != nil {
			return err
		}
	}

	d := &deps{amqpConn: amqpConn, amqpCh: amqpCh}

	if err := d.publish(ctx, requestQueue, []byte("request")); err != nil {
		return err
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			d.consumeThenReply(ctx)
			d.publishThenAwaitReply(ctx)
		}
	}
}

func (d *deps) consumeThenReply(ctx context.Context) {
	tracer := otel.Tracer("nrotel-example")
	ctx, rootSpan := tracer.Start(ctx, "consume-request", oteltrace.WithSpanKind(oteltrace.SpanKindConsumer), oteltrace.WithAttributes(
		attribute.String(string(semconv.MessagingDestinationNameKey), requestQueue),
		attribute.String(string(semconv.ServerAddressKey), d.amqpConn.RemoteAddr().String()),
		attribute.String(string(semconv.ServerPortKey), strconv.Itoa(d.amqpConn.RemoteAddr().(*net.TCPAddr).Port)),
	))
	defer rootSpan.End()

	msg, ok, err := d.amqpCh.Get(requestQueue, true)
	if err != nil {
		log.Println(err)
		return
	}
	if !ok {
		return
	}
	log.Printf("consumed request: %s", msg.Body)

	if err := d.publishSegment(ctx, replyQueue, []byte("reply")); err != nil {
		log.Println(err)
	}
}

func (d *deps) publishThenAwaitReply(ctx context.Context) {
	remoteParent := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    oteltrace.TraceID{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
		SpanID:     oteltrace.SpanID{8, 7, 6, 5, 4, 3, 2, 1},
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})
	ctx = oteltrace.ContextWithRemoteSpanContext(context.Background(), remoteParent)

	tracer := otel.Tracer("nrotel-example")
	ctx, rootSpan := tracer.Start(ctx, "publish-request", oteltrace.WithSpanKind(oteltrace.SpanKindProducer), oteltrace.WithAttributes(
		attribute.String(string(semconv.MessagingDestinationNameKey), requestQueue),
		attribute.String(string(semconv.ServerAddressKey), d.amqpConn.RemoteAddr().String()),
		attribute.String(string(semconv.ServerPortKey), strconv.Itoa(d.amqpConn.RemoteAddr().(*net.TCPAddr).Port)),
	))
	defer rootSpan.End()

	if err := d.amqpCh.PublishWithContext(ctx, "", requestQueue, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("request"),
	}); err != nil {
		log.Println(err)
		return
	}

	_, segSpan := tracer.Start(ctx, "await-reply", oteltrace.WithSpanKind(oteltrace.SpanKindConsumer), oteltrace.WithAttributes(
		attribute.String(string(semconv.MessagingDestinationNameKey), replyQueue),
		attribute.String(string(semconv.ServerAddressKey), d.amqpConn.RemoteAddr().String()),
		attribute.String(string(semconv.ServerPortKey), strconv.Itoa(d.amqpConn.RemoteAddr().(*net.TCPAddr).Port)),
	))
	defer segSpan.End()

	msg, ok, err := d.amqpCh.Get(replyQueue, true)
	if err != nil {
		log.Println(err)
		return
	}
	if ok {
		log.Printf("consumed reply: %s", msg.Body)
	}
}

func (d *deps) publish(ctx context.Context, queue string, body []byte) error {
	return d.amqpCh.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        body,
	})
}

func (d *deps) publishSegment(ctx context.Context, queue string, body []byte) error {
	tracer := otel.Tracer("nrotel-example")
	ctx, span := tracer.Start(ctx, "publish-reply", oteltrace.WithSpanKind(oteltrace.SpanKindProducer), oteltrace.WithAttributes(
		attribute.String(string(semconv.MessagingDestinationNameKey), queue),
		attribute.String(string(semconv.ServerAddressKey), d.amqpConn.RemoteAddr().String()),
		attribute.String(string(semconv.ServerPortKey), strconv.Itoa(d.amqpConn.RemoteAddr().(*net.TCPAddr).Port)),
	))
	defer span.End()

	return d.amqpCh.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        body,
	})
}
