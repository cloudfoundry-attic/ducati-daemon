package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"

	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter -o ../fakes/creator.go --fake-name Creator . creator
type creator interface {
	Setup(container.CreatorConfig) (models.Container, error)
}

type NetworksSetupContainer struct {
	Unmarshaler marshal.Unmarshaler
	Logger      lager.Logger
	Datastore   store.Store
	Creator     creator
}

func (h *NetworksSetupContainer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	logger := h.Logger.Session("networks-setup-containers")

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		// not tested
		logger.Error("body-read-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	var containerPayload models.NetworksSetupContainerPayload
	err = h.Unmarshaler.Unmarshal(bodyBytes, &containerPayload)
	if err != nil {
		logger.Error("unmarshal-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	containerConfig := container.CreatorConfig{
		NetworkID:       rata.Param(req, "network_id"),
		BridgeName:      fmt.Sprintf("vxlanbr%d", containerPayload.VNI),
		ContainerNsPath: containerPayload.ContainerNamespace,
		ContainerID:     rata.Param(req, "container_id"),
		HostIP:          containerPayload.HostIP,
		InterfaceName:   containerPayload.InterfaceName,
		VNI:             containerPayload.VNI,
		IPAMResult:      containerPayload.IPAM,
	}

	container, err := h.Creator.Setup(containerConfig)
	if err != nil {
		logger.Error("container-setup-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = h.Datastore.Create(container)
	if err != nil {
		logger.Error("datastore-create-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusCreated)
}
