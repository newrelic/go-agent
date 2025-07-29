// Copyright 2022 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"runtime/pprof"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/pprof/profile"
)

const (
	profileNilDest byte = iota
	profileLocalFile
	profileIngestOTEL
	profileIngestMELT
)

type profilerConfig struct {
	lock         sync.RWMutex     // protects creation of the ticker and access to map
	sampleTicker *time.Ticker     // once made, only read by monitor goroutine
	selected     ProfilingTypeSet // which profiling types we've selected to report
	done         chan byte
	ingestSwitch chan byte
	outputSwitch chan string
	switchResult chan error
}

func (a *app) StartProfiler() {
	if a == nil {
		return
	}

	a.profiler.lock.Lock()
	a.profiler.selected = a.config.Profiling.SelectedProfiles
	a.profiler.lock.Unlock()
	a.setProfileSampleInterval(a.config.Profiling.Interval)
}

func (a *Application) ShutdownProfiler() {
	if a == nil || a.app == nil {
		return
	}
	a.SetProfileSampleInterval(0)
	a.app.profiler.done <- 0
}

// SetProfileSampleInterval adjusts the sample time for profile data.
// If set to 0, the profiler is paused entirely, but its data are not deallocated
// nor are the profiles removed. Calling this method again with a positive interval
// resumes sampling again.
func (a *Application) SetProfileSampleInterval(interval time.Duration) {
	if a == nil || a.app == nil {
		return
	}

	a.app.setProfileSampleInterval(interval)
}

// SetProfileOutputDirectory changes the destination for the profiler's output so that
// all further profile data will be written to disk files in the specified directory
// instead of being sent to an ingest backend endpoint.
//
// This can be useful when debugging locally, if you want to get local profile data, or
// if you want manual control over where profile data gets reported.
func (a *Application) SetProfileOutputDirectory(dirname string) error {
	if a != nil && a.app != nil {
		a.app.profiler.outputSwitch <- dirname
		return <-a.app.profiler.switchResult
	}
	return fmt.Errorf("nil application")
}

// SetProfileOutputOTEL changes the destination for the profiler's output so that
// all further profile data will be written to an OTEL-compatible profiling signal
// endpoint.

func (a *Application) SetProfileOutputOTEL() error {
	if a != nil && a.app != nil {
		a.app.profiler.ingestSwitch <- profileIngestOTEL
		return <-a.app.profiler.switchResult
	}
	return fmt.Errorf("nil application")
}

// SetProfileOutputMELT changes the destination for the profiler's output so that
// all further profile data will be written to a New Relic MELT endpoint as custom
// log events
func (a *Application) SetProfileOutputMELT() error {
	if a != nil && a.app != nil {
		a.app.profiler.ingestSwitch <- profileIngestMELT
		return <-a.app.profiler.switchResult
	}
	return fmt.Errorf("nil application")
}

func (a *app) setProfileSampleInterval(interval time.Duration) {
	a.profiler.lock.Lock()
	defer a.profiler.lock.Unlock()

	if interval <= 0 {
		if a.profiler.sampleTicker != nil {
			a.profiler.sampleTicker.Stop()
		}
		return
	}

	if a.profiler.sampleTicker == nil {
		a.profiler.sampleTicker = time.NewTicker(interval)
		a.profiler.done = make(chan byte)
		a.profiler.ingestSwitch = make(chan byte)
		a.profiler.outputSwitch = make(chan string)
		a.profiler.switchResult = make(chan error)
		//a.profiler.profile = pprof.NewProfile("newrelic/go-agent/v3/profiler")
		go a.profiler.monitor(a)
	} else {
		a.profiler.sampleTicker.Reset(interval)
	}
}

func (a *Application) StartTraceSpan(txn *Transaction) {
	//	if a != nil && a.app != nil && a.app.profiler.sampleTicker != nil && txn != nil {
	//		pprof.Lookup("heap").Add(txn, 2)
	//	}
}

func (a *Application) StopTraceSpan(txn *Transaction) {
	//	if a != nil && a.app != nil && a.app.profiler.sampleTicker != nil && txn != nil {
	//		pprof.Lookup("heap").Remove(txn)
	//	}
}

func (pc *profilerConfig) isBlockSelected() bool {
	return (ProfilingType(pc.selected) & ProfilingTypeBlock) != 0
}

func (pc *profilerConfig) isCPUSelected() bool {
	return (ProfilingType(pc.selected) & ProfilingTypeCPU) != 0
}

func (pc *profilerConfig) isGoroutineSelected() bool {
	return (ProfilingType(pc.selected) & ProfilingTypeGoroutine) != 0
}

func (pc *profilerConfig) isHeapSelected() bool {
	return (ProfilingType(pc.selected) & ProfilingTypeHeap) != 0
}

func (pc *profilerConfig) isMutexSelected() bool {
	return (ProfilingType(pc.selected) & ProfilingTypeMutex) != 0
}

func (pc *profilerConfig) isThreadCreateSelected() bool {
	return (ProfilingType(pc.selected) & ProfilingTypeThreadCreate) != 0
}

