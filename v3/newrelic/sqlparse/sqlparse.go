// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package sqlparse

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

var (
	operations = map[string]string{
		"select":   "from",
		"delete":   "from",
		"insert":   "into",
		"update":   "",
		"call":     "",
		"create":   "",
		"drop":     "",
		"show":     "",
		"set":      "",
		"exec":     "",
		"execute":  "",
		"alter":    "",
		"commit":   "",
		"rollback": "",
	}
	updateModifiers = map[string]interface{}{
		"low_priority": nil,
		"ignore":       nil,
		"or":           nil,
		"rollback":     nil,
		"abort":        nil,
		"replace":      nil,
		"fail":         nil,
		"only":         nil,
	}
)

// Extracts the table name from the given string.
func extractTable(s string) string {
	if idx := strings.Index(s, "."); idx > 0 {
		s = s[idx+1:]
	}

	var buffer bytes.Buffer

	for _, c := range s {
		if unicode.IsSpace(c) || strings.ContainsRune("`'(){}[]\"", c) {
			continue
		} else {
			buffer.WriteRune(c)
		}
	}

	return buffer.String()
}

// Returns a new string with trailing comments removed.
func cutComment(query string) string {
	query = skipSpace(query)
	for i, c := range query {
		if strings.HasPrefix(query[i:], "/*") || strings.HasPrefix(query[i:], "--") || c == ';' || c == '#' {
			query = fmt.Sprintf("%s%s", query[:i], skipComment(query[i:]))
			return cutComment(query)
		}
	}
	return query
}

type sqlQuery struct {
	q string
	p uint64
}

// Returns a string slice of query without trailing spaces.
func skipSpace(query string) string {
	for i, c := range query {
		if !unicode.IsSpace(c) {
			return query[i:]
		}
	}
	return ""
}

// Returns a string slice of query with trailing comments removed.
func skipComment(query string) string {
	query = skipSpace(query)

	for {
		if strings.HasPrefix(query, "/*") {
			if commentEnd := strings.Index(query[2:], "*/"); commentEnd != -1 {
				query = query[commentEnd+4:]
				query = skipSpace(query)
			} else {
				return ""
			}
		} else if strings.HasPrefix(query, "--") {
			if commentEnd := strings.Index(query[2:], "\n"); commentEnd != -1 {
				query = query[commentEnd+3:]
				query = skipSpace(query)
			} else {
				return ""
			}
		} else if strings.HasPrefix(query, ";") || strings.HasPrefix(query, "#") {
			query = query[1:]
			query = skipSpace(query)
		} else {
			break
		}
	}

	return query
}

// Returns the first word and the remainder of the given string (with trailing
// comments removed).
//
// A word is either sequence of letters and digits, or a single special
// character.
func firstWord(query string) (string, string) {
	query = skipSpace(query)
	for i, c := range query {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			if unicode.IsSpace(c) || i != 0 {
				return query[0:i], skipComment(query[i+1:])
			} else {
				return query[0 : i+1], skipComment(query[i+1:])
			}
		}
	}
	return query, ""
}

// Returns the first token and the remainder of the given string (with trailing
// comments removed).
//
// A token is either an expression surrounded by [], or the charater sequence
// before the first space, '(', or '{'.
func firstToken(query string) (string, string) {
	query = skipSpace(query)

	if strings.HasPrefix(query, "[") {
		for i, c := range query {
			if c == ']' {
				return query[0 : i+1], skipComment(query[i+1:])
			}
		}
	}

	for i, c := range query {
		if unicode.IsSpace(c) || (i != 0 && strings.ContainsRune("({,", c)) {
			return query[0:i], skipComment(query[i+1:])
		}
	}
	return query, ""
}

// ParseQuery parses table and operation from the SQL query string.  It is
// a helper meant to be used when writing database/sql driver instrumentation.
// Check out full example usage here:
// https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrmysql/nrmysql.go
//
// ParseQuery is designed to work with MySQL, Postgres, and SQLite drivers.
// Ability to correctly parse queries for other SQL databases is not
// guaranteed.
func ParseQuery(segment *newrelic.DatastoreSegment, query string) {
	s := skipComment(query)
	op, s := firstWord(s)
	op = strings.ToLower(cutComment(op))
	if tablePrefix, ok := operations[op]; ok {
		segment.Operation = op
		if tablePrefix != "" {
			for {
				var token string
				token, s = firstWord(s)
				if token == "" {
					break
				}
				if strings.ToLower(token) == tablePrefix {
					var table string
					table, s = firstToken(s)
					segment.Collection = extractTable(cutComment(table))
				}
			}
		}
		if op == "update" {
			for {
				var token string
				token, s = firstToken(s)
				if token == "" {
					break
				}
				if _, ok := updateModifiers[strings.ToLower(token)]; !ok {
					segment.Collection = extractTable(cutComment(token))
					break
				}
			}
		}
	}
}
