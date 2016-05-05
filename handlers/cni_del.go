package handlers

import (
	"errors"
	"io/ioutil"
	"net/http"
	"runtime"

	"lib/marshal"

	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/del_controller.go --fake-name DelController . delController
type delController interface {
	Del(models.CNIDelPayload) error
}

type CNIDel struct {
	Unmarshaler marshal.Unmarshaler
	Marshaler   marshal.Marshaler
	Logger      lager.Logger
	Controller  delController
}

func (h *CNIDel) ServeHTTP(resp http.ResponseWriter, request *http.Request) {

	runtime.LockOSThread()

	logger := h.Logger.Session("cni-del")

	bodyBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Error("body-read-failed", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload models.CNIDelPayload
	err = h.Unmarshaler.Unmarshal(bodyBytes, &payload)
	if err != nil {
		logger.Error("unmarshal-failed", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	logger = logger.WithData(lager.Data{"payload": payload})
	defer logger.Info("container-del-complete")

	if payload.InterfaceName == "" {
		logger.Error("bad-request", errors.New("missing-interface_name"))
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

	err = h.Controller.Del(payload)
	if err != nil {
		logger.Error("controller-del", err)
		resp.WriteHeader(http.StatusInternalServerError)

		err = marshalError(resp, h.Marshaler, err)
		if err != nil {
			logger.Error("marshal-error", err)
		}
		return
	}

	resp.WriteHeader(http.StatusNoContent)
}
