package handlers

import (
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/store"
)

type marshaler interface {
	Marshal(input interface{}) ([]byte, error)
}

type ListHandler struct {
	Store     store.Store
	Marshaler marshaler
	Logger    Logger
}

func (h *ListHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	containers, err := h.Store.All()
	if err != nil {
		h.Logger.Error("store-list", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonResponse, err := h.Marshaler.Marshal(containers)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Write(jsonResponse)
}
