package handlers

import (
	"fmt"
	"net/http"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter -o ../fakes/ip_allocator.go --fake-name IPAllocator . ipAllocator
type ipAllocator interface {
	AllocateIP(networkID, containerID string) (*types.Result, error)
	ReleaseIP(networkID, containerID string) error
}

type IPAMAllocate struct {
	Marshaler   marshaler
	Logger      Logger
	IPAllocator ipAllocator
}

func (h *IPAMAllocate) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	h.Logger.Debug("handling allocate request")
	networkID := rata.Param(req, "network_id")
	containerID := rata.Param(req, "container_id")

	result, err := h.IPAllocator.AllocateIP(networkID, containerID)
	if err != nil {
		h.Logger.Error("allocate-ip", err)
		switch err {
		case ipam.NoMoreAddressesError:
			resp.WriteHeader(http.StatusConflict)
		default:
			resp.WriteHeader(http.StatusInternalServerError)
		}
		marshalError(h.Logger, resp, h.Marshaler, err)
		return
	}

	jsonBodyBytes, err := h.Marshaler.Marshal(result)
	if err != nil {
		h.Logger.Error("allocate-ip", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusCreated)

	_, err = resp.Write(jsonBodyBytes)
	if err != nil {
		h.Logger.Error("allocate-ip", fmt.Errorf("failed writing body: %s", err))
		return
	}
}
