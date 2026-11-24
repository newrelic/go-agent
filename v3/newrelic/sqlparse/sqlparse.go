// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package sqlparse

import (
	"regexp"
	"strings"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func extractTable(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case ' ', '\t', '\n', '\r', '\v', '\f', '`', '"', '\'', '(', ')', '{', '}', '[', ']':
			continue
		default:
			b = append(b, ch)
		}
	}
	s = string(b)
	if idx := strings.IndexByte(s, '.'); idx > 0 {
		s = s[idx+1:]
	}
	return s
}

var (
	basicTable    = `[^)(\]\[\}\{\s,;]+`
	enclosedTable = `[\[\(\{]` + `\s*` + basicTable + `\s*` + `[\]\)\}]`
	tablePattern  = `(` + `\s+` + basicTable + `|` + `\s*` + enclosedTable + `)`
	updateRegex   = regexp.MustCompile(`(?is)^update(?:\s+(?:low_priority|ignore|or|rollback|abort|replace|fail|only))*` + tablePattern)
	insertRegex   = regexp.MustCompile(`(?is)^insert(?:\s+(?:low_priority|delayed|high_priority|ignore))*(?:\s+into)?` + tablePattern)
	withRegex     = regexp.MustCompile(`(?is)^with(?:\s+recursive)?.*\)\s*select.*?\sfrom` + tablePattern)
	sqlOperations = map[string]*regexp.Regexp{
		"select":   regexp.MustCompile(`(?is)^.*\sfrom` + tablePattern),
		"delete":   regexp.MustCompile(`(?is)^.*\sfrom` + tablePattern),
		"insert":   insertRegex,
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
		"with":     withRegex,
	}
)

func isSQLSpace(ch byte) bool {
	switch ch {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	default:
		return false
	}
}

func isWordChar(ch byte) bool {
	return ch == '_' || ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ('0' <= ch && ch <= '9')
}

func toLowerASCII(s string) string {
	for i := 0; i < len(s); i++ {
		if 'A' <= s[i] && s[i] <= 'Z' {
			b := []byte(s)
			for j := i; j < len(b); j++ {
				if 'A' <= b[j] && b[j] <= 'Z' {
					b[j] += 'a' - 'A'
				}
			}
			return string(b)
		}
	}
	return s
}

func skipSQLPrefix(s string) string {
	i := 0
	for i < len(s) {
		if isSQLSpace(s[i]) || s[i] == ';' {
			i++
			continue
		}
		break
	}
	return s[i:]
}

func stripCommentsAndPrefix(query string) string {
	var out []byte
	n := len(query)
	last := 0
	i := 0
	for i < n {
		if i+1 < n && query[i] == '/' && query[i+1] == '*' {
			if out == nil {
				out = make([]byte, 0, n)
			}
			out = append(out, query[last:i]...)
			i += 2
			for i+1 < n && !(query[i] == '*' && query[i+1] == '/') {
				i++
			}
			if i+1 < n {
				i += 2
			} else {
				i = n
			}
			last = i
			continue
		}

		if (i+1 < n && query[i] == '-' && query[i+1] == '-') || query[i] == '#' {
			if out == nil {
				out = make([]byte, 0, n)
			}
			out = append(out, query[last:i]...)
			if query[i] == '#' {
				i++
			} else {
				i += 2
			}
			for i < n && query[i] != '\n' && query[i] != '\r' {
				i++
			}
			last = i
			continue
		}

		i++
	}

	if out == nil {
		return skipSQLPrefix(query)
	}

	out = append(out, query[last:]...)
	return skipSQLPrefix(string(out))
}

func operationFromQuery(s string) (string, int) {
	i := 0
	for i < len(s) && isWordChar(s[i]) {
		i++
	}
	if i == 0 {
		return "", 0
	}
	return toLowerASCII(s[:i]), i
}

func skipSpaces(s string, idx int) int {
	for idx < len(s) && isSQLSpace(s[idx]) {
		idx++
	}
	return idx
}

func readWordLower(s string, idx int) (word string, start int, end int, ok bool) {
	start = skipSpaces(s, idx)
	if start >= len(s) || !isWordChar(s[start]) {
		return "", start, start, false
	}
	end = start + 1
	for end < len(s) && isWordChar(s[end]) {
		end++
	}
	return toLowerASCII(s[start:end]), start, end, true
}

