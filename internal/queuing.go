package internal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	xRequestStart = "X-Request-Start"
	xQueueStart   = "X-Queue-Start"
	// Note that this is in canonical MIME header capitalization (consistent with http.Header).
	xNewrelicTimestampPrefix    = "X-Newrelic-Timestamp-"
	xNewrelicTimestampPrefixLen = len(xNewrelicTimestampPrefix)
	unknownIntermediary         = "Unknown"
)

var (
	earliestAcceptableSeconds = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	latestAcceptableSeconds   = time.Date(2050, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
)

func checkQueueTimeSeconds(secondsFloat float64) time.Time {
	seconds := int64(secondsFloat)
	nanos := int64((secondsFloat - float64(seconds)) * (1000.0 * 1000.0 * 1000.0))
	if seconds > earliestAcceptableSeconds && seconds < latestAcceptableSeconds {
		return time.Unix(seconds, nanos)
	}
	return time.Time{}
}

func parseQueueTime(s string) time.Time {
	f, err := strconv.ParseFloat(s, 64)
	if nil != err {
		return time.Time{}
	}
	if f <= 0 {
		return time.Time{}
	}

	// try microseconds
	if t := checkQueueTimeSeconds(f / (1000.0 * 1000.0)); !t.IsZero() {
		return t
	}
	// try milliseconds
	if t := checkQueueTimeSeconds(f / (1000.0)); !t.IsZero() {
		return t
	}
	// try seconds
	if t := checkQueueTimeSeconds(f); !t.IsZero() {
		return t
	}
	return time.Time{}
}

// Proxies contains information about queue time derived from inbound http headers.
type Proxies struct {
	durations map[string]time.Duration
	max       time.Duration
}

func (p Proxies) hasQueueing() bool {
	return len(p.durations) > 0
}

func (p *Proxies) addDuration(name string, d time.Duration) {
	if d > p.max {
		p.max = d
	}
	if nil == p.durations {
		p.durations = make(map[string]time.Duration)
	}
	p.durations[name] = d
}

// asAttributes facilitates testing.
func (p Proxies) asAttributes() map[string]interface{} {
	buf := bytes.NewBuffer(make([]byte, 0, 64*len(p.durations)))
	w := jsonFieldsWriter{buf: buf}
	buf.WriteByte('{')
	p.createIntrinsics(&w)
	buf.WriteByte('}')

	var attr map[string]interface{}
	json.Unmarshal(buf.Bytes(), &attr)
	return attr
}

func (p Proxies) createIntrinsics(w *jsonFieldsWriter) {
	if p.hasQueueing() {
		w.floatField("queueDuration", p.max.Seconds())
		for name, d := range p.durations {
			w.floatField("caller.transportDuration."+name, d.Seconds())
		}
	}
}

func (p Proxies) createMetrics(metrics *metricTable, caller payloadCaller, isWeb bool) {
	if p.hasQueueing() {
		metrics.addDuration(queueMetric, "", p.max, p.max, forced)
		for name, d := range p.durations {
			m := intermediaryMetric(caller, name)
			metrics.addDuration(m.all, "", d, d, unforced)
			metrics.addDuration(m.webOrOther(isWeb), "", d, d, unforced)
		}
	}
}

func (p *Proxies) addRaw(name string, val string, txnStart time.Time) {
	// The "t=" prefix might be being used by customers for the legacy queuing headers
	// ("X-Request-Start" and "X-Queue-Start").
	val = strings.TrimPrefix(val, "t=")
	tm := parseQueueTime(val)
	d := time.Duration(0)
	if !tm.IsZero() && !tm.After(txnStart) {
		d = txnStart.Sub(tm)
	}
	p.addDuration(name, d)
}

// NewProxies creates a new Proxies.
func NewProxies(hdr http.Header, txnStart time.Time) (p Proxies) {
	for key, vals := range hdr {
		if len(vals) == 0 {
			continue
		}
		if strings.HasPrefix(key, xNewrelicTimestampPrefix) {
			name := key[xNewrelicTimestampPrefixLen:]
			p.addRaw(name, vals[0], txnStart)
		}
	}
	v := hdr.Get(xQueueStart)
	if "" == v {
		v = hdr.Get(xRequestStart)
	}
	if "" != v {
		p.addRaw(unknownIntermediary, v, txnStart)
	}
	return
}
