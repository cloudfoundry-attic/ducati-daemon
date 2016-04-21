package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"lib/testsupport"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/rata"
)

var _ = Describe("CNIDel", func() {
	var (
		logger      *lagertest.TestLogger
		handler     http.Handler
		controller  *fakes.DelController
		request     *http.Request
		unmarshaler *fakes.Unmarshaler
		marshaler   *fakes.Marshaler
		payload     models.CNIDelPayload
	)

	var setPayload = func() {
		payloadBytes, err := json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())
		request.Body = ioutil.NopCloser(bytes.NewBuffer(payloadBytes))
	}

	BeforeEach(func() {
		unmarshaler = &fakes.Unmarshaler{}
		unmarshaler.UnmarshalStub = json.Unmarshal
		marshaler = &fakes.Marshaler{}
		marshaler.MarshalStub = json.Marshal
		logger = lagertest.NewTestLogger("test")
		controller = &fakes.DelController{}

		deleteHandler := &handlers.CNIDel{
			Marshaler:   marshaler,
			Unmarshaler: unmarshaler,
			Logger:      logger,
			Controller:  controller,
		}

		handler, request = rataWrap(deleteHandler, "POST", "/cni/del", rata.Params{})
		payload = models.CNIDelPayload{
			InterfaceName:      "some-interface-name",
			ContainerNamespace: "/some/container/namespace/path",
			ContainerID:        "some-container-id",
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
				Expect(logger).To(gbytes.Say("cni-del.*body-read-failed"))
			})
		})

		Context("when the request body is not valid JSON", func() {
			BeforeEach(func() {
				request.Body = ioutil.NopCloser(strings.NewReader(`{{{`))
			})

			It("should log and respond with status 400", func() {
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, request)

				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(logger).To(gbytes.Say("cni-del.*unmarshal-failed"))
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
					"cni-del.bad-request.*missing-%s", jsonName)))
			},
			Entry("interface", "InterfaceName", "interface_name"),
			Entry("container_namespace_path", "ContainerNamespace", "container_namespace"),
			Entry("container_id", "ContainerID", "container_id"),
		)

	})

	It("passes the payload to controller.Del", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(controller.DelCallCount()).To(Equal(1))
		Expect(controller.DelArgsForCall(0)).To(Equal(payload))
	})

	It("responds with status no content", func() {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, request)

		Expect(resp.Code).To(Equal(http.StatusNoContent))
	})

	Context("when the controller returns an error", func() {
		BeforeEach(func() {
			controller.DelReturns(errors.New("tomato"))
		})

		It("should respond with code 500 and log the error", func() {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, request)

			Expect(logger).To(gbytes.Say("cni-del.controller-del.*tomato"))
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

				Expect(logger).To(gbytes.Say("cni-del.controller-del.*tomato"))
				Expect(logger).To(gbytes.Say("cni-del.marshal-error.*potato"))
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})
})
