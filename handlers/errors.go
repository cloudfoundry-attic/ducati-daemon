package handlers

import (
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
)

type errorBody struct {
	Error string `json:"error"`
}

func marshalError(logger Logger, resp http.ResponseWriter, m marshal.Marshaler, err error) {
	marshaledError, err := m.Marshal(errorBody{Error: err.Error()})
	if err != nil {
		logger.Error("allocate-ip-error-marshaling", err)
		return
	}

	resp.Write(marshaledError)
}
