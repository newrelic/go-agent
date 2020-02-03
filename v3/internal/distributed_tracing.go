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
	DistributedTraceW3CTraceStateHeader = "Tracestate"
	// DistributedTraceW3CTraceParentHeader is one of two headers used by W3C
	// trace context
	DistributedTraceW3CTraceParentHeader = "Traceparent"
)

var (
	currentDistTraceVersion = distTraceVersion([2]int{0 /* Major */, 1 /* Minor */})
	callerUnknown           = payloadCaller{Type: "Unknown", App: "Unknown", Account: "Unknown", TransportType: "Unknown"}
	traceParentRegex        = regexp.MustCompile(`^([a-f0-9]{2})-` + // version
		`([a-f0-9]{32})-` + // traceId
		`([a-f0-9]{16})-` + // parentId
		`([a-f0-9]{2})(-.*)?$`) // flags
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

func (tm timestampMillis) unixMillisecondsString() string {
	ms := TimeToUnixMilliseconds(tm.Time())
	return strconv.FormatUint(ms, 10)
}

// Payload is the distributed tracing payload.
type Payload struct {
	Type          string   `json:"ty"`
	App           string   `json:"ap"`
	Account       string   `json:"ac"`
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
	NonTrustedTraceState string          `json:"-"`
	OriginalTraceState   string          `json:"-"`
}

type payloadCaller struct {
	TransportType string
	Type          string
	App           string
	Account       string
}

var (
	errPayloadMissingGUIDTxnID = errors.New("payload is missing both guid/id and TransactionId/tx")
	errPayloadMissingType      = errors.New("payload is missing Type/ty")
	errPayloadMissingAccount   = errors.New("payload is missing Account/ac")
	errPayloadMissingApp       = errors.New("payload is missing App/ap")
	errPayloadMissingTraceID   = errors.New("payload is missing TracedID/tr")
	errPayloadMissingTimestamp = errors.New("payload is missing Timestamp/ti")
	errPayloadMissingVersion   = errors.New("payload is missing Version/v")
)

// IsValid IsValidNewRelicData the payload data by looking for missing fields.
// Returns an error if there's a problem, nil if everything's fine
func (p Payload) validateNewRelicData() error {

	// If a payload is missing both `guid` and `transactionId` is received,
	// a ParseException supportability metric should be generated.
	if "" == p.TransactionID && "" == p.ID {
		return errPayloadMissingGUIDTxnID
	}

	if "" == p.Type {
		return errPayloadMissingType
	}

	if "" == p.Account {
		return errPayloadMissingAccount
	}

	if "" == p.App {
		return errPayloadMissingApp
	}

	if "" == p.TracedID {
		return errPayloadMissingTraceID
	}

	if p.Timestamp.Time().IsZero() || 0 == p.Timestamp.Time().Unix() {
		return errPayloadMissingTimestamp
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
	if p.isSampled() {
		flags = "01"
	} else {
		flags = "00"
	}
	traceID := strings.ToLower(p.TracedID)
	if idLen := len(traceID); idLen < traceIDHexStringLen {
		traceID = strings.Repeat("0", traceIDHexStringLen-idLen) + traceID
	} else if idLen > traceIDHexStringLen {
		traceID = traceID[idLen-traceIDHexStringLen:]
	}
	return w3cVersion + "-" + traceID + "-" + p.ID + "-" + flags
}

// W3CTraceState returns the W3C TraceState header for this payload
func (p Payload) W3CTraceState() string {
	var flags string

	if p.isSampled() {
		flags = "1"
	} else {
		flags = "0"
	}
	state := p.TrustedAccountKey + "@nr=" +
		traceStateVersion + "-" +
		typeMap[p.Type] + "-" +
		p.Account + "-" +
		p.App + "-" +
		p.ID + "-" +
		p.TransactionID + "-" +
		flags + "-" +
		p.Priority.traceStateFormat() + "-" +
		p.Timestamp.unixMillisecondsString()
	if p.NonTrustedTraceState != "" {
		state += "," + p.NonTrustedTraceState
	}
	return state
}

var (
	trueVal  = true
	falseVal = false
	boolPtrs = map[bool]*bool{
		true:  &trueVal,
		false: &falseVal,
	}
)

// SetSampled lets us set a value for our *bool,
// which we can't do directly since a pointer
// needs something to point at.
func (p *Payload) SetSampled(sampled bool) {
	p.Sampled = boolPtrs[sampled]
}

func (p Payload) isSampled() bool {
	return p.Sampled != nil && *p.Sampled
}

// AcceptPayload parses the inbound distributed tracing payload.
func AcceptPayload(hdrs http.Header, trustedAccountKey string, support *DistributedTracingSupport) (*Payload, error) {
	if hdrs.Get(DistributedTraceW3CTraceParentHeader) != "" {
		return processW3CHeaders(hdrs, trustedAccountKey, support)
	}
	return processNRDTString(hdrs.Get(DistributedTraceNewRelicHeader), support)
}

func processNRDTString(str string, support *DistributedTracingSupport) (*Payload, error) {
	if str == "" {
		return nil, nil
	}
	var decoded []byte
	if '{' == str[0] {
		decoded = []byte(str)
	} else {
		var err error
		decoded, err = base64.StdEncoding.DecodeString(str)
		if nil != err {
			support.AcceptPayloadParseException = true
			return nil, fmt.Errorf("unable to decode payload: %v", err)
		}
	}
	envelope := struct {
		Version distTraceVersion `json:"v"`
		Data    json.RawMessage  `json:"d"`
	}{}
	if err := json.Unmarshal(decoded, &envelope); nil != err {
		support.AcceptPayloadParseException = true
		return nil, fmt.Errorf("unable to unmarshal payload: %v", err)
	}

	if 0 == envelope.Version.major() && 0 == envelope.Version.minor() {
		support.AcceptPayloadParseException = true
		return nil, errPayloadMissingVersion
	}

	if envelope.Version.major() > currentDistTraceVersion.major() {
		support.AcceptPayloadIgnoredVersion = true
		return nil, fmt.Errorf("unsupported major version number %v",
			envelope.Version.major())
	}
	payload := new(Payload)
	if err := json.Unmarshal(envelope.Data, payload); nil != err {
		support.AcceptPayloadParseException = true
		return nil, fmt.Errorf("unable to unmarshal payload data: %v", err)
	}

	payload.HasNewRelicTraceInfo = true
	if err := payload.validateNewRelicData(); err != nil {
		support.AcceptPayloadParseException = true
		return nil, err
	}
	support.AcceptPayloadSuccess = true
	return payload, nil
}

func processW3CHeaders(hdrs http.Header, trustedAccountKey string, support *DistributedTracingSupport) (*Payload, error) {
	p, err := processTraceParent(hdrs)
	if nil != err {
		support.TraceContextParentParseException = true
		return nil, err
	}
	err = processTraceState(hdrs, trustedAccountKey, p)
	if nil != err {
		if err == errInvalidNRTraceState {
			support.TraceContextStateInvalidNrEntry = true
		} else {
			support.TraceContextStateNoNrEntry = true
		}
	}
	support.TraceContextAcceptSuccess = true
	return p, nil
}

var (
	errTooManyHdrs         = errors.New("too many TraceParent headers")
	errNumEntries          = errors.New("invalid number of TraceParent entries")
	errInvalidTraceID      = errors.New("invalid TraceParent trace ID")
	errInvalidParentID     = errors.New("invalid TraceParent parent ID")
	errInvalidFlags        = errors.New("invalid TraceParent flags for this version")
	errInvalidNRTraceState = errors.New("invalid NR entry in trace state")
	errMissingTrustedNR    = errors.New("no trusted NR entry found in trace state")
)

func processTraceParent(hdrs http.Header) (*Payload, error) {
	traceParents := hdrs[DistributedTraceW3CTraceParentHeader]
	if len(traceParents) > 1 {
		return nil, errTooManyHdrs
	}
	subMatches := traceParentRegex.FindStringSubmatch(traceParents[0])

	if subMatches == nil || len(subMatches) != 6 {
		return nil, errNumEntries
	}
	if !validateVersionAndFlags(subMatches) {
		return nil, errInvalidFlags
	}

	p := new(Payload)
	p.TracedID = subMatches[2]
	if p.TracedID == "00000000000000000000000000000000" {
		return nil, errInvalidTraceID
	}
	p.ID = subMatches[3]
	if p.ID == "0000000000000000" {
		return nil, errInvalidParentID
	}

	return p, nil
}

func validateVersionAndFlags(subMatches []string) bool {
	if subMatches[1] == w3cVersion {
		if subMatches[5] != "" {
			return false
		}
	}
	// Invalid version: https://w3c.github.io/trace-context/#version
	if subMatches[1] == "ff" {
		return false
	}
	return true
}

func processTraceState(hdrs http.Header, trustedAccountKey string, p *Payload) error {
	traceStates := hdrs[DistributedTraceW3CTraceStateHeader]
	fullTraceState := strings.Join(traceStates, ",")
	p.OriginalTraceState = fullTraceState

	var trustedVal string
	p.TracingVendors, p.NonTrustedTraceState, trustedVal = parseTraceState(fullTraceState, trustedAccountKey)
	if trustedVal == "" {
		return errMissingTrustedNR
	}

	matches := strings.Split(trustedVal, "-")
	if len(matches) < 9 {
		return errInvalidNRTraceState
	}

	// Required Fields:
	version := matches[0]
	parentType := typeMapReverse[matches[1]]
	account := matches[2]
	app := matches[3]
	timestamp, err := strconv.ParseUint(matches[8], 10, 64)

	if nil != err || "" == version || "" == parentType || "" == account || "" == app {
		return errInvalidNRTraceState
	}

	p.TrustedAccountKey = trustedAccountKey
	p.Type = parentType
	p.Account = account
	p.App = app
	p.TrustedParentID = matches[4]
	p.TransactionID = matches[5]

	// If sampled isn't "1" or "0", leave it unset
	if matches[6] == "1" {
		p.SetSampled(true)
	} else if matches[6] == "0" {
		p.SetSampled(false)
	}
	priority, err := strconv.ParseFloat(matches[7], 32)
	if nil == err {
		p.Priority = Priority(priority)
	}
	p.Timestamp = timestampMillis(timeFromUnixMilliseconds(timestamp))
	p.HasNewRelicTraceInfo = true
	return nil
}

func parseTraceState(fullState, trustedAccountKey string) (nonTrustedVendors string, nonTrustedState string, trustedEntryValue string) {
	trustedKey := trustedAccountKey + "@nr"
	pairs := strings.Split(fullState, ",")
	vendors := make([]string, 0, len(pairs))
	states := make([]string, 0, len(pairs))
	for _, entry := range pairs {
		entry = strings.TrimSpace(entry)
		m := strings.Split(entry, "=")
		if len(m) != 2 {
			continue
		}
		if key, val := m[0], m[1]; key == trustedKey {
			trustedEntryValue = val
		} else {
			vendors = append(vendors, key)
			states = append(states, entry)
		}
	}
	nonTrustedVendors = strings.Join(vendors, ",")
	nonTrustedState = strings.Join(states, ",")
	return
}
