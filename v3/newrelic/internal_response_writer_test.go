// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

type rwNoExtraMethods struct {
	hijackCalled      bool
	readFromCalled    bool
	flushCalled       bool
	closeNotifyCalled bool
}

type rwSetWriteDeadline struct{ rwNoExtraMethods }
type rwTwoExtraMethods struct{ rwNoExtraMethods }
type rwAllExtraMethods struct{ rwTwoExtraMethods }

func (rw *rwSetWriteDeadline) SetWriteDeadline(time time.Time) error {
	return nil
}

func (rw *rwAllExtraMethods) CloseNotify() <-chan bool {
	rw.closeNotifyCalled = true
	return nil
}
func (rw *rwAllExtraMethods) ReadFrom(r io.Reader) (int64, error) {
	rw.readFromCalled = true
	return 0, nil
}

func (rw *rwNoExtraMethods) Header() http.Header        { return nil }
func (rw *rwNoExtraMethods) Write([]byte) (int, error)  { return 0, nil }
func (rw *rwNoExtraMethods) WriteHeader(statusCode int) {}

func (rw *rwTwoExtraMethods) Flush() {
	rw.flushCalled = true
}
func (rw *rwTwoExtraMethods) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	rw.hijackCalled = true
	return nil, nil, nil
}

func TestReplacementResponseWriterUnwrap(t *testing.T) {
	original := &rwNoExtraMethods{}
	rw := &replacementResponseWriter{original: original}
	if got := rw.Unwrap(); got != original {
		t.Fatalf("Unwrap returned unexpected response writer: got %T, want %T", got, original)
	}
}

func TestReplacementResponseWriterUnwrapController(t *testing.T) {
	// Create a mock response writer
	original := &rwNoExtraMethods{}
	rw := &replacementResponseWriter{original: original}

	upgraded := upgradeResponseWriter(rw)

	unwrapper, ok := upgraded.(responseWriterUnwrapper)
	if !ok {
		t.Fatal("upgraded response writer does not expose Unwrap() method")
	}

	if got := unwrapper.Unwrap(); got != original {
		t.Errorf("Unwrap() returned wrong writer: got %v, want %v", got, original)
	}
}

func TestReplacementResponseWriterUnwrapSetWriteDeadline(t *testing.T) {
	// Create a mock response writer
	original := &rwSetWriteDeadline{}
	rw := &replacementResponseWriter{original: original}

	upgraded := upgradeResponseWriter(rw)

	unwrapper, ok := upgraded.(responseWriterUnwrapper)
	if !ok {
		t.Fatal("upgraded response writer does not expose Unwrap() method")
	}

	if got := unwrapper.Unwrap(); got != original {
		t.Errorf("Unwrap() returned wrong writer: got %v, want %v", got, original)
	}

	controller := http.NewResponseController(upgraded)
	if controller == nil {
		t.Fatal("http.NewResponseController returned nil")
	}
	var time time.Time
	err := controller.SetWriteDeadline(time)
	if err != nil {
		t.Fatalf("http.ResponseController.SetWriteDeadline failed with: %v", err)
	}
}

func TestTransactionAllExtraMethods(t *testing.T) {
	app := testApp(nil, nil, t)
	rw := &rwAllExtraMethods{}
	txn := app.StartTransaction("hello")
	w := txn.SetWebResponse(rw)
	if v, ok := w.(http.CloseNotifier); ok {
		v.CloseNotify()
	}
	if v, ok := w.(http.Flusher); ok {
		v.Flush()
	}
	if v, ok := w.(http.Hijacker); ok {
		v.Hijack()
	}
	if v, ok := w.(io.ReaderFrom); ok {
		v.ReadFrom(nil)
	}
	if !rw.hijackCalled ||
		!rw.readFromCalled ||
		!rw.flushCalled ||
		!rw.closeNotifyCalled {
		t.Error("wrong methods called", rw)
	}
}

func TestTransactionNoExtraMethods(t *testing.T) {
	app := testApp(nil, nil, t)
	rw := &rwNoExtraMethods{}
	txn := app.StartTransaction("hello")
	w := txn.SetWebResponse(rw)
	if _, ok := w.(http.CloseNotifier); ok {
		t.Error("unexpected CloseNotifier method")
	}
	if _, ok := w.(http.Flusher); ok {
		t.Error("unexpected Flusher method")
	}
	if _, ok := w.(http.Hijacker); ok {
		t.Error("unexpected Hijacker method")
	}
	if _, ok := w.(io.ReaderFrom); ok {
		t.Error("unexpected ReaderFrom method")
	}
}

func TestTransactionTwoExtraMethods(t *testing.T) {
	app := testApp(nil, nil, t)
	rw := &rwTwoExtraMethods{}
	txn := app.StartTransaction("hello")
	w := txn.SetWebResponse(rw)
	if _, ok := w.(http.CloseNotifier); ok {
		t.Error("unexpected CloseNotifier method")
	}
	if v, ok := w.(http.Flusher); ok {
		v.Flush()
	}
	if v, ok := w.(http.Hijacker); ok {
		v.Hijack()
	}
	if _, ok := w.(io.ReaderFrom); ok {
		t.Error("unexpected ReaderFrom method")
	}
	if !rw.hijackCalled ||
		rw.readFromCalled ||
		!rw.flushCalled ||
		rw.closeNotifyCalled {
		t.Error("wrong methods called", rw)
	}
}
