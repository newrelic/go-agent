package internal

import (
	"bytes"
	"fmt"
	"regexp"
	"time"

	"github.com/newrelic/go-sdk/internal/jsonx"
)

// https://newrelic.atlassian.net/wiki/display/eng/Custom+Events+in+New+Relic+Agents

var (
	eventTypeRegexRaw = `^[a-zA-Z0-9:_ ]+$`
	eventTypeRegex    = regexp.MustCompile(eventTypeRegexRaw)

	errEventTypeLength = fmt.Errorf("event type exceeds length limit of %d",
		attributeKeyLengthLimit)
	ErrEventTypeRegex = fmt.Errorf("event type must match %s", eventTypeRegexRaw)
	errNumAttributes  = fmt.Errorf("maximum of %d attributes exceeded",
		customEventAttributeLimit)
)

type customEvent struct {
	eventType       string
	timestamp       time.Time
	truncatedParams map[string]interface{}
}

func (e *customEvent) WriteJSON(buf *bytes.Buffer) {
	buf.WriteByte('[')
	buf.WriteByte('{')
	buf.WriteString(`"type":`)
	jsonx.AppendString(buf, e.eventType)
	buf.WriteByte(',')
	buf.WriteString(`"timestamp":`)
	jsonx.AppendFloat(buf, timeToFloatSeconds(e.timestamp))
	buf.WriteByte('}')

	buf.WriteByte(',')
	buf.WriteByte('{')
	first := true
	for key, val := range e.truncatedParams {
		if first {
			first = false
		} else {
			buf.WriteByte(',')
		}
		jsonx.AppendString(buf, key)
		buf.WriteByte(':')

		switch v := val.(type) {
		case nil:
			buf.WriteString(`null`)
		case string:
			jsonx.AppendString(buf, v)
		case bool:
			if v {
				buf.WriteString(`true`)
			} else {
				buf.WriteString(`false`)
			}
		case uint8:
			jsonx.AppendUint(buf, uint64(v))
		case uint16:
			jsonx.AppendUint(buf, uint64(v))
		case uint32:
			jsonx.AppendUint(buf, uint64(v))
		case uint64:
			jsonx.AppendUint(buf, v)
		case uint:
			jsonx.AppendUint(buf, uint64(v))
		case uintptr:
			jsonx.AppendUint(buf, uint64(v))
		case int8:
			jsonx.AppendInt(buf, int64(v))
		case int16:
			jsonx.AppendInt(buf, int64(v))
		case int32:
			jsonx.AppendInt(buf, int64(v))
		case int64:
			jsonx.AppendInt(buf, v)
		case int:
			jsonx.AppendInt(buf, int64(v))
		case float32:
			jsonx.AppendFloat(buf, float64(v))
		case float64:
			jsonx.AppendFloat(buf, v)
		default:
			jsonx.AppendString(buf, fmt.Sprintf("%T", v))
		}
	}
	buf.WriteByte('}')

	buf.WriteByte(',')
	buf.WriteByte('{')
	buf.WriteByte('}')
	buf.WriteByte(']')
}

func (e *customEvent) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	e.WriteJSON(buf)

	return buf.Bytes(), nil
}

func eventTypeValidate(eventType string) error {
	if len(eventType) > attributeKeyLengthLimit {
		return errEventTypeLength
	}
	if !eventTypeRegex.MatchString(eventType) {
		return ErrEventTypeRegex
	}
	return nil
}

func createCustomEvent(eventType string, params map[string]interface{}, now time.Time) (*customEvent, error) {
	if err := eventTypeValidate(eventType); nil != err {
		return nil, err
	}

	if len(params) > customEventAttributeLimit {
		return nil, errNumAttributes
	}

	truncatedParams := make(map[string]interface{})
	for key, val := range params {
		if err := validAttributeKey(key); nil != err {
			return nil, err
		}

		val = truncateLongStringValue(val)

		if err := valueIsValid(val); nil != err {
			return nil, err
		}
		truncatedParams[key] = val
	}

	return &customEvent{
		eventType:       eventType,
		timestamp:       now,
		truncatedParams: truncatedParams,
	}, nil
}

func (event *customEvent) mergeIntoHarvest(h *harvest) {
	h.customEvents.Add(event)
}
