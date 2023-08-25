package nrfasthttp

import (
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/valyala/fasthttp"
)

type FastHTTPClient interface {
	Do(req *fasthttp.Request, resp *fasthttp.Response) error
}

func Do(client FastHTTPClient, txn *newrelic.Transaction, req *fasthttp.Request, res *fasthttp.Response) error {
	seg := txn.StartSegment("fasthttp-do")
	err := client.Do(req, res)
	if err != nil {
		txn.NoticeError(err)
	}
	seg.End()

	return err
}
