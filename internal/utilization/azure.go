package utilization

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	azureHostname     = "169.254.169.254"
	azureEndpointPath = "/metadata/instance/compute?api-version=2017-03-01"
	azureEndpoint     = "http://" + azureHostname + azureEndpointPath
)

type azure struct {
	Location string `json:"location,omitempty"`
	Name     string `json:"name,omitempty"`
	VMID     string `json:"vmId,omitempty"`
	VMSize   string `json:"vmSize,omitempty"`

	client *http.Client
}

func gatherAzure(util *Data) error {
	az := newAzure()
	if err := az.Gather(); err != nil {
		return fmt.Errorf("Azure not detected: %s", err)
	}
	util.Vendors.Azure = az

	return nil
}

func newAzure() *azure {
	return &azure{
		client: &http.Client{Timeout: providerTimeout},
	}
}

func (az *azure) Gather() error {
	// Azure's metadata service requires a Metadata header to avoid accidental
	// redirects.
	req, err := http.NewRequest("GET", azureEndpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Metadata", "true")

	response, err := az.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("got response code %d", response.StatusCode)
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, az); err != nil {
		return err
	}

	if err := az.validate(); err != nil {
		*az = azure{client: az.client}
		return err
	}

	return nil
}

func (az *azure) validate() (err error) {
	az.Location, err = normalizeValue(az.Location)
	if err != nil {
		return fmt.Errorf("Invalid Azure location: %v", err)
	}

	az.Name, err = normalizeValue(az.Name)
	if err != nil {
		return fmt.Errorf("Invalid Azure name: %v", err)
	}

	az.VMID, err = normalizeValue(az.VMID)
	if err != nil {
		return fmt.Errorf("Invalid Azure VM ID: %v", err)
	}

	az.VMSize, err = normalizeValue(az.VMSize)
	if err != nil {
		return fmt.Errorf("Invalid Azure VM size: %v", err)
	}

	return
}
