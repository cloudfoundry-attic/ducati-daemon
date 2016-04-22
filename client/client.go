package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/appc/cni/pkg/skel"
	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"lib/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
)

var RecordNotFoundError error = errors.New("record not found")

func New(baseURL string, httpClient *http.Client) *DaemonClient {
	return &DaemonClient{
		JSONClient: JSONClient{
			BaseURL:     baseURL,
			Marshaler:   marshal.MarshalFunc(json.Marshal),
			Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
			HttpClient:  httpClient,
		},
	}
}

//go:generate counterfeiter -o ../fakes/http_client.go --fake-name HTTPClient . httpClient
type httpClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

type DaemonClient struct {
	JSONClient JSONClient
}

func (d *DaemonClient) CNIAdd(input *skel.CmdArgs) (types.Result, error) {
	var stdinStruct struct {
		Network models.NetworkPayload `json:"network"`
	}
	err := json.Unmarshal(input.StdinData, &stdinStruct)
	if err != nil {
		return types.Result{}, fmt.Errorf("invalid network spec: %s", err)
	}

	network := models.NetworkPayload{
		ID:  stdinStruct.Network.ID,
		App: stdinStruct.Network.App,
	}

	return d.ContainerUp(models.CNIAddPayload{
		ContainerID:        input.ContainerID,
		ContainerNamespace: input.Netns,
		InterfaceName:      input.IfName,
		Args:               input.Args,
		Network:            network,
	})
}

func (d *DaemonClient) CNIDel(input *skel.CmdArgs) error {
	return d.ContainerDown(models.CNIDelPayload{
		ContainerID:        input.ContainerID,
		ContainerNamespace: input.Netns,
		InterfaceName:      input.IfName,
	})
}

func (d *DaemonClient) ContainerUp(payload models.CNIAddPayload) (types.Result, error) {
	var ipamResult types.Result

	err := d.JSONClient.BuildAndDo(ClientConfig{
		Action:            "ContainerUp",
		Method:            "POST",
		URL:               "/cni/add",
		RequestPayload:    payload,
		ResponseResult:    &ipamResult,
		SuccessStatusCode: http.StatusCreated,
		MeaningfulErrors: map[int]error{
			http.StatusBadRequest: ipam.AlreadyOnNetworkError,
			http.StatusConflict:   ipam.NoMoreAddressesError,
		},
	})
	return ipamResult, err
}

func (d *DaemonClient) ContainerDown(payload models.CNIDelPayload) error {
	return d.JSONClient.BuildAndDo(ClientConfig{
		Action:            "ContainerDown",
		Method:            "POST",
		URL:               "/cni/del",
		RequestPayload:    payload,
		SuccessStatusCode: http.StatusNoContent,
	})
}

func (d *DaemonClient) GetContainer(containerID string) (models.Container, error) {
	var container models.Container

	err := d.JSONClient.BuildAndDo(ClientConfig{
		Action:            "GetContainer",
		Method:            "GET",
		URL:               path.Join("containers", containerID),
		RequestPayload:    nil,
		ResponseResult:    &container,
		SuccessStatusCode: http.StatusOK,
		MeaningfulErrors: map[int]error{
			http.StatusNotFound: RecordNotFoundError,
		},
	})
	return container, err
}

func (d *DaemonClient) ListNetworkContainers(networkID string) ([]models.Container, error) {
	var containers []models.Container

	err := d.JSONClient.BuildAndDo(ClientConfig{
		Action:            "ListNetworkContainers",
		Method:            "GET",
		URL:               path.Join("networks", networkID),
		RequestPayload:    nil,
		ResponseResult:    &containers,
		SuccessStatusCode: http.StatusOK,
	})
	return containers, err
}

func (d *DaemonClient) ListContainers() ([]models.Container, error) {
	var containers []models.Container

	err := d.JSONClient.BuildAndDo(ClientConfig{
		Action:            "ListContainers",
		Method:            "GET",
		URL:               "containers",
		RequestPayload:    nil,
		ResponseResult:    &containers,
		SuccessStatusCode: http.StatusOK,
	})
	return containers, err
}

func checkStatus(method string, receivedStatus, expectedStatus int) error {
	if receivedStatus != expectedStatus {
		return fmt.Errorf("unexpected status code on %s: expected %d but got %d", method, expectedStatus, receivedStatus)
	}

	return nil
}

//go:generate counterfeiter -o ../fakes/round_tripper.go --fake-name RoundTripper . roundTripper
type roundTripper interface {
	http.RoundTripper
}
