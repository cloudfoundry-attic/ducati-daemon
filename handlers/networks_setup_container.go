package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/ossupport"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter -o ../fakes/creator.go --fake-name Creator . creator
type creator interface {
	Setup(container.CreatorConfig) (models.Container, error)
}

type NetworksSetupContainer struct {
	Unmarshaler    marshal.Unmarshaler
	Logger         lager.Logger
	Datastore      store.Store
	Creator        creator
	OSThreadLocker ossupport.OSThreadLocker
	Marshaler      marshal.Marshaler
	IPAllocator    ipam.IPAllocator
}

func (h *NetworksSetupContainer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	h.OSThreadLocker.LockOSThread()
	defer h.OSThreadLocker.UnlockOSThread()

	logger := h.Logger.Session("networks-setup-containers")

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error("body-read-failed", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	var containerPayload models.NetworksSetupContainerPayload
	err = h.Unmarshaler.Unmarshal(bodyBytes, &containerPayload)
	if err != nil {
		logger.Error("unmarshal-failed", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	networkID := rata.Param(req, "network_id")
	containerID := rata.Param(req, "container_id")

	ipamResult, err := h.IPAllocator.AllocateIP(networkID, containerID)
	if err != nil {
		logger.Error("allocate-ip", err)

		switch err {
		case ipam.NoMoreAddressesError:
			resp.WriteHeader(http.StatusConflict)
		default:
			resp.WriteHeader(http.StatusInternalServerError)
		}

		marshalError(logger, resp, h.Marshaler, err)
		return
	}

	jsonBodyBytes, err := h.Marshaler.Marshal(ipamResult)
	if err != nil {
		logger.Error("allocate-ip", err)
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
		IPAMResult:      ipamResult,
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
	_, err = resp.Write(jsonBodyBytes)
	if err != nil {
		logger.Error("allocate-ip", fmt.Errorf("failed writing body: %s", err))
		return
	}
}
