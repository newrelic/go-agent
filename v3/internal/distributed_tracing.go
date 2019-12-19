package internal

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type distTraceVersion [2]int

func (v distTraceVersion) major() int { return v[0] }
func (v distTraceVersion) minor() int { return v[1] }

const (
	// CallerTypeApp is the Type field's value for outbound payloads.
	CallerTypeApp = "App"
	// CallerTypeBrowser is the Type field's value for browser payloads
	CallerTypeBrowser = "Browser"
	// CallerTypeMobile is the Type field's value for mobile payloads
	CallerTypeMobile = "Mobile"

	// DistributedTraceNewRelicHeader is the header used by New Relic agents
	// for automatic trace payload instrumentation.
	DistributedTraceNewRelicHeader = "Newrelic"
	// DistributedTraceW3CTraceStateHeader is one of two headers used by W3C
	// trace context
	DistributedTraceW3CTraceStateHeader = "tracestate"
	// DistributedTraceW3CTraceParentHeader is one of two headers used by W3C
	// trace context
	DistributedTraceW3CTraceParentHeader = "traceparent"
)

var (
	currentDistTraceVersion = distTraceVersion([2]int{0 /* Major */, 1 /* Minor */})
	callerUnknown           = payloadCaller{Type: "Unknown", App: "Unknown", Account: "Unknown", TransportType: "Unknown"}
	traceParentRegex        = regexp.MustCompile(`^([a-f0-9]{2})-([a-f0-9]{32})-([a-f0-9]{16})-([a-f0-9]{2})$`)
	fullTraceStateRegex     = regexp.MustCompile(`\d+@nr=[^,=]+,?`)
	newRelicTraceStateRegex = regexp.MustCompile(`(\d+)@nr=(\d)-(\d)-(\d+)-(\d+)-([a-f0-9]{16})-([a-f0-9]{16})-(\d)?-(\d\.\d+)?-(\d+),?`)
	traceStateVendorsRegex  = regexp.MustCompile(`((?:\w*@)?[\w_\-*\s/]+)=[^,]*`)
)

// timestampMillis allows raw payloads to use exact times, and marshalled
// payloads to use times in millis.
type timestampMillis time.Time

func (tm *timestampMillis) UnmarshalJSON(data []byte) error {
	var millis uint64
	if err := json.Unmarshal(data, &millis); nil != err {
		return err
	}
	*tm = timestampMillis(timeFromUnixMilliseconds(millis))
	return nil
}

func (tm timestampMillis) MarshalJSON() ([]byte, error) {
	return json.Marshal(TimeToUnixMilliseconds(tm.Time()))
}

func (tm timestampMillis) Time() time.Time  { return time.Time(tm) }
func (tm *timestampMillis) Set(t time.Time) { *tm = timestampMillis(t) }

func (tm *timestampMillis) UnixMilliseconds() uint64 {
	return TimeToUnixMilliseconds(tm.Time())
}

// Payload is the distributed tracing payload.
type Payload struct {
	payloadCaller
	TransactionID string   `json:"tx,omitempty"`
	ID            string   `json:"id,omitempty"`
	TracedID      string   `json:"tr"`
	Priority      Priority `json:"pr"`
	// This is a *bool instead of a normal bool so we can tell the different between unset and false.
	Sampled              *bool           `json:"sa"`
	Timestamp            timestampMillis `json:"ti"`
	TransportDuration    time.Duration   `json:"-"`
	TrustedParentID      string          `json:"-"`
	TracingVendors       string          `json:"-"`
	HasNewRelicTraceInfo bool            `json:"-"`
	TrustedAccountKey    string          `json:"tk,omitempty"`
	OriginalTraceState   string          `json:"-"`
}

type payloadCaller struct {
	TransportType string `json:"-"`
	Type          string `json:"ty"`
	App           string `json:"ap"`
	Account       string `json:"ac"`
}

// IsValid IsValidNewRelicData the payload data by looking for missing fields.
// Returns an error if there's a problem, nil if everything's fine
func (p Payload) validateNewRelicData() error {

	// If a payload is missing both `guid` and `transactionId` is received,
	// a ParseException supportability metric should be generated.
	if "" == p.TransactionID && "" == p.ID {
		return ErrPayloadMissingField{message: "missing both guid/id and TransactionId/tx"}
	}

	if "" == p.Type {
		return ErrPayloadMissingField{message: "missing Type/ty"}
	}

	if "" == p.Account {
		return ErrPayloadMissingField{message: "missing Account/ac"}
	}

	if "" == p.App {
		return ErrPayloadMissingField{message: "missing App/ap"}
	}

	if "" == p.TracedID {
		return ErrPayloadMissingField{message: "missing TracedID/tr"}
	}

	if p.Timestamp.Time().IsZero() || 0 == p.Timestamp.Time().Unix() {
		return ErrPayloadMissingField{message: "missing Timestamp/ti"}
	}

	return nil
}

