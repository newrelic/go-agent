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

// Returns a string slice of query without leading spaces.
func skipSpace(query string) string {
	for i, c := range query {
		if !unicode.IsSpace(c) {
			return query[i:]
		}
	}
	return query
}

// Returns a string slice of query with leading comments removed.
func skipComment(query string) string {
	query = skipSpace(query)

	if strings.HasPrefix(query, "/*") {
		if commentEnd := strings.Index(query[2:], "*/"); commentEnd != -1 {
			return skipComment(query[commentEnd+4:])
		}
		return ""
	} else if strings.HasPrefix(query, "--") || strings.HasPrefix(query, "#") {
		if commentEnd := strings.Index(query[1:], "\n"); commentEnd != -1 {
			return skipComment(query[commentEnd+2:])
		}
		return ""
	} else if strings.HasPrefix(query, ";") {
		return skipComment(query[1:])
	}

	return query
}

// Returns a new string with all comments removed.
func removeAllComments(query string) string {
	query = skipComment(query)
	for i, c := range query {
		if strings.HasPrefix(query[i:], "/*") || strings.HasPrefix(query[i:], "--") || c == '#' {
			query = fmt.Sprintf("%s %s", query[:i], skipComment(query[i:]))
			return removeAllComments(query)
		}
	}
	return query
}

// A SQL tokenizer that extracts tokens from an SQL string.
type sqlTokenizer struct {
	query string
}

// Returns the first word and the remainder of the given string (with trailing
// comments removed).
//
// A word is either sequence of letters and digits, or a single special
// character.
func (q *sqlTokenizer) nextWord() string {
	q.query = skipComment(q.query)

	for i, c := range q.query {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			var word string
			if unicode.IsSpace(c) || i != 0 {
				word = q.query[0:i]
				q.query = q.query[i:]
			} else {
				word = q.query[0 : i+1]
				q.query = q.query[i+1:]
			}
			q.query = skipComment(q.query)
			return word
		}
	}

	word := q.query
	q.query = ""
	return word
}

// Returns the first token and the remainder of the given string (with trailing
// comments removed).
//
// A token is either an expression surrounded by [], or the charater sequence
// before the first space, '(', or '{'.
func (q *sqlTokenizer) nextToken() string {
	q.query = skipComment(q.query)

	if strings.HasPrefix(q.query, "[") {
		for i, c := range q.query {
			if c == ']' {
				token := q.query[0 : i+1]
				q.query = q.query[i+1:]
				return token
			}
		}
	}

	for i, c := range q.query {
		if unicode.IsSpace(c) || (i != 0 && strings.ContainsRune("({,", c)) {
			token := q.query[0:i]
			q.query = q.query[i+1:]
			return token
		}
	}

	token := q.query
	q.query = ""
	return token
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
	sql := sqlTokenizer{query: query}
	op := strings.ToLower(sql.nextWord())
	if tablePrefix, ok := operations[op]; ok {
		segment.Operation = op
		if tablePrefix != "" {
			for {
				word := sql.nextWord()
				if word == "" {
					break
				}
				if strings.ToLower(word) == tablePrefix {
					table := sql.nextToken()
					segment.Collection = extractTable(removeAllComments(table))
				}
			}
		}
		if op == "update" {
			for {
				token := sql.nextToken()
				if token == "" {
					break
				}
				if _, ok := updateModifiers[strings.ToLower(token)]; !ok {
					segment.Collection = extractTable(removeAllComments(token))
					break
				}
			}
		}
	}
}