func (pc *profilerConfig) isTraceSelected() bool {
	return (ProfilingType(pc.selected) & ProfilingTypeTrace) != 0
}

func (pc *profilerConfig) monitor(a *app) {
	if pc == nil {
		return
	}

	profileDestination := profileNilDest
	var heap_f, goroutine_f, threadcreate_f, block_f, mutex_f, cpu_f *os.File
	var err error
	var harvestNumber int64

	closeLocalFiles := func() {
		_ = heap_f.Close()
		_ = goroutine_f.Close()
		_ = threadcreate_f.Close()
		_ = block_f.Close()
		_ = mutex_f.Close()
		_ = cpu_f.Close()
		if pc.isCPUSelected() {
			pprof.StopCPUProfile()
		}
		profileDestination = profileNilDest
	}
	defer closeLocalFiles()

	for {
		select {
		// To prevent interthread contention without the need for mutexes, we use channels here
		// to let user threads request switching profile output destinations here and only this
		// monitor thread ever writes anything.
		case newDestination := <-pc.ingestSwitch:
			switch newDestination {
			case profileIngestOTEL, profileIngestMELT, profileNilDest:
				if profileDestination == profileLocalFile {
					closeLocalFiles()
				}
				profileDestination = newDestination
				pc.switchResult <- nil
			default:
				pc.switchResult <- fmt.Errorf("Invalid profile destination code %v", newDestination)
			}

		case newDirectory := <-pc.outputSwitch:
			var err error
			if pc.isHeapSelected() {
				if heap_f, err = os.OpenFile(path.Join(newDirectory, "heap.pprof"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
					pc.switchResult <- err
					return
				}
			}
			if pc.isGoroutineSelected() {
				if goroutine_f, err = os.OpenFile(path.Join(newDirectory, "goroutine.pprof"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
					pc.switchResult <- err
					return
				}
			}
			if pc.isThreadCreateSelected() {
				if threadcreate_f, err = os.OpenFile(path.Join(newDirectory, "threadcreate.pprof"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
					pc.switchResult <- err
					return
				}
			}
			if pc.isBlockSelected() {
				if block_f, err = os.OpenFile(path.Join(newDirectory, "block.pprof"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
					pc.switchResult <- err
					return
				}
			}
			if pc.isMutexSelected() {
				if mutex_f, err = os.OpenFile(path.Join(newDirectory, "mutex.pprof"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
					pc.switchResult <- err
					return
				}
			}
			if pc.isCPUSelected() {
				if cpu_f, err = os.OpenFile(path.Join(newDirectory, "cpu.pprof"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
					pc.switchResult <- err
					return
				}
				if err = pprof.StartCPUProfile(cpu_f); err != nil {
					pc.switchResult <- err
					return
				}
			}
			profileDestination = profileLocalFile
			pc.switchResult <- nil

		case <-pc.sampleTicker.C:
			if profileDestination == profileNilDest {
				continue
			}

			if profileDestination == profileLocalFile {
				if pc.isHeapSelected() {
					pprof.Lookup("heap").WriteTo(heap_f, 1)
				}
				if pc.isGoroutineSelected() {
					pprof.Lookup("goroutine").WriteTo(goroutine_f, 1)
				}
				if pc.isThreadCreateSelected() {
					pprof.Lookup("threadcreate").WriteTo(threadcreate_f, 1)
				}
				if pc.isBlockSelected() {
					pprof.Lookup("block").WriteTo(block_f, 1)
				}
				if pc.isMutexSelected() {
					pprof.Lookup("mutex").WriteTo(mutex_f, 1)
				}
			} else {
				// Otherwise, we need to process the profile data internally and report it out somewhere.
				reportProfileSample := func(profileName, eventType string, debug bool) {
					var data bytes.Buffer
					pprof.Lookup(profileName).WriteTo(&data, 0)
					var p *profile.Profile
					if p, err = profile.ParseData(data.Bytes()); err == nil {
						attrs := map[string]any{"harvest_seq": harvestNumber}
						for sampleNumber, sampleData := range p.Sample {
							attrs["sample_seq"] = sampleNumber
							for i, dataValue := range sampleData.Value {
								attrs[normalizeAttrNameFromSampleValueType(p.SampleType[i].Type, p.SampleType[i].Unit)] = dataValue
							}
							for i, codeLoc := range sampleData.Location {
								if codeLoc.Line != nil && len(codeLoc.Line) > 0 {
									attrs[fmt.Sprintf("location.%d", i)] = fmt.Sprintf("%s:%d", codeLoc.Line[0].Function.Name, codeLoc.Line[0].Line)
								}
							}
							attrs["time_ns"] = p.TimeNanos
							attrs["duration_ns"] = p.DurationNanos
							attrs[normalizeAttrNameFromSampleValueType("sample_period_"+p.PeriodType.Type, p.PeriodType.Unit)] = p.Period
							if debug {
								fmt.Printf("EVENT %s: %v\n", eventType, attrs)
							} else {
								if err = a.RecordCustomEvent(eventType, attrs); err != nil {
									a.Error("unable to record heap profiling data as custom event", map[string]any{
										"event-type": eventType,
										"reason":     err.Error(),
									})
								}
							}
						}
					} else {
						if debug {
							fmt.Printf("ERROR parsing %s: %v\n", eventType, err)
						} else {
							a.Error("unable to parse "+eventType+" profiling data", map[string]any{
								"event-type": eventType,
								"reason":     err.Error(),
							})
						}
					}
				}

				if pc.isHeapSelected() {
					reportProfileSample("heap", "ProfileHeap", false)
				}
				if pc.isGoroutineSelected() {
					reportProfileSample("goroutine", "ProfileGoroutine", false)
				}
				if pc.isThreadCreateSelected() {
					reportProfileSample("threadcreate", "ProfileThreadCreate", false)
				}
				if pc.isBlockSelected() {
					reportProfileSample("block", "ProfileBlock", false)
				}
				if pc.isMutexSelected() {
					reportProfileSample("mutex", "ProfileMutex", false)
				}
				if pc.isCPUSelected() {
					// wip
				}
				harvestNumber++
			}
		case <-pc.done:
			log.Println("pc.done!")
			// We were told to terminate our profile monitoring
			return
		}
	}
}

//// HeapHighWaterMarkAlarmShutdown stops the monitoring goroutine and deallocates the entire
//// monitoring completely. All alarms are calcelled and disabled.
//func (a *Application) HeapHighWaterMarkAlarmShutdown() {
//	if a == nil || a.app == nil {
//		return
//	}
//
//	a.app.heapHighWaterMarkAlarms.lock.Lock()
//	defer a.app.heapHighWaterMarkAlarms.lock.Unlock()
//	a.app.heapHighWaterMarkAlarms.sampleTicker.Stop()
//	if a.app.heapHighWaterMarkAlarms.done != nil {
//		a.app.heapHighWaterMarkAlarms.done <- 0
//	}
//	if a.app.heapHighWaterMarkAlarms.alarms != nil {
//		clear(a.app.heapHighWaterMarkAlarms.alarms)
//		a.app.heapHighWaterMarkAlarms.alarms = nil
//	}
//	a.app.heapHighWaterMarkAlarms.sampleTicker = nil
//}
//
//// HeapHighWaterMarkAlarmDisable stops sampling the heap memory allocation started by
//// HeapHighWaterMarkAlarmEnable. It is safe to call even if HeapHighWaterMarkAlarmEnable was
//// never called or the alarms were already disabled.
//func (a *Application) HeapHighWaterMarkAlarmDisable() {
//	if a == nil || a.app == nil {
//		return
//	}
//
//}
//
//// HeapHighWaterMarkAlarmSet adds a heap memory high water mark alarm to the set of alarms
//// being tracked by the running heap monitor. Memory is checked on the interval specified to
//// the last call to HeapHighWaterMarkAlarmEnable, and if at that point the globally allocated heap
//// memory is at least the specified size, the provided callback function will be invoked. This
//// method may be called multiple times to register any number of callback functions to respond
//// to different memory thresholds. For example, you may wish to make measurements or warnings
//// of various urgency levels before finally taking action.
////
//// If HeapHighWaterMarkAlarmSet is called with the same memory limit as a previous call, the
//// supplied callback function will replace the one previously registered for that limit. If
//// the function is given as nil, then that memory limit alarm is removed from the list.
//func (a *Application) HeapHighWaterMarkAlarmSet(limit uint64, f func(uint64, *runtime.MemStats)) {
//	if a == nil || a.app == nil {
//		return
//	}
//
//	a.app.heapHighWaterMarkAlarms.lock.Lock()
//	defer a.app.heapHighWaterMarkAlarms.lock.Unlock()
//
//	if a.app.heapHighWaterMarkAlarms.alarms == nil {
//		a.app.heapHighWaterMarkAlarms.alarms = make(map[uint64]func(uint64, *runtime.MemStats))
//	}
//
//	if f == nil {
//		delete(a.app.heapHighWaterMarkAlarms.alarms, limit)
//	} else {
//		a.app.heapHighWaterMarkAlarms.alarms[limit] = f
//	}
//}
//
//// HeapHighWaterMarkAlarmClearAll removes all high water mark alarms from the memory monitor
//// set.
//func (a *Application) HeapHighWaterMarkAlarmClearAll() {
//	if a == nil || a.app == nil {
//		return
//	}
//
//	a.app.heapHighWaterMarkAlarms.lock.Lock()
//	defer a.app.heapHighWaterMarkAlarms.lock.Unlock()
//
//	if a.app.heapHighWaterMarkAlarms.alarms == nil {
//		return
//	}
//
//	clear(a.app.heapHighWaterMarkAlarms.alarms)
//}

func normalizeAttrNameFromSampleValueType(typeName, unitName string) string {
	return strings.Map(func(s rune) rune {
		if unicode.IsSpace(s) {
			return '_'
		}
		if unicode.IsLetter(s) {
			return s
		}
		if s == '_' {
			return s
		}
		return -1
	}, strings.ToLower(typeName+"_"+unitName))
}
