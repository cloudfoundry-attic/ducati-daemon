package handlers

import (
	"encoding/json"
	"net/http"
)

type Container struct {
	ID string `json:"id"`
}

type ListHandler struct {
	Containers []Container
}

func (h *ListHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	containers := h.Containers
	if containers == nil {
		containers = []Container{}
	}

	jsonResponse, err := json.Marshal(containers)
	if err != nil {
		panic(err)
	}
	resp.Write(jsonResponse)
}
