package newrelic

import (
	"net/http"

	"github.com/newrelic/go-agent/datastore"
)

const noTransactionToken Token = 0

var _ Application = &NullApplication{}

// NullApplication is a null-object version of newrelic.Application.
// Useful in tests.
type NullApplication struct {
}

func (a *NullApplication) StartTransaction(name string, w http.ResponseWriter, r *http.Request) Transaction {
	return &NullTransaction{w}
}

func (a *NullApplication) RecordCustomEvent(eventType string, params map[string]interface{}) error {
	return nil
}

type NullTransaction struct {
	http.ResponseWriter
}

func (t *NullTransaction) End() error {
	return nil
}

func (t *NullTransaction) Ignore() error {
	return nil
}

func (t *NullTransaction) SetName(name string) error {
	return nil
}

func (t *NullTransaction) NoticeError(err error) error {
	return nil
}

func (t *NullTransaction) AddAttribute(key string, value interface{}) error {
	return nil
}

func (t *NullTransaction) StartSegment() Token {
	return noTransactionToken
}

func (t *NullTransaction) EndSegment(token Token, name string) {
}

func (t *NullTransaction) EndExternal(token Token, url string) {
}

func (t *NullTransaction) EndDatastore(token Token, seg datastore.Segment) {
}

func (t *NullTransaction) PrepareRequest(token Token, r *http.Request) {
}

func (t *NullTransaction) EndRequest(token Token, r *http.Request, resp *http.Response) {
}
