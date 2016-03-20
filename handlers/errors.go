package handlers

import (
	"fmt"
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
)

type errorBody struct {
	Error string `json:"error"`
}

func marshalError(resp http.ResponseWriter, m marshal.Marshaler, err error) error {
	marshaledError, err := m.Marshal(errorBody{Error: err.Error()})
	if err != nil {
		return fmt.Errorf("marshal: %s", err)
	}

	_, err = resp.Write(marshaledError)
	if err != nil {
		return fmt.Errorf("response write: %s", err)
	}

	return nil
}
