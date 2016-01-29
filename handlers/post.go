package handlers

import (
	"io/ioutil"
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/models"
)

type storePutter interface {
	Put(container models.Container) error
}

type unmarshaler interface {
	Unmarshal(input []byte, output interface{}) error
}

type PostHandler struct {
	Store       storePutter
	Unmarshaler unmarshaler
	Logger      Logger
}

func (h *PostHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
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

	err = h.Store.Put(container)
	if err != nil {
		h.Logger.Error("store-put", err, nil)
		resp.WriteHeader(http.StatusBadGateway)
		return
	}

	resp.WriteHeader(http.StatusNoContent)
}
