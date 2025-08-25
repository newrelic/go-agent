package newrelic

import (
	"runtime"
	"testing"
	"time"
)

func TestHeapHighWaterMarkAlarmNilAppSafety(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*Application)
	}{
		{"Enable", func(app *Application) { app.HeapHighWaterMarkAlarmEnable(100 * time.Millisecond) }},
		{"Disable", func(app *Application) { app.HeapHighWaterMarkAlarmDisable() }},
		{"Shutdown", func(app *Application) { app.HeapHighWaterMarkAlarmShutdown() }},
		{"Set", func(app *Application) { app.HeapHighWaterMarkAlarmSet(1024, func(uint64, *runtime.MemStats) {}) }},
		{"ClearAll", func(app *Application) { app.HeapHighWaterMarkAlarmClearAll() }},
	}

	appCases := []struct {
		name string
		app  *Application
	}{
		{"nil app", nil},
		{"nil app.app", &Application{app: nil}},
	}

	for _, appCase := range appCases {
		for _, tt := range tests {
			t.Run(appCase.name+"_"+tt.name, func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s panicked on %s: %v", tt.name, appCase.name, r)
					}
				}()

				tt.fn(appCase.app)
			})
		}
	}
}

func TestHeapHighWaterMarkAlarmEnable(t *testing.T) {
	app := testApp(nil, ConfigEnabled(true), t)
	defer app.Shutdown(10 * time.Second)

	app.HeapHighWaterMarkAlarmEnable(100 * time.Millisecond)
	defer app.HeapHighWaterMarkAlarmShutdown()

	app.app.heapHighWaterMarkAlarms.lock.RLock()
	ticker := app.app.heapHighWaterMarkAlarms.sampleTicker
	done := app.app.heapHighWaterMarkAlarms.done
	app.app.heapHighWaterMarkAlarms.lock.RUnlock()

	if ticker == nil || done == nil {
		t.Error("Expected ticker and done channel to be created")
	}
}

func TestHeapHighWaterMarkAlarmEnableResetTicker(t *testing.T) {
	app := testApp(nil, ConfigEnabled(true), t)
	defer app.Shutdown(10 * time.Second)

	// Enable with initial interval
	app.HeapHighWaterMarkAlarmEnable(100 * time.Millisecond)
	defer app.HeapHighWaterMarkAlarmShutdown()

	app.app.heapHighWaterMarkAlarms.lock.RLock()
	oldTicker := app.app.heapHighWaterMarkAlarms.sampleTicker
	app.app.heapHighWaterMarkAlarms.lock.RUnlock()

	if oldTicker == nil {
		t.Fatal("Expected ticker to be created on first enable")
	}

	// Enable with new interval - should reset existing ticker
	app.HeapHighWaterMarkAlarmEnable(400 * time.Millisecond)

	app.app.heapHighWaterMarkAlarms.lock.RLock()
	newTicker := app.app.heapHighWaterMarkAlarms.sampleTicker
	app.app.heapHighWaterMarkAlarms.lock.RUnlock()

	if newTicker == nil {
		t.Fatal("Expected ticker to exist after reset")
	}

	if oldTicker != newTicker {
		t.Error("Expected ticker to be the same instance after reset")
	}

	// Verify ticker is functional and doesn't tick too frequently
	tickCount := 0
	timeout := time.After(1200 * time.Millisecond) // Give enough time for 2+ ticks

	for {
		select {
		case <-newTicker.C:
			tickCount++
			if tickCount == 2 { // Stop after getting expected ticks
				return
			} else if tickCount > 2 {
				t.Errorf("Ticker ticked more than expected: %d times", tickCount)
				return
			}
		case <-timeout:
			if tickCount == 0 {
				t.Error("Ticker should have ticked at least once within timeout")
			}
			return
		}
	}
}

