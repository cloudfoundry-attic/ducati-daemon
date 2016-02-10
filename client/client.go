package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
)

var RecordNotFoundError error = errors.New("record not found")

func New(baseURL string) *DaemonClient {
	return &DaemonClient{
		BaseURL:     baseURL,
		Marshaler:   marshal.MarshalFunc(json.Marshal),
		Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
	}
}

type DaemonClient struct {
	BaseURL     string
	Marshaler   marshal.Marshaler
	Unmarshaler marshal.Unmarshaler
}

func (d *DaemonClient) SaveContainer(container models.Container) error {
	postData, err := d.Marshaler.Marshal(container)
	if err != nil {
		return fmt.Errorf("failed to marshal container: %s", err)
	}

	resp, err := http.Post(d.BaseURL+"/containers", "application/json", bytes.NewReader(postData))
	if err != nil {
		return fmt.Errorf("failed to perform request: %s", err)
	}
	defer resp.Body.Close()

	if statusError := checkStatus("SaveContainer", resp.StatusCode, http.StatusCreated); statusError != nil {
		return statusError
	}

	return nil
}

func (d *DaemonClient) RemoveContainer(containerID string) error {
	req, err := http.NewRequest("DELETE", d.BaseURL+"/containers/"+containerID, nil)
	if err != nil {
		return fmt.Errorf("failed to construct request: %s", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return RecordNotFoundError
	}

	if statusError := checkStatus("RemoveContainer", resp.StatusCode, http.StatusNoContent); statusError != nil {
		return statusError
	}

	return nil
}

func (d *DaemonClient) AllocateIP(networkID, containerID string) (types.Result, error) {
	resp, err := http.Post(d.BaseURL+"/ipam/"+networkID+"/"+containerID, "application/json", nil)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to perform request: %s", err)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return types.Result{}, nil // not tested
	}

	switch resp.StatusCode {
	case http.StatusCreated:
	case http.StatusConflict:
		return types.Result{}, ipam.NoMoreAddressesError
	default:
		if statusError := checkStatus("AllocateIP", resp.StatusCode, http.StatusCreated); statusError != nil {
			return types.Result{}, statusError
		}
	}

	var ipamResult types.Result
	err = d.Unmarshaler.Unmarshal(respBytes, &ipamResult)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to unmarshal IPAM result: %s", err)
	}

	return ipamResult, nil
}

func (d *DaemonClient) ReleaseIP(networkID, containerID string) error {
	url := fmt.Sprintf("%s/ipam/%s/%s", d.BaseURL, networkID, containerID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to construct request: %s", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if statusError := checkStatus("ReleaseIP", resp.StatusCode, http.StatusNoContent); statusError != nil {
		return statusError
	}

	return nil
}

func checkStatus(method string, receivedStatus, expectedStatus int) error {
	if receivedStatus != expectedStatus {
		return fmt.Errorf("unexpected status code on %s: expected %d but got %d", method, expectedStatus, receivedStatus)
	}

	return nil
}
