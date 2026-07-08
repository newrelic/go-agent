package newrelic

import "bytes"

type spanEventEvent struct {
	spanEventType string
	timestamp     int64
	spanId        string //span.id
	traceId       string //trace.id
	name          string
}

func (e *spanEventEvent) WriteJSON(buf *bytes.Buffer) {
	w := jsonFieldsWriter{buf: buf}
	buf.WriteByte('[')
	buf.WriteByte('{')
	w.stringField("type", e.spanEventType)
	w.stringField("trace.id", e.traceId)
	w.stringField("name", e.name)
	w.stringField("span.id", e.spanId)
	w.intField("timestamp", e.timestamp)
	buf.WriteByte('}')
	buf.WriteByte(',')
	buf.WriteByte('{')
	// user attributes (link attributes are not yet captured)
	buf.WriteByte('}')
	buf.WriteByte(',')
	buf.WriteByte('{')
	// agent attributes
	buf.WriteByte('}')
	buf.WriteByte(']')
}

// MarshalJSON is used for testing.
func (e *spanEventEvent) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	e.WriteJSON(buf)

	return buf.Bytes(), nil
}
