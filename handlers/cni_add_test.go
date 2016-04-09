package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/testsupport"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/rata"
)

var _ = Describe("CNIAdd", func() {
	var (
		unmarshaler         *fakes.Unmarshaler
		logger              *lagertest.TestLogger
		controller          *fakes.AddController
		handler             http.Handler
		request             *http.Request
		marshaler           *fakes.Marshaler
		expectedResultBytes []byte
		payload             models.CNIAddPayload
	)

	var setPayload = func() {
		payloadBytes, err := json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())
		request.Body = ioutil.NopCloser(bytes.NewBuffer(payloadBytes))
	}

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		marshaler = &fakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal
		unmarshaler = &fakes.Unmarshaler{}
		unmarshaler.UnmarshalStub = json.Unmarshal
		controller = &fakes.AddController{}

		setupHandler := &handlers.CNIAdd{
			Logger:      logger,
			Unmarshaler: unmarshaler,
			Marshaler:   marshaler,
			Controller:  controller,
		}

		ipamResult := &types.Result{
			IP4: &types.IPConfig{
				IP: net.IPNet{
					IP: net.ParseIP("192.168.100.2"),
				},
			},
		}
		controller.AddReturns(ipamResult, nil)

		var err error
		expectedResultBytes, err = json.Marshal(ipamResult)
		Expect(err).NotTo(HaveOccurred())

		handler, request = rataWrap(setupHandler, "POST", "/cni/add", rata.Params{})
		payload = models.CNIAddPayload{
			Args:               "FOO=BAR;ABC=123",
			ContainerNamespace: "/some/namespace/path",
			InterfaceName:      "interface-name",
			ContainerID:        "container-id",
			Network: models.NetworkPayload{
				ID:  "network-id-1",
				App: "some-app-guid",
			},
		}
		setPayload()
	})

	Describe("parsing and validating input", func() {
		Context("when the request body cannot be read", func() {
			BeforeEach(func() {
				request.Body = ioutil.NopCloser(&testsupport.BadReader{})
			})

			It("should log and respond with status 400", func() {
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(logger).To(gbytes.Say("cni-add.*body-read-failed"))
			})
		})

		Context("when unmarshalling payload fails", func() {
			BeforeEach(func() {
				request.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(`{}`)))
			})

			It("logs an error and responds with code 400", func() {
				unmarshaler.UnmarshalReturns(errors.New("some-unmarshal-error"))
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(logger).To(gbytes.Say("cni-add.unmarshal-failed.*some-unmarshal-error"))

				Expect(controller.AddCallCount()).To(BeZero())
			})
		})

		DescribeTable("missing payload fields",
			func(paramToRemove, jsonName string) {
				field := reflect.ValueOf(&payload).Elem().FieldByName(paramToRemove)
				if !field.IsValid() {
					Fail("invalid test: payload does not have a field named " + paramToRemove)
				}
				field.Set(reflect.Zero(field.Type()))
				setPayload()

				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(logger).To(gbytes.Say(fmt.Sprintf(
					"cni-add.bad-request.*missing-%s", jsonName)))
			},
			Entry("interface", "InterfaceName", "interface_name"),
			Entry("container_namespace_path", "ContainerNamespace", "container_namespace"),
			Entry("container_id", "ContainerID", "container_id"),
		)

		DescribeTable("missing payload fields",
			func(paramToRemove, jsonName string) {
				field := reflect.ValueOf(&payload.Network).Elem().FieldByName(paramToRemove)
				if !field.IsValid() {
					Fail("invalid test: payload does not have a field named " + paramToRemove)
				}
				field.Set(reflect.Zero(field.Type()))
				setPayload()

				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(logger).To(gbytes.Say(fmt.Sprintf(
					"cni-add.bad-request.*missing-%s", jsonName)))
			},
			Entry("network id", "ID", "network_id"),
			Entry("app", "App", "app"),
		)
	})

	It("passes the payload to controller.Add", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(resp.Code).To(Equal(http.StatusCreated))
		Expect(controller.AddCallCount()).To(Equal(1))
		Expect(controller.AddArgsForCall(0)).To(Equal(payload))
	})

	It("responds with the JSON encoding of the IPAM result", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)
		Expect(resp.Body.String()).To(MatchJSON(expectedResultBytes))
	})

	Context("when the controller returns a AlreadyOnNetworkError", func() {
		BeforeEach(func() {
			controller.AddReturns(nil, ipam.AlreadyOnNetworkError)
		})
		It("should log and return a 400", func() {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(resp.Body.String()).To(MatchJSON(`{ "error": "already on this network"}`))
			Expect(logger).To(gbytes.Say(`cni-add.controller-add.*already on this network`))
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Context("when the controller returns an ipam.NoMoreAddressesError", func() {
		BeforeEach(func() {
			controller.AddReturns(nil, ipam.NoMoreAddressesError)
		})

		It("should log and return a 409 status with JSON body encoding the error message", func() {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(resp.Body.String()).To(MatchJSON(`{ "error": "no addresses available" }`))
			Expect(logger).To(gbytes.Say(`cni-add.controller-add.*no addresses available`))
			Expect(resp.Code).To(Equal(http.StatusConflict))
		})
	})

	Context("when the controller returns any other error", func() {
		BeforeEach(func() {
			controller.AddReturns(nil, errors.New("tomato"))
		})

		It("should respond with code 500 and log the error", func() {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(logger).To(gbytes.Say("cni-add.controller-add.*tomato"))
			Expect(resp.Body.String()).To(MatchJSON(`{ "error": "tomato" }`))
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		})

		Context("when writing the error response fails", func() {
			BeforeEach(func() {
				marshaler.MarshalReturns(nil, errors.New("potato"))
			})
			It("should log both errors", func() {
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(logger).To(gbytes.Say("cni-add.controller-add.*tomato"))
				Expect(logger).To(gbytes.Say("cni-add.marshal-error.*potato"))
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})

	Context("when marshaling the result fails", func() {
		It("should return 500 and log the error", func() {
			marshaler.MarshalReturns([]byte(`bad`), errors.New("banana"))

			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(logger).To(gbytes.Say("cni-add.marshal-result.*banana"))
			Expect(resp.Body.String()).To(BeEmpty())
			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Context("when writing the response body fails", func() {
		It("should log the error", func() {
			badResponseWriter := &badResponseWriter{}
			handler.ServeHTTP(badResponseWriter, request)

			Expect(logger).To(gbytes.Say("cni-add.marshal-error.*failed writing body: some bad writer"))
		})
	})
})
