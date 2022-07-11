package newrelic

import (
	"bytes"
	"testing"
)

func BenchmarkStringFieldShort(b *testing.B) {
	writer := jsonFieldsWriter{
		buf: bytes.NewBuffer(make([]byte, 300)),
	}

	for i := 0; i < b.N; i++ {
		writer.stringField("testkey", "this is a short string")
	}
}

func BenchmarkStringFieldLong(b *testing.B) {
	writer := jsonFieldsWriter{
		buf: bytes.NewBuffer(make([]byte, 300)),
	}

	for i := 0; i < b.N; i++ {
		writer.stringField("testkey", "this is a long string that will capture the runtime performance impact that writing more bytes has on this function")
	}
}
