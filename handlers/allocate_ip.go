package handlers

import (
	"fmt"
	"net/http"

	"github.com/appc/cni/pkg/types"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter -o ../fakes/ip_allocator.go --fake-name IPAllocator . ipAllocator
type ipAllocator interface {
	AllocateIP(networkID, containerID string) (*types.Result, error)
	ReleaseIP(networkID, containerID string) error
}

type AllocateIPHandler struct {
	Marshaler   marshaler
	Logger      Logger
	IPAllocator ipAllocator
}

func (h *AllocateIPHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	h.Logger.Debug("handling allocate request")
	networkID := rata.Param(req, "network_id")
	containerID := rata.Param(req, "container_id")

	result, err := h.IPAllocator.AllocateIP(networkID, containerID)
	if err != nil {
		fmt.Printf("foo error: %s\n", err)
		h.Logger.Error("allocate-ip", err)
		resp.WriteHeader(http.StatusInternalServerError)
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
