package handlers

import (
	"net/http"

	"lib/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

type GetContainer struct {
	Marshaler marshal.Marshaler
	Logger    lager.Logger
	Datastore store.Store
}

func (h *GetContainer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := h.Logger.Session("get-container")
	id := rata.Param(req, "container_id")

	container, err := h.Datastore.Get(id)
	if err != nil {
		if err == store.RecordNotFoundError {
			logger.Error("record-not-found", err)
			resp.WriteHeader(http.StatusNotFound)
			return
		}
		logger.Error("database-error", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	payload, err := h.Marshaler.Marshal(container)
	if err != nil {
		logger.Error("marshal-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(payload)
}
