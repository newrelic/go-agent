package internal

import (
	"testing"
	"time"
)

var (
	payloadTime   = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	samplePayload = PayloadV1{
		payloadCaller: payloadCaller{
			Type:    CallerType,
			Account: "123",
			App:     "456",
		},
		ID:       "myid",
		Trip:     "mytrip",
		Priority: "mypriority",
		Order:    12,
		Depth:    34,
		Time:     payloadTime,
		TimeMS:   TimeToUnixMilliseconds(payloadTime),
		Host:     "myhost",
	}
)

func TestPayloadRaw(t *testing.T) {
	out, err := AcceptPayload(samplePayload)
	if err != nil || out == nil {
		t.Fatal(err, out)
	}
	if samplePayload != *out {
		t.Fatal(samplePayload, out)
	}
}

func TestPayloadText(t *testing.T) {
	out, err := AcceptPayload(samplePayload.Text())
	if err != nil || out == nil {
		t.Fatal(err, out)
	}
	out.Time = samplePayload.Time // account for timezone differences
	if samplePayload != *out {
		t.Fatal(samplePayload, out)
	}
}

func TestPayloadHTTPSafe(t *testing.T) {
	out, err := AcceptPayload(samplePayload.HTTPSafe())
	if err != nil || nil == out {
		t.Fatal(err, out)
	}
	out.Time = samplePayload.Time // account for timezone differences
	if samplePayload != *out {
		t.Fatal(samplePayload, out)
	}
}

func TestPayloadInvalidBase64(t *testing.T) {
	out, err := AcceptPayload("======")
	if err == nil {
		t.Fatal(err)
	}
	if nil != out {
		t.Fatal(out)
	}
}

func TestPayloadEmptyString(t *testing.T) {
	out, err := AcceptPayload("")
	if err != nil {
		t.Fatal(err)
	}
	if nil != out {
		t.Fatal(out)
	}
}

func TestPayloadUnexpectedType(t *testing.T) {
	out, err := AcceptPayload(1)
	if err != nil {
		t.Fatal(err)
	}
	if nil != out {
		t.Fatal(out)
	}
}

func BenchmarkPayloadText(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		samplePayload.Text()
	}
}
