package internal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

type distTraceVersion [2]int

func (v distTraceVersion) major() int { return v[0] }
func (v distTraceVersion) minor() int { return v[1] }

const (
	// CallerType is the value that should be used for the Caller.Type field for outbound
	// payloads.
	CallerType = "App"
)

var (
	currentDistTraceVersion = distTraceVersion([2]int{
		1, // Major
		0, // Minor
	})
	callerUnknown = payloadCaller{
		Type:          "Unknown",
		App:           "Unknown",
		Account:       "Unknown",
		TransportType: "Unknown",
	}
	sampleCaller = payloadCaller{
		Type:          "App",
		Account:       "123",
		App:           "456",
		TransportType: "HTTP",
	}
)

type payloadSynthetics struct {
	Resource string `json:"r"`
	Job      string `json:"j"`
	Monitor  string `json:"m"`
}

// PayloadV1 is the distributed tracing payload.
type PayloadV1 struct {
	payloadCaller
	ID                string             `json:"id"`
	Trip              string             `json:"tr"`
	Priority          string             `json:"pr"`
	Sequence          int                `json:"se"`
	Depth             int                `json:"de"`
	Time              time.Time          `json:"-"`
	TimeMS            uint64             `json:"ti"`
	Host              string             `json:"ho,omitempty"`
	Synthetics        *payloadSynthetics `json:"sy,omitempty"`
	TransportDuration time.Duration      `json:"-"`
}

type payloadCaller struct {
	TransportType string `json:"-"`
	Type          string `json:"ty"`
	App           string `json:"ap"`
	Account       string `json:"ac"`
}

func (p PayloadV1) text() []byte {
	js, _ := json.Marshal(struct {
		Version distTraceVersion `json:"v"`
		Data    PayloadV1        `json:"d"`
	}{
		Version: currentDistTraceVersion,
		Data:    p,
	})
	return js
}

// Text implements newrelic.DistributedTracePayload.
func (p PayloadV1) Text() string {
	t := p.text()
	return string(t)
}

// HTTPSafe implements newrelic.DistributedTracePayload.
func (p PayloadV1) HTTPSafe() string {
	t := p.text()
	return base64.StdEncoding.EncodeToString(t)
}

// AcceptPayload parses the inbound distributed tracing payload.
func AcceptPayload(p interface{}) (*PayloadV1, error) {
	var payload PayloadV1
	switch v := p.(type) {
	case string:
		if "" == v {
			return nil, nil
		}
		var decoded []byte
		if '{' == v[0] {
			decoded = []byte(v)
		} else {
			var err error
			decoded, err = base64.StdEncoding.DecodeString(v)
			if nil != err {
				return nil, err
			}
		}
		envelope := struct {
			Version distTraceVersion `json:"v"`
			Data    json.RawMessage  `json:"d"`
		}{}
		if err := json.Unmarshal(decoded, &envelope); nil != err {
			return nil, err
		}
		if envelope.Version.major() > currentDistTraceVersion.major() {
			return nil, fmt.Errorf("unsupported major version number %d",
				envelope.Version.major())
		}
		payload.Depth = -1
		payload.Sequence = -1
		if err := json.Unmarshal(envelope.Data, &payload); nil != err {
			return nil, err
		}
	case PayloadV1:
		payload = v
	default:
		// Could be a shim payload (if the app is not yet connected).
		return nil, nil
	}
	if payload.Time.IsZero() {
		payload.Time = timeFromUnixMilliseconds(payload.TimeMS)
	}
	alloc := new(PayloadV1)
	*alloc = payload
	return alloc, nil
}
