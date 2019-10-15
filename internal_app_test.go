package newrelic

import (
	"fmt"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/logger"
)

func TestConnectBackoff(t *testing.T) {
	attempts := map[int]int{
		0:   15,
		2:   30,
		5:   300,
		6:   300,
		100: 300,
		-5:  300,
	}

	for k, v := range attempts {
		if b := getConnectBackoffTime(k); b != v {
			t.Error(fmt.Sprintf("Invalid connect backoff for attempt #%d:", k), v)
		}
	}
}

func TestExtraDebugLogging(t *testing.T) {
	event, err := internal.CreateCustomEvent("myCustomEvent", map[string]interface{}{"zip": "zap"}, time.Now())
	if nil != err {
		t.Fatal(err)
	}
	debug(event, logger.ShimLogger{})
}
