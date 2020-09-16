// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package sqlparse

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func extractTable(s string) string {
	s = extractTableRegex.ReplaceAllString(s, "")
	if idx := strings.Index(s, "."); idx > 0 {
		s = s[idx+1:]
	}
	return s
}

var (
	basicTable        = `[^)(\]\[\}\{\s,;]+`
	enclosedTable     = `[\[\(\{]` + `\s*` + basicTable + `\s*` + `[\]\)\}]`
	tablePattern      = `(` + `\s+` + basicTable + `|` + `\s*` + enclosedTable + `)`
	extractTableRegex = regexp.MustCompile(`[\s` + "`" + `"'\(\)\{\}\[\]]*`)
	updateRegex       = regexp.MustCompile(`(?is)^update(?:\s+(?:low_priority|ignore|or|rollback|abort|replace|fail|only))*` + tablePattern)
	sqlOperations     = map[string]*regexp.Regexp{
		"select":   regexp.MustCompile(tablePattern),
		"delete":   regexp.MustCompile(tablePattern),
		"insert":   regexp.MustCompile(tablePattern),
		"update":   updateRegex,
		"call":     nil,
		"create":   nil,
		"drop":     nil,
		"show":     nil,
		"set":      nil,
		"exec":     nil,
		"execute":  nil,
		"alter":    nil,
		"commit":   nil,
		"rollback": nil,
	}
	firstWordRegex   = regexp.MustCompile(`^\w+`)
	cCommentRegex    = regexp.MustCompile(`(?is)/\*.*?\*/`)
	lineCommentRegex = regexp.MustCompile(`(?im)(?:--|#).*?$`)
	sqlPrefixRegex   = regexp.MustCompile(`^[\s;]*`)
)

func skipSpace(query string) string {
	for i, c := range query {
		if !unicode.IsSpace(c) {
			return query[i:]
		}
	}
	return ""
}

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

func cutComment(query string) string {
	query = skipSpace(query)
	for i, _ := range query {
		if strings.HasPrefix(query[i:], "/*") || strings.HasPrefix(query[i:], "--") || strings.HasPrefix(query[i:], ";") {
			query = fmt.Sprintf("%s%s", query[:i], skipComment(query[i:]))
			return cutComment(query)
		}
	}
	return query
}

func firstWord(query string) (string, string) {
	query = skipSpace(query)
	for i, c := range query {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			if unicode.IsSpace(c) || c == '(' || c == '[' || c == '{' {
				return query[0:i], skipSpace(query[i+1:])
			} else {
				return query[0 : i+1], skipSpace(query[i+1:])
			}
		}
	}
	return query, ""
}

func firstWordOnlySpace(query string) (string, string) {
	query = skipSpace(query)

	if strings.HasPrefix(query, "[") {
		for i, c := range query {
			if c == ']' {
				return query[0 : i+1], skipSpace(query[i+1:])
			}
		}
	}

	for i, c := range query {
		if unicode.IsSpace(c) || (i != 0 && (c == '(' || c == ',' || c == '{')) {
			return query[0:i], skipSpace(query[i+1:])
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
	//fmt.Printf("### Original       : %v\n", query)
	s := skipComment(query)
	//fmt.Printf("### Skipped comment: %v\n", s)
	op, s := firstWord(s)
	op = strings.ToLower(op)
	//fmt.Printf("### First word     : %v.\n", op)
	fmt.Printf("")
	op = cutComment(op)
	if _, ok := sqlOperations[op]; ok {
		segment.Operation = op
		if op == "select" || op == "delete" || op == "insert" {
			var token string
			if op == "insert" {
				token = "into"
			} else {
				token = "from"
			}
			for {
				s = skipComment(s)
				from, next := firstWord(s)
				next = skipComment(next)
				s = next
				//fmt.Printf("### query: %v, first word %v\n", query, from)
				if strings.ToLower(from) == token {
					table, next := firstWordOnlySpace(s)
					//fmt.Printf("### s: %v\n", s)
					//fmt.Printf("### table: %v, next: %v\n", table, next)
					s = next
					segment.Collection = extractTable(cutComment(table))
				} else if from == "" {
					break
				}
			}
		} else if op == "update" {
			for {
				modifiers := map[string]interface{}{
					"low_priority": nil,
					"ignore":       nil,
					"or":           nil,
					"rollback":     nil,
					"abort":        nil,
					"replace":      nil,
					"fail":         nil,
					"only":         nil,
				}

				s = skipComment(s)
				from, next := firstWordOnlySpace(s)
				next = skipComment(next)
				s = next
				//fmt.Printf("### query: %v, first word %v\n", query, from)
				if _, ok := modifiers[strings.ToLower(from)]; ok {
					continue
				} else {
					segment.Collection = extractTable(cutComment(from))
					break
				}
			}
		}
	}
}

func ParseQuery2(segment *newrelic.DatastoreSegment, query string) {
	s := cCommentRegex.ReplaceAllString(query, "")
	s = lineCommentRegex.ReplaceAllString(s, "")
	s = sqlPrefixRegex.ReplaceAllString(s, "")
	op := strings.ToLower(firstWordRegex.FindString(query))
	if rg, ok := sqlOperations[op]; ok {
		segment.Operation = op
		if nil != rg {
			if m := rg.FindStringSubmatch(query); len(m) > 1 {
				segment.Collection = extractTable(m[1])
			}
		}
	}
}
