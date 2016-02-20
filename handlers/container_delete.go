package handlers

import (
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/tedsuo/rata"
)

type DeleteHandler struct {
	Store  store.Store
	Logger Logger
}

func (h *DeleteHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	id := rata.Param(req, "container_id")

	err := h.Store.Delete(id)
	if err != nil {
		if err == store.NotFoundError {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
		h.Logger.Error("store-delete", err, nil)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusNoContent)
}
