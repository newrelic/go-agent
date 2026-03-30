// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrgocqlx

import (
	"context"
	"testing"

	gocql "github.com/gocql/gocql"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
)

type mockObserver struct {
	called   bool
	original interface {
		ObserveQuery(ctx context.Context, query gocql.ObservedQuery)
	}
}

func (m *mockObserver) ObserveQuery(ctx context.Context, q gocql.ObservedQuery) {
	m.called = true
}

func TestObserveQuery(t *testing.T) {
	app := integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		integrationsupport.ConfigFullTraces,
	)

	hostWithID := new(gocql.HostInfo)
	hostWithID.SetHostID("test-host-id")

	tests := []struct {
		name           string
		useTransaction bool
		original       *mockObserver
		query          gocql.ObservedQuery
		wantOrigCalled bool
		wantQuery      string
		wantKeyspace   string
		wantHost       string
		wantPort       string
	}{
		{
			name:           "nil transaction returns early",
			useTransaction: false,
			original:       nil,
			query:          gocql.ObservedQuery{Statement: "SELECT 1"},
		},
		{
			name:           "original observer is called",
			useTransaction: true,
			original:       &mockObserver{},
			query:          gocql.ObservedQuery{Statement: "SELECT * FROM users", Keyspace: "ks"},
			wantOrigCalled: true,
			wantQuery:      "SELECT * FROM users",
			wantKeyspace:   "ks",
		},
		{
			name:           "segment is enriched",
			useTransaction: true,
			query:          gocql.ObservedQuery{Statement: "INSERT INTO t", Keyspace: "mykeyspace"},
			wantQuery:      "INSERT INTO t",
			wantKeyspace:   "mykeyspace",
		},
		{
			name:           "host info is captured",
			useTransaction: true,
			query: gocql.ObservedQuery{
				Statement: "SELECT * FROM t",
				Keyspace:  "ks",
				Host:      hostWithID,
			},
			wantQuery:    "SELECT * FROM t",
			wantKeyspace: "ks",
			wantHost:     "test-host-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var seg *newrelic.DatastoreSegment

			if tt.useTransaction {
				txn := app.StartTransaction("test-txn")
				defer txn.End()
				ctx = newrelic.NewContext(ctx, txn)
				seg = &newrelic.DatastoreSegment{StartTime: txn.StartSegmentNow()}
				defer seg.End()
				ctx = context.WithValue(ctx, "nrGocqlxSegment", seg)
			}

			NewQueryObserver(tt.original).ObserveQuery(ctx, tt.query)

			if tt.original != nil && tt.original.called != tt.wantOrigCalled {
				t.Errorf("original.called = %v, want %v", tt.original.called, tt.wantOrigCalled)
			}
			if seg != nil {
				if seg.ParameterizedQuery != tt.wantQuery {
					t.Errorf("ParameterizedQuery = %q, want %q", seg.ParameterizedQuery, tt.wantQuery)
				}
				if seg.DatabaseName != tt.wantKeyspace {
					t.Errorf("DatabaseName = %q, want %q", seg.DatabaseName, tt.wantKeyspace)
				}
				if seg.Host != tt.wantHost {
					t.Errorf("Host = %q, want %q", seg.Host, tt.wantHost)
				}
			}
		})
	}
}
