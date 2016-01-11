package internal

// https://newrelic.atlassian.net/wiki/display/eng/Language+agent+transaction+segment+terms+rules

import (
	"encoding/json"
	"regexp"
	"strings"

	"go.datanerd.us/p/will/newrelic/log"
)

const (
	placeholder = "*"
)

type segmentRule struct {
	Prefix        string   `json:"prefix"`
	Terms         []string `json:"terms"`
	TermsRegexRaw string
	TermsRegex    *regexp.Regexp
}

// The key is the rules Prefix field with any trailing slash removed.
type SegmentRules map[string]*segmentRule

func buildTermsRegex(terms []string) string {
	if 0 == len(terms) {
		// If there aren't any terms, then the expected behaviour is not
		// to match anything. We'll return a regex that can't possibly
		// match anything.
		return "$."
	}
	groups := make([]string, len(terms))
	for i, t := range terms {
		groups[i] = "(" + t + ")"
	}
	return strings.Join(groups, "|")
}

func (rules *SegmentRules) UnmarshalJSON(b []byte) error {
	var raw []*segmentRule

	if err := json.Unmarshal(b, &raw); nil != err {
		return err
	}

	rs := make(map[string]*segmentRule)

	for _, rule := range raw {
		prefix := strings.TrimSuffix(rule.Prefix, "/")
		if len(strings.Split(prefix, "/")) != 2 {
			log.Warn("invalid segment term rule prefix",
				log.Context{"prefix": rule.Prefix})
			continue
		}

		if nil == rule.Terms {
			log.Warn("segment term rule has missing terms",
				log.Context{"prefix": rule.Prefix})
			continue
		}

		rule.TermsRegexRaw = buildTermsRegex(rule.Terms)
		re, err := regexp.Compile(rule.TermsRegexRaw)
		if nil != err {
			log.Warn("unable to compile segment term rule terms regexp",
				log.Context{
					"prefix": rule.Prefix,
					"terms":  rule.Terms,
					"regex":  rule.TermsRegexRaw,
				})
			continue
		}
		rule.TermsRegex = re

		rs[prefix] = rule
	}

	*rules = rs
	return nil
}

func collapse(name string) string {
	segments := strings.Split(name, "/")
	collapsed := make([]string, 0, len(segments))

	for i, segment := range segments {
		if segment == placeholder &&
			i < len(segments)-1 &&
			segments[i+1] == placeholder {
			continue
		}
		collapsed = append(collapsed, segment)
	}

	return strings.Join(collapsed, "/")
}

func (rule *segmentRule) apply(name string) string {
	if !strings.HasPrefix(name, rule.Prefix) {
		return name
	}

	s := strings.TrimPrefix(name, rule.Prefix)

	leadingSlash := ""
	if strings.HasPrefix(s, "/") {
		leadingSlash = "/"
		s = strings.TrimPrefix(s, "/")
	}

	if "" != s {
		segments := strings.Split(s, "/")
		replaced := make([]string, len(segments))

		for i, segment := range segments {
			if rule.TermsRegex.MatchString(segment) {
				replaced[i] = segment
			} else {
				replaced[i] = placeholder
			}
		}
		s = strings.Join(replaced, "/")
	}

	return collapse(rule.Prefix + leadingSlash + s)
}

func (rules SegmentRules) Apply(name string) string {
	if nil == rules {
		return name
	}
	segments := strings.Split(name, "/")
	if len(segments) < 3 {
		return name
	}

	key := strings.Join(segments[0:2], "/")
	rule, ok := rules[key]
	if !ok {
		return name
	}

	return rule.apply(name)
}
