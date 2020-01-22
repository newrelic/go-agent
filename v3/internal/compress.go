package internal

import (
	"bytes"
	"compress/gzip"
)

// Compress compresses.
func Compress(b []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(b)
	w.Close()

	if nil != err {
		return nil, err
	}

	return &buf, nil
}
