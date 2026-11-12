package nrgorm

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/newrelic/go-agent/v3/newrelic"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const testLicenseKey = "1234567890123456789012345678901234567890"

type noopConn struct{}

func (noopConn) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (noopConn) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return nil, errors.New("not implemented")
}

func (noopConn) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, errors.New("not implemented")
}

func (noopConn) QueryRowContext(context.Context, string, ...interface{}) *sql.Row {
	return nil
}

func newTestTransaction() (*newrelic.Application, *newrelic.Transaction, error) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigEnabled(false),
		newrelic.ConfigAppName("nrgorm-test"),
		newrelic.ConfigLicense(testLicenseKey),
	)
	if err != nil {
		return nil, nil, err
	}

	txn := app.StartTransaction("txn")
	if txn == nil {
		app.Shutdown(0)
		return nil, nil, errors.New("start transaction returned nil")
	}

	return app, txn, nil
}

func withTestTransaction(fn func(txn *newrelic.Transaction)) error {
	app, txn, err := newTestTransaction()
	if err != nil {
		return err
	}
	defer txn.End()
	defer app.Shutdown(0)

	fn(txn)
	return nil
}

func TestAPMPluginName(t *testing.T) {
	plugin := APMPlugin{}
	gotName := plugin.Name()
	wantName := "newrelic"

	if gotName != wantName {
		t.Fatalf("unexpected plugin name: got %q want %q", gotName, wantName)
	}
}

func TestAPMPluginInitializeRegistersCallbacks(t *testing.T) {
	db, err := gorm.Open(
		postgres.New(postgres.Config{Conn: noopConn{}}),
		&gorm.Config{
			DisableAutomaticPing: true,
			Plugins: map[string]gorm.Plugin{
				APMPlugin{}.Name(): APMPlugin{},
			},
		},
	)
	if err != nil {
		t.Fatalf("open gorm db: %v", err)
	}

	tests := []struct {
		name string
		get  func() func(*gorm.DB)
	}{
		{name: "create/before", get: func() func(*gorm.DB) { return db.Callback().Create().Get("newrelic:before_create") }},
		{name: "query/before", get: func() func(*gorm.DB) { return db.Callback().Query().Get("newrelic:before_query") }},
		{name: "update/before", get: func() func(*gorm.DB) { return db.Callback().Update().Get("newrelic:before_update") }},
		{name: "delete/before", get: func() func(*gorm.DB) { return db.Callback().Delete().Get("newrelic:before_delete") }},
		{name: "create/after", get: func() func(*gorm.DB) { return db.Callback().Create().Get("newrelic:after_create") }},
		{name: "query/after", get: func() func(*gorm.DB) { return db.Callback().Query().Get("newrelic:after_query") }},
		{name: "update/after", get: func() func(*gorm.DB) { return db.Callback().Update().Get("newrelic:after_update") }},
		{name: "delete/after", get: func() func(*gorm.DB) { return db.Callback().Delete().Get("newrelic:after_delete") }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.get(); got == nil {
				t.Fatalf("callback %q was not registered", tc.name)
			}
		})
	}
}

func TestBeforeCallback(t *testing.T) {
	t.Run("sets start time when transaction exists", func(t *testing.T) {
		conn := postgres.New(postgres.Config{Conn: noopConn{}})
		db, err := gorm.Open(
			conn,
			&gorm.Config{DisableAutomaticPing: true},
		)
		if err != nil {
			t.Fatalf("open gorm db: %v", err)
		}
		db = db.Session(&gorm.Session{Initialized: true})

		err = withTestTransaction(func(txn *newrelic.Transaction) {
			db.Statement.Context = newrelic.NewContext(context.Background(), txn)

			beforeCallback(db)

			startTime, ok := db.Get(startTimeKey)
			if !ok {
				t.Fatalf("expected start time to be set")
			}

			if _, ok := startTime.(newrelic.SegmentStartTime); !ok {
				t.Fatalf("unexpected start time type: %T", startTime)
			}
		})
		if err != nil {
			t.Fatalf("setup transaction: %v", err)
		}
	})

	t.Run("does nothing when transaction is absent", func(t *testing.T) {
		conn := postgres.New(postgres.Config{Conn: noopConn{}})
		db, err := gorm.Open(
			conn,
			&gorm.Config{DisableAutomaticPing: true},
		)
		if err != nil {
			t.Fatalf("open gorm db: %v", err)
		}
		db = db.Session(&gorm.Session{Initialized: true})
		db.Statement.Context = context.Background()

		beforeCallback(db)

		if _, ok := db.Get(startTimeKey); ok {
			t.Fatalf("start time should not be set when transaction is absent")
		}
	})
}

func TestAfterCallback(t *testing.T) {
	tests := []struct {
		name         string
		setStartTime bool
		withTxn      bool
		operation    string
		query        string
	}{
		{name: "returns safely when start time is missing", setStartTime: false, withTxn: false, operation: "SELECT", query: "SELECT * FROM users"},
		{name: "ends segment when start time exists", setStartTime: true, withTxn: true, operation: "SELECT", query: "SELECT * FROM users"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(
				postgres.New(postgres.Config{Conn: noopConn{}}),
				&gorm.Config{DisableAutomaticPing: true},
			)
			if err != nil {
				t.Fatalf("open gorm db: %v", err)
			}
			db = db.Session(&gorm.Session{Initialized: true})
			db.Statement.Table = "users"
			db.Statement.SQL.WriteString(tc.query)

			if tc.setStartTime {
				err := withTestTransaction(func(txn *newrelic.Transaction) {
					db.Set(startTimeKey, txn.StartSegmentNow())
					afterCallback(tc.operation)(db)
				})
				if err != nil {
					t.Fatalf("setup transaction: %v", err)
				}
				return
			}

			afterCallback(tc.operation)(db)
		})
	}
}
