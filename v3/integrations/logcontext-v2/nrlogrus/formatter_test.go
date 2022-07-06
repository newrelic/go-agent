package nrlogrus

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	testTime      = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	matchAnything = struct{}{}
	//host, _       = sysinfo.Hostname()
)

func newTestLogger(out io.Writer) *logrus.Logger {
	l := logrus.New()
	l.Formatter = ContextFormatter{}
	l.SetReportCaller(true)
	l.SetOutput(out)
	return l
}

func validateOutput(t *testing.T, out *bytes.Buffer, expected map[string]interface{}) {
	var actual map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &actual); nil != err {
		t.Fatal("failed to unmarshal log output:", err)
	}
	for k, v := range expected {
		found, ok := actual[k]
		if !ok {
			t.Errorf("key %s not found:\nactual=%s", k, actual)
		}
		if v != matchAnything && found != v {
			t.Errorf("value for key %s is incorrect:\nactual=%s\nexpected=%s", k, found, v)
		}
	}
	for k, v := range actual {
		if _, ok := expected[k]; !ok {
			t.Errorf("unexpected key found:\nkey=%s\nvalue=%s", k, v)
		}
	}
}

func TestLogNoContext(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	log.WithTime(testTime).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"log.level": "info",
		"message":   "Hello World!",
		"timestamp": float64(1417136460000),
	})
}
