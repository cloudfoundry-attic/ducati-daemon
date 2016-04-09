package handlers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/add_controller.go --fake-name AddController . addController
type addController interface {
	Add(models.CNIAddPayload) (*types.Result, error)
}

type CNIAdd struct {
	Unmarshaler marshal.Unmarshaler
	Logger      lager.Logger
	Marshaler   marshal.Marshaler
	Controller  addController
}

func (h *CNIAdd) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := h.Logger.Session("cni-add")

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

	logger = logger.WithData(lager.Data{"payload": payload})
	defer logger.Info("container-add-complete")

	if payload.InterfaceName == "" {
		logger.Error("bad-request", errors.New("missing-interface_name"))
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.Network.ID == "" {
		logger.Error("bad-request", errors.New("missing-network_id"))
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.Network.App == "" {
		logger.Error("bad-request", errors.New("missing-app"))
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

	ipamResult, err := h.Controller.Add(payload)
	if err != nil {
		logger.Error("controller-add", err)
		switch err {
		case ipam.AlreadyOnNetworkError:
			resp.WriteHeader(http.StatusBadRequest)
		case ipam.NoMoreAddressesError:
			resp.WriteHeader(http.StatusConflict)
		default:
			resp.WriteHeader(http.StatusInternalServerError)
		}

		err = marshalError(resp, h.Marshaler, err)
		if err != nil {
			logger.Error("marshal-error", err)
		}
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
		logger.Error("marshal-error", fmt.Errorf("failed writing body: %s", err))
		return
	}
}
