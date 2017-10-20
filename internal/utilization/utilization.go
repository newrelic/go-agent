// Package utilization implements the Utilization spec, available at
// https://source.datanerd.us/agents/agent-specs/blob/master/Utilization.md
//
package utilization

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/newrelic/go-agent/internal/logger"
	"github.com/newrelic/go-agent/internal/sysinfo"
)

const (
	metadataVersion = 3
)

type Config struct {
	DetectAWS         bool
	DetectAzure       bool
	DetectDocker      bool
	LogicalProcessors int
	TotalRamMIB       int
	BillingHostname   string
}

type override struct {
	LogicalProcessors *int   `json:"logical_processors,omitempty"`
	TotalRamMIB       *int   `json:"total_ram_mib,omitempty"`
	BillingHostname   string `json:"hostname,omitempty"`
}

type Data struct {
	MetadataVersion int `json:"metadata_version"`
	// Although `runtime.NumCPU()` will never fail, this field is a pointer
	// to facilitate the cross agent tests.
	LogicalProcessors *int      `json:"logical_processors"`
	RamMiB            *uint64   `json:"total_ram_mib"`
	Hostname          string    `json:"hostname"`
	BootID            string    `json:"boot_id,omitempty"`
	Vendors           *vendors  `json:"vendors,omitempty"`
	Config            *override `json:"config,omitempty"`
}

var (
	sampleRAMMib    = uint64(1024)
	sampleLogicProc = int(16)
	// SampleData contains sample utilization data useful for testing.
	SampleData = Data{
		MetadataVersion:   metadataVersion,
		LogicalProcessors: &sampleLogicProc,
		RamMiB:            &sampleRAMMib,
		Hostname:          "my-hostname",
	}
)

type docker struct {
	ID string `json:"id",omitempty`
}

type vendors struct {
	AWS    *aws    `json:"aws,omitempty"`
	Azure  *azure  `json:"azure,omitempty"`
	Docker *docker `json:"docker,omitempty"`
}

func (v *vendors) isEmpty() bool {
	return v.AWS == nil && v.Azure == nil && v.Docker == nil
}

func overrideFromConfig(config Config) *override {
	ov := &override{}

	if 0 != config.LogicalProcessors {
		x := config.LogicalProcessors
		ov.LogicalProcessors = &x
	}
	if 0 != config.TotalRamMIB {
		x := config.TotalRamMIB
		ov.TotalRamMIB = &x
	}
	ov.BillingHostname = config.BillingHostname

	if "" == ov.BillingHostname &&
		nil == ov.LogicalProcessors &&
		nil == ov.TotalRamMIB {
		ov = nil
	}
	return ov
}

func Gather(config Config, lg logger.Logger) *Data {
	var wg sync.WaitGroup

	uDat := &Data{
		MetadataVersion: metadataVersion,
		Vendors:         &vendors{},
	}

	// This closure allows us to run each gather function in a separate goroutine
	// and wait for them at the end by closing over the wg WaitGroup we
	// instantiated at the start of the function.
	goGather := func(gather func(util *Data) error, util *Data) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := gather(util); err != nil {
				lg.Warn("%s", map[string]interface{}{
					"error gathering data": err.Error(),
				})
			}
		}()
	}

	// System things we gather no matter what.
	goGather(GatherBootID, uDat)
	goGather(GatherCPU, uDat)
	goGather(GatherHostname, uDat)
	goGather(GatherMemory, uDat)

	// Now things the user can turn off.
	if config.DetectDocker {
		goGather(GatherDockerID, uDat)
	}

	if config.DetectAWS {
		goGather(GatherAWS, uDat)
	}

	if config.DetectAzure {
		goGather(GatherAzure, uDat)
	}

	// Now we wait for everything!
	wg.Wait()

	// Override whatever needs to be overridden.
	uDat.Config = overrideFromConfig(config)

	if uDat.Vendors.isEmpty() {
		// Per spec, we MUST NOT send any vendors hash if it's empty.
		uDat.Vendors = nil
	}

	return uDat
}

func GatherBootID(util *Data) error {
	id, err := sysinfo.BootID()
	if err != nil {
		if err != sysinfo.ErrFeatureUnsupported {
			return fmt.Errorf("Invalid boot ID detected: %s", err)
		}
	} else {
		util.BootID = id
	}

	return nil
}

func GatherCPU(util *Data) error {
	cpu := runtime.NumCPU()
	util.LogicalProcessors = &cpu
	return nil
}

func GatherDockerID(util *Data) error {
	id, err := sysinfo.DockerID()
	if err != nil {
		if err != sysinfo.ErrFeatureUnsupported {
			return fmt.Errorf("Did not detect Docker on this platform: %s", err)
		}
	} else {
		util.Vendors.Docker = &docker{ID: id}
	}

	return nil
}

func GatherHostname(util *Data) error {
	hostname, err := sysinfo.Hostname()
	if nil == err {
		util.Hostname = hostname
	} else {
		return fmt.Errorf("Could not find hostname: %s", err)
	}

	return nil
}

func GatherMemory(util *Data) error {
	ram, err := sysinfo.PhysicalMemoryBytes()
	if nil == err {
		ram = ram / (1024 * 1024) // bytes -> MiB
		util.RamMiB = &ram
	} else {
		return fmt.Errorf("Could not find host memory: %s", err)
	}

	return nil
}
