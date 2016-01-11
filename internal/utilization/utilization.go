// Package utilization implements the Utilization spec, available at
// https://source.datanerd.us/agents/agent-specs/blob/master/Utilization.md
package utilization

import (
	"runtime"

	"go.datanerd.us/p/will/newrelic/internal/sysinfo"
	"go.datanerd.us/p/will/newrelic/log"
)

const metadataVersion = 1

type Config struct {
	DetectAWS    bool
	DetectDocker bool
}

type Data struct {
	MetadataVersion   int      `json:"metadata_version"`
	LogicalProcessors int      `json:"logical_processors"`
	RamMib            *uint64  `json:"total_ram_mib"`
	Hostname          string   `json:"hostname"`
	Vendors           *vendors `json:"vendors,omitempty"`
}

type vendor struct {
	Id   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Zone string `json:"zone,omitempty"`
}

type vendors struct {
	AWS    *vendor `json:"aws,omitempty"`
	Docker *vendor `json:"docker,omitempty"`
}

func Gather(config Config) *Data {
	uDat := Data{
		MetadataVersion:   metadataVersion,
		Vendors:           &vendors{},
		LogicalProcessors: runtime.NumCPU(),
	}

	if config.DetectDocker {
		id, err := sysinfo.DockerID()
		if err != nil && err != sysinfo.ErrDockerUnsupported {
			log.Warn("error gathering Docker information", log.Context{
				"error": err.Error(),
			})
		} else if id != "" {
			uDat.Vendors.Docker = &vendor{Id: id}
		}
	}

	if config.DetectAWS {
		aws, err := getAWS()
		if nil == err {
			uDat.Vendors.AWS = aws
		} else if isAWSValidationError(err) {
			log.Error("AWS validation error", log.Context{
				"error": err.Error(),
			})
		} else {
			log.Debug("unable to connect to AWS", log.Context{
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
		log.Error("error getting hostname", log.Context{
			"error": err.Error(),
		})
	}

	bts, err := sysinfo.PhysicalMemoryBytes()
	if nil == err {
		mib := sysinfo.BytesToMebibytes(bts)
		uDat.RamMib = &mib
	} else {
		log.Error("error getting memory", log.Context{
			"error": err.Error(),
		})
	}

	return &uDat
}
