// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal/logger"
)

func serverlessGetenvShim(s string) string {
	if s == "AWS_EXECUTION_ENV" {
		return "the-execution-env"
	}
	return ""
}

func TestServerlessHarvest(t *testing.T) {
	// Test the expected ServerlessHarvest use.
	sh := NewServerlessHarvest(logger.ShimLogger{}, "the-version", serverlessGetenvShim)
	event, err := CreateCustomEvent("myEvent", nil, time.Now())
	if nil != err {
		t.Fatal(err)
	}
	sh.Consume(event)
	buf := &bytes.Buffer{}
	sh.Write("arn", buf)
	metadata, data, err := ParseServerlessPayload(buf.Bytes())
	if nil != err {
		t.Fatal(err)
	}
	if v := string(metadata["metadata_version"]); v != `2` {
		t.Error(v)
	}
	if v := string(metadata["arn"]); v != `"arn"` {
		t.Error(v)
	}
	if v := string(metadata["protocol_version"]); v != `17` {
		t.Error(v)
	}
	if v := string(metadata["execution_environment"]); v != `"the-execution-env"` {
		t.Error(v)
	}
	if v := string(metadata["agent_version"]); v != `"the-version"` {
		t.Error(v)
	}
	if v := string(metadata["agent_language"]); v != `"go"` {
		t.Error(v)
	}
	eventData := string(data["custom_event_data"])
	if !strings.Contains(eventData, `"type":"myEvent"`) {
		t.Error(eventData)
	}
	if len(data) != 1 {
		t.Fatal(data)
	}
	// Test that the harvest was replaced with a new harvest.
	buf = &bytes.Buffer{}
	sh.Write("arn", buf)
	if 0 != buf.Len() {
		t.Error(buf.String())
	}
}

func TestServerlessHarvestNil(t *testing.T) {
	// The public ServerlessHarvest methods should not panic if the
	// receiver is nil.
	var sh *ServerlessHarvest
	event, err := CreateCustomEvent("myEvent", nil, time.Now())
	if nil != err {
		t.Fatal(err)
	}
	sh.Consume(event)
	buf := &bytes.Buffer{}
	sh.Write("arn", buf)
}

func TestServerlessHarvestEmpty(t *testing.T) {
	// Test that ServerlessHarvest.Write doesn't do anything if the harvest
	// is empty.
	sh := NewServerlessHarvest(logger.ShimLogger{}, "the-version", serverlessGetenvShim)
	buf := &bytes.Buffer{}
	sh.Write("arn", buf)
	if 0 != buf.Len() {
		t.Error(buf.String())
	}
}

func BenchmarkServerless(b *testing.B) {
	// The JSON creation in ServerlessHarvest.Write has not been optimized.
	// This benchmark would be useful for doing so.
	sh := NewServerlessHarvest(logger.ShimLogger{}, "the-version", serverlessGetenvShim)
	event, err := CreateCustomEvent("myEvent", nil, time.Now())
	if nil != err {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sh.Consume(event)
		buf := &bytes.Buffer{}
		sh.Write("arn", buf)
		if buf.Len() == 0 {
			b.Fatal(buf.String())
		}
	}
}
