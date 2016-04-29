package handlers

import (
	"net/http"

	"lib/marshal"

	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/pivotal-golang/lager"
)

type ListContainers struct {
	Marshaler marshal.Marshaler
	Logger    lager.Logger
	Datastore store.Store
}

func (h *ListContainers) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := h.Logger.Session("list-containers")

	resp.Header().Set("content-type", "application/json")

	containers, err := h.Datastore.All()
	if err != nil {
		logger.Error("datastore-all-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	payload, err := h.Marshaler.Marshal(containers)
	if err != nil {
		logger.Error("marshal-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Write(payload)
}
