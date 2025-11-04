// Copyright 2022 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"sync"
	"time"
	"unicode"

	xtrace "golang.org/x/exp/trace"

	"github.com/google/pprof/profile"
)

const (
	profileNilDest byte = iota
	profileLocalFile
	profileIngestOTEL
	profileIngestMELT
)

type profilerAuditRecord struct {
	EventType  string         `json:"event_type"`
	HarvestSeq int64          `json:"harvest_seq"`
	SampleSeq  int            `json:"sample_seq"`
	Reason     string         `json:"error,omitempty"`
	Attributes int            `json:"attr_count"`
	RawData    map[string]any `json:"raw_data,omitempty"`
}

func auditQty(audit io.Writer, eventType string, harvestNumber int64, samples int) {
	if audit != nil {
		if b, jerr := json.Marshal(profilerAuditRecord{
			EventType:  "INFO_QTY:" + eventType,
			HarvestSeq: harvestNumber,
			Attributes: samples,
		}); jerr == nil {
			audit.Write(b)
			audit.Write([]byte{'\n'})
		}
	}
}

func auditLog(audit io.Writer, format string, data ...any) {
	if audit != nil {
		if b, jerr := json.Marshal(profilerAuditRecord{
			EventType: "INFO",
			Reason:    fmt.Sprintf(format, data...),
		}); jerr == nil {
			audit.Write(b)
			audit.Write([]byte{'\n'})
		}
	}
}

func profilerError(a *app, audit io.Writer, eventType string, harvestSeq int64, err error, debug bool, format string, data ...any) {
	if debug {
		fmt.Printf("ERROR "+format, data...)
		fmt.Println(err.Error())
	} else {
		a.Error(fmt.Sprintf(format, data...), map[string]any{
			"event-type": eventType,
			"reason":     err.Error(),
		})
		if audit != nil {
			auditError(audit, eventType, harvestSeq, err, format, data...)
		}
	}
}

func auditError(audit io.Writer, eventType string, harvestSeq int64, e error, format string, data ...any) {
	if audit != nil {
		m := fmt.Sprintf(format, data...)
		if b, jerr := json.Marshal(profilerAuditRecord{
			EventType:  eventType,
			HarvestSeq: harvestSeq,
			Reason:     fmt.Sprintf("%s: %v", m, e.Error()),
		}); jerr == nil {
			audit.Write(b)
			audit.Write([]byte{'\n'})
		}
	}
}

type profilerConfig struct {
	lock            sync.RWMutex // protects creation of the ticker and access to map
	segLock         sync.RWMutex // protects access to segment list
	sampleTicker    *time.Ticker // once made, only read by monitor goroutine
	cpuReportTicker *time.Ticker // once made, only read by monitor goroutine
	isRunning       bool
	selected        ProfilingType // which profiling types we've selected to report
	auditFile       *os.File      // debugging audit file of profile data (nil for normal production runs)
	done            chan byte
	outputDirectory string
	ingestSwitch    chan byte
	outputSwitch    chan string
	switchResult    chan error
	activeSegments  map[string]struct{}
	blockRate       int
	mutexRate       int
	cpuSampleRateHz int
	ingestEndpoint  string
	ingestClient    *http.Client
	apiKey          string
	serviceName     string
}

func (p *profilerConfig) IsRunning() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.isRunning
}

func (p *profilerConfig) SetRunning(state bool) {
	p.lock.Lock()
	p.isRunning = state
	p.lock.Unlock()
}

