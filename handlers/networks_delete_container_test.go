package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	exec_fakes "github.com/cloudfoundry-incubator/ducati-daemon/executor/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/rata"
)

var _ = Describe("NetworksDeleteContainer", func() {
	var (
		logger    *lagertest.TestLogger
		datastore *fakes.Store
		executor  *exec_fakes.Executor
		deletor   *fakes.Deletor
		handler   http.Handler
		request   *http.Request
		osLocker  *fakes.OSThreadLocker
	)

	BeforeEach(func() {
		osLocker = &fakes.OSThreadLocker{}

		logger = lagertest.NewTestLogger("test")
		datastore = &fakes.Store{}
		executor = &exec_fakes.Executor{}
		deletor = &fakes.Deletor{}

		deleteHandler := &handlers.NetworksDeleteContainer{
			Logger:         logger,
			Datastore:      datastore,
			Deletor:        deletor,
			OSThreadLocker: osLocker,
		}

		handler, request = rataWrap(deleteHandler, "DELETE", "/networks/:network_id/:container_id", rata.Params{
			"network_id":   "some-network-id",
			"container_id": "some-container-id",
		})

	})

	It("deletes the container from the network", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(deletor.DeleteCallCount()).To(Equal(1))
		networkID, containerID := deletor.DeleteArgsForCall(0)
		Expect(networkID).To(Equal("some-network-id"))
		Expect(containerID).To(Equal("some-container-id"))
	})

	It("deletes the container from the datastore", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(datastore.DeleteCallCount()).To(Equal(1))
		containerID := datastore.DeleteArgsForCall(0)
		Expect(containerID).To(Equal("some-container-id"))
	})

	It("responds with status no content", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(resp.Code).To(Equal(http.StatusNoContent))
	})

	It("locks and unlocks the os thread", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(osLocker.LockOSThreadCallCount()).To(Equal(1))
		Expect(osLocker.UnlockOSThreadCallCount()).To(Equal(1))
	})

	Context("when deleting the container from the network fails", func() {
		BeforeEach(func() {
			deletor.DeleteReturns(errors.New("some-deletor-error"))
		})

		It("should log and respond with status 500", func() {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(logger).To(gbytes.Say("networks-delete-containers.deletor.delete-failed.*some-deletor-error"))
		})

		It("should not remove the container from the datastore", func() {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(datastore.DeleteCallCount()).To(Equal(0))
		})

		It("locks and unlocks the os thread", func() {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(osLocker.LockOSThreadCallCount()).To(Equal(1))
			Expect(osLocker.UnlockOSThreadCallCount()).To(Equal(1))
		})
	})

	Context("when deleting from the datastore fails", func() {
		BeforeEach(func() {
			datastore.DeleteReturns(errors.New("some-datastore-error"))
		})

		It("should log and respond with status 500", func() {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			Expect(logger).To(gbytes.Say("networks-delete-containers.datastore.delete-failed.*some-datastore-error"))
		})

		It("locks and unlocks the os thread", func() {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(osLocker.LockOSThreadCallCount()).To(Equal(1))
			Expect(osLocker.UnlockOSThreadCallCount()).To(Equal(1))
		})
	})
})