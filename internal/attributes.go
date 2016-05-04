package internal

import (
	"fmt"
	"sort"
	"strings"

	"go.datanerd.us/p/will/go-sdk/log"
)

type destinationConfig struct {
	Enabled bool
	Include []string
	Exclude []string
}

type attributeConfig struct {
	All               destinationConfig
	BrowserMonitoring destinationConfig
	ErrorCollector    destinationConfig
	TransactionTracer destinationConfig
	TransactionEvents destinationConfig
}

var defaultAttributeConfig = attributeConfig{
	All:               destinationConfig{Enabled: true},
	BrowserMonitoring: destinationConfig{Enabled: false},
	ErrorCollector:    destinationConfig{Enabled: true},
	TransactionTracer: destinationConfig{Enabled: true},
	TransactionEvents: destinationConfig{Enabled: true},
}

// https://newrelic.atlassian.net/wiki/display/eng/Agent+Attributes

type destination int

const (
	destinationEvent destination = 1 << iota
	destinationTrace
	destinationError
	destinationBrowser
)

const (
	destinationNone destination = 0
	destinationAll  destination = destinationEvent |
		destinationTrace |
		destinationError |
		destinationBrowser
)

func (d destination) String() string {
	s := ""
	for _, x := range []struct {
		string
		destination
	}{
		{"event", destinationEvent},
		{"trace", destinationTrace},
		{"error", destinationError},
		{"browser", destinationBrowser},
	} {
		if destinationNone != d&x.destination {
			if "" != s {
				s += "+"
			}
			s += x.string
		}
	}
	if "" == s {
		s = "none"
	}
	return s
}

func (a *attributes) processDestination(dc *destinationConfig, d destination) {
	if !dc.Enabled {
		a.disabledDestinations |= d
	}
	for _, s := range dc.Include {
		a.addModifier(s, d, 0)
	}
	for _, s := range dc.Exclude {
		a.addModifier(s, 0, d)
	}
}

func makeModifier(match string, include, exclude destination) *attributeModifier {
	if "" == match {
		return nil
	}
	wildcardSuffix := false
	if match[len(match)-1] == '*' {
		wildcardSuffix = true
		match = match[0 : len(match)-1]
	}

	return &attributeModifier{
		wildcardSuffix: wildcardSuffix,
		match:          match,
		include:        include,
		exclude:        exclude,
	}
}

func (a *attributes) addModifier(match string, include, exclude destination) {
	modifier := makeModifier(match, include, exclude)
	if nil == modifier {
		return
	}

	if !modifier.wildcardSuffix {
		if m, ok := a.exactMatchModifiers[modifier.match]; ok {
			m.include |= modifier.include
			m.exclude |= modifier.exclude
		} else {
			a.exactMatchModifiers[modifier.match] = modifier
		}
		return
	}

	for _, m := range a.wildcardModifiers {
		// Important: Duplicate entries for the same match string would
		// not work because exclude needs precedence over include.
		if m.match == modifier.match && m.wildcardSuffix == modifier.wildcardSuffix {
			m.include |= modifier.include
			m.exclude |= modifier.exclude
			return
		}
	}

	a.wildcardModifiers = append(a.wildcardModifiers, modifier)
}

type attributeModifier struct {
	wildcardSuffix bool
	match          string // This will not contain a trailing '*'.
	include        destination
	exclude        destination
}

type attributeModifiers []*attributeModifier

func (m attributeModifiers) Len() int {
	return len(m)
}
func (m attributeModifiers) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m attributeModifiers) Less(i, j int) bool {
	if m[i].match == m[j].match {
		return m[i].wildcardSuffix
	}
	return m[i].match < m[j].match
}

type attribute struct {
	value        interface{}
	destinations destination
}

type attributes struct {
	disabledDestinations destination
	// The order of this list is important.  Modifiers appearing
	// later have precedence over modifiers appearing earlier.
	exactMatchModifiers map[string]*attributeModifier
	wildcardModifiers   attributeModifiers

	agentAttributes map[string]*attribute
	userAttributes  map[string]*attribute
}

func createAttributes(cfg *attributeConfig) *attributes {
	a := &attributes{
		disabledDestinations: destinationNone,
		exactMatchModifiers:  make(map[string]*attributeModifier),
		wildcardModifiers:    make([]*attributeModifier, 0, 64),

		agentAttributes: make(map[string]*attribute),
		userAttributes:  make(map[string]*attribute),
	}

	a.processDestination(&cfg.All, destinationAll)
	a.processDestination(&cfg.BrowserMonitoring, destinationBrowser)
	a.processDestination(&cfg.ErrorCollector, destinationError)
	a.processDestination(&cfg.TransactionTracer, destinationTrace)
	a.processDestination(&cfg.TransactionEvents, destinationEvent)

	sort.Sort(a.wildcardModifiers)

	return a
}

