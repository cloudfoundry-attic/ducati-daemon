package handlers

import (
	"net/http"

	"github.com/tedsuo/rata"
)

type ReleaseIPHandler struct {
	Logger      Logger
	Marshaler   marshaler
	IPAllocator ipAllocator
}

func (h *ReleaseIPHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	networkID := rata.Param(req, "network_id")
	containerID := rata.Param(req, "container_id")

	err := h.IPAllocator.ReleaseIP(networkID, containerID)
	if err != nil {
		h.Logger.Error("release-ip", err)
		resp.WriteHeader(http.StatusInternalServerError)
		marshalError(h.Logger, resp, h.Marshaler, err)
		return
	}

	resp.WriteHeader(http.StatusNoContent)
}
