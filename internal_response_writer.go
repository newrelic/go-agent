package newrelic

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

const (
	hasC = 1 << iota // CloseNotifier
	hasF             // Flusher
	hasH             // Hijacker
	hasR             // ReaderFrom
)

type wrap struct{ *txn }
type wrapR struct{ *txn }
type wrapH struct{ *txn }
type wrapHR struct{ *txn }
type wrapF struct{ *txn }
type wrapFR struct{ *txn }
type wrapFH struct{ *txn }
type wrapFHR struct{ *txn }
type wrapC struct{ *txn }
type wrapCR struct{ *txn }
type wrapCH struct{ *txn }
type wrapCHR struct{ *txn }
type wrapCF struct{ *txn }
type wrapCFR struct{ *txn }
type wrapCFH struct{ *txn }
type wrapCFHR struct{ *txn }

// Note that these methods use getWriter() which can return nil.  This is not a
// problem because the transaction will only be upgraded to a type with these
// methods if a response writer is present.

func (x wrapC) CloseNotify() <-chan bool    { return x.getWriter().(http.CloseNotifier).CloseNotify() }
func (x wrapCR) CloseNotify() <-chan bool   { return x.getWriter().(http.CloseNotifier).CloseNotify() }
func (x wrapCH) CloseNotify() <-chan bool   { return x.getWriter().(http.CloseNotifier).CloseNotify() }
func (x wrapCHR) CloseNotify() <-chan bool  { return x.getWriter().(http.CloseNotifier).CloseNotify() }
func (x wrapCF) CloseNotify() <-chan bool   { return x.getWriter().(http.CloseNotifier).CloseNotify() }
func (x wrapCFR) CloseNotify() <-chan bool  { return x.getWriter().(http.CloseNotifier).CloseNotify() }
func (x wrapCFH) CloseNotify() <-chan bool  { return x.getWriter().(http.CloseNotifier).CloseNotify() }
func (x wrapCFHR) CloseNotify() <-chan bool { return x.getWriter().(http.CloseNotifier).CloseNotify() }

func (x wrapF) Flush()    { x.getWriter().(http.Flusher).Flush() }
func (x wrapFR) Flush()   { x.getWriter().(http.Flusher).Flush() }
func (x wrapFH) Flush()   { x.getWriter().(http.Flusher).Flush() }
func (x wrapFHR) Flush()  { x.getWriter().(http.Flusher).Flush() }
func (x wrapCF) Flush()   { x.getWriter().(http.Flusher).Flush() }
func (x wrapCFR) Flush()  { x.getWriter().(http.Flusher).Flush() }
func (x wrapCFH) Flush()  { x.getWriter().(http.Flusher).Flush() }
func (x wrapCFHR) Flush() { x.getWriter().(http.Flusher).Flush() }

func (x wrapH) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return x.getWriter().(http.Hijacker).Hijack()
}
func (x wrapHR) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return x.getWriter().(http.Hijacker).Hijack()
}
func (x wrapFH) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return x.getWriter().(http.Hijacker).Hijack()
}
func (x wrapFHR) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return x.getWriter().(http.Hijacker).Hijack()
}
func (x wrapCH) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return x.getWriter().(http.Hijacker).Hijack()
}
func (x wrapCHR) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return x.getWriter().(http.Hijacker).Hijack()
}
func (x wrapCFH) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return x.getWriter().(http.Hijacker).Hijack()
}
func (x wrapCFHR) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return x.getWriter().(http.Hijacker).Hijack()
}

func (x wrapR) ReadFrom(r io.Reader) (int64, error)  { return x.getWriter().(io.ReaderFrom).ReadFrom(r) }
func (x wrapHR) ReadFrom(r io.Reader) (int64, error) { return x.getWriter().(io.ReaderFrom).ReadFrom(r) }
func (x wrapFR) ReadFrom(r io.Reader) (int64, error) { return x.getWriter().(io.ReaderFrom).ReadFrom(r) }
func (x wrapFHR) ReadFrom(r io.Reader) (int64, error) {
	return x.getWriter().(io.ReaderFrom).ReadFrom(r)
}
func (x wrapCR) ReadFrom(r io.Reader) (int64, error) { return x.getWriter().(io.ReaderFrom).ReadFrom(r) }
func (x wrapCHR) ReadFrom(r io.Reader) (int64, error) {
	return x.getWriter().(io.ReaderFrom).ReadFrom(r)
}
func (x wrapCFR) ReadFrom(r io.Reader) (int64, error) {
	return x.getWriter().(io.ReaderFrom).ReadFrom(r)
}
func (x wrapCFHR) ReadFrom(r io.Reader) (int64, error) {
	return x.getWriter().(io.ReaderFrom).ReadFrom(r)
}

func upgradeTxn(txn *txn) Transaction {
	// Note that txn.getWriter() is not used here.  The transaction is
	// locked (or under construction) when this function is used.
	x := 0
	if _, ok := txn.writer.(http.CloseNotifier); ok {
		x |= hasC
	}
	if _, ok := txn.writer.(http.Flusher); ok {
		x |= hasF
	}
	if _, ok := txn.writer.(http.Hijacker); ok {
		x |= hasH
	}
	if _, ok := txn.writer.(io.ReaderFrom); ok {
		x |= hasR
	}

	switch x {
	default:
		// Wrap the transaction even when there are no methods needed to
		// ensure consistent error stack trace depth.
		return wrap{txn}
	case hasR:
		return wrapR{txn}
	case hasH:
		return wrapH{txn}
	case hasH | hasR:
		return wrapHR{txn}
	case hasF:
		return wrapF{txn}
	case hasF | hasR:
		return wrapFR{txn}
	case hasF | hasH:
		return wrapFH{txn}
	case hasF | hasH | hasR:
		return wrapFHR{txn}
	case hasC:
		return wrapC{txn}
	case hasC | hasR:
		return wrapCR{txn}
	case hasC | hasH:
		return wrapCH{txn}
	case hasC | hasH | hasR:
		return wrapCHR{txn}
	case hasC | hasF:
		return wrapCF{txn}
	case hasC | hasF | hasR:
		return wrapCFR{txn}
	case hasC | hasF | hasH:
		return wrapCFH{txn}
	case hasC | hasF | hasH | hasR:
		return wrapCFHR{txn}
	}
}
