// Package utilization implements the Utilization spec, available at
// https://source.datanerd.us/agents/agent-specs/blob/master/Utilization.md
package utilization

import (
	"runtime"

	"github.com/newrelic/go-agent/internal/sysinfo"
	"github.com/newrelic/go-agent/log"
)

const metadataVersion = 1

// Config controls the behavior of utilization information capture.
type Config struct {
	DetectAWS    bool
	DetectDocker bool
}

// Data contains utilization system information.
type Data struct {
	MetadataVersion   int      `json:"metadata_version"`
	LogicalProcessors int      `json:"logical_processors"`
	RAMMib            *uint64  `json:"total_ram_mib"`
	Hostname          string   `json:"hostname"`
	Vendors           *vendors `json:"vendors,omitempty"`
}

var (
	sampleRAMMib = uint64(1024)
	// SampleData contains sample utilization data useful for testing.
	SampleData = Data{
		MetadataVersion:   metadataVersion,
		LogicalProcessors: 16,
		RAMMib:            &sampleRAMMib,
		Hostname:          "my-hostname",
	}
)

type vendor struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Zone string `json:"zone,omitempty"`
}

type vendors struct {
	AWS    *vendor `json:"aws,omitempty"`
	Docker *vendor `json:"docker,omitempty"`
}

// Gather gathers system utilization data.
func Gather(config Config) *Data {
	uDat := Data{
		MetadataVersion:   metadataVersion,
		Vendors:           &vendors{},
		LogicalProcessors: runtime.NumCPU(),
	}

	if config.DetectDocker {
		id, err := sysinfo.DockerID()
		if err != nil &&
			err != sysinfo.ErrDockerUnsupported &&
			err != sysinfo.ErrDockerNotFound {
			log.Warn("error gathering Docker information", log.Context{
				"error": err.Error(),
			})
		} else if id != "" {
			uDat.Vendors.Docker = &vendor{ID: id}
		}
	}

	if config.DetectAWS {
		aws, err := getAWS()
		if nil == err {
			uDat.Vendors.AWS = aws
		} else if isAWSValidationError(err) {
			log.Warn("AWS validation error", log.Context{
				"error": err.Error(),
			})
		}
	}

	if uDat.Vendors.AWS == nil && uDat.Vendors.Docker == nil {
		uDat.Vendors = nil
	}

	host, err := sysinfo.Hostname()
	if nil == err {
		uDat.Hostname = host
	} else {
		log.Warn("error getting hostname", log.Context{
			"error": err.Error(),
		})
	}

	bts, err := sysinfo.PhysicalMemoryBytes()
	if nil == err {
		mib := sysinfo.BytesToMebibytes(bts)
		uDat.RAMMib = &mib
	} else {
		log.Warn("error getting memory", log.Context{
			"error": err.Error(),
		})
	}

	return &uDat
}
