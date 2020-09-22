// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package sqlparse

import (
	"testing"

	"github.com/newrelic/go-agent/v3/internal/crossagent"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

type sqlTestcase struct {
	Input     string `json:"input"`
	Operation string `json:"operation"`
	Table     string `json:"table"`
}

func (tc sqlTestcase) test(t *testing.T) {
	var segment newrelic.DatastoreSegment
	ParseQuery(&segment, tc.Input)
	if tc.Operation == "other" {
		// Allow for matching of Operation "other" to ""
		if segment.Operation != "" {
			t.Errorf("operation mismatch query='%s' wanted='%s' got='%s'",
				tc.Input, tc.Operation, segment.Operation)
		}
	} else if segment.Operation != tc.Operation {
		t.Errorf("operation mismatch query='%s' wanted='%s' got='%s'",
			tc.Input, tc.Operation, segment.Operation)
	}
	// The Go agent subquery behavior does not match the PHP Agent.
	if tc.Table == "(subquery)" {
		return
	}
	if tc.Table != segment.Collection {
		t.Errorf("table mismatch query='%s' wanted='%s' got='%s'",
			tc.Input, tc.Table, segment.Collection)
	}
}

func TestParseSQLCrossAgent(t *testing.T) {
	var tcs []sqlTestcase
	err := crossagent.ReadJSON("sql_parsing.json", &tcs)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tcs {
		tc.test(t)
	}
}

func TestParseSQLSubQuery(t *testing.T) {
	for _, tc := range []sqlTestcase{
		{Input: "SELECT * FROM (SELECT * FROM foobar)", Operation: "select", Table: "foobar"},
		{Input: "SELECT * FROM (SELECT * FROM foobar) WHERE x > y", Operation: "select", Table: "foobar"},
		{Input: "SELECT * FROM(SELECT * FROM foobar) WHERE x > y", Operation: "select", Table: "foobar"},
	} {
		tc.test(t)
	}
}

func TestParseSQLOther(t *testing.T) {
	for _, tc := range []sqlTestcase{
		// Test that we handle table names enclosed in brackets.
		{Input: "SELECT * FROM [foo]", Operation: "select", Table: "foo"},
		{Input: "SELECT * FROM[foo]", Operation: "select", Table: "foo"},
		{Input: "SELECT * FROM [ foo ]", Operation: "select", Table: "foo"},
		{Input: "SELECT * FROM [ 'foo' ]", Operation: "select", Table: "foo"},
		{Input: "SELECT * FROM[ `something`.'foo' ]", Operation: "select", Table: "foo"},
		// Test that we handle the cheese.
		{Input: "SELECT fromage FROM fromagier", Operation: "select", Table: "fromagier"},
	} {
		tc.test(t)
	}
}

func TestParseSQLUpdateExtraKeywords(t *testing.T) {
	for _, tc := range []sqlTestcase{
		{Input: "update or rollback foo", Operation: "update", Table: "foo"},
		{Input: "update only foo", Operation: "update", Table: "foo"},
		{Input: "update low_priority ignore{foo}", Operation: "update", Table: "foo"},
	} {
		tc.test(t)
	}
}

func TestLineComment(t *testing.T) {
	for _, tc := range []sqlTestcase{
		{
			Input: `SELECT -- * FROM tricky
			* FROM foo`,
			Operation: "select",
			Table:     "foo",
		},
		{
			Input: `SELECT # * FROM tricky
			* FROM foo`,
			Operation: "select",
			Table:     "foo",
		},
		{
			Input: `    -- SELECT * FROM tricky
			SELECT * FROM foo`,
			Operation: "select",
			Table:     "foo",
		},
		{
			Input: `    # SELECT * FROM tricky
			SELECT * FROM foo`,
			Operation: "select",
			Table:     "foo",
		},
		{
			Input: `SELECT * FROM -- tricky
			foo`,
			Operation: "select",
			Table:     "foo",
		},
	} {
		tc.test(t)
	}
}

func TestSemicolonPrefix(t *testing.T) {
	for _, tc := range []sqlTestcase{
		{
			Input:     `;select * from foo`,
			Operation: "select",
			Table:     "foo",
		},
		{
			Input:     `  ;;  ; select * from foo`,
			Operation: "select",
			Table:     "foo",
		},
		{
			Input: ` ;
			SELECT * FROM foo`,
			Operation: "select",
			Table:     "foo",
		},
	} {
		tc.test(t)
	}
}

func TestDollarSignTable(t *testing.T) {
	for _, tc := range []sqlTestcase{
		{
			Input:     `select * from $dollar_100_$`,
			Operation: "select",
			Table:     "$dollar_100_$",
		},
	} {
		tc.test(t)
	}
}

func TestPriorityQuery(t *testing.T) {
	// Test that we handle:
	// https://dev.mysql.com/doc/refman/8.0/en/insert.html
	//     INSERT [LOW_PRIORITY | DELAYED | HIGH_PRIORITY] [IGNORE] [INTO] tbl_name
	for _, tc := range []sqlTestcase{
		{
			Input:     `INSERT HIGH_PRIORITY INTO employee VALUES('Tom',12345,'Sales',100)`,
			Operation: "insert",
			Table:     "employee",
		},
	} {
		tc.test(t)
	}
}

func TestExtractTable(t *testing.T) {
	for idx, tc := range []string{
		"table",
		"`table`",
		`"table"`,
		"`database.table`",
		"`database`.table",
		"database.`table`",
		"`database`.`table`",
		"  { table }",
		"\n[table]",
		"\t    ( 'database'.`table`  ) ",
	} {
		table := extractTable(tc)
		if table != "table" {
			t.Error(idx, table)
		}
	}
}

func TestSqlTokenizerNextWord(t *testing.T) {
	for input, words := range map[string][]string{
		"SELECT * FROM table":            {"SELECT", "*", "FROM", "table"},
		"SELECT * FROM [ table ]":        {"SELECT", "*", "FROM", "[", "table", "]"},
		"SELECT a,b FROM table":          {"SELECT", "a", ",", "b", "FROM", "table"},
		"SELECT * FROM (SELECT COUNT())": {"SELECT", "*", "FROM", "(", "SELECT", "COUNT", "(", ")", ")"},
		"UPDATE rollback{foo}":           {"UPDATE", "rollback", "{", "foo", "}"},
	} {
		tokenizer := sqlTokenizer{input}

		for _, word := range words {
			if next := tokenizer.nextWord(); word != next {
				t.Errorf("query '%s': expected %s, got %s", input, word, next)
			}
		}

		if next := tokenizer.nextWord(); next != "" {
			t.Errorf("query '%s': expected empty word, got %s", input, next)
		}
	}
}

func TestSqlTokenizerNextToken(t *testing.T) {
	for input, tokens := range map[string][]string{
		"SELECT * FROM [ table ]":        {"SELECT", "*", "FROM", "[ table ]"},
		"SELECT a,b FROM table":          {"SELECT", "a", "b", "FROM", "table"},
		"SELECT * FROM (SELECT COUNT())": {"SELECT", "*", "FROM", "(SELECT", "COUNT", "))"},
		"UPDATE rollback{foo}":           {"UPDATE", "rollback", "foo}"},
	} {
		tokenizer := sqlTokenizer{input}

		for _, token := range tokens {
			if next := tokenizer.nextToken(); token != next {
				t.Errorf("query '%s': expected %s, got %s", input, token, next)
			}
		}

		if next := tokenizer.nextToken(); next != "" {
			t.Errorf("query '%s': expected empty token , got %s", input, next)
		}
	}
}

func TestRemoveAllComments(t *testing.T) {
	for input, expected := range map[string]string{
		"query -- comment":                                  "query  ",
		"query -- comment\nother":                           "query  other",
		"query /* comment */":                               "query  ",
		"query /* comment */ other":                         "query  other",
		"query /* comment */ other /* other comment */ end": "query  other  end",
		"query # comment":                                   "query  ",
		"query# comment\nother":                             "query other",
	} {
		if result := removeAllComments(input); result != expected {
			t.Errorf("query '%s': expected %s, got %s", input, expected, result)
		}
	}
}

func TestSkipComment(t *testing.T) {
	for input, expected := range map[string]string{
		" -- comment\nquery":   "query",
		"/* comment */ query":  "query",
		"/* comment */query":   "query",
		"\t/* comment */query": "query",
		"# comment\nquery":     "query",
		"# comment\nquery ":    "query ",
		" query":               "query",
	} {
		if result := skipComment(input); result != expected {
			t.Errorf("query '%s': expected %s, got %s", input, expected, result)
		}
	}
}

func TestSkipSpace(t *testing.T) {
	for input, expected := range map[string]string{
		"query":    "query",
		"  query":  "query",
		"\t query": "query",
		"\n query": "query",
	} {
		if result := skipSpace(input); result != expected {
			t.Errorf("query: '%s': expected %s, got %s", input, expected, result)
		}
	}
}
