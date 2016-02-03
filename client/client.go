package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
)

var RecordNotFoundError error = errors.New("record not found")

func New(baseURL string) *DaemonClient {
	return &DaemonClient{
		BaseURL:   baseURL,
		Marshaler: marshal.MarshalFunc(json.Marshal),
	}
}

type DaemonClient struct {
	BaseURL   string
	Marshaler marshal.Marshaler
}

func (d *DaemonClient) SaveContainer(container models.Container) error {
	postData, err := d.Marshaler.Marshal(container)
	if err != nil {
		return fmt.Errorf("failed to marshal container: %s", err)
	}

	resp, err := http.Post(d.BaseURL+"/containers", "application/json", bytes.NewReader(postData))
	if err != nil {
		return fmt.Errorf("failed to construct request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("expected to receive %d but got %d for data %s", http.StatusCreated, resp.StatusCode, postData)
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
		return fmt.Errorf("failed to construct request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		if resp.StatusCode == http.StatusNotFound {
			return RecordNotFoundError
		}
		return fmt.Errorf("expected to receive %d but got %d", http.StatusNoContent, resp.StatusCode)
	}

	return nil
}
