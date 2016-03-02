package handlers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/cloudfoundry-incubator/ducati-daemon/threading"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
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
	OSThreadLocker threading.OSThreadLocker
	SandboxRepo    repository
}

func (h *NetworksDeleteContainer) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	h.OSThreadLocker.LockOSThread()
	defer h.OSThreadLocker.UnlockOSThread()

	logger := h.Logger.Session("networks-delete-containers")

	containerID := rata.Param(request, "container_id")
	_ = rata.Param(request, "network_id") // we may want this later

	bodyBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Error("body-read-failed", err)
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload models.NetworksDeleteContainerPayload
	err = h.Unmarshaler.Unmarshal(bodyBytes, &payload)
	if err != nil {
		logger.Error("unmarshal-failed", err)
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.InterfaceName == "" {
		logger.Error("bad-request", errors.New("missing-interface_name"))
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.ContainerNamespace == "" {
		logger.Error("bad-request", errors.New("missing-container_namespace"))
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.VNI == 0 {
		logger.Error("bad-request", errors.New("missing-vni"))
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	sandboxName := fmt.Sprintf("vni-%d", payload.VNI)
	sandboxNS, err := h.SandboxRepo.Get(sandboxName)
	if err != nil {
		logger.Error("sandbox-repo", err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	deletorConfig := container.DeletorConfig{
		InterfaceName:   payload.InterfaceName,
		ContainerNSPath: payload.ContainerNamespace,
		SandboxNSPath:   sandboxNS.Path(),
	}

	err = h.Deletor.Delete(deletorConfig)
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
