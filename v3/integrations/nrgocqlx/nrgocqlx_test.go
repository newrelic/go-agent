// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrgocqlx

import (
	"context"
	"fmt"
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

// func sgmtStartedCheck(t *testing.T, sgmtStarted bool, sgmt *newrelic.DatastoreSegment) {
// 	if !sgmtStarted {
// 		if sgmt != nil {
// 			t.Errorf("execOriginal() began segment unexpectedly")
// 		}
// 	} else {
// 		if sgmt == nil {
// 			t.Errorf("execOriginal() segment not started unexpectedly")
// 		}
// 	}
// }

func Test_execOriginal(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		fn               func(ctx context.Context, dest any) error
		wantErr          bool
		ctx              context.Context
		startTransaction bool
	}{
		{
			name: "Context is nil, should execute function and not begin a segment",
			fn: func(ctx context.Context, dest any) error {
				return nil
			},
			wantErr:          false,
			ctx:              nil,
			startTransaction: false,
		},
		{
			name: "Context exists, should execute function and not begin a segment",
			fn: func(ctx context.Context, dest any) error {
				return nil
			},
			wantErr:          false,
			ctx:              context.Background(),
			startTransaction: false,
		},
		{
			name: "Context is nil, should execute function that returns error and not begin a segment",
			fn: func(ctx context.Context, dest any) error {
				return fmt.Errorf("testing error")
			},
			wantErr:          true,
			ctx:              nil,
			startTransaction: false,
		},
		{
			name: "Context exists, should execute function that returns error and not begin a segment",
			fn: func(ctx context.Context, dest any) error {
				return fmt.Errorf("testing error")
			},
			wantErr:          true,
			ctx:              context.Background(),
			startTransaction: false,
		},
		{
			name: "Context exists, should execute function and begin segment",
			fn: func(ctx context.Context, dest any) error {
				return nil
			},
			wantErr:          false,
			ctx:              context.Background(),
			startTransaction: true,
		},
		{
			name: "Context exists, should execute function that returns error and begin segment",
			fn: func(ctx context.Context, dest any) error {
				return fmt.Errorf("testing error")
			},
			wantErr:          true,
			ctx:              context.Background(),
			startTransaction: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.ctx
			if tt.startTransaction {
				app := integrationsupport.NewTestApp(
					integrationsupport.SampleEverythingReplyFn,
					integrationsupport.ConfigFullTraces,
				)
				txn := app.StartTransaction("test-txn")
				defer txn.End()
				ctx = newrelic.NewContext(ctx, txn)
			}
			gotErr := execOriginal(ctx, tt.fn, struct{}{})

			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("execOriginal() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("execOriginal() succeeded unexpectedly")
			}

		})
	}
}

func Test_execOriginalCAS(t *testing.T) {
	tests := []struct {
		name             string
		fn               func(ctx context.Context, dest any) (bool, error)
		wantErr          bool
		wantApplied      bool
		ctx              context.Context
		startTransaction bool
	}{
		{
			name: "Context is nil, should execute function and not begin a segment",
			fn: func(ctx context.Context, dest any) (bool, error) {
				return true, nil
			},
			wantErr:          false,
			wantApplied:      true,
			ctx:              nil,
			startTransaction: false,
		},
		{
			name: "Context exists, should execute function and not begin a segment",
			fn: func(ctx context.Context, dest any) (bool, error) {
				return true, nil
			},
			wantErr:          false,
			wantApplied:      true,
			ctx:              context.Background(),
			startTransaction: false,
		},
		{
			name: "Context is nil, should execute function that returns error and not begin a segment",
			fn: func(ctx context.Context, dest any) (bool, error) {
				return false, fmt.Errorf("testing error")
			},
			wantErr:          true,
			wantApplied:      false,
			ctx:              nil,
			startTransaction: false,
		},
		{
			name: "Context exists, should execute function that returns error and not begin a segment",
			fn: func(ctx context.Context, dest any) (bool, error) {
				return false, fmt.Errorf("testing error")
			},
			wantErr:          true,
			wantApplied:      false,
			ctx:              context.Background(),
			startTransaction: false,
		},
		{
			name: "Context exists, should execute function and begin segment, applied true",
			fn: func(ctx context.Context, dest any) (bool, error) {
				return true, nil
			},
			wantErr:          false,
			wantApplied:      true,
			ctx:              context.Background(),
			startTransaction: true,
		},
		{
			name: "Context exists, should execute function and begin segment, applied false",
			fn: func(ctx context.Context, dest any) (bool, error) {
				return false, nil
			},
			wantErr:          false,
			wantApplied:      false,
			ctx:              context.Background(),
			startTransaction: true,
		},
		{
			name: "Context exists, should execute function that returns error and begin segment",
			fn: func(ctx context.Context, dest any) (bool, error) {
				return false, fmt.Errorf("testing error")
			},
			wantErr:          true,
			wantApplied:      false,
			ctx:              context.Background(),
			startTransaction: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.ctx
			if tt.startTransaction {
				app := integrationsupport.NewTestApp(
					integrationsupport.SampleEverythingReplyFn,
					integrationsupport.ConfigFullTraces,
				)
				txn := app.StartTransaction("test-txn")
				defer txn.End()
				ctx = newrelic.NewContext(ctx, txn)
			}
			gotApplied, gotErr := execOriginalCAS(ctx, tt.fn, struct{}{})

			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("execOriginalCAS() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("execOriginalCAS() succeeded unexpectedly")
			}
			if gotApplied != tt.wantApplied {
				t.Errorf("execOriginalCAS() applied = %v, want %v", gotApplied, tt.wantApplied)
			}
		})
	}
}
