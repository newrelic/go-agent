package nrslog

import (
	"os"
	"reflect"
	"testing"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
)

func Test_linkingCache_getAgentLinkingMetadata(t *testing.T) {
	hostname, _ := os.Hostname()
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	md := app.GetLinkingMetadata()

	tests := []struct {
		name         string
		obj          *linkingCache
		app          *newrelic.Application
		wantMetadata newrelic.LinkingMetadata
		wantCache    linkingCache
	}{
		{
			name: "empty cache",
			obj:  &linkingCache{},
			app:  app.Application,
			wantMetadata: newrelic.LinkingMetadata{
				EntityGUID: md.EntityGUID,
				EntityName: "my app",
				Hostname:   hostname,
			},
			wantCache: linkingCache{
				loaded:     true,
				entityGUID: md.EntityGUID,
				entityName: "my app",
				hostname:   hostname,
			},
		},
		{
			name: "loaded cache preserved",
			obj: &linkingCache{
				loaded:     true,
				entityGUID: "test entity GUID",
				entityName: "test app",
				hostname:   "test hostname",
			},
			app: app.Application,
			wantMetadata: newrelic.LinkingMetadata{
				EntityGUID: "test entity GUID",
				EntityName: "test app",
				Hostname:   "test hostname",
			},
			wantCache: linkingCache{
				loaded:     true,
				entityGUID: "test entity GUID",
				entityName: "test app",
				hostname:   "test hostname",
			},
		},
		{
			name: "cache replaced when GUID is empty",
			obj: &linkingCache{
				entityGUID: "",
				entityName: "test app",
				hostname:   "test hostname",
			},
			app: app.Application,
			wantMetadata: newrelic.LinkingMetadata{
				EntityGUID: md.EntityGUID,
				EntityName: "my app",
				Hostname:   hostname,
			},
			wantCache: linkingCache{
				loaded:     true,
				entityGUID: md.EntityGUID,
				entityName: "my app",
				hostname:   hostname,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.obj.getAgentLinkingMetadata(tt.app)
			if got.EntityGUID != tt.wantMetadata.EntityGUID {
				t.Errorf("got incorrect entity GUID for agent = %v, want %v", got.EntityGUID, tt.wantMetadata.EntityGUID)
			}
			if got.EntityName != tt.wantMetadata.EntityName {
				t.Errorf("got incorrect entity name for agent = %v, want %v", got.EntityName, tt.wantMetadata.EntityName)
			}
			if got.Hostname != tt.wantMetadata.Hostname {
				t.Errorf("got incorrect hostname for agent = %v, want %v", got.Hostname, tt.wantMetadata.Hostname)
			}
			if got.TraceID != tt.wantMetadata.TraceID {
				t.Errorf("got incorrect trace ID for transaction = %v, want %v", got.TraceID, tt.wantMetadata.TraceID)
			}
			if got.SpanID != tt.wantMetadata.SpanID {
				t.Errorf("got incorrect span ID for transaction = %v, want %v", got.SpanID, tt.wantMetadata.SpanID)
			}
			if !reflect.DeepEqual(tt.obj, &tt.wantCache) {
				t.Errorf("linkingCache state is incorrect = %v, want %v", tt.obj, tt.wantCache)
			}
		})
	}
}

func Test_linkingCache_getTransactionLinkingMetadata(t *testing.T) {
	hostname, _ := os.Hostname()
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	txn := app.StartTransaction("txn")
	defer txn.End()

	md := txn.GetLinkingMetadata()

	tests := []struct {
		name         string
		obj          *linkingCache
		txn          *newrelic.Transaction
		wantMetadata newrelic.LinkingMetadata
		wantCache    linkingCache
	}{
		{
			name: "empty cache",
			obj:  &linkingCache{},
			txn:  txn,
			wantMetadata: newrelic.LinkingMetadata{
				EntityGUID: md.EntityGUID,
				EntityName: "my app",
				Hostname:   hostname,
				TraceID:    md.TraceID,
				SpanID:     md.SpanID,
			},
			wantCache: linkingCache{
				loaded:     true,
				entityGUID: md.EntityGUID,
				entityName: "my app",
				hostname:   hostname,
			},
		},
		{
			name: "cache preserved when loaded",
			obj: &linkingCache{
				loaded:     true,
				entityGUID: "test entity GUID",
				entityName: "test app",
				hostname:   "test hostname",
			},
			txn: txn,
			wantMetadata: newrelic.LinkingMetadata{
				EntityGUID: "test entity GUID",
				EntityName: "test app",
				Hostname:   "test hostname",
				TraceID:    md.TraceID,
				SpanID:     md.SpanID,
			},
			wantCache: linkingCache{
				loaded:     true,
				entityGUID: "test entity GUID",
				entityName: "test app",
				hostname:   "test hostname",
			},
		},
		{
			name: "cache replaced not fully loaded",
			obj: &linkingCache{
				loaded:     false,
				entityGUID: "",
				entityName: "test app",
				hostname:   "test hostname",
			},
			txn: txn,
			wantMetadata: newrelic.LinkingMetadata{
				EntityGUID: md.EntityGUID,
				EntityName: "my app",
				Hostname:   hostname,
				TraceID:    md.TraceID,
				SpanID:     md.SpanID,
			},
			wantCache: linkingCache{
				loaded:     true,
				entityGUID: md.EntityGUID,
				entityName: "my app",
				hostname:   hostname,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.obj.getTransactionLinkingMetadata(tt.txn)
			if got.EntityGUID != tt.wantMetadata.EntityGUID {
				t.Errorf("got incorrect entity GUID for agent = %v, want %v", got.EntityGUID, tt.wantMetadata.EntityGUID)
			}
			if got.EntityName != tt.wantMetadata.EntityName {
				t.Errorf("got incorrect entity name for agent = %v, want %v", got.EntityName, tt.wantMetadata.EntityName)
			}
			if got.Hostname != tt.wantMetadata.Hostname {
				t.Errorf("got incorrect hostname for agent = %v, want %v", got.Hostname, tt.wantMetadata.Hostname)
			}
			if got.TraceID != tt.wantMetadata.TraceID {
				t.Errorf("got incorrect trace ID for transaction = %v, want %v", got.TraceID, tt.wantMetadata.TraceID)
			}
			if got.SpanID != tt.wantMetadata.SpanID {
				t.Errorf("got incorrect span ID for transaction = %v, want %v", got.SpanID, tt.wantMetadata.SpanID)
			}
			if !reflect.DeepEqual(tt.obj, &tt.wantCache) {
				t.Errorf("linkingCache state is incorrect = %+v, want %+v", tt.obj, tt.wantCache)
			}
		})
	}
}

func BenchmarkGetAgentLinkingMetadata(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	cache := &linkingCache{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.getAgentLinkingMetadata(app.Application)
	}
}

func BenchmarkGetTransactionLinkingMetadata(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	txn := app.StartTransaction("txn")
	defer txn.End()

	// cache := &linkingCache{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		txn.GetTraceMetadata()
	}
}
