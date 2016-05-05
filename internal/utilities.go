package internal

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

func removeFirstSegment(name string) string {
	idx := strings.Index(name, "/")
	if -1 == idx {
		return name
	}
	return name[idx+1:]
}

func timeToFloatSeconds(t time.Time) float64 {
	return float64(t.UnixNano()) / float64(1000*1000*1000)
}

func timeToFloatMilliseconds(t time.Time) float64 {
	return float64(t.UnixNano()) / float64(1000*1000)
}

func floatSecondsToDuration(seconds float64) time.Duration {
	nanos := seconds * 1000 * 1000 * 1000
	return time.Duration(nanos) * time.Nanosecond
}

func absTimeDiff(t1, t2 time.Time) time.Duration {
	if t1.After(t2) {
		return t1.Sub(t2)
	}
	return t2.Sub(t1)
}

func compactJSON(js []byte) []byte {
	buf := new(bytes.Buffer)
	if err := json.Compact(buf, js); err != nil {
		return nil
	}
	return buf.Bytes()
}

func compactJSONString(js string) string {
	out := compactJSON([]byte(js))
	return string(out)
}
