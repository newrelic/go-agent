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
	traceParentFlagRegex    = regexp.MustCompile(`^([a-f0-9]{2})$`)
	fullTraceStateRegex     = regexp.MustCompile(`\d+@nr=[^,=]+`)
	newRelicTraceStateRegex = regexp.MustCompile(`(\d+)@nr=` + // trustKey@nr=
		`(\d)-` + // version
		`(\d)-` + // parentType
		`(\d+)-` + // accountId
		`([0-9a-zA-Z]+)-` + // appId
		`([a-f0-9]{16})?-` + // spanId
		`([a-f0-9]{16})?-` + // transactionId
		`(\d)?-` + // sampled
		`(\d\.\d+)?-` + // priority
		`(\d+),?`) // timestamp
	traceStateVendorsRegex = regexp.MustCompile(`((?:[\w_\-*\s/]*@)?[\w_\-*\s/]+)=[^,]*`)
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
	NonTrustedTraceState string          `json:"-"`
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
	traceID := p.TracedID
	if idLen := len(traceID); idLen < traceIDHexStringLen {
		traceID = strings.Repeat("0", traceIDHexStringLen-idLen) + traceID
	}
	return w3cVersion + "-" + traceID + "-" + p.ID + "-" + flags
}

// W3CTraceState returns the W3C TraceState header for this payload
func (p Payload) W3CTraceState() string {
	var flags string

	if p.Sampled != nil && *p.Sampled {
		flags = "1"
	} else {
		flags = "0"
	}
	newRelicTraceState := getTraceStatePrefix(p.TrustedAccountKey) + "=" +
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
		newRelicTraceState = newRelicTraceState + "," + p.NonTrustedTraceState
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
func AcceptPayload(hdrs http.Header, trustedAccountKey string) (p *Payload, err error) {
	if hdrs.Get(DistributedTraceW3CTraceParentHeader) != "" {
		p, err = processW3CHeaders(hdrs, trustedAccountKey)
	} else if nrPayload := hdrs.Get(DistributedTraceNewRelicHeader); nrPayload != "" {
		p, err = processNRDTString(nrPayload)
	}
	return
}

func processNRDTString(str string) (*Payload, error) {
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
			return nil, ErrPayloadParse{err: err}
		}
	}
	envelope := struct {
		Version distTraceVersion `json:"v"`
		Data    json.RawMessage  `json:"d"`
	}{}
	if err := json.Unmarshal(decoded, &envelope); nil != err {
		return nil, ErrPayloadParse{err: err}
	}

	if 0 == envelope.Version.major() && 0 == envelope.Version.minor() {
		return nil, ErrPayloadMissingField{message: "missing v"}
	}

	if envelope.Version.major() > currentDistTraceVersion.major() {
		return nil, ErrUnsupportedPayloadVersion{
			version: envelope.Version.major(),
		}
	}
	payload := new(Payload)
	if err := json.Unmarshal(envelope.Data, payload); nil != err {
		return nil, ErrPayloadParse{err: err}
	}

	payload.HasNewRelicTraceInfo = true
	if err := payload.validateNewRelicData(); err != nil {
		return nil, err
	}
	return payload, nil
}

func processW3CHeaders(hdrs http.Header, trustedAccountKey string) (*Payload, error) {
	p, err := processTraceParent(hdrs)
	if nil != err {
		return nil, err
	}
	processTraceState(hdrs, trustedAccountKey, p)
	return p, nil
}

var (
	errNumEntries      = ErrPayloadParse{errors.New("invalid number of TraceParent entries")}
	errInvalidTraceID  = ErrPayloadParse{errors.New("invalid TraceParent trace ID")}
	errInvalidParentID = ErrPayloadParse{errors.New("invalid TraceParent parent ID")}
	errInvalidFlags    = ErrPayloadParse{errors.New("invalid TraceParent flags for this version")}
)

func processTraceParent(hdrs http.Header) (*Payload, error) {
	traceParent := hdrs.Get(DistributedTraceW3CTraceParentHeader)
	subMatches := traceParentRegex.FindStringSubmatch(traceParent)

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
		return isValidFlag(subMatches[4])
	}
	// Invalid version: https://w3c.github.io/trace-context/#version
	if subMatches[1] == "ff" {
		return false
	}
	return true
}

func isValidFlag(f string) bool {
	return traceParentFlagRegex.MatchString(f)
}

func processTraceState(hdrs http.Header, trustedAccountKey string, p *Payload) {
	traceStates := hdrs[DistributedTraceW3CTraceStateHeader]
	fullTraceState := strings.Join(traceStates, ",")
	p.OriginalTraceState = fullTraceState

	nrTraceState := findTrustedNREntry(fullTraceState, trustedAccountKey)
	p.TracingVendors, p.NonTrustedTraceState = parseNonTrustedTraceStates(fullTraceState, nrTraceState)
	if nrTraceState == "" {
		return
	}
	matches := newRelicTraceStateRegex.FindStringSubmatch(nrTraceState)
	if len(matches) != 11 {
		return
	}

	p.TrustedAccountKey = matches[1]
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
	p.HasNewRelicTraceInfo = true
	return
}

func parseNonTrustedTraceStates(fullTraceState string, trustedTraceState string) (tVendors, tState string) {
	vendorMatches := traceStateVendorsRegex.FindAllStringSubmatch(fullTraceState, -1)
	if len(vendorMatches) == 0 {
		return
	}
	vendors := make([]string, 0, len(vendorMatches))
	states := make([]string, 0, len(vendorMatches))
	for _, vendorMatch := range vendorMatches {
		if vendorMatch[0] == trustedTraceState {
			continue
		}
		if len(vendorMatch) != 2 {
			break
		}
		if vendorMatch[1] != "" {
			vendors = append(vendors, vendorMatch[1])
			states = append(states, vendorMatch[0])
		}
	}

	tVendors = strings.Join(vendors, ",")
	tState = strings.Join(states, ",")
	return
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

func getTraceStatePrefix(trustedAccount string) string {
	return trustedAccount + "@nr"
}