func readTableToken(s string, idx int) (string, bool) {
	idx = skipSpaces(s, idx)
	if idx >= len(s) {
		return "", false
	}

	switch s[idx] {
	case '[', '(', '{':
		closing := byte(']')
		switch s[idx] {
		case '(':
			closing = ')'
		case '{':
			closing = '}'
		}

		end := idx + 1
		for end < len(s) && s[end] != closing {
			end++
		}
		if end >= len(s) {
			return "", false
		}
		contentStart := skipSpaces(s, idx+1)
		contentEnd := end
		for contentEnd > contentStart && isSQLSpace(s[contentEnd-1]) {
			contentEnd--
		}
		if contentEnd <= contentStart {
			return "", false
		}
		for i := contentStart; i < contentEnd; i++ {
			ch := s[i]
			if isSQLSpace(ch) || ch == ',' || ch == ';' || ch == '(' || ch == ')' || ch == '[' || ch == ']' || ch == '{' || ch == '}' {
				return "", false
			}
		}
		return extractTable(s[idx : end+1]), true
	default:
		end := idx
		for end < len(s) {
			ch := s[end]
			if isSQLSpace(ch) || ch == ',' || ch == ';' || ch == '(' || ch == ')' || ch == '[' || ch == ']' || ch == '{' || ch == '}' {
				break
			}
			end++
		}
		if end == idx {
			return "", false
		}
		return extractTable(s[idx:end]), true
	}
}

func parseInsertTableFast(s string, opEnd int) (string, bool) {
	idx := opEnd
	for {
		word, start, end, ok := readWordLower(s, idx)
		if !ok {
			return readTableToken(s, start)
		}

		switch word {
		case "low_priority", "delayed", "high_priority", "ignore":
			idx = end
		case "into":
			return readTableToken(s, end)
		default:
			return readTableToken(s, start)
		}
	}
}

func parseUpdateTableFast(s string, opEnd int) (string, bool) {
	idx := opEnd
	for {
		word, start, end, ok := readWordLower(s, idx)
		if !ok {
			return readTableToken(s, start)
		}

		switch word {
		case "low_priority", "ignore", "or", "rollback", "abort", "replace", "fail", "only":
			idx = end
		default:
			return readTableToken(s, start)
		}
	}
}

func skipQuoted(s string, idx int) (int, bool) {
	quote := s[idx]
	idx++
	for idx < len(s) {
		if s[idx] == quote {
			if (quote == '\'' || quote == '"') && idx+1 < len(s) && s[idx+1] == quote {
				idx += 2
				continue
			}
			return idx + 1, true
		}
		idx++
	}
	return idx, false
}

func findLastFromTableFast(s string) (string, bool) {
	depth := 0
	fromEnd := -1

	for idx := 0; idx < len(s); {
		ch := s[idx]
		switch ch {
		case '\'', '"', '`':
			next, ok := skipQuoted(s, idx)
			if !ok {
				return "", false
			}
			idx = next
			continue
		case '[':
			end := idx + 1
			for end < len(s) && s[end] != ']' {
				end++
			}
			if end >= len(s) {
				return "", false
			}
			idx = end + 1
			continue
		case '(':
			depth++
			idx++
			continue
		case ')':
			if depth > 0 {
				depth--
			}
			idx++
			continue
		default:
			if depth == 0 && isWordChar(ch) {
				start := idx
				idx++
				for idx < len(s) && isWordChar(s[idx]) {
					idx++
				}
				if toLowerASCII(s[start:idx]) == "from" {
					fromEnd = idx
				}
				continue
			}
			idx++
		}
	}

	if fromEnd < 0 {
		return "", false
	}
	return readTableToken(s, fromEnd)
}

func parseFastTable(op string, s string, opEnd int) (string, bool) {
	switch op {
	case "insert":
		return parseInsertTableFast(s, opEnd)
	case "update":
		return parseUpdateTableFast(s, opEnd)
	case "select", "delete", "with":
		return findLastFromTableFast(s)
	default:
		return "", false
	}
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
	s := stripCommentsAndPrefix(query)
	op, opEnd := operationFromQuery(s)
	if rg, ok := sqlOperations[op]; ok {
		segment.Operation = op
		segment.RawQuery = query
		if table, ok := parseFastTable(op, s, opEnd); ok {
			segment.Collection = table
			return
		}

		if nil != rg {
			if m := rg.FindStringSubmatch(s); len(m) > 1 {
				segment.Collection = extractTable(m[1])
			}
		}
	}
}
