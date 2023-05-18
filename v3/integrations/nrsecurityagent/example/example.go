// Copyright 2022 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/newrelic/go-agent/v3/integrations/nrsecurityagent"
	_ "github.com/newrelic/go-agent/v3/integrations/nrsqlite3"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func index(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello world")
}

func mysql(w http.ResponseWriter, r *http.Request) {

	var user_id = r.URL.Query().Get("input")
	var db *sql.DB
	db, err := sql.Open("nrsqlite3", "./csectest.db")
	defer db.Close()
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("<h1>Unable to Connect DATABASE</h1>"))
	}

	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS USER (id INTEGER, name TEXT)")
	statement.Exec()
	defer statement.Close()
	txn := newrelic.FromContext(r.Context())
	ctx := newrelic.NewContext(context.Background(), txn)

	res := db.QueryRowContext(ctx, "SELECT * FROM USER WHERE name = '"+user_id+"'")

	if err != nil {
		fmt.Println(err)
		w.Write([]byte("<h1>ERROR</h1>"))
	} else {
		fmt.Println(res)
		w.Write([]byte("Executed Query : SELECT * FROM USER WHERE name = '" + user_id + "'"))
	}

}

func async(w http.ResponseWriter, r *http.Request) {
	var filename = r.URL.Query().Get("input")
	txn := newrelic.FromContext(r.Context())
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(txn *newrelic.Transaction) {
		defer wg.Done()
		defer txn.StartSegment("async").End()
		os.Open(filename)
	}(txn.NewGoroutine())

	segment := txn.StartSegment("wg.Wait")
	wg.Wait()
	segment.End()
	w.Write([]byte("done!"))
}

func rxss(w http.ResponseWriter, r *http.Request) {
	var input = r.URL.Query().Get("input")
	io.WriteString(w, input)
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Example App"),
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigAppLogForwardingEnabled(true),
		newrelic.ConfigCodeLevelMetricsEnabled(true),
		newrelic.ConfigCodeLevelMetricsPathPrefix("go-agent/v3"),
		func(config *newrelic.Config) {
			config.Host = "staging-collector.newrelic.com"

		},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = nrsecurityagent.InitSecurityAgent(
		app,
		nrsecurityagent.ConfigSecurityMode("IAST"),
		nrsecurityagent.ConfigSecurityValidatorServiceEndPointUrl("wss://csec-staging.nr-data.net"),
		nrsecurityagent.ConfigSecurityEnable(true),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc(newrelic.WrapHandleFunc(app, "/", index))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/mysql", mysql))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/rxss", rxss))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/async", async))
	http.ListenAndServe(newrelic.WrapListen(":8000"), nil)
}
