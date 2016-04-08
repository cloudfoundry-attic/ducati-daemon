package client

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
)

type JSONClient struct {
	BaseURL     string
	Marshaler   marshal.Marshaler
	Unmarshaler marshal.Unmarshaler
	HttpClient  httpClient
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

func (d *JSONClient) BuildAndDo(config ClientConfig) error {
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

func (d *JSONClient) buildURL(routeElements ...string) (string, error) {
	parsedURL, err := url.Parse(d.BaseURL)
	if err != nil {
		return "", err
	}
	pathElements := append([]string{parsedURL.Path}, routeElements...)
	parsedURL.Path = path.Join(pathElements...)
	return parsedURL.String(), nil
}

func (d *JSONClient) unmarshalResponse(responseBody io.Reader, output interface{}) error {
	respBytes, err := ioutil.ReadAll(responseBody)
	if err != nil {
		return fmt.Errorf("failed to read response body: %s", err)
	}

	err = d.Unmarshaler.Unmarshal(respBytes, output)
	if err != nil {
		return fmt.Errorf("failed to unmarshal result: %s", err)
	}

	return nil
}
