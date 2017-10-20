package utilization

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	gcpHostname     = "metadata.google.internal"
	gcpEndpointPath = "/computeMetadata/v1/instance/?recursive=true"
	gcpEndpoint     = "http://" + gcpHostname + gcpEndpointPath
)

func GatherGCP(util *Data) error {
	gcp := newGCP()
	if err := gcp.Gather(); err != nil {
		return fmt.Errorf("GCP not detected: %s", err)
	} else {
		util.Vendors.GCP = gcp
	}

	return nil
}

// numericString is used rather than json.Number because we want the output when
// marshalled to be a string, rather than a number.
type numericString string

func (ns *numericString) MarshalJSON() ([]byte, error) {
	return json.Marshal(ns.String())
}

func (ns *numericString) String() string {
	return string(*ns)
}

func (ns *numericString) UnmarshalJSON(data []byte) error {
	var n int64

	// Try to unmarshal as an integer first.
	if err := json.Unmarshal(data, &n); err == nil {
		*ns = numericString(fmt.Sprintf("%d", n))
		return nil
	}

	// Otherwise, unmarshal as a string, and verify that it's numeric (for our
	// definition of numeric, which is actually integral).
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	for _, r := range s {
		if r < '0' || r > '9' {
			return fmt.Errorf("invalid numeric character: %c", r)
		}
	}

	*ns = numericString(s)
	return nil
}

type gcp struct {
	ID          numericString `json:"id"`
	MachineType string        `json:"machineType,omitempty"`
	Name        string        `json:"name,omitempty"`
	Zone        string        `json:"zone,omitempty"`

	client *http.Client
}

func newGCP() *gcp {
	return &gcp{
		client: &http.Client{Timeout: providerTimeout},
	}
}

func (g *gcp) Gather() error {
	// GCP's metadata service requires a Metadata-Flavor header because... hell, I
	// don't know, maybe they really like Guy Fieri?
	req, err := http.NewRequest("GET", gcpEndpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Metadata-Flavor", "Google")

	response, err := g.client.Do(req)
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

	if err := json.Unmarshal(data, g); err != nil {
		return err
	}

	if err := g.validate(); err != nil {
		*g = gcp{client: g.client}
		return err
	}

	return nil
}

func (gcp *gcp) validate() (err error) {
	id, err := normalizeValue(gcp.ID.String())
	if err != nil {
		return fmt.Errorf("Invalid GCP ID: %v", err)
	}
	gcp.ID = numericString(id)

	mt, err := normalizeValue(gcp.MachineType)
	if err != nil {
		return fmt.Errorf("Invalid GCP machine type: %v", err)
	}
	gcp.MachineType = stripGCPPrefix(mt)

	gcp.Name, err = normalizeValue(gcp.Name)
	if err != nil {
		return fmt.Errorf("Invalid GCP name: %v", err)
	}

	zone, err := normalizeValue(gcp.Zone)
	if err != nil {
		return fmt.Errorf("Invalid GCP zone: %v", err)
	}
	gcp.Zone = stripGCPPrefix(zone)

	return
}

// We're only interested in the last element of slash separated paths for the
// machine type and zone values, so this function handles stripping the parts
// we don't need.
func stripGCPPrefix(s string) string {
	parts := strings.Split(s, "/")
	return parts[len(parts)-1]
}
