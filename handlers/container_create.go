package handlers

import (
	"io/ioutil"
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
)

type storeCreator interface {
	Create(container models.Container) error
}

type unmarshaler interface {
	Unmarshal(input []byte, output interface{}) error
}

type ContainerCreate struct {
	Store       storeCreator
	Unmarshaler unmarshaler
	Logger      Logger
}

func (h *ContainerCreate) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	requestBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	var container models.Container
	err = h.Unmarshaler.Unmarshal(requestBytes, &container)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.Store.Create(container)
	if err != nil {
		if err == store.RecordExistsError {
			resp.WriteHeader(http.StatusConflict)
			return
		}

		h.Logger.Error("store-put", err, nil)
		resp.WriteHeader(http.StatusBadGateway)
		return
	}

	resp.WriteHeader(http.StatusCreated)
}
