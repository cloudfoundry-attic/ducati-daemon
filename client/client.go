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
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/dghubble/sling"
)

var RecordNotFoundError error = errors.New("record not found")

type DaemonClient struct {
	slingClient *sling.Sling
}

func New(baseURL string, httpClient *http.Client) *DaemonClient {
	return &DaemonClient{
		slingClient: sling.New().Client(httpClient).Base(baseURL).Set("Accept", "application/json"),
	}
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
		Properties: stdinStruct.Network.Properties,
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

	resp, err := d.slingClient.New().Post("/cni/add").BodyJSON(payload).ReceiveSuccess(&ipamResult)
	if err != nil {
		return types.Result{}, fmt.Errorf("container up: %s", err)
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		return ipamResult, nil
	case http.StatusBadRequest:
		return types.Result{}, ipam.AlreadyOnNetworkError
	case http.StatusConflict:
		return types.Result{}, ipam.NoMoreAddressesError
	default:
		return types.Result{}, fmt.Errorf("unexpected status code on %s: expected %d but got %d", "ContainerUp", http.StatusCreated, resp.StatusCode)
	}
}

func (d *DaemonClient) ContainerDown(payload models.CNIDelPayload) error {
	resp, err := d.slingClient.New().Post("/cni/del").BodyJSON(payload).ReceiveSuccess(nil)
	if err != nil {
		return fmt.Errorf("container down: %s", err)
	}

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	default:
		return fmt.Errorf("unexpected status code on %s: expected %d but got %d", "ContainerDown", http.StatusNoContent, resp.StatusCode)
	}
}

func (d *DaemonClient) GetContainer(containerID string) (models.Container, error) {
	var container models.Container

	resp, err := d.slingClient.New().Get(path.Join("containers", containerID)).ReceiveSuccess(&container)
	if err != nil {
		return models.Container{}, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return container, nil
	case http.StatusNotFound:
		return models.Container{}, RecordNotFoundError
	default:
		return models.Container{}, fmt.Errorf("unexpected status code on %s: expected %d but got %d", "GetContainer", http.StatusOK, resp.StatusCode)
	}
}

func (d *DaemonClient) ListNetworkContainers(networkID string) ([]models.Container, error) {
	var containers []models.Container

	resp, err := d.slingClient.New().Get(path.Join("networks", networkID)).ReceiveSuccess(&containers)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return containers, nil
	default:
		return nil, fmt.Errorf("unexpected status code on %s: expected %d but got %d", "ListNetworkContainers", http.StatusOK, resp.StatusCode)
	}
}

func (d *DaemonClient) ListContainers() ([]models.Container, error) {
	var containers []models.Container

	resp, err := d.slingClient.New().Get("containers").ReceiveSuccess(&containers)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return containers, nil
	default:
		return nil, fmt.Errorf("unexpected status code on %s: expected %d but got %d", "ListContainers", http.StatusOK, resp.StatusCode)
	}
}

//go:generate counterfeiter -o ../fakes/round_tripper.go --fake-name RoundTripper . roundTripper
type roundTripper interface {
	http.RoundTripper
}
