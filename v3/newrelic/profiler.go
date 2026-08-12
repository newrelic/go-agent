// Copyright 2022 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/btree"
	"github.com/google/pprof/profile"
)

const (
	profileNilDest byte = iota
	profileLocalFile
	profileIngestPPROF
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

// pfofileSpanData holds each active transaction we noticed happening while we were doing
// CPU profiling. We hold onto these in a cache until we report out the CPU profiles we
// collect and know we've attached these to the outgoing samples that were collected at the
// same time frames the transactions were active.
// We have to cache them away because the CPU profiler runs asynchronously (and in a totally
// different system facility) from us, and we'll collect its output later, possibly after these
// transactions have finished.
//
// The combination of TimeNanos+TxnID must be unique within the cache of span data we're holding.
// The data are kept ordered by that combination of values as well. "TxnID" here is somewhat
// arbitrary and for our purposes is created (by us here) from the transaction name and internal
// ID if possible, and from the memory object pointer as a last resort.
type profileSpanData struct {
	TimeNanos     int64  // nanoseconds on clock of span start since Jan 1 1970 UTC
	DurationNanos int64  // nanoseconds span was alive or 0 if still running
	TxnID         string // transaction ID
	TxnName       string // transaction Name
	SpanID        string // trace spanID
	TraceID       string // trace traceID
}

// Derive our unique key from transaction pointer
func profileSpanDataID(txn *Transaction) string {
	TxnID := txn.thread.TxnID + ":" + txn.Name()
	if TxnID == ":" {
		TxnID = fmt.Sprintf("TXN<%v>", txn)
	}

	return strings.Map(func(s rune) rune {
		if unicode.IsSpace(s) {
			return '_'
		}
		return s
	}, TxnID)
}

// Return a new profileSpanData value from an existing transaction, start time, and associated
// TraceMetadata, ready to use in our cache.
func profileSpanDataFromTxn(txn *Transaction) profileSpanData {
	md := txn.GetTraceMetadata()
	return profileSpanData{
		TimeNanos: txn.thread.Start.UnixNano(),
		TxnID:     profileSpanDataID(txn),
		TxnName:   txn.Name(),
		SpanID:    md.SpanID,
		TraceID:   md.TraceID,
	}
}

// End records the end time of a profileSpanData value in our cache, possibly also
// updating the TraceMetadata (in case that's changed or in case it wasn't available
// at the time the transaction was first created).
func (sd *profileSpanData) End(duration time.Duration, md TraceMetadata) {
	sd.DurationNanos = duration.Nanoseconds()

	// Force non-zero just in case somehow the span lasted less than a nanosecond, since
	// zero here means it's still running.
	if sd.DurationNanos == 0 {
		sd.DurationNanos = 1
	}

	// Only update the metadata if what is passed to this method actually has new data
	if md.SpanID != "" && sd.SpanID != md.SpanID {
		sd.SpanID = md.SpanID
	}
	if md.TraceID != "" && sd.TraceID != md.TraceID {
		sd.TraceID = md.TraceID
	}
}

// LessThan takes a pair of profileSpanData values and returns True if the first of them
// is less than the second. This is used by the btree library to order the cache data and
// to iterate over and search through them.
func PSDLessThan(a, b profileSpanData) bool {
	if a.TimeNanos != b.TimeNanos {
		return a.TimeNanos < b.TimeNanos
	}
	return a.TxnID < b.TxnID
}

type profilerConfig struct {
	lock              sync.RWMutex // protects creation of the ticker and access to map
	segLock           sync.RWMutex // protects access to segment list
	sampleTicker      *time.Ticker // once made, only read by monitor goroutine
	cpuReportTicker   *time.Ticker // once made, only read by monitor goroutine
	delayToStart      time.Duration
	delayToStop       time.Duration
	isRunning         bool
	matchSpans        bool
	selected          ProfilingType // which profiling types we've selected to report
	auditFile         *os.File      // debugging audit file of profile data (nil for normal production runs)
	done              chan byte
	outputDirectory   string
	outputDebug       int
	ingestSwitch      chan byte
	outputSwitch      chan string
	switchResult      chan error
	spanCache         *btree.BTreeG[profileSpanData]
	blockRate         int
	mutexRate         int
	cpuSampleRateHz   int
	ingestClient      *http.Client
	apiKey            string
	serviceName       string
	hostname          string
	entityGUID        string
	methodRpmCmd      rpmCmd
	methodRpmControls rpmControls
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

func (a *app) UpdateProfiler() {
	if a == nil {
		return
	}
	a.Debug("trying to get profiler configuration data", nil)

	reply, err := a.getState()
	if err == nil {
		a.profiler.lock.Lock()
		a.profiler.entityGUID = reply.Reply.EntityGUID
		a.profiler.methodRpmCmd.Collector = reply.Reply.Collector
		a.profiler.methodRpmCmd.RunID = reply.Reply.RunID.String()
		a.profiler.methodRpmCmd.RequestHeadersMap = reply.Reply.RequestHeadersMap
		a.profiler.methodRpmCmd.MaxPayloadSize = reply.Reply.MaxPayloadSizeInBytes
		a.profiler.methodRpmCmd.MethodParams = make(map[string]string)
		a.profiler.lock.Unlock()
		if a.run != nil && a.profiler.ableToSend() && !a.run.Config.Profiling.Enabled {
			// we got disabled, probably from server-side config
			a.Info("shutting down profiler due to configuration change", nil)
			a.ShutdownProfiler()
			return
		}
	}
}

func (p *profilerConfig) ableToSend() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.methodRpmCmd.Collector != "" && p.methodRpmCmd.RunID != ""
}

func (a *app) StartProfiler() {
	if a == nil {
		return
	}
	if a.config.HighSecurity {
		a.config.Logger.Error("refusing to start profiler in high security mode", nil)
		return
	}

	if a.profiler.IsRunning() {
		return
	}

	a.profiler.lock.Lock()
	a.profiler.delayToStart = a.config.Profiling.Delay
	a.profiler.delayToStop = a.config.Profiling.Duration
	a.profiler.selected = a.config.Profiling.SelectedProfiles
	a.profiler.blockRate = a.config.Profiling.BlockRate
	a.profiler.mutexRate = a.config.Profiling.MutexRate
	a.profiler.cpuSampleRateHz = a.config.Profiling.CPUSampleRateHz
	a.profiler.serviceName = a.config.AppName
	a.profiler.apiKey = a.config.License
	reply, err := a.getState()
	a.profiler.hostname = a.config.hostname
	a.profiler.matchSpans = a.config.Profiling.MatchSpans
	if err == nil {
		a.profiler.entityGUID = reply.Reply.EntityGUID
		a.profiler.methodRpmCmd.Collector = reply.Reply.Collector
		a.profiler.methodRpmCmd.RunID = reply.Reply.RunID.String()
		a.profiler.methodRpmCmd.RequestHeadersMap = reply.Reply.RequestHeadersMap
		a.profiler.methodRpmCmd.MaxPayloadSize = reply.Reply.MaxPayloadSizeInBytes
	}
	a.profiler.methodRpmControls.License = a.rpmControls.License // string value; make local copy we can use
	a.profiler.methodRpmControls.Client = a.rpmControls.Client   // *http.Client is goroutine-safe for us to use concurrently
	a.profiler.methodRpmControls.Logger = a.rpmControls.Logger   // logger.Logger is goroutine-safe for us to use concurrently
	a.profiler.methodRpmControls.GzipWriterPool = nil            // our data is already gzip compressed so don't let our collecter request code compress it again
	a.profiler.lock.Unlock()
	a.setProfileSampleInterval(a.config.Profiling.Interval)
	a.setProfileCPUReportInterval(a.config.Profiling.CPUReportInterval)
	a.profiler.methodRpmCmd.Name = cmdPprofData
}

// ProfilerStartSpan notifies the CPU profiler that a transaction has started, so that it can later
// be associated with any CPU profile data collected during that transaction. Normally you won't need
// to call this; the agent's transaction-handling code will automatically call this when setting up
// the transaction if CPU profiling is enabled and running at the time.
func (app *Application) ProfilerStartSpan(txn *Transaction) {
	if app == nil {
		return
	}
	app.app.profilerStartSpan(txn)
}

func (app *app) profilerStartSpan(txn *Transaction) {
	if txn == nil {
		app.Error("ProfilerStartSpan called on nil transaction", nil)
		return
	}
	if !app.profiler.matchSpans {
		return
	}
	newSpanDatum := profileSpanDataFromTxn(txn)
	app.profiler.segLock.Lock()
	if app.profiler.spanCache == nil {
		app.profiler.spanCache = btree.NewG[profileSpanData](64, PSDLessThan)
	}
	app.profiler.spanCache.ReplaceOrInsert(newSpanDatum)
	app.profiler.segLock.Unlock()
	app.Debug("profiler: recorded transaction", map[string]any{
		"txn-id":            newSpanDatum.TxnID,
		"span_id":           newSpanDatum.SpanID,
		"trace_id":          newSpanDatum.TraceID,
		"cache-size":        app.profiler.spanCache.Len(),
		"start-time":        newSpanDatum.TimeNanos,
		"start-time-string": time.Unix(newSpanDatum.TimeNanos/1_000_000_000, newSpanDatum.TimeNanos%1_000_000_000).String(),
	})
}

// ProfilerEndSpan notifies the profiler that a transaction which it had previously been notified
// about via ProfilerStartSpan has now ended. This will record the total duration of the transaction
// and the span and trace ID associated with it for association with CPU profile data also being
// reported by the CPU profiler. Normally, you won't need to call this yourself, since the agent's
// transaction handling code will call it for you when you end the transaction, if the CPU profiler
// is enabled and running at the time.
func (app *Application) ProfilerEndSpan(txn *Transaction) {
	if app == nil {
		return
	}
	if txn == nil || txn.thread == nil {
		app.app.Error("ProfilerEndSpan called on nil transaction", nil)
		return
	}
	if !app.app.profiler.matchSpans {
		return
	}
	key := profileSpanDataID(txn)
	app.app.profiler.segLock.Lock()
	defer app.app.profiler.segLock.Unlock()
	if app.app.profiler.spanCache == nil {
		// we don't even have the btree at all, so we already know we have nothing to do.
		return
	}

	targetValue, isInCache := app.app.profiler.spanCache.Get(profileSpanData{
		TimeNanos: txn.thread.Start.UnixNano(),
		TxnID:     key,
	})
	if isInCache {
		targetValue.End(txn.thread.Duration, txn.GetTraceMetadata())
		app.app.profiler.spanCache.ReplaceOrInsert(targetValue)
		app.app.Debug("profiler: ended transaction", map[string]any{
			"txn-id":            targetValue.TxnID,
			"span_id":           targetValue.SpanID,
			"trace_id":          targetValue.TraceID,
			"cache-size":        app.app.profiler.spanCache.Len(),
			"start-time":        targetValue.TimeNanos,
			"duration":          targetValue.DurationNanos,
			"start-time-string": time.Unix(targetValue.TimeNanos/1_000_000_000, targetValue.TimeNanos%1_000_000_000).String(),
			"duration-ms":       targetValue.DurationNanos / 1_000_000,
		})
	} else {
		// we didn't find the exact match we were hoping for. Search the cache to see if we
		// can find the transaction ID, possibly with a different start time
		// TODO: remove this warning after preview release
		app.app.Warn("ProfilerEndSpan unable to directly target transaction; searching cache", map[string]any{
			"key":     key,
			"EndTime": txn.thread.Stop.UnixNano(),
		})

		app.app.profiler.spanCache.Ascend(func(sd profileSpanData) bool {
			if sd.TxnID == key {
				sd.End(txn.thread.Duration, txn.GetTraceMetadata())
				app.app.profiler.spanCache.ReplaceOrInsert(sd)
				app.app.Debug("profiler: ended transaction", map[string]any{
					"txn-id":            sd.TxnID,
					"span_id":           sd.SpanID,
					"trace_id":          sd.TraceID,
					"cache-size":        app.app.profiler.spanCache.Len(),
					"start-time":        sd.TimeNanos,
					"duration":          sd.DurationNanos,
					"start-time-string": time.Unix(sd.TimeNanos/1_000_000_000, sd.TimeNanos%1_000_000_000).String(),
					"duration-ms":       sd.DurationNanos / 1_000_000,
				})
				return false
			}
			return true
		})
	}
}

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

func (app *Application) SetProfileOutputDebugLevel(level int) {
	if app == nil || app.app == nil {
		return
	}
	app.app.profiler.outputDebug = level
}

// ShutdownProfiler stops the collection and reporting of profile data and stops the
// monitor background goroutine. If the waitForShutdown parameter is true, it will block
// until the monitor goroutine has completed its final harvest of profile samples and fully
// shut down before returning.
func (app *Application) ShutdownProfiler(waitForShutdown bool) {
	if app == nil || app.app == nil {
		return
	}
	app.app.ShutdownProfiler()
	if waitForShutdown {
		for app.app.profiler.IsRunning() {
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func (app *app) ShutdownProfiler() {
	app.setProfileSampleInterval(0)
	if app.profiler.IsRunning() {
		app.profiler.done <- 0
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

// SetProfileOutputPPROF changes the destination for the profiler's output so that
// all further profile data will be written to a PPROF-compatible profiling signal
// endpoint. It also refreshes the linking metadata including entityGUID in case
// this needs to be updated.
func (app *Application) SetProfileOutputPPROF() error {
	if app != nil && app.app != nil {
		md := app.GetLinkingMetadata()
		app.app.profiler.lock.Lock()
		defer app.app.profiler.lock.Unlock()
		app.app.profiler.hostname = md.Hostname
		app.app.profiler.entityGUID = md.EntityGUID
		app.app.profiler.ingestSwitch <- profileIngestPPROF
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

func localProfileFileName(ptype string) string {
	now := time.Now()
	return fmt.Sprintf("%s.%04d-%03d-%02d%02d.*.pprof", ptype, now.Year(), now.YearDay(), now.Hour(), now.Minute())
}

func (pc *profilerConfig) monitor(a *app) {
	if pc == nil {
		return
	}

	if pc.delayToStart > 0 {
		a.Info(fmt.Sprintf("profiler delaying startup %s", a.config.Config.Profiling.Delay.String()), nil)
		time.Sleep(pc.delayToStart)
		a.Info("profiler starting after delay", nil)
	}

	var autoShutdown *time.Timer
	if pc.delayToStop > 0 {
		a.Info(fmt.Sprintf("profiler will run for %s before stopping", a.config.Config.Profiling.Duration.String()), nil)
		autoShutdown = time.NewTimer(pc.delayToStop)
	} else {
		autoShutdown = time.NewTimer(0)
		if !autoShutdown.Stop() {
			<-autoShutdown.C
		}
	}
	a.Info("profiler starting", nil)

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
	var heap_f, goroutine_f, threadcreate_f, block_f, mutex_f, cpu_f *os.File //trace_f *os.File
	var cpuData bytes.Buffer
	//traceData
	var err error
	var harvestNumber int64

	go func() {
		pc.ingestSwitch <- profileIngestPPROF
		err := <-pc.switchResult
		a.Debug("profiler: launching ingest switch", map[string]any{"err": err})
		if err != nil {
			a.Error("unable to switch to default PPROF ingest destination", map[string]any{
				"reason": err,
			})
		}
	}()

	reportBufferedProfileSamples := func(profileData *bytes.Buffer, eventType string, debug bool, audit io.Writer) {
		p, err := profile.ParseData(profileData.Bytes())
		if err != nil {
			a.Error("profiler: unable to parse profile data to inject span information", map[string]any{
				"event-type": eventType,
				"reason":     err.Error(),
			})
		} else {
			for _, sample := range p.Sample {
				var span_id, trace_id string
				if sample != nil {
					if len(sample.Label) > 0 {
						if s, ok := sample.Label["span_id"]; ok {
							span_id = s[0]
						}
						if s, ok := sample.Label["trace_id"]; ok {
							trace_id = s[0]
						}
					}
					if span_id != "" || trace_id != "" {
						var location string

						if len(sample.Location) > 0 {
							var locs []string
							for _, l := range sample.Location {
								if l != nil && l.Mapping != nil {
									locs = append(locs, l.Mapping.File)
								}
							}
							location = strings.Join(locs, ", ")
						}

						a.Debug("profiler: sample includes embedded span data", map[string]any{
							"event-type": eventType,
							"location":   location,
							"span_id":    span_id,
							"trace_id":   trace_id,
						})
					}
				}
			}
		}

		if eventType == "cpu" && pc.matchSpans && pc.spanCache != nil {
			// inject cached span IDs into CPU samples before reporting them
			spanIDs := make([]string, 0, 2)
			traceIDs := make([]string, 0, 2)
			p, err := profile.ParseData(profileData.Bytes())
			if err != nil {
				a.Error("profiler: unable to parse profile data to inject span information", map[string]any{
					"event-type": eventType,
					"reason":     err.Error(),
				})
			} else {
				// Search for all spans covering this time frame
				// Because we don't know what spans may be long-running from a while ago, we need
				// to start at the earliest end of the tree. We know when to *stop* searching, just
				// not when to *start*. So the optimization here is to occasionally clean up the btree
				// when we don't need these in the cache anymore.
				trashList := make([]profileSpanData, 0, 64)
				pc.segLock.Lock()
				pc.spanCache.Ascend(func(sd profileSpanData) bool {
					if sd.TimeNanos > p.TimeNanos+p.DurationNanos {
						return false
					}
					if sd.DurationNanos == 0 || sd.TimeNanos+sd.DurationNanos >= p.TimeNanos {
						spanIDs = append(spanIDs, sd.SpanID)
						traceIDs = append(traceIDs, sd.TraceID)
					} else {
						// this item exists entirely before the time this profile covers.
						// assuming we receive all our profile data in order, we can discard this one
						// but we can't do it until we stop iterating over the tree, so save it for now.
						trashList = append(trashList, sd)
					}
					return true
				})
				if len(trashList) > 0 {
					a.Debug("profiler: purging old transaction data from cache", map[string]any{
						"event-type":      eventType,
						"spans-recorded":  len(spanIDs),
						"cache-size":      pc.spanCache.Len(),
						"items-to-remove": len(trashList),
					})
					for i, oldItem := range trashList {
						if _, removed := pc.spanCache.Delete(oldItem); !removed {
							a.Debug("profiler: failed to remove item from cache", map[string]any{
								"event-type":      eventType,
								"spans-recorded":  len(spanIDs),
								"cache-size":      pc.spanCache.Len(),
								"items-to-remove": len(trashList),
								"failed-index":    i,
								"failed-id":       oldItem.TxnID,
							})
						}
					}
				}
				pc.segLock.Unlock()

				if len(spanIDs) == 1 && len(traceIDs) == 1 {
					//TODO: more robust encoding here
					p.SetLabel("span_id", spanIDs)
					p.SetLabel("trace_id", traceIDs)

					profileData.Reset()
					p.Write(profileData)
					a.Debug("profiler: profile data with labels recorded", map[string]any{
						"event-type":        eventType,
						"profile-data":      p.String(),
						"spans-recorded":    len(spanIDs),
						"cache-size":        pc.spanCache.Len(),
						"start-time":        p.TimeNanos,
						"duration":          p.DurationNanos,
						"start-time-string": time.Unix(p.TimeNanos/1_000_000_000, p.TimeNanos%1_000_000_000).String(),
						"duration-ms":       p.DurationNanos / 1_000_000,
					})
				} else if len(spanIDs) > 1 {
					a.Debug("profiler: profile data skipped adding labels (could not map exactly one span to the sample set)", map[string]any{
						"event-type":        eventType,
						"profile-data":      p.String(),
						"spans-recorded":    len(spanIDs),
						"cache-size":        pc.spanCache.Len(),
						"start-time":        p.TimeNanos,
						"duration":          p.DurationNanos,
						"start-time-string": time.Unix(p.TimeNanos/1_000_000_000, p.TimeNanos%1_000_000_000).String(),
						"duration-ms":       p.DurationNanos / 1_000_000,
					})
				} else {
					a.Debug("profiler: profile data skipped adding labels (no active spans found)", map[string]any{
						"event-type":        eventType,
						"profile-data":      p.String(),
						"spans-recorded":    len(spanIDs),
						"cache-size":        pc.spanCache.Len(),
						"start-time":        p.TimeNanos,
						"duration":          p.DurationNanos,
						"start-time-string": time.Unix(p.TimeNanos/1_000_000_000, p.TimeNanos%1_000_000_000).String(),
						"duration-ms":       p.DurationNanos / 1_000_000,
					})
				}
			}
		}

		if profileDestination == profileIngestPPROF {
			pc.sendProfilePprofMethod(eventType, eventType, profileData, a)
			return
		}
		a.Error("profiler: non-PPROF report destination", nil)
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
		} else {
			// we're sending to an ingest endpoint of some sort
			if pc.isCPUSelected() {
				pprof.StopCPUProfile()
				reportBufferedProfileSamples(&cpuData, "cpu", false, pc.auditFile)
				cpuData.Reset()
			}
		}
		profileDestination = profileNilDest
	}
	defer closeLocalFiles()

	recheckConfig := time.NewTicker(100 * time.Millisecond)
	saveLocalProfileSample := func(ptype, etype string, bits ProfilingType) {
		var err error
		var f *os.File
		a.config.Logger.Debug(fmt.Sprintf("saveLocalProfileSample(%s)", ptype), nil)
		if f, err = os.CreateTemp(pc.outputDirectory, localProfileFileName(ptype)); err == nil {
			a.config.Logger.Debug(fmt.Sprintf("Created file %s", f.Name()), nil)
			pprof.Lookup(ptype).WriteTo(f, pc.outputDebug)
			a.config.Logger.Debug("Wrote data", nil)
			if err = f.Close(); err != nil {
				a.config.Logger.Debug("ERROR CLOSING", map[string]any{
					"error": err.Error(),
				})
				a.Error(fmt.Sprintf("error closing local output file for %s profile data", ptype), map[string]any{
					"event-type": etype,
					"reason":     err.Error(),
				})
			}
		} else {
			a.config.Logger.Debug("ERROR OPENING", map[string]any{
				"error": err.Error(),
			})
			a.Error(fmt.Sprintf("error opening local output file for %s profile data", ptype), map[string]any{
				"event-type": etype,
				"reason":     err.Error(),
			})
			pc.selected &= ^bits
		}
	}

	for {
		select {
		case <-recheckConfig.C:
			if pc.ableToSend() {
				// we have successfully obtained a connect payload, so we can stop checking.
				a.Debug("stopping timer", nil)
				recheckConfig.Stop()
			} else {
				a.Debug("checking for updates", nil)
				a.UpdateProfiler() // check to see if we have config data to get yet
				a.Debug("done checking for updates", nil)
			}

		// To prevent interthread contention without the need for mutexes, we use channels here
		// to let user threads request switching profile output destinations here and only this
		// monitor thread ever writes anything.
		//
		case newDestination := <-pc.ingestSwitch:
			switch newDestination {
			case profileIngestPPROF, profileNilDest:
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
				profileDestination = newDestination
				pc.switchResult <- nil
			default:
				pc.switchResult <- fmt.Errorf("Invalid profile destination code %v", newDestination)
			}

		case newDirectory := <-pc.outputSwitch:
			var err error

			pc.outputDirectory = newDirectory
			closeLocalFiles()
			heap_f = nil
			goroutine_f = nil
			threadcreate_f = nil
			block_f = nil
			mutex_f = nil
			if pc.isCPUSelected() {
				if cpu_f, err = os.CreateTemp(newDirectory, localProfileFileName("cpu")); err != nil {
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
						if cpu_f, err = os.CreateTemp(pc.outputDirectory, localProfileFileName("cpu")); err != nil {
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
						reportBufferedProfileSamples(&cpuData, "cpu", false, pc.auditFile)
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
					saveLocalProfileSample("heap", "ProfileHeap", ProfilingTypeHeap)
				}
				if pc.isGoroutineSelected() {
					saveLocalProfileSample("goroutine", "ProfileGoroutine", ProfilingTypeGoroutine)
				}
				if pc.isThreadCreateSelected() {
					saveLocalProfileSample("threadcreate", "ProfileThreadCreate", ProfilingTypeThreadCreate)
				}
				if pc.isBlockSelected() {
					saveLocalProfileSample("block", "ProfileBlock", ProfilingTypeBlock)
				}
				if pc.isMutexSelected() {
					saveLocalProfileSample("mutex", "ProfileMutex", ProfilingTypeMutex)
				}
				// The tracer writes to the file on its own, as does the cpu profiler,
				// so we don't need to do anything for them here.
			} else {
				// Otherwise, we need to process the profile data internally and report it out somewhere.
				reportProfileSample := func(profileName, eventType string, debug bool, audit io.Writer) {
					var data bytes.Buffer
					pprof.Lookup(profileName).WriteTo(&data, 0)
					if profileDestination == profileIngestPPROF {
						pc.sendProfilePprofMethod(profileName, eventType, &data, a)
						return
					}
					a.Error("profiler: non-PPROF destination", nil)
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
				harvestNumber++
			}
		case <-pc.done:
			// We were told to terminate our profile monitoring
			auditLog(pc.auditFile, "monitor stopped")
			a.Info("profiler stopped", nil)
			return
		case <-autoShutdown.C:
			auditLog(pc.auditFile, "monitor auto-stopped")
			a.ShutdownProfiler()
			a.Info("profiler automatically stopped due to configured duration", nil)
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

func (pc *profilerConfig) sendProfilePprofMethod(profileName, eventType string, data *bytes.Buffer, app *app) {
	if !pc.ableToSend() {
		app.Error("profiler did not send sample data (endpoint not yet ready to receive)", map[string]any{
			"profile":       eventType,
			"payload.bytes": len(data.Bytes()),
		})
		return
	}

	pc.methodRpmCmd.Data = data.Bytes()
	app.Debug(fmt.Sprintf("Sending %s payload of %d bytes\n", eventType, len(pc.methodRpmCmd.Data)), nil)
	if profileName == "mutex" || profileName == "block" {
		pc.methodRpmCmd.MethodParams["profile_metadata"] = "category=" + profileName
	} else {
		delete(pc.methodRpmCmd.MethodParams, "profile_metadata")
	}

	resp := collectorRequest(pc.methodRpmCmd, pc.methodRpmControls)
	if resp.IsDisconnect() || resp.IsRestartException() {
		// TODO: handle shutdown condition
		app.Debug("shutdown event from pprof endpoint", nil)
		return
	}
	if resp.GetError() != nil {
		// TODO
		app.Error("profiler received error response from collector", map[string]any{
			"error":   resp.GetError().Error(),
			"profile": eventType,
		})
		return
	}
	app.Debug("pprof response", map[string]any{
		"statusCode": resp.statusCode,
		"body":       resp.body,
		"profile":    eventType,
	})
}

func ProfilerWrapCall(txn *Transaction, f func(context.Context)) {
	ProfilerWrapCallWithContext(txn, context.TODO(), f)
}

func ProfilerWrapCallWithContext(txn *Transaction, ctx context.Context, f func(context.Context)) {
	var labels pprof.LabelSet
	if txn != nil {
		md := txn.GetTraceMetadata()
		labels = pprof.Labels("span_id", md.SpanID, "trace_id", md.TraceID)
	}
	pprof.Do(ctx, labels, f)
}
