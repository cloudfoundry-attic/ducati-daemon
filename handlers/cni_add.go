package handlers

import (
	"errors"
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
)

//go:generate counterfeiter -o ../fakes/creator.go --fake-name Creator . creator
type creator interface {
	Setup(container.CreatorConfig) (models.Container, error)
}

type CNIAdd struct {
	Unmarshaler    marshal.Unmarshaler
	Logger         lager.Logger
	Datastore      store.Store
	Creator        creator
	OSThreadLocker ossupport.OSThreadLocker
	Marshaler      marshal.Marshaler
	IPAllocator    ipam.IPAllocator
	NetworkMapper  ipam.NetworkMapper
}

func (h *CNIAdd) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	h.OSThreadLocker.LockOSThread()
	defer h.OSThreadLocker.UnlockOSThread()

	logger := h.Logger.Session("networks-setup-containers")

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error("body-read-failed", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload models.CNIAddPayload
	err = h.Unmarshaler.Unmarshal(bodyBytes, &payload)
	if err != nil {
		logger.Error("unmarshal-failed", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.InterfaceName == "" {
		logger.Error("bad-request", errors.New("missing-interface_name"))
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.NetworkID == "" {
		logger.Error("bad-request", errors.New("missing-network_id"))
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.ContainerID == "" {
		logger.Error("bad-request", errors.New("missing-container_id"))
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.ContainerNamespace == "" {
		logger.Error("bad-request", errors.New("missing-container_namespace"))
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	vni, err := h.NetworkMapper.GetVNI(payload.NetworkID)
	if err != nil {
		logger.Error("network-mapper-get-vni", err)
		resp.WriteHeader(http.StatusInternalServerError)
	}

	ipamResult, err := h.IPAllocator.AllocateIP(payload.NetworkID, payload.ContainerID)
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

	containerConfig := container.CreatorConfig{
		NetworkID:       payload.NetworkID,
		ContainerNsPath: payload.ContainerNamespace,
		ContainerID:     payload.ContainerID,
		InterfaceName:   payload.InterfaceName,
		VNI:             vni,
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

	jsonBodyBytes, err := h.Marshaler.Marshal(ipamResult)
	if err != nil {
		logger.Error("marshal-result", err)
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