func (m *attributeModifier) isMatch(key string) bool {
	if !m.wildcardSuffix {
		// Exact match expected
		return m.match == key
	}
	// Note: match does NOT include '*'
	return strings.HasPrefix(key, m.match)
}

func (m *attributeModifier) Apply(key string, d destination) destination {
	if m.isMatch(key) {
		// Include before exclude, since exclude has priority.
		d |= m.include
		d &^= m.exclude
	}
	return d
}

func (a *attributes) Apply(key string, d destination) destination {
	// Important: The wildcard modifiers must be applied before the exact match
	// modifiers.
	// Important: The slice must be iterated in a forward direction
	for _, m := range a.wildcardModifiers {
		d = m.Apply(key, d)
	}

	if m, ok := a.exactMatchModifiers[key]; ok {
		d = m.Apply(key, d)
	}

	d &^= a.disabledDestinations

	return d
}

type invalidAttributeError struct{ typeString string }

func (e invalidAttributeError) Error() string {
	return fmt.Sprintf("attribute value type %s is invalid", e.typeString)
}

func valueIsValid(val interface{}) error {
	// TODO(willhf): Is there a more elegant way to do this?
	switch val.(type) {
	case string, bool, nil,
		uint8, uint16, uint32, uint64,
		int8, int16, int32, int64,
		float32, float64,
		uint, int, uintptr:
		return nil
	default:
		return invalidAttributeError{typeString: fmt.Sprintf("%T", val)}
	}
}

type invalidAttributeKeyErr struct {
	key string
}

func (e invalidAttributeKeyErr) Error() string {
	return fmt.Sprintf("attribute key '%.32s...' exceeds length limit %d",
		e.key, attributeKeyLengthLimit)
}

func validAttributeKey(key string) error {
	if len(key) > attributeKeyLengthLimit {
		return invalidAttributeKeyErr{key: key}
	}
	return nil
}

func truncateLongStringValue(val interface{}) interface{} {
	str, ok := val.(string)
	if ok && len(str) > attributeValueLengthLimit {
		val = interface{}(str[0:attributeValueLengthLimit])
	}
	return val
}

func (a *attributes) add(key string, val interface{},
	defaultDests destination, ats map[string]*attribute, limit int) error {

	// Dropping attributes whose keys are excessively long rather than
	// truncating the keys was chosen by product management to avoid
	// worrying about the application of configuration to truncated values,
	// or performing the truncation after configuration.
	if err := validAttributeKey(key); nil != err {
		return err
	}

	val = truncateLongStringValue(val)

	if err := valueIsValid(val); nil != err {
		return err
	}

	finalDestinations := a.Apply(key, defaultDests)
	if finalDestinations != defaultDests {
		log.Debug("attribute destinations modified",
			log.Context{
				"key":    key,
				"input":  defaultDests.String(),
				"output": finalDestinations.String(),
			})
	}

	attribute := &attribute{
		value:        val,
		destinations: finalDestinations,
	}

	// If the attribute being added has a key which is the same as the key
	// of an attribute which already exists, the existing attribute will be
	// removed:  The last attribute in wins.
	if _, ok := ats[key]; ok {
		log.Debug("attribute overridden", log.Context{"key": key})
		ats[key] = attribute
		return nil
	}

	if len(ats) >= limit {
		return fmt.Errorf("attribute '%.128s' discarded: limit of %d reached", key, limit)
	}

	ats[key] = attribute

	return nil
}

func (a *attributes) addAgent(key string, val interface{}, defaultDests destination) error {
	return a.add(key, val, defaultDests, a.agentAttributes, attributeAgentLimit)
}

func (a *attributes) addUser(key string, val interface{}, defaultDests destination) error {
	return a.add(key, val, defaultDests, a.userAttributes, attributeUserLimit)
}

func (a *attributes) get(d destination, ats map[string]*attribute) map[string]interface{} {
	out := make(map[string]interface{})
	for key, attribute := range ats {
		if destinationNone != d&attribute.destinations {
			out[key] = attribute.value
		}
	}
	return out
}

func (a *attributes) GetAgent(d destination) map[string]interface{} {
	return a.get(d, a.agentAttributes)
}

func (a *attributes) GetUser(d destination) map[string]interface{} {
	return a.get(d, a.userAttributes)
}
