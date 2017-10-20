package utilization

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	awsHostname     = "169.254.169.254"
	awsEndpointPath = "/2016-09-02/dynamic/instance-identity/document"
	awsEndpoint     = "http://" + awsHostname + awsEndpointPath
)

type aws struct {
	InstanceID       string `json:"instanceId,omitempty"`
	InstanceType     string `json:"instanceType,omitempty"`
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	client *http.Client
}

func GatherAWS(util *Data) error {
	aws := newAWS()
	if err := aws.Gather(); err != nil {
		return fmt.Errorf("AWS not detected: %s", err)
	} else {
		util.Vendors.AWS = aws
	}

	return nil
}

func newAWS() *aws {
	return &aws{
		client: &http.Client{Timeout: providerTimeout},
	}
}

func (a *aws) Gather() error {
	response, err := a.client.Get(awsEndpoint)
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

	if err := json.Unmarshal(data, a); err != nil {
		return err
	}

	if err := a.validate(); err != nil {
		*a = aws{client: a.client}
		return err
	}

	return nil
}

func (aws *aws) validate() (err error) {
	aws.InstanceID, err = normalizeValue(aws.InstanceID)
	if err != nil {
		return fmt.Errorf("Invalid AWS instance ID: %v", err)
	}

	aws.InstanceType, err = normalizeValue(aws.InstanceType)
	if err != nil {
		return fmt.Errorf("Invalid AWS instance type: %v", err)
	}

	aws.AvailabilityZone, err = normalizeValue(aws.AvailabilityZone)
	if err != nil {
		return fmt.Errorf("Invalid AWS availability zone: %v", err)
	}

	return
}
