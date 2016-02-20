package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/rata"
)

var _ = Describe("Release IP", func() {
	var request *http.Request
	var marshaler *fakes.Marshaler
	var logger *fakes.Logger
	var ipAllocator *fakes.IPAllocator
	var handler http.Handler

	BeforeEach(func() {
		logger = &fakes.Logger{}

		marshaler = &fakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal

		ipAllocator = &fakes.IPAllocator{}

		releaseIPHandler := &handlers.IPAMRelease{
			Marshaler:   marshaler,
			IPAllocator: ipAllocator,
			Logger:      logger,
		}

		handler, request = rataWrap(releaseIPHandler, "DELETE", "/ipam/:network_id/:container_id", rata.Params{
			"network_id":   "some-network",
			"container_id": "some-container",
		})

		ipAllocator.ReleaseIPReturns(nil)
	})

	It("releases an IP", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(ipAllocator.ReleaseIPCallCount()).To(Equal(1))
		Expect(resp.Body.String()).To(BeEmpty())

		Expect(resp.Code).To(Equal(http.StatusNoContent))

		networkID, containerID := ipAllocator.ReleaseIPArgsForCall(0)
		Expect(networkID).To(Equal("some-network"))
		Expect(containerID).To(Equal("some-container"))
	})

	Context("when the allocator.ReleaseIP call fails", func() {
		BeforeEach(func() {
			ipAllocator.ReleaseIPReturns(errors.New(`{"boom":"bang"}`))
		})

		It("should return 500 and log the error", func() {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(marshaler.MarshalCallCount()).To(Equal(1))
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))

			Expect(resp.Body.String()).To(MatchJSON(`{ "error": "{\"boom\":\"bang\"}" }`))

			Expect(logger.ErrorCallCount()).To(Equal(1))
			action, err, _ := logger.ErrorArgsForCall(0)
			Expect(action).To(Equal("release-ip"))
			Expect(err).To(MatchError(`{"boom":"bang"}`))
		})

	})
})
