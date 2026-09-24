package nrrueidis

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidishook"
)

type contextKeyType struct{}

var (
	segmentContextKey = contextKeyType(struct{}{})
)

func multiOperation(typ string, cmds []rueidis.Completed) string {
	operations := make([]string, 0, len(cmds))
	for _, cmd := range cmds {
		// XXX: Is it always true that there is only one command here?
		operations = append(operations, cmd.Commands()[0])
	}
	return typ + ":" + strings.Join(operations, ",")
}

type hook struct {
	segment newrelic.DatastoreSegment
}

func (h hook) before(ctx context.Context, operation string) context.Context {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return ctx
	}
	s := h.segment
	s.Operation = operation
	s.StartTime = txn.StartSegmentNow()
	ctx = context.WithValue(ctx, segmentContextKey, &s)
	return ctx
}

func (h hook) after(ctx context.Context) {
	if segment, ok := ctx.Value(segmentContextKey).(interface{ End() }); ok {
		segment.End()
	}
}

func (h *hook) Do(client rueidis.Client, ctx context.Context, cmd rueidis.Completed) (resp rueidis.RedisResult) {
	// XXX: Is it always true that there is only one command here?
	ctx = h.before(ctx, cmd.Commands()[0])
	res := client.Do(ctx, cmd)
	h.after(ctx)
	return res
}

func (h *hook) DoMulti(client rueidis.Client, ctx context.Context, multi ...rueidis.Completed) (resps []rueidis.RedisResult) {
	ctx = h.before(ctx, multiOperation("pipeline", multi))
	res := client.DoMulti(ctx, multi...)
	h.after(ctx)
	return res
}

func (h *hook) DoCache(client rueidis.Client, ctx context.Context, cmd rueidis.Cacheable, ttl time.Duration) (resp rueidis.RedisResult) {
	// XXX: Is it always true that there is only one command here?
	ctx = h.before(ctx, cmd.Commands()[0])
	h.segment.AddAttribute("cachable", true)
	res := client.DoCache(ctx, cmd, ttl)
	h.after(ctx)
	return res
}

func (h *hook) DoMultiCache(client rueidis.Client, ctx context.Context, multi ...rueidis.CacheableTTL) (resps []rueidis.RedisResult) {
	cmds := make([]rueidis.Completed, 0, len(multi))
	for _, m := range multi {
		cmds = append(cmds, rueidis.Completed(m.Cmd))
	}
	ctx = h.before(ctx, multiOperation("pipeline", cmds))
	h.segment.AddAttribute("cachable", true)
	res := client.DoMultiCache(ctx, multi...)
	h.after(ctx)
	return res
}

func (h *hook) Receive(client rueidis.Client, ctx context.Context, subscribe rueidis.Completed, fn func(msg rueidis.PubSubMessage)) (err error) {
	ctx = h.before(ctx, subscribe.Commands()[0])
	res := client.Receive(ctx, subscribe, fn)
	h.after(ctx)
	return res
}

func (h *hook) DoStream(client rueidis.Client, ctx context.Context, cmd rueidis.Completed) rueidis.RedisResultStream {
	// XXX: Is it always true that there is only one command here?
	// XXX: Should we mark this as "streaming"
	ctx = h.before(ctx, cmd.Commands()[0])
	h.segment.AddAttribute("streaming", true)
	res := client.DoStream(ctx, cmd)
	h.after(ctx)
	return res
}

func (h *hook) DoMultiStream(client rueidis.Client, ctx context.Context, multi ...rueidis.Completed) rueidis.MultiRedisResultStream {
	ctx = h.before(ctx, multiOperation("pipeline", multi))
	h.segment.AddAttribute("streaming", true)
	res := client.DoMultiStream(ctx, multi...)
	h.after(ctx)
	return res
}

var _ rueidishook.Hook = (*hook)(nil)

// nrrueidisHook creates a rueidis hook to instrument Redis calls.  Add it to your
// client, then ensure that all calls contain a context which includes the
// transaction.  The options are optional.  Provide them to get instance metrics
// broken out by host and port.  The hook returned can be used with
// redis.Client, redis.ClusterClient, and redis.Ring.
func nrrueidisHook(opt rueidis.ClientOption) *hook {
	h := &hook{}
	h.segment.Product = newrelic.DatastoreRedis

	if len(opt.InitAddress) == 0 {
		return h
	}

	// Parse the first init address to get host and port.
	// XXX: Does rueidis support unix sockets?
	if host, port, err := net.SplitHostPort(opt.InitAddress[0]); err == nil {
		if host == "" {
			host = "localhost"
		}
		h.segment.Host = host
		h.segment.PortPathOrID = port
	}
	return h
}
