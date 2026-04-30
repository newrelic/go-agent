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
	"github.com/scylladb/gocqlx/v3"
)

type mockObserver struct {
	called   bool
	original interface {
		ObserveQuery(ctx context.Context, query gocql.ObservedQuery)
	}
}

type mockBatchObserver struct {
	called   bool
	original interface {
		ObserveBatch(ctx context.Context, batch gocql.ObservedBatch)
	}
}

func (m *mockObserver) ObserveQuery(ctx context.Context, q gocql.ObservedQuery) {
	m.called = true
}

func (m *mockBatchObserver) ObserveBatch(ctx context.Context, batch gocql.ObservedBatch) {
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
		name             string
		startTransaction bool
		storeSegment     bool
		original         *mockObserver
		query            gocql.ObservedQuery
		wantOrigCalled   bool
		wantQuery        string
		wantKeyspace     string
		wantHost         string
	}{
		{
			name:             "nil transaction returns early",
			startTransaction: false,
			storeSegment:     false,
			original:         nil,
			query:            gocql.ObservedQuery{Statement: "SELECT 1"},
		},
		{
			name:             "original observer is called",
			startTransaction: true,
			storeSegment:     true,
			original:         &mockObserver{},
			query:            gocql.ObservedQuery{Statement: "SELECT * FROM users", Keyspace: "ks"},
			wantOrigCalled:   true,
			wantQuery:        "SELECT * FROM users",
			wantKeyspace:     "ks",
		},
		{
			name:             "segment is enriched",
			startTransaction: true,
			storeSegment:     true,
			query:            gocql.ObservedQuery{Statement: "INSERT INTO t", Keyspace: "mykeyspace"},
			wantQuery:        "INSERT INTO t",
			wantKeyspace:     "mykeyspace",
		},
		{
			name:             "host info is captured",
			startTransaction: true,
			storeSegment:     true,
			query: gocql.ObservedQuery{
				Statement: "SELECT * FROM t",
				Keyspace:  "ks",
				Host:      hostWithID,
			},
			wantQuery:    "SELECT * FROM t",
			wantKeyspace: "ks",
			wantHost:     "test-host-id",
		},
		{
			name:             "early return no segment",
			startTransaction: true,
			storeSegment:     false,
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

			if tt.startTransaction {
				txn := app.StartTransaction("test-txn")
				defer txn.End()
				ctx = newrelic.NewContext(ctx, txn)

				if tt.storeSegment {
					seg = &newrelic.DatastoreSegment{StartTime: newrelic.SegmentStartTime{}}
					defer seg.End()
					ctx = context.WithValue(ctx, queryKey, seg)

				}
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

func Test_execOriginal(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		fn               func(ctx context.Context) error
		wantErr          bool
		ctx              context.Context
		startTransaction bool
	}{
		{
			name: "Context is nil, should execute function and not begin a segment",
			fn: func(ctx context.Context) error {
				return nil
			},
			wantErr:          false,
			ctx:              nil,
			startTransaction: false,
		},
		{
			name: "Context exists, should execute function and not begin a segment",
			fn: func(ctx context.Context) error {
				return nil
			},
			wantErr:          false,
			ctx:              context.Background(),
			startTransaction: false,
		},
		{
			name: "Context is nil, should execute function that returns error and not begin a segment",
			fn: func(ctx context.Context) error {
				return fmt.Errorf("testing error")
			},
			wantErr:          true,
			ctx:              nil,
			startTransaction: false,
		},
		{
			name: "Context exists, should execute function that returns error and not begin a segment",
			fn: func(ctx context.Context) error {
				return fmt.Errorf("testing error")
			},
			wantErr:          true,
			ctx:              context.Background(),
			startTransaction: false,
		},
		{
			name: "Context exists, should execute function and begin segment",
			fn: func(ctx context.Context) error {
				return nil
			},
			wantErr:          false,
			ctx:              context.Background(),
			startTransaction: true,
		},
		{
			name: "Context exists, should execute function that returns error and begin segment",
			fn: func(ctx context.Context) error {
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
			gotErr := execOriginal(ctx, tt.fn, queryKey)

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
		fn               func(ctx context.Context) (bool, error)
		wantErr          bool
		wantApplied      bool
		ctx              context.Context
		startTransaction bool
	}{
		{
			name: "Context is nil, should execute function and not begin a segment",
			fn: func(ctx context.Context) (bool, error) {
				return true, nil
			},
			wantErr:          false,
			wantApplied:      true,
			ctx:              nil,
			startTransaction: false,
		},
		{
			name: "Context exists, should execute function and not begin a segment",
			fn: func(ctx context.Context) (bool, error) {
				return true, nil
			},
			wantErr:          false,
			wantApplied:      true,
			ctx:              context.Background(),
			startTransaction: false,
		},
		{
			name: "Context is nil, should execute function that returns error and not begin a segment",
			fn: func(ctx context.Context) (bool, error) {
				return false, fmt.Errorf("testing error")
			},
			wantErr:          true,
			wantApplied:      false,
			ctx:              nil,
			startTransaction: false,
		},
		{
			name: "Context exists, should execute function that returns error and not begin a segment",
			fn: func(ctx context.Context) (bool, error) {
				return false, fmt.Errorf("testing error")
			},
			wantErr:          true,
			wantApplied:      false,
			ctx:              context.Background(),
			startTransaction: false,
		},
		{
			name: "Context exists, should execute function and begin segment, applied true",
			fn: func(ctx context.Context) (bool, error) {
				return true, nil
			},
			wantErr:          false,
			wantApplied:      true,
			ctx:              context.Background(),
			startTransaction: true,
		},
		{
			name: "Context exists, should execute function and begin segment, applied false",
			fn: func(ctx context.Context) (bool, error) {
				return false, nil
			},
			wantErr:          false,
			wantApplied:      false,
			ctx:              context.Background(),
			startTransaction: true,
		},
		{
			name: "Context exists, should execute function that returns error and begin segment",
			fn: func(ctx context.Context) (bool, error) {
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
			gotApplied, gotErr := execOriginalCAS(ctx, tt.fn, queryKey)

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

func Test_newNRGocqlxQueryxWrapper(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		queryx *gocqlx.Queryx
	}{
		{
			name:   "Wrapper with runners set and queryx set",
			queryx: &gocqlx.Queryx{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newNRGocqlxQueryxWrapper(tt.queryx)

			if got.segmentRunner == nil {
				t.Errorf("newNRGocqlxQueryxWrapper() segmentRunner is nil")
			}

			if got.CASSegmentRunner == nil {
				t.Errorf("newNRGocqlxQueryxWrapper() CASSegmentRunner is nil")
			}
		})
	}
}

func Test_segmentRunner_Query(t *testing.T) {

	app := integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		integrationsupport.ConfigFullTraces,
	)
	w := newNRGocqlxQueryxWrapper(&gocqlx.Queryx{Query: &gocql.Query{}})
	ctx := context.Background()
	txn := app.StartTransaction("test-txn")
	defer txn.End()
	ctx = newrelic.NewContext(ctx, txn)
	w.WithContext(ctx)

	seg := &newrelic.DatastoreSegment{StartTime: txn.StartSegmentNow()}
	defer seg.End()
	ctx = context.WithValue(ctx, queryKey, seg)

	t.Run("runWithSegment and runCASWithSegment no error", func(t *testing.T) {
		err := w.segmentRunner(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("runWithSegment returning error while should be nil")
		}
		_, err = w.CASSegmentRunner(func() (bool, error) {
			return true, nil
		})
		if err != nil {
			t.Errorf("runCASWithSegment returning error while should be nil")
		}
	})
}

func Test_segmentRunner_Batch(t *testing.T) {

	app := integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		integrationsupport.ConfigFullTraces,
	)
	w := newNRGocqlxBatchWrapper(&gocqlx.Batch{Batch: &gocql.Batch{}})
	ctx := context.Background()
	txn := app.StartTransaction("test-txn")
	defer txn.End()
	ctx = newrelic.NewContext(ctx, txn)
	w.WithContext(ctx)

	seg := &newrelic.DatastoreSegment{StartTime: txn.StartSegmentNow()}
	defer seg.End()
	ctx = context.WithValue(ctx, batchKey, seg)

	t.Run("runWithSegment and runCASWithSegment no error", func(t *testing.T) {
		err := w.segmentRunner(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("runWithSegment returning error while should be nil")
		}
	})
}

func TestNewQueryObserver(t *testing.T) {
	var explicitNil *queryObserver = nil
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		original interface {
			ObserveQuery(ctx context.Context, query gocql.ObservedQuery)
		}
		want *queryObserver
	}{
		{
			name:     "Original is explicit nil return original as nil",
			original: nil,
			want:     &queryObserver{nil},
		},
		{
			name:     "Original is type nil return original as nil",
			original: explicitNil,
			want:     &queryObserver{nil},
		},
		{
			name:     "Original is an observer, return original as set",
			original: &mockObserver{},
			want:     &queryObserver{original: &mockObserver{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewQueryObserver(tt.original)
			if tt.want.original == nil && got.original != nil {
				t.Errorf("NewQueryObserver() = %v, want %v", got.original, tt.want.original)
			}
			if tt.want.original != nil && got.original == nil {
				t.Errorf("NewQueryObserver() = %v, want %v", got.original, tt.want.original)
			}
		})
	}
}

func TestObserveBatch(t *testing.T) {
	app := integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		integrationsupport.ConfigFullTraces,
	)

	hostWithID := new(gocql.HostInfo)
	hostWithID.SetHostID("test-host-id")

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		batch            gocql.ObservedBatch
		startTransaction bool
		storeSegment     bool
		original         *mockBatchObserver
		wantOrigCalled   bool
		wantBatch        string
		wantKeyspace     string
		wantHost         string
	}{
		{
			name:             "nil transaction returns early",
			startTransaction: false,
			storeSegment:     false,
			original:         nil,
			batch:            gocql.ObservedBatch{Statements: []string{}},
		},
		{
			name:             "original observer is called",
			startTransaction: true,
			storeSegment:     true,
			original:         &mockBatchObserver{},
			batch:            gocql.ObservedBatch{Statements: []string{"SELECT * FROM USERS", "INSERT INTO t"}, Keyspace: "mykeyspace"},
			wantOrigCalled:   true,
			wantBatch:        "SELECT * FROM USERS; INSERT INTO t",
			wantKeyspace:     "mykeyspace",
		},
		{
			name:             "segment is enriched",
			startTransaction: true,
			storeSegment:     true,
			batch:            gocql.ObservedBatch{Statements: []string{"INSERT INTO t", "DELETE FROM t"}, Keyspace: "mykeyspace"},
			wantBatch:        "INSERT INTO t; DELETE FROM t",
			wantKeyspace:     "mykeyspace",
		},
		{
			name:             "host info is captured",
			startTransaction: true,
			storeSegment:     true,
			batch: gocql.ObservedBatch{
				Statements: []string{"SELECT * FROM t"},
				Keyspace:   "ks",
				Host:       hostWithID,
			},
			wantBatch:    "SELECT * FROM t",
			wantKeyspace: "ks",
			wantHost:     "test-host-id",
		},
		{
			name:             "early return no segment",
			startTransaction: true,
			storeSegment:     false,
			batch: gocql.ObservedBatch{
				Statements: []string{"SELECT * FROM t"},
				Keyspace:   "ks",
				Host:       hostWithID,
			},
		},
		{
			name:             "more than two statements joined",
			startTransaction: true,
			storeSegment:     true,
			batch: gocql.ObservedBatch{
				Statements: []string{"SELECT * FROM a", "INSERT INTO b", "DELETE FROM c"},
				Keyspace:   "ks",
			},
			wantBatch:    "SELECT * FROM a; INSERT INTO b; DELETE FROM c",
			wantKeyspace: "ks",
		},
		{
			name:             "empty statements",
			startTransaction: true,
			storeSegment:     true,
			batch:            gocql.ObservedBatch{Statements: []string{}, Keyspace: "ks"},
			wantBatch:        "",
			wantKeyspace:     "ks",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var seg *newrelic.DatastoreSegment

			if tt.startTransaction {
				txn := app.StartTransaction("test-txn")
				defer txn.End()
				ctx = newrelic.NewContext(ctx, txn)
				if tt.storeSegment {
					seg = &newrelic.DatastoreSegment{StartTime: newrelic.SegmentStartTime{}}
					defer seg.End()
					ctx = context.WithValue(ctx, batchKey, seg)
				}
			}
			NewBatchObserver(tt.original).ObserveBatch(ctx, tt.batch)

			if tt.original != nil && tt.original.called != tt.wantOrigCalled {
				t.Errorf("original.called = %v, want %v", tt.original.called, tt.wantOrigCalled)
			}
			if seg != nil {
				if seg.ParameterizedQuery != tt.wantBatch {
					t.Errorf("ParameterizedQuery = %q, want %q", seg.ParameterizedQuery, tt.wantBatch)
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

func TestNewBatchObserver(t *testing.T) {
	var explicitNil *batchObserver = nil

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		original interface {
			ObserveBatch(ctx context.Context, batch gocql.ObservedBatch)
		}
		want *batchObserver
	}{
		{
			name:     "Original is explicit nil return original as nil",
			original: nil,
			want:     &batchObserver{nil},
		},
		{
			name:     "Orignal is type nil return original as nil",
			original: explicitNil,
			want:     &batchObserver{nil},
		},
		{
			name:     "Orignal is an observer return original as observer",
			original: &mockBatchObserver{},
			want:     &batchObserver{original: &mockBatchObserver{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewBatchObserver(tt.original)
			if tt.want.original == nil && got.original != nil {
				t.Errorf("NewBatchObserver() = %v, want %v", got, tt.want)
			}
			if tt.want.original != nil && got.original == nil {
				t.Errorf("NewBatchObserver() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newNRGocqlxBatchWrapper(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		batch *gocqlx.Batch
	}{
		{
			name:  "Wrapper with runner set and batch set",
			batch: &gocqlx.Batch{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newNRGocqlxBatchWrapper(tt.batch)
			if got.segmentRunner == nil {
				t.Errorf("newNRGocqlxBatchWrapper() segmentRunner is nil")
			}
		})
	}
}
