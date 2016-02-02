package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
)

func New(listenAddr string) *DaemonClient {
	return &DaemonClient{
		BaseURL:   fmt.Sprintf("http://%s", listenAddr),
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

	if resp.StatusCode != 201 {
		return fmt.Errorf("expected to receive 201 but got %d for data %s", resp.StatusCode, postData)
	}

	return nil
}
