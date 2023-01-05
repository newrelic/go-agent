// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"compress/gzip"
	"sync"
)

func compress(b []byte, gzipWriterPool *sync.Pool) (*bytes.Buffer, error) {
	w := gzipWriterPool.Get().(*gzip.Writer)
	defer gzipWriterPool.Put(w)

	var buf bytes.Buffer
	w.Reset(&buf)
	_, err := w.Write(b)
	w.Close()

	if nil != err {
		return nil, err
	}

	return &buf, nil
}
