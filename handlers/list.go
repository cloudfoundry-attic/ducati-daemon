package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/store"
)

type ListHandler struct {
	Store store.Store
}

func (h *ListHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	containers, err := h.Store.All()
	if err != nil {
		panic(err)
	}

	jsonResponse, err := json.Marshal(containers)
	if err != nil {
		panic(err)
	}
	resp.Write(jsonResponse)
}
