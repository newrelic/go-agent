// Copyright 2022 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"runtime"
	"sync"
	"time"
)

type heapHighWaterMarkAlarmSet struct {
	lock         sync.RWMutex // protects creation of the ticker and access to map
	sampleTicker *time.Ticker // once made, only read by monitor goroutine
	alarms       map[uint64]func(uint64, *runtime.MemStats)
	done         chan byte
}

// This is a gross, high-level whole-heap memory monitor which can be used to monitor, track,
// and trigger an application's response to running out of memory as an initial step or when
// more expensive or sophisticated analysis such as per-routine memory usage tracking is not
// needed.
//
// For this, we simply configure one or more heap memory limits and for each, register a callback
// function to be called any time we notice that the total heap allocation reaches or exceeds that
// limit. Note that this means if the allocation size crosses multiple limits, then multiple
// callbacks will be triggered since each of their criteria will be met.
//
// HeapHighWaterMarkAlarmEnable starts the periodic sampling of the runtime heap allocation
// of the application, at the user-provided sampling interval. Calling HeapHighWaterMarkAlarmEnable
// with an interval less than or equal to 0 is equivalent to calling HeapHighWaterMarkAlarmDisable.
//
// If there was already a running heap monitor, this merely changes its sample interval time.
func (app *Application) HeapHighWaterMarkAlarmEnable(interval time.Duration) {
	if app == nil || app.app == nil {
		return
	}

	if interval <= 0 {
		app.HeapHighWaterMarkAlarmDisable()
		return
	}

	app.app.heapHighWaterMarkAlarms.lock.Lock()
	defer app.app.heapHighWaterMarkAlarms.lock.Unlock()
	if app.app.heapHighWaterMarkAlarms.sampleTicker == nil {
		app.app.heapHighWaterMarkAlarms.sampleTicker = time.NewTicker(interval)
		app.app.heapHighWaterMarkAlarms.done = make(chan byte)
		go app.app.heapHighWaterMarkAlarms.monitor()
	} else {
		app.app.heapHighWaterMarkAlarms.sampleTicker.Reset(interval)
	}
}

func (as *heapHighWaterMarkAlarmSet) monitor() {
	for {
		select {
		case <-as.sampleTicker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			as.lock.RLock()
			if as.alarms != nil {
				for limit, callback := range as.alarms {
					if m.HeapAlloc >= limit {
						callback(limit, &m)
					}
				}
			}
			as.lock.RUnlock()
		case <-as.done:
			return
		}
	}
}

// HeapHighWaterMarkAlarmShutdown stops the monitoring goroutine and deallocates the entire

// monitoring completely. All alarms are calcelled and disabled.
func (app *Application) HeapHighWaterMarkAlarmShutdown() {
	if app == nil || app.app == nil {
		return
	}

	app.app.heapHighWaterMarkAlarms.lock.Lock()
	defer app.app.heapHighWaterMarkAlarms.lock.Unlock()
	if app.app.heapHighWaterMarkAlarms.sampleTicker != nil {
		app.app.heapHighWaterMarkAlarms.sampleTicker.Stop()
	}
	if app.app.heapHighWaterMarkAlarms.done != nil {
		app.app.heapHighWaterMarkAlarms.done <- 0
	}
	if app.app.heapHighWaterMarkAlarms.alarms != nil {
		clear(app.app.heapHighWaterMarkAlarms.alarms)
		app.app.heapHighWaterMarkAlarms.alarms = nil
	}
	app.app.heapHighWaterMarkAlarms.sampleTicker = nil
}

// HeapHighWaterMarkAlarmDisable stops sampling the heap memory allocation started by
// HeapHighWaterMarkAlarmEnable. It is safe to call even if HeapHighWaterMarkAlarmEnable was
// never called or the alarms were already disabled.
func (app *Application) HeapHighWaterMarkAlarmDisable() {
	if app == nil || app.app == nil {
		return
	}

	app.app.heapHighWaterMarkAlarms.lock.Lock()
	defer app.app.heapHighWaterMarkAlarms.lock.Unlock()
	if app.app.heapHighWaterMarkAlarms.sampleTicker != nil {
		app.app.heapHighWaterMarkAlarms.sampleTicker.Stop()
	}
}

// HeapHighWaterMarkAlarmSet adds a heap memory high water mark alarm to the set of alarms
// being tracked by the running heap monitor. Memory is checked on the interval specified to
// the last call to HeapHighWaterMarkAlarmEnable, and if at that point the globally allocated heap
// memory is at least the specified size, the provided callback function will be invoked. This
// method may be called multiple times to register any number of callback functions to respond
// to different memory thresholds. For example, you may wish to make measurements or warnings
// of various urgency levels before finally taking action.
//
// If HeapHighWaterMarkAlarmSet is called with the same memory limit as a previous call, the
// supplied callback function will replace the one previously registered for that limit. If
// the function is given as nil, then that memory limit alarm is removed from the list.
func (app *Application) HeapHighWaterMarkAlarmSet(limit uint64, f func(uint64, *runtime.MemStats)) {
	if app == nil || app.app == nil {
		return
	}

	app.app.heapHighWaterMarkAlarms.lock.Lock()
	defer app.app.heapHighWaterMarkAlarms.lock.Unlock()

	if app.app.heapHighWaterMarkAlarms.alarms == nil {
		app.app.heapHighWaterMarkAlarms.alarms = make(map[uint64]func(uint64, *runtime.MemStats))
	}

	if f == nil {
		delete(app.app.heapHighWaterMarkAlarms.alarms, limit)
	} else {
		app.app.heapHighWaterMarkAlarms.alarms[limit] = f
	}
}

// HeapHighWaterMarkAlarmClearAll removes all high water mark alarms from the memory monitor
// set.
func (app *Application) HeapHighWaterMarkAlarmClearAll() {
	if app == nil || app.app == nil {
		return
	}

	app.app.heapHighWaterMarkAlarms.lock.Lock()
	defer app.app.heapHighWaterMarkAlarms.lock.Unlock()

	if app.app.heapHighWaterMarkAlarms.alarms == nil {
		return
	}

	clear(app.app.heapHighWaterMarkAlarms.alarms)
}