func (p Payload) text(v distTraceVersion) []byte {
	// TrustedAccountKey should only be attached to the outbound payload if its value differs
	// from the Account field.
	if p.TrustedAccountKey == p.Account {
		p.TrustedAccountKey = ""
	}
	js, _ := json.Marshal(struct {
		Version distTraceVersion `json:"v"`
		Data    Payload          `json:"d"`
	}{
		Version: v,
		Data:    p,
	})
	return js
}

// NRText implements newrelic.DistributedTracePayload.
func (p Payload) NRText() string {
	t := p.text(currentDistTraceVersion)
	return string(t)
}

// NRHTTPSafe implements newrelic.DistributedTracePayload.
func (p Payload) NRHTTPSafe() string {
	t := p.text(currentDistTraceVersion)
	return base64.StdEncoding.EncodeToString(t)
}

var (
	typeMap = map[string]string{
		CallerTypeApp:     "0",
		CallerTypeBrowser: "1",
		CallerTypeMobile:  "2",
	}
	typeMapReverse = func() map[string]string {
		reversed := make(map[string]string)
		for k, v := range typeMap {
			reversed[v] = k
		}
		return reversed
	}()
)

const (
	w3cVersion        = "00"
	traceStateVersion = "0"
)

// W3CTraceParent returns the W3C TraceParent header for this payload
func (p Payload) W3CTraceParent() string {
	var flags string
	if p.Sampled != nil && *p.Sampled {
		flags = "01"
	} else {
		flags = "00"
	}
	return fmt.Sprintf(w3cVersion + "-" + p.TracedID + "-" + p.ID + "-" + flags)
}

// W3CTraceState returns the W3C TraceState header for this payload
func (p Payload) W3CTraceState() string {
	var flags string

	if p.Sampled != nil && *p.Sampled {
		flags = "1"
	} else {
		flags = "0"
	}
	nonNewRelicTraceState := fullTraceStateRegex.ReplaceAllString(p.OriginalTraceState, "")
	if strings.HasSuffix(nonNewRelicTraceState, ",") {
		nonNewRelicTraceState = nonNewRelicTraceState[0 : len(nonNewRelicTraceState)-1]
	}
	newRelicTraceState := getTraceStatePrefix(p.TrustedAccountKey) + "=" +
		traceStateVersion + "-" +
		typeMap[p.Type] + "-" +
		p.Account + "-" +
		p.App + "-" +
		p.ID + "-" +
		p.TransactionID + "-" +
		flags + "-" +
		strconv.FormatFloat(float64(p.Priority), 'f', 5, 32) + "-" +
		strconv.FormatUint(p.Timestamp.UnixMilliseconds(), 10)
	if nonNewRelicTraceState != "" {
		return newRelicTraceState + "," + nonNewRelicTraceState
	}
	return newRelicTraceState
}

// SetSampled lets us set a value for our *bool,
// which we can't do directly since a pointer
// needs something to point at.
func (p *Payload) SetSampled(sampled bool) {
	p.Sampled = &sampled
}

// ErrPayloadParse indicates that the payload was malformed.
type ErrPayloadParse struct{ err error }

func (e ErrPayloadParse) Error() string {
	return fmt.Sprintf("unable to parse inbound payload: %s", e.err.Error())
}

// ErrPayloadMissingField indicates there's a required field that's missing
type ErrPayloadMissingField struct{ message string }

func (e ErrPayloadMissingField) Error() string {
	return fmt.Sprintf("payload is missing required fields: %s", e.message)
}

// ErrUnsupportedPayloadVersion indicates that the major version number is
// unknown.
type ErrUnsupportedPayloadVersion struct{ version int }

func (e ErrUnsupportedPayloadVersion) Error() string {
	return fmt.Sprintf("unsupported major version number %d", e.version)
}

// AcceptPayload parses the inbound distributed tracing payload.
func AcceptPayload(hdrs http.Header, trustedAccountKey string) (*Payload, error) {
	var payload Payload
	nrPayload := hdrs.Get(DistributedTraceNewRelicHeader)
	w3cTraceParentHdr := hdrs.Get(DistributedTraceW3CTraceParentHeader)

	// If we get both types of headers, first attempt to extract a New Relic entry from tracestate.
	// If there is no New Relic entry in tracestate, use the New Relic header instead.
	if nrPayload != "" && w3cTraceParentHdr != "" {
		err := processW3CHeaders(hdrs, trustedAccountKey, &payload)
		if err != nil {
			err := processNRDTString(nrPayload, &payload)
			if err != nil {
				return nil, err
			}
		}
	} else if nrPayload != "" {
		err := processNRDTString(nrPayload, &payload)
		if nil != err {
			return nil, err
		}
	} else if w3cTraceParentHdr != "" {
		if err := processW3CHeaders(hdrs, trustedAccountKey, &payload); nil != err {
			return nil, err
		}
	} else {
		return nil, nil
	}
	// Ensure that we don't have a reference to the input payload: we don't
	// want to change it, it could be used multiple times.
	alloc := new(Payload)
	*alloc = payload

	return alloc, nil
}

