package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
)

var RecordNotFoundError error = errors.New("record not found")

func New(baseURL string, httpClient *http.Client) *DaemonClient {
	return &DaemonClient{
		BaseURL:     baseURL,
		Marshaler:   marshal.MarshalFunc(json.Marshal),
		Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
		HttpClient:  httpClient,
	}
}

type DaemonClient struct {
	BaseURL     string
	Marshaler   marshal.Marshaler
	Unmarshaler marshal.Unmarshaler
	HttpClient  *http.Client
}

func (d *DaemonClient) ContainerUp(networkID, containerID string, payload models.NetworksSetupContainerPayload) error {
	postData, err := d.Marshaler.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal cni payload: %s", err)
	}

	url := d.buildURL("networks", networkID, containerID)
	resp, err := d.HttpClient.Post(url, "application/json", bytes.NewReader(postData))
	if err != nil {
		return fmt.Errorf("failed to perform request: %s", err)
	}
	defer resp.Body.Close()

	if statusError := checkStatus("ContainerUp", resp.StatusCode, http.StatusCreated); statusError != nil {
		return statusError
	}
	return nil
}

func (d *DaemonClient) ContainerDown(networkID, containerID string, payload models.NetworksDeleteContainerPayload) error {
	deleteData, err := d.Marshaler.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal cni payload: %s", err)
	}

	url := d.buildURL("networks", networkID, containerID)
	req, err := http.NewRequest("DELETE", url, bytes.NewReader(deleteData))
	if err != nil {
		return fmt.Errorf("failed to construct request: %s", err)
	}
	req.Header.Set("Content-type", "application/json")
	resp, err := d.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %s", err)
	}
	defer resp.Body.Close()

	if statusError := checkStatus("ContainerDown", resp.StatusCode, http.StatusNoContent); statusError != nil {
		return statusError
	}
	return nil
}

func (d *DaemonClient) SaveContainer(container models.Container) error {
	postData, err := d.Marshaler.Marshal(container)
	if err != nil {
		return fmt.Errorf("failed to marshal container: %s", err)
	}

	url := d.buildURL("containers")
	resp, err := d.HttpClient.Post(url, "application/json", bytes.NewReader(postData))
	if err != nil {
		return fmt.Errorf("failed to perform request: %s", err)
	}
	defer resp.Body.Close()

	if statusError := checkStatus("SaveContainer", resp.StatusCode, http.StatusCreated); statusError != nil {
		return statusError
	}

	return nil
}

func (d *DaemonClient) ListContainers() ([]models.Container, error) {
	url := d.buildURL("containers")
	resp, err := d.HttpClient.Get(url)
	if err != nil {
		return []models.Container{}, fmt.Errorf("failed to perform request: %s", err)
	}
	defer resp.Body.Close()

	if statusError := checkStatus("ListContainers", resp.StatusCode, http.StatusOK); statusError != nil {
		return []models.Container{}, statusError
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []models.Container{}, nil // not tested
	}

	var containers []models.Container
	err = d.Unmarshaler.Unmarshal(respBytes, &containers)
	if err != nil {
		return []models.Container{}, fmt.Errorf("failed to unmarshal containers: %s", err)
	}

	return containers, nil
}

func (d *DaemonClient) RemoveContainer(containerID string) error {
	url := d.buildURL("containers", containerID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to construct request: %s", err)
	}

	resp, err := d.HttpClient.Do(req)
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
	url := d.buildURL("ipam", networkID, containerID)
	resp, err := d.HttpClient.Post(url, "application/json", nil)
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
	url := d.buildURL("ipam", networkID, containerID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to construct request: %s", err)
	}

	resp, err := d.HttpClient.Do(req)
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

func (d *DaemonClient) buildURL(routeElements ...string) string {
	return d.BaseURL + "/" + strings.Join(routeElements, "/")
}

//go:generate counterfeiter -o ../fakes/round_tripper.go --fake-name RoundTripper . roundTripper
type roundTripper interface {
	http.RoundTripper
}
