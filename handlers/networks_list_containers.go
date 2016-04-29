package handlers

import (
	"net/http"

	"lib/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

type NetworksListContainers struct {
	Marshaler marshal.Marshaler
	Logger    lager.Logger
	Datastore store.Store
}

func (h *NetworksListContainers) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := h.Logger.Session("networks-list-containers")
	id := rata.Param(req, "network_id")

	resp.Header().Set("content-type", "application/json")

	allContainers, err := h.Datastore.All()
	if err != nil {
		logger.Error("datastore-all-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	containers := []models.Container{}
	for _, c := range allContainers {
		if c.NetworkID == id {
			containers = append(containers, c)
		}
	}

	payload, err := h.Marshaler.Marshal(containers)
	if err != nil {
		logger.Error("marshal-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(payload)
}
