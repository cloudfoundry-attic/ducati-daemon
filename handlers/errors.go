package handlers

import "net/http"

type errorBody struct {
	Error string `json:"error"`
}

func marshalError(logger Logger, resp http.ResponseWriter, m marshaler, err error) {
	marshaledError, err := m.Marshal(errorBody{Error: err.Error()})
	if err != nil {
		logger.Error("allocate-ip-error-marshaling", err)
		return
	}

	resp.Write(marshaledError)
}
