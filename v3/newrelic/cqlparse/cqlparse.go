// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cqlparse

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

/*
CQL is similar to SQL in syntax with a few key differences.  There are no Common Table Expressions (CTE),
so the WITH keyword is not captured with regular expressions below.  Additionally, there are no
sub-queries in CQL.  In some SQL queries, INTO may be an optional keyword to use with INSERT.  However, in
CQL is required.  We only capture the DatastoreSegmemt.Collection (table in CQL) for DML queries (SELECT, UPDATE,
INSERT, DELETE) and TRUNCATE.
*/
var (
	basicTable        = `[^)(\]\[\}\{\s,;]+`
	enclosedTable     = `[\[\(\{]` + `\s*` + basicTable + `\s*` + `[\]\)\}]`
	tablePattern      = `(` + `\s+` + basicTable + `|` + `\s*` + enclosedTable + `)`
	extractTableRegex = regexp.MustCompile(`[\s` + "`" + `"'\(\)\{\}\[\]]*`)
	updateRegex       = regexp.MustCompile(`(?is)^update` + tablePattern)
	truncateRegex     = regexp.MustCompile(`(?is)^truncate(?:\s+table)?` + tablePattern)
	cqlOperations     = map[string]*regexp.Regexp{
		"select":   regexp.MustCompile(`(?is)^.*\sfrom` + tablePattern),
		"delete":   regexp.MustCompile(`(?is)^.*\sfrom` + tablePattern),
		"insert":   regexp.MustCompile(`(?is)^.*\sinto` + tablePattern), // INTO is a required keyword after INSERT
		"update":   updateRegex,
		"create":   nil,
		"drop":     nil,
		"alter":    nil,
		"truncate": truncateRegex, // drops all rows from table
		"use":      nil,
		"begin":    nil, // BEGIN BATCH
		"apply":    nil, // APPLY BATCH
	}
	firstWordRegex   = regexp.MustCompile(`^\w+`)
	cCommentRegex    = regexp.MustCompile(`(?is)/\*.*?\*/`)
	lineCommentRegex = regexp.MustCompile(`(?im)--.*?$`)
	cqlPrefixRegex   = regexp.MustCompile(`^[\s;]*`)
)

/*
ParseQuery parses table and operation from a CQL query string. It is a
helper meant to be used when writing Cassandra driver instrumentation.
This is not meant to be used to parse SQL.
*/
func ParseQuery(segment *newrelic.DatastoreSegment, query string) {
	s := cCommentRegex.ReplaceAllString(query, "")
	s = lineCommentRegex.ReplaceAllString(s, "")
	s = cqlPrefixRegex.ReplaceAllString(s, "")
	op := strings.ToLower(firstWordRegex.FindString(s))
	if rg, ok := cqlOperations[op]; ok {
		segment.Operation = op
		segment.RawQuery = query
		if nil != rg {
			if m := rg.FindStringSubmatch(s); len(m) > 1 {
				segment.Collection = extractTable(m[1])
			}
		}
	}
}
