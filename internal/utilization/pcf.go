package utilization

import (
	"errors"
	"fmt"
	"os"
)

type pcf struct {
	InstanceGUID string `json:"cf_instance_guid,omitempty"`
	InstanceIP   string `json:"cf_instance_ip,omitempty"`
	MemoryLimit  string `json:"memory_limit,omitempty"`

	// Having a custom getter allows the unit tests to mock os.Getenv().
	environmentVariableGetter func(key string) string
}

func GatherPCF(util *Data) error {
	pcf := newPCF()
	if err := pcf.Gather(); err != nil {
		return fmt.Errorf("PCF not detected: %s", err)
	} else {
		util.Vendors.PCF = pcf
	}

	return nil
}

func newPCF() *pcf {
	return &pcf{
		environmentVariableGetter: os.Getenv,
	}
}

func (pcf *pcf) Gather() error {
	pcf.InstanceGUID = pcf.environmentVariableGetter("CF_INSTANCE_GUID")
	pcf.InstanceIP = pcf.environmentVariableGetter("CF_INSTANCE_IP")
	pcf.MemoryLimit = pcf.environmentVariableGetter("MEMORY_LIMIT")

	if err := pcf.validate(); err != nil {
		return err
	}

	return nil
}

func (pcf *pcf) validate() (err error) {
	pcf.InstanceGUID, err = normalizeValue(pcf.InstanceGUID)
	if err != nil {
		return fmt.Errorf("Invalid PCF instance GUID: %v", err)
	}

	pcf.InstanceIP, err = normalizeValue(pcf.InstanceIP)
	if err != nil {
		return fmt.Errorf("Invalid PCF instance IP: %v", err)
	}

	pcf.MemoryLimit, err = normalizeValue(pcf.MemoryLimit)
	if err != nil {
		return fmt.Errorf("Invalid PCF memory limit: %v", err)
	}

	if pcf.InstanceGUID == "" || pcf.InstanceIP == "" || pcf.MemoryLimit == "" {
		err = errors.New("One or more PCF environment variables are unavailable")
	}

	return
}
