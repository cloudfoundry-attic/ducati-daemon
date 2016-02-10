package handlers_test

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/rata"
)

func rataWrap(handler http.Handler, method, path string, params rata.Params) (http.Handler, *http.Request) {
	testRoutes := rata.Routes{
		{Name: "wicked_smat", Method: method, Path: path},
	}
	requestGenerator := rata.NewRequestGenerator("", testRoutes)
	testHandlers := rata.Handlers{
		"wicked_smat": handler,
	}

	router, err := rata.NewRouter(testRoutes, testHandlers)
	Expect(err).NotTo(HaveOccurred())

	request, err := requestGenerator.CreateRequest("wicked_smat", params, nil)
	Expect(err).NotTo(HaveOccurred())

	return router, request
}

var _ = Describe("Allocate IP", func() {
	var request *http.Request
	var marshaler *fakes.Marshaler
	var logger *fakes.Logger
	var ipAllocator *fakes.IPAllocator
	var expectedResultBytes []byte
	var handler http.Handler

	BeforeEach(func() {
		logger = &fakes.Logger{}

		marshaler = &fakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal

		ipAllocator = &fakes.IPAllocator{}

		allocateIPHandler := &handlers.AllocateIPHandler{
			Marshaler:   marshaler,
			IPAllocator: ipAllocator,
			Logger:      logger,
		}

		handler, request = rataWrap(allocateIPHandler, "POST", "/ipam/:network_id/:container_id", rata.Params{
			"network_id":   "some-network",
			"container_id": "some-container",
		})

		expectedResult := &types.Result{
			IP4: &types.IPConfig{
				IP: net.IPNet{
					IP:   net.ParseIP("192.168.100.1"),
					Mask: net.ParseIP("192.168.100.1").DefaultMask(),
				},
				Gateway: net.ParseIP("192.168.100.1"),
				Routes: []types.Route{
					{
						Dst: net.IPNet{
							IP:   net.ParseIP("192.168.1.5"),
							Mask: net.ParseIP("192.168.1.5").DefaultMask(),
						},
						GW: net.ParseIP("192.168.1.1"),
					},
				},
			},
		}
		var err error
		expectedResultBytes, err = json.Marshal(expectedResult)
		Expect(err).NotTo(HaveOccurred())

		ipAllocator.AllocateIPReturns(expectedResult, nil)
	})

	It("allocates an IP and returns the result", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(ipAllocator.AllocateIPCallCount()).To(Equal(1))
		Expect(resp.Body.String()).To(MatchJSON(expectedResultBytes))

		Expect(marshaler.MarshalCallCount()).To(Equal(1))
		Expect(resp.Code).To(Equal(http.StatusCreated))

		networkID, containerID := ipAllocator.AllocateIPArgsForCall(0)
		Expect(networkID).To(Equal("some-network"))
		Expect(containerID).To(Equal("some-container"))
	})

	Context("when things go wrong", func() {
		Context("when the allocator fails to allocate", func() {
			BeforeEach(func() {
				ipAllocator.AllocateIPReturns(&types.Result{}, errors.New("failed to allocate"))
			})

			It("should return 500 and log the error", func() {
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(marshaler.MarshalCallCount()).To(Equal(0))
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))

				Expect(logger.ErrorCallCount()).To(Equal(1))
				action, err, _ := logger.ErrorArgsForCall(0)
				Expect(action).To(Equal("allocate-ip"))
				Expect(err).To(MatchError("failed to allocate"))
			})
		})
	})

	Context("when marshaling the result fails", func() {
		It("should return 500 and log the error", func() {
			marshaler.MarshalReturns([]byte(`bad`), errors.New("banana"))

			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))

			Expect(logger.ErrorCallCount()).To(Equal(1))
			action, err, _ := logger.ErrorArgsForCall(0)
			Expect(action).To(Equal("allocate-ip"))
			Expect(err).To(MatchError("banana"))

			Expect(resp.Body.String()).To(BeEmpty())
		})
	})

	Context("when writing the response body fails", func() {
		It("should log the error", func() {
			badResponseWriter := &badResponseWriter{}
			handler.ServeHTTP(badResponseWriter, request)

			Expect(logger.ErrorCallCount()).To(Equal(1))
			action, err, _ := logger.ErrorArgsForCall(0)
			Expect(action).To(Equal("allocate-ip"))
			Expect(err).To(MatchError("failed writing body: some bad writer"))
		})
	})
})
