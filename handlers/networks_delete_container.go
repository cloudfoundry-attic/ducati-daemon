package handlers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/ossupport"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/deletor.go --fake-name Deletor . deletor
type deletor interface {
	Delete(deletorConfig container.DeletorConfig) error
}

type repository interface {
	Get(string) (namespace.Namespace, error)
}

type NetworksDeleteContainer struct {
	Unmarshaler    marshal.Unmarshaler
	Logger         lager.Logger
	Datastore      store.Store
	Deletor        deletor
	OSThreadLocker ossupport.OSThreadLocker
	SandboxRepo    repository
	NetworkMapper  ipam.NetworkMapper
}

func (h *NetworksDeleteContainer) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	h.OSThreadLocker.LockOSThread()
	defer h.OSThreadLocker.UnlockOSThread()

	logger := h.Logger.Session("networks-delete-containers")

	bodyBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Error("body-read-failed", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload models.NetworksDeleteContainerPayload
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

	sandboxName := fmt.Sprintf("vni-%d", vni)
	sandboxNS, err := h.SandboxRepo.Get(sandboxName)
	if err != nil {
		logger.Error("sandbox-repo", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	deletorConfig := container.DeletorConfig{
		InterfaceName:   payload.InterfaceName,
		ContainerNSPath: payload.ContainerNamespace,
		SandboxNSPath:   sandboxNS.Path(),
		VxlanDeviceName: fmt.Sprintf("vxlan%d", vni),
	}

	err = h.Deletor.Delete(deletorConfig)
	if err != nil {
		logger.Error("deletor.delete-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = h.Datastore.Delete(payload.ContainerID)
	if err != nil {
		logger.Error("datastore.delete-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return // untested
	}

	resp.WriteHeader(http.StatusNoContent)
}
