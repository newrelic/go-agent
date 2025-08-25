// Copyright 2022 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"fmt"
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
	lock           sync.RWMutex     // protects creation of the ticker and access to map
	segLock        sync.RWMutex     // protects access to segment list
	sampleTicker   *time.Ticker     // once made, only read by monitor goroutine
	selected       ProfilingTypeSet // which profiling types we've selected to report
	done           chan byte
	ingestSwitch   chan byte
	outputSwitch   chan string
	switchResult   chan error
	activeSegments map[string]struct{}
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

// AddSegmentToProfiler signals that a segment has started which the profiler should report as being
// in play during all subsequent samples until RemoveSegmentFromProfiler is called with the same segment
// name. If the ConfigProfilingWithSegments(true) option is set, this will automatically be called when
// txn.StartSegment is invoked, but if you start a custom segment in any other way, you'll need to
// call AddSegmentToProfiler manually yourself since otherwise the profiler won't be able to be told
// the segment name to track.
//
// Note that this assumes segment names are unique at any given point in the program's runtime.
func (app *Application) AddSegmentToProfiler(name string) {
	app.app.profiler.segLock.Lock()
	if app.app.profiler.activeSegments == nil {
		app.app.profiler.activeSegments = make(map[string]struct{})
	}
	app.app.profiler.activeSegments[name] = struct{}{}
	app.app.profiler.segLock.Unlock()
}

// RemoveSegmentFromProfiler signals that a segment has terminated and the profiler should stop
// tracking that segment name to collected samples. If the ConfigProfilingWithSegments(true) option is
// set, this will automatically be called when the segment's End method is invoked.
func (app *Application) RemoveSegmentFromProfiler(name string) {
	app.app.profiler.segLock.Lock()
	if app.app.profiler.activeSegments != nil {
		delete(app.app.profiler.activeSegments, name)
	}
	app.app.profiler.segLock.Unlock()
}

func (app *Application) ShutdownProfiler() {
	if app == nil || app.app == nil {
		return
	}
	app.SetProfileSampleInterval(0)
	app.app.profiler.done <- 0
}

// SetProfileSampleInterval adjusts the sample time for profile data.
// If set to 0, the profiler is paused entirely, but its data are not deallocated
// nor are the profiles removed. Calling this method again with a positive interval
// resumes sampling again.
func (app *Application) SetProfileSampleInterval(interval time.Duration) {
	if app == nil || app.app == nil {
		return
	}

	app.app.setProfileSampleInterval(interval)
}

// SetProfileOutputDirectory changes the destination for the profiler's output so that
// all further profile data will be written to disk files in the specified directory
// instead of being sent to an ingest backend endpoint.
//
// This can be useful when debugging locally, if you want to get local profile data, or
// if you want manual control over where profile data gets reported.
func (app *Application) SetProfileOutputDirectory(dirname string) error {
	if app != nil && app.app != nil {
		app.app.profiler.outputSwitch <- dirname
		return <-app.app.profiler.switchResult
	}
	return fmt.Errorf("nil application")
}

// SetProfileOutputOTEL changes the destination for the profiler's output so that
// all further profile data will be written to an OTEL-compatible profiling signal
// endpoint.

func (app *Application) SetProfileOutputOTEL() error {
	if app != nil && app.app != nil {
		app.app.profiler.ingestSwitch <- profileIngestOTEL
		return <-app.app.profiler.switchResult
	}
	return fmt.Errorf("nil application")
}

// SetProfileOutputMELT changes the destination for the profiler's output so that
// all further profile data will be written to a New Relic MELT endpoint as custom
// log events
func (app *Application) SetProfileOutputMELT() error {
	if app != nil && app.app != nil {
		app.app.profiler.ingestSwitch <- profileIngestMELT
		return <-app.app.profiler.switchResult
	}
	return fmt.Errorf("nil application")
}

func (app *app) setProfileSampleInterval(interval time.Duration) {
	app.profiler.lock.Lock()
	defer app.profiler.lock.Unlock()

	if interval <= 0 {
		if app.profiler.sampleTicker != nil {
			app.profiler.sampleTicker.Stop()
		}
		return
	}

	if app.profiler.sampleTicker == nil {
		app.profiler.sampleTicker = time.NewTicker(interval)
		app.profiler.done = make(chan byte)
		app.profiler.ingestSwitch = make(chan byte)
		app.profiler.outputSwitch = make(chan string)
		app.profiler.switchResult = make(chan error)
		go app.profiler.monitor(app)
	} else {
		app.profiler.sampleTicker.Reset(interval)
	}
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
	var cpuData bytes.Buffer
	var err error
	var harvestNumber int64

	reportCPUProfileSamples := func(profileData *bytes.Buffer, eventType string, debug bool) {
		var p *profile.Profile
		if p, err = profile.ParseData(profileData.Bytes()); err == nil {
			attrs := map[string]any{"harvest_seq": 0} // we only get one harvest, at the close
			for sampleNumber, sampleData := range p.Sample {
				attrs["sample_seq"] = sampleNumber
				for i, dataValue := range sampleData.Value {
					attrs[normalizeAttrNameFromSampleValueType(p.SampleType[i].Type, p.SampleType[i].Unit)] = dataValue
				}
				pc.segLock.RLock()
				if pc.activeSegments != nil {
					segmentSeq := 0
					for segmentName, _ := range pc.activeSegments {
						attrs[fmt.Sprintf("segment.%d", segmentSeq)] = segmentName
						segmentSeq++
					}
				}
				pc.segLock.RUnlock()
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
						a.Error("unable to record CPU profiling data as custom event", map[string]any{
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

	closeLocalFiles := func() {
		if profileDestination == profileNilDest {
			// no action needed
		} else if profileDestination == profileLocalFile {
			_ = heap_f.Close()
			_ = goroutine_f.Close()
			_ = threadcreate_f.Close()
			_ = block_f.Close()
			_ = mutex_f.Close()
			if pc.isCPUSelected() {
				pprof.StopCPUProfile()
				_ = cpu_f.Close()
			}
		} else {
			// we're sending to an ingest endpoint of some sort
			if pc.isCPUSelected() {
				pprof.StopCPUProfile()
				reportCPUProfileSamples(&cpuData, "ProfileCPU", false)
				cpuData.Reset()
			}
		}
		profileDestination = profileNilDest
	}
	defer closeLocalFiles()

	for {
		select {
		// To prevent interthread contention without the need for mutexes, we use channels here
		// to let user threads request switching profile output destinations here and only this
		// monitor thread ever writes anything.
		//
		case newDestination := <-pc.ingestSwitch:
			switch newDestination {
			case profileIngestOTEL, profileIngestMELT, profileNilDest:
				if profileDestination == profileLocalFile {
					closeLocalFiles()
				}
				if pc.isCPUSelected() {
					if err = pprof.StartCPUProfile(&cpuData); err != nil {
						pc.switchResult <- err
						return
					}
				}
				profileDestination = newDestination
				pc.switchResult <- nil
			default:
				pc.switchResult <- fmt.Errorf("Invalid profile destination code %v", newDestination)
			}

		case newDirectory := <-pc.outputSwitch:
			var err error

			closeLocalFiles()
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
							pc.segLock.RLock()
							if pc.activeSegments != nil {
								segmentSeq := 0
								for segmentName, _ := range pc.activeSegments {
									attrs[fmt.Sprintf("segment.%d", segmentSeq)] = segmentName
									segmentSeq++
								}
							}
							pc.segLock.RUnlock()
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
				harvestNumber++
			}
		case <-pc.done:
			// We were told to terminate our profile monitoring
			return
		}
	}
}

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