func TestHeapHighWaterMarkAlarmCallbackTriggered(t *testing.T) {
	app := testApp(nil, ConfigEnabled(true), t)
	defer app.Shutdown(10 * time.Second)

	app.HeapHighWaterMarkAlarmEnable(10 * time.Millisecond)
	defer app.HeapHighWaterMarkAlarmShutdown()

	callbackCh := make(chan struct{}, 1)
	var calledLimit uint64
	var calledMemStats *runtime.MemStats

	app.HeapHighWaterMarkAlarmSet(1, func(limit uint64, memStats *runtime.MemStats) {
		calledLimit = limit
		calledMemStats = memStats
		select {
		case callbackCh <- struct{}{}:
		default:
		}
	})

	select {
	case <-callbackCh:
		// Callback triggered
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout: callback was not called when heap allocation exceeds limit")
	}

	if calledLimit != 1 {
		t.Errorf("Expected callback to be called with limit 1, got %d", calledLimit)
	}

	if calledMemStats == nil {
		t.Error("Expected callback to be called with valid MemStats")
	}

	if calledMemStats != nil && calledMemStats.HeapAlloc < 1 {
		t.Errorf("Expected HeapAlloc to be >= 1, got %d", calledMemStats.HeapAlloc)
	}
}

func TestHeapHighWaterMarkWithInvalidInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
	}{
		{"zero interval", 0},
		{"negative interval", -1 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := testApp(nil, ConfigEnabled(true), t)
			defer app.Shutdown(10 * time.Second)

			app.HeapHighWaterMarkAlarmEnable(tt.interval)
			defer app.HeapHighWaterMarkAlarmShutdown()

			app.app.heapHighWaterMarkAlarms.lock.RLock()
			ticker := app.app.heapHighWaterMarkAlarms.sampleTicker
			app.app.heapHighWaterMarkAlarms.lock.RUnlock()

			if ticker != nil {
				t.Error("Expected ticker to be nil with invalid interval")
			}
		})
	}
}

func TestHeapHighWaterMarkAlarmDisable(t *testing.T) {
	app := testApp(nil, ConfigEnabled(true), t)
	defer app.Shutdown(10 * time.Second)

	app.HeapHighWaterMarkAlarmEnable(100 * time.Millisecond)
	defer app.HeapHighWaterMarkAlarmShutdown()

	app.HeapHighWaterMarkAlarmDisable()

	app.app.heapHighWaterMarkAlarms.lock.RLock()
	ticker := app.app.heapHighWaterMarkAlarms.sampleTicker
	app.app.heapHighWaterMarkAlarms.lock.RUnlock()

	if ticker == nil {
		t.Error("Expected ticker to still exist after disable")
	}
}

func TestHeapHighWaterMarkAlarmShutdown(t *testing.T) {
	app := testApp(nil, ConfigEnabled(true), t)
	defer app.Shutdown(10 * time.Second)

	app.HeapHighWaterMarkAlarmEnable(100 * time.Millisecond)
	app.HeapHighWaterMarkAlarmSet(1024, func(uint64, *runtime.MemStats) {})
	app.HeapHighWaterMarkAlarmShutdown()

	app.app.heapHighWaterMarkAlarms.lock.RLock()
	ticker := app.app.heapHighWaterMarkAlarms.sampleTicker
	alarms := app.app.heapHighWaterMarkAlarms.alarms
	app.app.heapHighWaterMarkAlarms.lock.RUnlock()

	if ticker != nil || alarms != nil {
		t.Error("Expected ticker and alarms to be nil after shutdown")
	}
}

