package handlers

import (
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/cloudfoundry-incubator/ducati-daemon/threading"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter -o ../fakes/deletor.go --fake-name Deletor . deletor
type deletor interface {
	Delete(networkID, containerID string) error
}

type NetworksDeleteContainer struct {
	Logger         lager.Logger
	Datastore      store.Store
	Deletor        deletor
	OSThreadLocker threading.OSThreadLocker
}

func (h *NetworksDeleteContainer) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	h.OSThreadLocker.LockOSThread()
	defer h.OSThreadLocker.UnlockOSThread()

	logger := h.Logger.Session("networks-delete-containers")

	containerID := rata.Param(request, "container_id")
	networkID := rata.Param(request, "network_id")

	err := h.Deletor.Delete(networkID, containerID)
	if err != nil {
		logger.Error("deletor.delete-failed", err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = h.Datastore.Delete(containerID)
	if err != nil {
		logger.Error("datastore.delete-failed", err)
		response.WriteHeader(http.StatusInternalServerError)
		return // untested
	}

	response.WriteHeader(http.StatusNoContent)
}