func processNRDTString(str string, payload *Payload) error {
	if str == "" {
		return nil
	}
	var decoded []byte
	if '{' == str[0] {
		decoded = []byte(str)
	} else {
		var err error
		decoded, err = base64.StdEncoding.DecodeString(str)
		if nil != err {
			return ErrPayloadParse{err: err}
		}
	}
	envelope := struct {
		Version distTraceVersion `json:"v"`
		Data    json.RawMessage  `json:"d"`
	}{}
	if err := json.Unmarshal(decoded, &envelope); nil != err {
		return ErrPayloadParse{err: err}
	}

	if 0 == envelope.Version.major() && 0 == envelope.Version.minor() {
		return ErrPayloadMissingField{message: "missing v"}
	}

	if envelope.Version.major() > currentDistTraceVersion.major() {
		return ErrUnsupportedPayloadVersion{
			version: envelope.Version.major(),
		}
	}
	if err := json.Unmarshal(envelope.Data, payload); nil != err {
		return ErrPayloadParse{err: err}
	}
	payload.HasNewRelicTraceInfo = true
	return payload.validateNewRelicData()
}

func processW3CHeaders(hdrs http.Header, trustedAccountKey string, p *Payload) error {
	traceParentStr := hdrs.Get(DistributedTraceW3CTraceParentHeader)
	if err := processTraceParent(traceParentStr, p); nil != err {
		return err
	}

	traceStateStr := hdrs.Get(DistributedTraceW3CTraceStateHeader)
	if err := processTraceState(traceStateStr, trustedAccountKey, p); nil != err {
		return err
	}

	return nil
}

func processTraceParent(traceParentStr string, p *Payload) error {
	subMatches := traceParentRegex.FindStringSubmatch(traceParentStr)

	if subMatches == nil || len(subMatches) != 5 {
		return ErrPayloadParse{errors.New("invalid number of TraceParent entries")}
	}
	p.TracedID = subMatches[2]
	if p.TracedID == "00000000000000000000000000000000" {
		return ErrPayloadParse{errors.New("invalid TraceParent trace ID")}
	}
	p.ID = subMatches[3]
	if p.ID == "0000000000000000" {
		return ErrPayloadParse{errors.New("invalid TraceParent parent ID")}
	}

	return nil
}

var errFieldNum = ErrPayloadParse{errors.New("incorrect number of fields in TraceState")}

func processTraceState(fullTraceState string, trustedAccountKey string, p *Payload) error {
	p.OriginalTraceState = fullTraceState

	nrTraceState := findTrustedNREntry(fullTraceState, trustedAccountKey)
	if nrTraceState == "" {
		return nil
	}
	matches := newRelicTraceStateRegex.FindStringSubmatch(nrTraceState)
	if len(matches) != 11 {
		return errFieldNum
	}

	p.TrustedAccountKey = matches[1]
	if matches[2] != "0" { // currently we only know about version 0
		return ErrPayloadParse{errors.New("unknown TraceState version: " + matches[2])}
	}
	p.Type = typeMapReverse[matches[3]]
	p.Account = matches[4]
	p.App = matches[5]
	p.TrustedParentID = matches[6]
	p.TransactionID = matches[7]

	// If sampled isn't "1" or "0", leave it unset
	if matches[8] == "1" {
		p.SetSampled(true)
	} else if matches[8] == "0" {
		p.SetSampled(false)
	}
	priority, err := strconv.ParseFloat(matches[9], 32)
	if nil == err {
		p.Priority = Priority(priority)
	}
	ts, err := strconv.ParseUint(matches[10], 10, 64)
	if nil == err {
		p.Timestamp = timestampMillis(timeFromUnixMilliseconds(ts))
	}
	p.TracingVendors = tracingVendors(p, p.Account)
	p.HasNewRelicTraceInfo = true
	return nil
}

func findTrustedNREntry(fullTraceState string, trustedAccount string) string {
	submatches := fullTraceStateRegex.FindAllStringSubmatch(fullTraceState, -1)
	accountStr := getTraceStatePrefix(trustedAccount)
	for _, str := range submatches {
		nrString := str[0]
		if strings.HasPrefix(nrString, accountStr) {
			return nrString
		}
	}
	return ""
}

func tracingVendors(p *Payload, thisAccount string) string {
	prefix := getTraceStatePrefix(thisAccount)
	vendorMatches := traceStateVendorsRegex.FindAllStringSubmatch(p.OriginalTraceState, -1)
	if len(vendorMatches) == 0 {
		return ""
	}
	vendors := make([]string, 0, len(vendorMatches))
	for _, vendorMatch := range vendorMatches {
		if len(vendorMatch) != 2 {
			break
		}
		s := vendorMatch[1]
		if s != "" && !strings.HasPrefix(s, prefix) {
			vendors = append(vendors, s)
		}
	}
	return strings.Join(vendors, ",")
}

func getTraceStatePrefix(trustedAccount string) string {
	return trustedAccount + "@nr"
}
