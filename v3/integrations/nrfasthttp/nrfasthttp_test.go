package nrfasthttp

import (
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"

	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

type mockFastHTTPClient struct {
	err error
}

func (m *mockFastHTTPClient) Do(req *fasthttp.Request, resp *fasthttp.Response) error {

	if m.err != nil {
		return m.err
	}
	return nil
}

func TestFastHTTPWrapperResponseHeader(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("X-Test-Header", "test")
	wrapper := fasthttpWrapperResponse{ctx: ctx}
	hdrs := wrapper.Header()
	assert.Equal(t, "test", hdrs.Get("X-Test-Header"))
}

func TestFastHTTPWrapperResponseWrite(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	wrapper := fasthttpWrapperResponse{ctx: ctx}

	w, err := wrapper.Write([]byte("Hello World!"))
	assert.Nil(t, err)
	assert.Equal(t, 12, w)
	assert.Equal(t, "Hello World!", string(ctx.Response.Body()))
}

func TestFastHTTPWrapperResponseWriteHeader(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	wrapper := fasthttpWrapperResponse{ctx: ctx}
	wrapper.WriteHeader(http.StatusForbidden)
	assert.Equal(t, http.StatusForbidden, ctx.Response.StatusCode())
}

func TestGetTransaction(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	txn := &newrelic.Transaction{}
	ctx.SetUserValue("transaction", txn)
	assert.Equal(t, txn, GetTransaction(ctx))
}

func TestNRHandler(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	original := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("Hello World!")
	}
	nrhandler := NRHandler(app.Application, original)
	ctx := &fasthttp.RequestCtx{}
	nrhandler(ctx)
	assert.Equal(t, "Hello World!", string(ctx.Response.Body()))
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction"},
		{Name: "Apdex/Go/"},
		{Name: "Custom/fasthttp-set-response"},
		{Name: "WebTransactionTotalTime/Go/"},
		{Name: "Custom/fasthttp-set-request"},
		{Name: "WebTransaction/Go/"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb"},
		{Name: "WebTransactionTotalTime"},
		{Name: "Apdex"},
		{Name: "Custom/fasthttp-set-request", Scope: "WebTransaction/Go/", Forced: false, Data: nil},
		{Name: "Custom/fasthttp-set-response", Scope: "WebTransaction/Go/", Forced: false, Data: nil},
		{Name: "HttpDispatcher"},
	})
}

func TestDo(t *testing.T) {
	client := &mockFastHTTPClient{}
	txn := &newrelic.Transaction{}
	req := &fasthttp.Request{}
	resp := &fasthttp.Response{}

	// check for no error
	err := Do(client, txn, req, resp)
	assert.NoError(t, err)

	// check for error
	client.err = errors.New("ahh!!")
	err = Do(client, txn, req, resp)

	assert.Error(t, err)
}
