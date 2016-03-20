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

func (d *DaemonClient) ContainerUp(payload models.CNIAddPayload) (types.Result, error) {
	postData, err := d.Marshaler.Marshal(payload)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to marshal cni payload: %s", err)
	}

	url := d.buildURL("cni", "add")
	resp, err := d.HttpClient.Post(url, "application/json", bytes.NewReader(postData))
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to perform request: %s", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusConflict:
		return types.Result{}, ipam.NoMoreAddressesError
	default:
		if statusError := checkStatus("ContainerUp", resp.StatusCode, http.StatusCreated); statusError != nil {
			return types.Result{}, statusError
		}
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return types.Result{}, fmt.Errorf("reading response body: %s", err)
	}

	var ipamResult types.Result
	err = d.Unmarshaler.Unmarshal(respBytes, &ipamResult)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to unmarshal IPAM result: %s", err)
	}

	return ipamResult, nil
}

func (d *DaemonClient) ContainerDown(payload models.CNIDelPayload) error {
	deleteData, err := d.Marshaler.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal cni payload: %s", err)
	}

	url := d.buildURL("cni", "del")
	resp, err := d.HttpClient.Post(url, "application/json", bytes.NewReader(deleteData))
	if err != nil {
		return fmt.Errorf("failed to perform request: %s", err)
	}
	defer resp.Body.Close()

	if statusError := checkStatus("ContainerDown", resp.StatusCode, http.StatusNoContent); statusError != nil {
		return statusError
	}
	return nil
}

func (d *DaemonClient) ListNetworkContainers(networkID string) ([]models.Container, error) {
	url := d.buildURL("networks", networkID)
	resp, err := d.HttpClient.Get(url)
	if err != nil {
		return []models.Container{}, fmt.Errorf("failed to perform request: %s", err)
	}
	defer resp.Body.Close()

	if statusError := checkStatus("ListNetworkContainers", resp.StatusCode, http.StatusOK); statusError != nil {
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
