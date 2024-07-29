// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package sqlparse

import (
	"regexp"
	"strings"

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
		"select":   regexp.MustCompile(`(?is)^.*\sfrom` + tablePattern),
		"delete":   regexp.MustCompile(`(?is)^.*\sfrom` + tablePattern),
		"insert":   regexp.MustCompile(`(?is)^.*\sinto?` + tablePattern),
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

// ParseQuery parses table and operation from the SQL query string.  It is
// a helper meant to be used when writing database/sql driver instrumentation.
// Check out full example usage here:
// https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrmysql/nrmysql.go
//
// ParseQuery is designed to work with MySQL, Postgres, and SQLite drivers.
// Ability to correctly parse queries for other SQL databases is not
// guaranteed.
func ParseQuery(segment *newrelic.DatastoreSegment, query string) {
	s := cCommentRegex.ReplaceAllString(query, "")
	s = lineCommentRegex.ReplaceAllString(s, "")
	s = sqlPrefixRegex.ReplaceAllString(s, "")
	op := strings.ToLower(firstWordRegex.FindString(s))
	if rg, ok := sqlOperations[op]; ok {
		segment.Operation = op
		segment.RawQuery = query
		if nil != rg {
			if m := rg.FindStringSubmatch(s); len(m) > 1 {
				segment.Collection = extractTable(m[1])
			}
		}
	}
}
