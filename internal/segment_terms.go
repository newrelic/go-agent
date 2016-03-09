package internal

// https://newrelic.atlassian.net/wiki/display/eng/Language+agent+transaction+segment+terms+rules

import (
	"encoding/json"
	"strings"

	"go.datanerd.us/p/will/newrelic/log"
)

const (
	placeholder = "*"
	separator   = "/"
)

type segmentRule struct {
	Prefix   string   `json:"prefix"`
	Terms    []string `json:"terms"`
	TermsMap map[string]struct{}
}

// The key is the rules Prefix field with any trailing slash removed.
type SegmentRules map[string]*segmentRule

func buildTermsMap(terms []string) map[string]struct{} {
	m := make(map[string]struct{}, len(terms))
	for _, t := range terms {
		m[t] = struct{}{}
	}
	return m
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

		rule.TermsMap = buildTermsMap(rule.Terms)

		rs[prefix] = rule
	}

	*rules = rs
	return nil
}

func (rule *segmentRule) apply(name string) string {
	if !strings.HasPrefix(name, rule.Prefix) {
		return name
	}

	s := strings.TrimPrefix(name, rule.Prefix)

	leadingSlash := ""
	if strings.HasPrefix(s, separator) {
		leadingSlash = separator
		s = strings.TrimPrefix(s, separator)
	}

	if "" != s {
		segments := strings.Split(s, separator)
		replaced := make([]string, len(segments))

		for i, segment := range segments {
			_, whitelisted := rule.TermsMap[segment]
			if whitelisted {
				replaced[i] = segment
			} else {
				replaced[i] = placeholder
			}
		}

		s = collapsePlaceholders(replaced)
	}

	return rule.Prefix + leadingSlash + s
}

func (rules SegmentRules) Apply(name string) string {
	if nil == rules {
		return name
	}

	rule, ok := rules[firstTwoSegments(name)]
	if !ok {
		return name
	}

	return rule.apply(name)
}

func firstTwoSegments(name string) string {
	firstSlashIdx := strings.Index(name, separator)
	if firstSlashIdx == -1 {
		return name
	}

	secondSlashIdx := strings.Index(name[firstSlashIdx+1:], separator)
	if secondSlashIdx == -1 {
		return name
	}

	return name[0 : firstSlashIdx+secondSlashIdx+1]
}

func collapsePlaceholders(segments []string) string {
	prevStar := false
	collapsed := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == placeholder {
			if !prevStar {
				collapsed = append(collapsed, segment)
			}
			prevStar = true
		} else {
			collapsed = append(collapsed, segment)
			prevStar = false
		}
	}

	return strings.Join(collapsed, separator)
}
