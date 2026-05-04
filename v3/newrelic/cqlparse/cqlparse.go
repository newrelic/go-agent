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
		"insert":   regexp.MustCompile(`(?is)^.*\sinto` + tablePattern),
		"update":   updateRegex,
		"create":   nil,
		"drop":     nil,
		"alter":    nil,
		"truncate": truncateRegex,
		"use":      nil,
		"begin":    nil, // BEGIN [UNLOGGED|COUNTER] BATCH
		"apply":    nil, // APPLY BATCH
	}
	firstWordRegex   = regexp.MustCompile(`^\w+`)
	cCommentRegex    = regexp.MustCompile(`(?is)/\*.*?\*/`)
	lineCommentRegex = regexp.MustCompile(`(?im)--.*?$`)
	cqlPrefixRegex   = regexp.MustCompile(`^[\s;]*`)
)

// ParseQuery parses table and operation from a CQL query string. It is a
// helper meant to be used when writing Cassandra driver instrumentation.
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
