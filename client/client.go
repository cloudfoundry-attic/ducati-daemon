package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/appc/cni/pkg/skel"
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

//go:generate counterfeiter -o ../fakes/http_client.go --fake-name HTTPClient . httpClient
type httpClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

type DaemonClient struct {
	BaseURL     string
	Marshaler   marshal.Marshaler
	Unmarshaler marshal.Unmarshaler
	HttpClient  httpClient
}

func (d *DaemonClient) CNIAdd(input *skel.CmdArgs) (types.Result, error) {
	var stdinStruct struct {
		Network models.NetworkPayload `json:"network"`
	}
	err := json.Unmarshal(input.StdinData, &stdinStruct)
	if err != nil {
		panic(err)
	}
	if stdinStruct.Network.ID == "" {
		stdinStruct.Network.ID = "legacy"
	}

	network := models.NetworkPayload{ID: stdinStruct.Network.ID}

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

type ClientConfig struct {
	Action            string
	Method            string
	URL               string
	RequestPayload    interface{}
	ResponseResult    interface{}
	SuccessStatusCode int
	MeaningfulErrors  map[int]error
}

func (d *DaemonClient) buildAndDo(config ClientConfig) error {
	var err error
	url, err := d.buildURL(config.URL)
	if err != nil {
		return fmt.Errorf("build url: %s", err)
	}

	var requestBody io.Reader
	if config.RequestPayload != nil {
		postData, err := d.Marshaler.Marshal(config.RequestPayload)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %s", err)
		}
		requestBody = bytes.NewReader(postData)
	}
	req, err := http.NewRequest(config.Method, url, requestBody)
	if err != nil {
		return fmt.Errorf("build request: %s", err) // not tested
	}
	if config.RequestPayload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := d.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != config.SuccessStatusCode {
		for statusCode, meaningfulError := range config.MeaningfulErrors {
			if statusCode == resp.StatusCode {
				return meaningfulError
			}
		}
		if statusError := checkStatus(config.Action, resp.StatusCode, config.SuccessStatusCode); statusError != nil {
			return statusError
		}
	}

	if config.ResponseResult != nil {
		err = d.unmarshalResponse(resp.Body, config.ResponseResult)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DaemonClient) ContainerUp(payload models.CNIAddPayload) (types.Result, error) {
	var ipamResult types.Result

	err := d.buildAndDo(ClientConfig{
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
	return d.buildAndDo(ClientConfig{
		Action:            "ContainerDown",
		Method:            "POST",
		URL:               "/cni/del",
		RequestPayload:    payload,
		SuccessStatusCode: http.StatusNoContent,
	})
}

func (d *DaemonClient) GetContainer(containerID string) (models.Container, error) {
	var container models.Container

	err := d.buildAndDo(ClientConfig{
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

	err := d.buildAndDo(ClientConfig{
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

	err := d.buildAndDo(ClientConfig{
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

func (d *DaemonClient) buildURL(routeElements ...string) (string, error) {
	parsedURL, err := url.Parse(d.BaseURL)
	if err != nil {
		return "", err
	}
	pathElements := append([]string{parsedURL.Path}, routeElements...)
	parsedURL.Path = path.Join(pathElements...)
	return parsedURL.String(), nil
}

func (d *DaemonClient) unmarshalResponse(responseBody io.Reader, output interface{}) error {
	respBytes, err := ioutil.ReadAll(responseBody)
	if err != nil {
		return fmt.Errorf("reading response body: %s", err)
	}

	err = d.Unmarshaler.Unmarshal(respBytes, output)
	if err != nil {
		return fmt.Errorf("failed to unmarshal result: %s", err)
	}

	return nil
}

//go:generate counterfeiter -o ../fakes/round_tripper.go --fake-name RoundTripper . roundTripper
type roundTripper interface {
	http.RoundTripper
}
