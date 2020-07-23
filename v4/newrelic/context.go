// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"context"
	"net/http"
)

// NewContext returns a new context.Context that carries the provided
// transaction.
func NewContext(ctx context.Context, txn *Transaction) context.Context {
	return ctx
}

// FromContext returns the Transaction from the context if present, and nil
// otherwise.
func FromContext(ctx context.Context) *Transaction {
	return nil
}

// RequestWithTransactionContext adds the Transaction to the request's context.
func RequestWithTransactionContext(req *http.Request, txn *Transaction) *http.Request {
	return req
}