func TestHeapHighWaterMarkAlarmSet(t *testing.T) {
	app := testApp(nil, ConfigEnabled(true), t)
	defer app.Shutdown(10 * time.Second)

	app.HeapHighWaterMarkAlarmEnable(100 * time.Millisecond)
	defer app.HeapHighWaterMarkAlarmShutdown()

	app.HeapHighWaterMarkAlarmSet(1024, func(uint64, *runtime.MemStats) {})

	if app.app == nil {
		t.Error("app.app is nil")
		return
	}

	app.app.heapHighWaterMarkAlarms.lock.RLock()
	alarms := app.app.heapHighWaterMarkAlarms.alarms
	app.app.heapHighWaterMarkAlarms.lock.RUnlock()

	if alarms == nil || len(alarms) != 1 || alarms[1024] == nil {
		t.Error("Expected one alarm with a valid callback")
	}
}

func TestHeapHighWaterMarkAlarmSetReplaceCallback(t *testing.T) {
	app := testApp(nil, ConfigEnabled(true), t)
	defer app.Shutdown(10 * time.Second)

	app.HeapHighWaterMarkAlarmEnable(100 * time.Millisecond)
	defer app.HeapHighWaterMarkAlarmShutdown()

	app.HeapHighWaterMarkAlarmSet(1024, func(uint64, *runtime.MemStats) {})
	app.HeapHighWaterMarkAlarmSet(1024, func(uint64, *runtime.MemStats) {})

	app.app.heapHighWaterMarkAlarms.lock.RLock()
	alarms := app.app.heapHighWaterMarkAlarms.alarms
	app.app.heapHighWaterMarkAlarms.lock.RUnlock()

	if len(alarms) != 1 {
		t.Error("Expected one alarm after replacing callback")
	}
}

func TestHeapHighWaterMarkAlarmSetRemoveWithNil(t *testing.T) {
	app := testApp(nil, ConfigEnabled(true), t)
	defer app.Shutdown(10 * time.Second)

	app.HeapHighWaterMarkAlarmEnable(100 * time.Millisecond)
	defer app.HeapHighWaterMarkAlarmShutdown()

	app.HeapHighWaterMarkAlarmSet(1024, func(uint64, *runtime.MemStats) {})
	app.HeapHighWaterMarkAlarmSet(1024, nil)

	app.app.heapHighWaterMarkAlarms.lock.RLock()
	alarms := app.app.heapHighWaterMarkAlarms.alarms
	app.app.heapHighWaterMarkAlarms.lock.RUnlock()

	if len(alarms) != 0 {
		t.Error("Expected no alarms after removing with nil")
	}
}

func TestHeapHighWaterMarkAlarmClearAll(t *testing.T) {
	app := testApp(nil, ConfigEnabled(true), t)
	defer app.Shutdown(10 * time.Second)

	app.HeapHighWaterMarkAlarmEnable(100 * time.Millisecond)
	defer app.HeapHighWaterMarkAlarmShutdown()

	app.HeapHighWaterMarkAlarmSet(1024, func(uint64, *runtime.MemStats) {})
	app.HeapHighWaterMarkAlarmSet(2048, func(uint64, *runtime.MemStats) {})
	app.HeapHighWaterMarkAlarmClearAll()

	app.app.heapHighWaterMarkAlarms.lock.RLock()
	alarms := app.app.heapHighWaterMarkAlarms.alarms
	app.app.heapHighWaterMarkAlarms.lock.RUnlock()

	if len(alarms) != 0 {
		t.Error("Expected no alarms after clearing all")
	}
}

func TestHeapHighWaterMarkAlarmClearAllWithNilAlarms(t *testing.T) {
	app := testApp(nil, ConfigEnabled(true), t)
	defer app.Shutdown(10 * time.Second)

	app.HeapHighWaterMarkAlarmEnable(100 * time.Millisecond)
	defer app.HeapHighWaterMarkAlarmShutdown()

	// Ensure alarms is nil
	app.app.heapHighWaterMarkAlarms.lock.Lock()
	app.app.heapHighWaterMarkAlarms.alarms = nil
	app.app.heapHighWaterMarkAlarms.lock.Unlock()

	// Should not panic or error
	app.HeapHighWaterMarkAlarmClearAll()
}