func (a *app) StartProfiler() {
	if a == nil {
		return
	}

	a.profiler.lock.Lock()
	a.profiler.selected = a.config.Profiling.SelectedProfiles
	a.profiler.blockRate = a.config.Profiling.BlockRate
	a.profiler.mutexRate = a.config.Profiling.MutexRate
	a.profiler.cpuSampleRateHz = a.config.Profiling.CPUSampleRateHz
	a.profiler.ingestEndpoint = a.config.Profiling.Host
	a.profiler.serviceName = a.config.AppName
	a.profiler.apiKey = a.config.License
	a.profiler.lock.Unlock()
	a.setProfileSampleInterval(a.config.Profiling.Interval)
	a.setProfileCPUReportInterval(a.config.Profiling.CPUReportInterval)
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

// The following are undocumented because they're intended for internal testing
// purposes. They open and close an audit log of the profile samples collected
// and reported for the profiler.

func (app *Application) OpenProfileAuditLog(filename string) error {
	var err error
	if app == nil || app.app == nil {
		return fmt.Errorf("nil application")
	}
	app.app.profiler.lock.Lock()
	app.app.profiler.auditFile, err = os.Create(filename)
	app.app.profiler.lock.Unlock()
	return err
}

func (app *Application) CloseProfileAuditLog() error {
	var err error
	if app == nil || app.app == nil {
		return fmt.Errorf("nil application")
	}
	app.app.profiler.lock.Lock()
	err = app.app.profiler.auditFile.Close()
	app.app.profiler.auditFile = nil
	app.app.profiler.lock.Unlock()
	return err
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

// ShutdownProfiler stops the collection and reporting of profile data and stops the
// monitor background goroutine. If the waitForShutdown parameter is true, it will block
// until the monitor goroutine has completed its final harvest of profile samples and fully
// shut down before returning.
func (app *Application) ShutdownProfiler(waitForShutdown bool) {
	if app == nil || app.app == nil {
		return
	}
	app.SetProfileSampleInterval(0)
	app.app.profiler.done <- 0
	if waitForShutdown {
		for app.app.profiler.IsRunning() {
			time.Sleep(time.Millisecond * 100)
		}
	}
}

// SetProfileCPUSampleRateHz adjusts the sample time for CPU profile data.
// Changing this value does not actually take effect until the next time the
// CPU profiler is restarted. This will be when it is explicitly started, or
// when app.ReportCPUProfileStats is called (either manually or via the periodic
// timer set in motion via the ConfigProfilingCPUReportInterval option).
func (app *Application) SetProfileCPUSampleRateHz(hz int) {
	if app == nil || app.app == nil {
		return
	}
	app.app.profiler.lock.Lock()
	app.app.profiler.cpuSampleRateHz = hz
	app.app.profiler.lock.Unlock()
}

// SetProfileCPUReportInterval adjusts the pace at which we report the collected CPU profile data, just
// like the ConfigProfilingCPUReportInverval agent configuration option does, but this allows the value
// to be adjusted at runtime at will. Setting this to 0 stops the interruption of the CPU profiler, allowing
// it to run until explicitly stopped when the overall agent profiler is shut down.
func (app *Application) SetProfileCPUReportInterval(interval time.Duration) {
	if app == nil || app.app == nil {
		return
	}

	app.app.setProfileCPUReportInterval(interval)
}

func (app *app) setProfileCPUReportInterval(interval time.Duration) {
	app.profiler.lock.Lock()
	defer app.profiler.lock.Unlock()

	if interval <= 0 {
		if app.profiler.cpuReportTicker != nil {
			app.profiler.cpuReportTicker.Stop()
		}
		return
	}

	if app.profiler.cpuReportTicker == nil {
		app.profiler.cpuReportTicker = time.NewTicker(interval)
	} else {
		app.profiler.cpuReportTicker.Reset(interval)
	}
}

// SetProfileSampleInterval adjusts the sample time for profile data.
// If set to 0, the profiler is paused entirely, but its data are not deallocated
// nor are the profiles removed. Calling this method again with a positive interval
// resumes sampling again.
//
// This does not affect sample rates for CPU data. Use SetProfileCPUSampleInterval
// and/or SetProfileCPUReportingInterval for that instead.
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
	return (pc.selected & ProfilingTypeBlock) != 0
}

func (pc *profilerConfig) isCPUSelected() bool {
	return (pc.selected & ProfilingTypeCPU) != 0
}

func (pc *profilerConfig) isGoroutineSelected() bool {
	return (pc.selected & ProfilingTypeGoroutine) != 0
}

func (pc *profilerConfig) isHeapSelected() bool {
	return (pc.selected & ProfilingTypeHeap) != 0
}

func (pc *profilerConfig) isMutexSelected() bool {
	return (pc.selected & ProfilingTypeMutex) != 0
}

func (pc *profilerConfig) isThreadCreateSelected() bool {
	return (pc.selected & ProfilingTypeThreadCreate) != 0
}

func (pc *profilerConfig) isTraceSelected() bool {
	return (pc.selected & ProfilingTypeTrace) != 0
}

func sanitizeProfileEventAttrs(attrs map[string]any) {
	if len(attrs) > customEventAttributeLimit {
		// too many attributes for an event; sacrifice some location names
		for i := len(attrs) - 1; i >= 0 && len(attrs) > customEventAttributeLimit; i-- {
			key := fmt.Sprintf("location.%d", i)
			if _, exists := attrs[key]; exists {
				delete(attrs, key)
			}
		}
	}
	if len(attrs) > customEventAttributeLimit {
		// still too many? kill the span ids too, then, as a move of desperation...
		for i := len(attrs) - 1; i >= 0 && len(attrs) > customEventAttributeLimit; i-- {
			key := fmt.Sprintf("span.%d", i)
			if _, exists := attrs[key]; exists {
				delete(attrs, key)
			}
		}
	}
}

func (pc *profilerConfig) monitor(a *app) {
	if pc == nil {
		return
	}

	pc.SetRunning(true)
	defer pc.SetRunning(false)

	auditLog(pc.auditFile, "monitor started")
	if pc.isBlockSelected() {
		runtime.SetBlockProfileRate(pc.blockRate)
	}
	if pc.isMutexSelected() {
		_ = runtime.SetMutexProfileFraction(pc.mutexRate)
	}

	profileDestination := profileNilDest
	var heap_f, goroutine_f, threadcreate_f, block_f, mutex_f, cpu_f, trace_f *os.File
	var cpuData, traceData bytes.Buffer
	var err error
	var harvestNumber int64

	reportBufferedTraceSamples := func(profileData *bytes.Buffer, eventType string, debug bool, audit io.Writer) {
		var err error
		var event xtrace.Event

		reader, err := xtrace.NewReader(profileData)
		if err != nil {
			profilerError(a, pc.auditFile, eventType, harvestNumber, err, debug, "cannot create trace reader")
			return
		}
		sampleNumber := 0
		for {
			event, err = reader.ReadEvent()
			if err == io.EOF {
				break
			}
			if err != nil {
				profilerError(a, pc.auditFile, eventType, harvestNumber, err, debug, "error reading trace events")
				return
			}
			attrs := map[string]any{
				"harvest_seq": harvestNumber,
				"sample_seq":  sampleNumber,
				"kind":        event.Kind().String(),
			}
			sampleNumber++

			switch event.Kind() {
			case xtrace.EventLog:
				elog := event.Log()
				attrs["task"] = uint64(elog.Task)
				attrs["category"] = elog.Category
				attrs["message"] = elog.Message
			case xtrace.EventMetric:
				em := event.Metric()
				attrs["name"] = em.Name
				if em.Value.Kind() == xtrace.ValueUint64 {
					attrs["value"] = em.Value.Uint64()
				} else if em.Value.Kind() == xtrace.ValueString {
					attrs["value"] = em.Value.String()
				}

				/* event.Kind().String() EventSync EventMetric EventLabel EventStackSample EventRangeBegin EventRangeActive EventRangeEnd EventTaskBegin EventTaskEnd EventRegionBegin EventRegionEnd EventLog EventStateTransition EventExperimental*/
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
			sanitizeProfileEventAttrs(attrs)
			if debug {
				fmt.Printf("EVENT %s: %v\n", eventType, attrs)
			} else {
				if err = a.RecordCustomEvent(eventType, attrs); err != nil {
					profilerError(a, pc.auditFile, eventType, harvestNumber, err, debug, "unable to record profiling data as custom event")
				} else if audit != nil {
					// the custom event succeeded. add that to the audit trail too
					if b, jerr := json.Marshal(profilerAuditRecord{
						EventType:  eventType,
						HarvestSeq: harvestNumber,
						SampleSeq:  sampleNumber,
					}); jerr == nil {
						audit.Write(b)
						audit.Write([]byte{'\n'})
					}
				}
			}
		}
	}
	reportBufferedProfileSamples := func(profileData *bytes.Buffer, eventType string, debug bool, audit io.Writer) {
		if profileDestination == profileIngestOTEL {
			pc.sendOTELProfileRawData(eventType, eventType, profileData)
			return
		}

		var p *profile.Profile
		if p, err = profile.ParseData(profileData.Bytes()); err == nil {
			auditQty(audit, eventType, harvestNumber, len(p.Sample))
			for sampleNumber, sampleData := range p.Sample {
				attrs := map[string]any{
					"harvest_seq": harvestNumber,
					"sample_seq":  sampleNumber,
				}
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
				sanitizeProfileEventAttrs(attrs)
				if debug {
					fmt.Printf("EVENT %s: %v\n", eventType, attrs)
				} else {
					if err = a.RecordCustomEvent(eventType, attrs); err != nil {
						a.Error("unable to record profiling data as custom event", map[string]any{
							"event-type": eventType,
							"reason":     err.Error(),
						})
						if audit != nil {
							// add note in our audit record that we failed to record this sample
							if b, jerr := json.Marshal(profilerAuditRecord{
								EventType:  eventType,
								HarvestSeq: harvestNumber,
								SampleSeq:  sampleNumber,
								Reason:     err.Error(),
							}); jerr == nil {
								audit.Write(b)
								audit.Write([]byte{'\n'})
							}
						}
					} else if audit != nil {
						// the custom event succeeded. add that to the audit trail too
						if b, jerr := json.Marshal(profilerAuditRecord{
							EventType:  eventType,
							HarvestSeq: harvestNumber,
							SampleSeq:  sampleNumber,
						}); jerr == nil {
							audit.Write(b)
							audit.Write([]byte{'\n'})
						}
					}
				}
			}
		} else {
			if debug {
				fmt.Printf("ERROR parsing %s: %v\n", eventType, err)
			} else {
				a.Error("unable to parse profiling data", map[string]any{
					"event-type": eventType,
					"reason":     err.Error(),
				})
				if audit != nil {
					if b, jerr := json.Marshal(profilerAuditRecord{
						EventType:  eventType,
						HarvestSeq: harvestNumber,
						Reason:     err.Error(),
					}); jerr == nil {
						audit.Write(b)
						audit.Write([]byte{'\n'})
					}
				}
			}
		}
	}

	closeLocalFiles := func() {
		auditLog(pc.auditFile, "closeLocalFiles called")
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
			if pc.isTraceSelected() {
				trace.Stop()
				_ = trace_f.Close()
			}
		} else {
			// we're sending to an ingest endpoint of some sort
			if pc.isCPUSelected() {
				pprof.StopCPUProfile()
				reportBufferedProfileSamples(&cpuData, "ProfileCPU", false, pc.auditFile)
				cpuData.Reset()
			}
			if pc.isTraceSelected() {
				trace.Stop()
				reportBufferedTraceSamples(&traceData, "ProfileTrace", false, pc.auditFile)
				traceData.Reset()
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
					runtime.SetCPUProfileRate(pc.cpuSampleRateHz)
					if err = pprof.StartCPUProfile(&cpuData); err != nil {
						pc.switchResult <- err
						return
					}
				}
				if pc.isTraceSelected() {
					if err = trace.Start(&traceData); err != nil {
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

			pc.outputDirectory = newDirectory
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
			if pc.isTraceSelected() {
				if trace_f, err = os.OpenFile(path.Join(newDirectory, "trace.pprof"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
					pc.switchResult <- err
					return
				}
				if err = trace.Start(trace_f); err != nil {
					pc.switchResult <- err
					return
				}
			}
			if pc.isCPUSelected() {
				if cpu_f, err = os.OpenFile(path.Join(newDirectory, "cpu.pprof"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
					pc.switchResult <- err
					return
				}
				runtime.SetCPUProfileRate(pc.cpuSampleRateHz)
				if err = pprof.StartCPUProfile(cpu_f); err != nil {
					pc.switchResult <- err
					return
				}
			}
			profileDestination = profileLocalFile
			pc.switchResult <- nil

		case <-pc.cpuReportTicker.C:
			if pc.isCPUSelected() {
				if profileDestination == profileNilDest {
					// nothing to do here
				} else {
					// shut down the profiler, let it report out
					pprof.StopCPUProfile()
					runtime.SetCPUProfileRate(pc.cpuSampleRateHz)
					if profileDestination == profileLocalFile {
						// cycle to a new destination file
						_ = cpu_f.Close()
						if cpu_f, err = os.OpenFile(path.Join(pc.outputDirectory, fmt.Sprintf("cpu.pprof%v", harvestNumber)), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
							a.Error("error restarting CPU profiler", map[string]any{
								"event-type": "ProfileCPU",
								"filename":   fmt.Sprintf("cpu.pprof%v", harvestNumber),
								"operation":  "OpenFile",
								"reason":     err.Error(),
							})
							pc.selected &= ^ProfilingTypeCPU
						} else if err = pprof.StartCPUProfile(cpu_f); err != nil {
							a.Error("error restarting CPU profiler", map[string]any{
								"event-type": "ProfileCPU",
								"operation":  "StartCPUProfile",
								"reason":     err.Error(),
							})
							pc.selected &= ^ProfilingTypeCPU
						}
					} else {
						// report to ingest endpoint
						reportBufferedProfileSamples(&cpuData, "ProfileCPU", false, pc.auditFile)
						cpuData.Reset()
						if err = pprof.StartCPUProfile(&cpuData); err != nil {
							a.Error("error restarting CPU profiler", map[string]any{
								"event-type": "ProfileCPU",
								"operation":  "StartCPUProfile",
								"reason":     err.Error(),
							})
							pc.selected &= ^ProfilingTypeCPU
						}
					}
				}
				harvestNumber++
			}

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
				// The tracer writes to the file on its own, as does the cpu profiler,
				// so we don't need to do anything for them here.
			} else {
				// Otherwise, we need to process the profile data internally and report it out somewhere.
				reportProfileSample := func(profileName, eventType string, debug bool, audit io.Writer) {
					var data bytes.Buffer
					pprof.Lookup(profileName).WriteTo(&data, 0)
					if profileDestination == profileIngestOTEL {
						pc.sendOTELProfileRawData(profileName, eventType, &data)
						return
					}
					var p *profile.Profile
					if p, err = profile.ParseData(data.Bytes()); err == nil {
						auditQty(audit, eventType, harvestNumber, len(p.Sample))
						for sampleNumber, sampleData := range p.Sample {
							attrs := map[string]any{
								"harvest_seq": harvestNumber,
								"sample_seq":  sampleNumber,
							}
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
							sanitizeProfileEventAttrs(attrs)
							if debug {
								fmt.Printf("EVENT %s: %v\n", eventType, attrs)
							} else {
								if err = a.RecordCustomEvent(eventType, attrs); err != nil {
									a.Error("unable to record "+eventType+" profiling data as custom event", map[string]any{
										"event-type": eventType,
										"reason":     err.Error(),
									})
									if audit != nil {
										// add not in our audit record that we failed to record this sample
										if b, jerr := json.Marshal(profilerAuditRecord{
											EventType:  eventType,
											HarvestSeq: harvestNumber,
											SampleSeq:  sampleNumber,
											Reason:     err.Error(),
											Attributes: len(attrs),
											RawData:    attrs,
										}); jerr == nil {
											audit.Write(b)
											audit.Write([]byte{'\n'})
										}
									}
								} else if audit != nil {
									// the custom event succeeded. add that to the audit trail too
									if b, jerr := json.Marshal(profilerAuditRecord{
										EventType:  eventType,
										HarvestSeq: harvestNumber,
										SampleSeq:  sampleNumber,
										Attributes: len(attrs),
										RawData:    attrs,
									}); jerr == nil {
										audit.Write(b)
										audit.Write([]byte{'\n'})
									}
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
							if audit != nil {
								if b, jerr := json.Marshal(profilerAuditRecord{
									EventType:  eventType,
									HarvestSeq: harvestNumber,
									Reason:     err.Error(),
								}); jerr == nil {
									audit.Write(b)
									audit.Write([]byte{'\n'})
								}
							}
						}
					}
				}

				if pc.isHeapSelected() {
					reportProfileSample("heap", "ProfileHeap", false, pc.auditFile)
				}
				if pc.isGoroutineSelected() {
					reportProfileSample("goroutine", "ProfileGoroutine", false, pc.auditFile)
				}
				if pc.isThreadCreateSelected() {
					reportProfileSample("threadcreate", "ProfileThreadCreate", false, pc.auditFile)
				}
				if pc.isBlockSelected() {
					reportProfileSample("block", "ProfileBlock", false, pc.auditFile)
				}
				if pc.isMutexSelected() {
					reportProfileSample("mutex", "ProfileMutex", false, pc.auditFile)
				}
				// We don't get trace data until we stop the profiler (but we can use the flight recorder instead
				// if we need that)
				harvestNumber++
			}
		case <-pc.done:
			// We were told to terminate our profile monitoring
			auditLog(pc.auditFile, "monitor stopped")
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

func (pc *profilerConfig) sendOTELProfileRawData(profileName, eventType string, data *bytes.Buffer) {
	if pc.ingestClient == nil {
		pc.ingestClient = &http.Client{Transport: &http.Transport{
			DisableKeepAlives:  false,
			MaxIdleConns:       0,
			IdleConnTimeout:    0,
			DisableCompression: true, // our data format is already compressed
		}}
		if pc.auditFile != nil {
			auditLog(pc.auditFile, "OTEL ingest endpoint transport setup to %s", pc.ingestEndpoint)
		}
	}
	req, err := http.NewRequest(http.MethodPost, pc.ingestEndpoint, data)
	if err != nil {
		auditLog(pc.auditFile, "error creating HTTP request to send %s profile data to %s: %v", profileName, pc.ingestEndpoint, err)
		return
	}

	req.Header.Add("api-key", pc.apiKey)
	req.Header.Add("service-name", pc.serviceName)
	req.Header.Add("profile-type", profileName)
	req.Header.Add("event-type", eventType)
	req.Header.Add("content-type", "application/octet-stream")

	response, err := pc.ingestClient.Do(req)
	if err != nil {
		auditLog(pc.auditFile, "error sending %s profile data to %s: %v", profileName, pc.ingestEndpoint, err)
		return
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		auditLog(pc.auditFile, "HTTP response %d sending %s profile data to %s: %v", response.StatusCode, profileName, pc.ingestEndpoint, err)
	}
	if _, err := io.ReadAll(response.Body); err != nil {
		auditLog(pc.auditFile, "error reading response from sending %s profile data to %s: %v", profileName, pc.ingestEndpoint, err)
	}
	if err := response.Body.Close(); err != nil {
		auditLog(pc.auditFile, "error closing response from sending %s profile data to %s: %v", profileName, pc.ingestEndpoint, err)
	}
	if pc.auditFile != nil {
		auditLog(pc.auditFile, "posted %s data -> %d %s", profileName, response.StatusCode, response.Status)
		fmt.Printf("posted %s data -> %s", profileName, response.Status)
	}
}
