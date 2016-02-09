package handlers

import (
	"fmt"
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/tedsuo/rata"
)

type GetHandler struct {
	Store     store.Store
	Marshaler marshaler
	Logger    Logger
}

func (h *GetHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	id := rata.Param(req, "container_id")

	container, err := h.Store.Get(id)
	if err != nil {
		if err == store.NotFoundError {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
		h.Logger.Error("store-get", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonResponse, err := h.Marshaler.Marshal(container)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = resp.Write(jsonResponse)
	if err != nil {
		h.Logger.Error("store-get", fmt.Errorf("failed writing body: %s", err))
		return
	}
}
