package client_test

import (
	"encoding/json"
	"errors"
	"lib/testsupport"
	"net/http"

	"github.com/cloudfoundry-incubator/ducati-daemon/client"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("JSON Client", func() {
	Describe("BuildAndDo", func() {
		var (
			jsonClient  client.JSONClient
			server      *ghttp.Server
			marshaler   *fakes.Marshaler
			unmarshaler *fakes.Unmarshaler

			roundTripper *fakes.RoundTripper
			httpClient   *http.Client

			config client.ClientConfig

			requestPayload []int
			responseResult map[string]string
		)

		BeforeEach(func() {
			server = ghttp.NewServer()
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/some/path"),
				ghttp.RespondWithJSONEncoded(http.StatusCreated, map[string]string{"foo": "bar"}),
			))

			marshaler = &fakes.Marshaler{}
			unmarshaler = &fakes.Unmarshaler{}

			roundTripper = &fakes.RoundTripper{}
			roundTripper.RoundTripStub = http.DefaultTransport.RoundTrip

			httpClient = &http.Client{
				Transport: roundTripper,
			}

			jsonClient = client.JSONClient{
				BaseURL:     server.URL(),
				Marshaler:   marshaler,
				Unmarshaler: unmarshaler,
				HttpClient:  httpClient,
			}

			marshaler.MarshalStub = json.Marshal
			unmarshaler.UnmarshalStub = json.Unmarshal

			requestPayload = []int{1, 2, 3}

			config = client.ClientConfig{
				Action:            "SomeAction",
				Method:            "POST",
				URL:               "/some/path",
				RequestPayload:    &requestPayload,
				ResponseResult:    &responseResult,
				SuccessStatusCode: http.StatusCreated,
				MeaningfulErrors: map[int]error{
					http.StatusTeapot:     errors.New("TEAPOT!"),
					http.StatusBadRequest: errors.New("bad request"),
				},
			}
		})

		AfterEach(func() {
			server.Close()
		})

		It("marshals the request payload into the request body", func() {
			err := jsonClient.BuildAndDo(config)
			Expect(err).NotTo(HaveOccurred())

			Expect(marshaler.MarshalCallCount()).To(Equal(1))
		})

		Context("when the request payload is missing", func() {
			BeforeEach(func() {
				config.RequestPayload = nil
			})

			It("should not try to marshal", func() {
				err := jsonClient.BuildAndDo(config)
				Expect(err).NotTo(HaveOccurred())

				Expect(marshaler.MarshalCallCount()).To(Equal(0))
			})

			It("should not set header content type to application json", func() {
				err := jsonClient.BuildAndDo(config)
				Expect(err).NotTo(HaveOccurred())

				requests := server.ReceivedRequests()
				Expect(requests).Should(HaveLen(1))
				Expect(requests[0].Header["Content-Type"]).To(BeEmpty())
			})
		})

		It("performs the request using the JSON representation of the request payload", func() {
			err := jsonClient.BuildAndDo(config)
			Expect(err).NotTo(HaveOccurred())

			requests := server.ReceivedRequests()
			Expect(requests).Should(HaveLen(1))
			Expect(requests[0].Method).To(Equal("POST"))
			Expect(requests[0].URL.Path).To(Equal("/some/path"))
			Expect(requests[0].Header["Content-Type"][0]).To(Equal("application/json"))
		})

		It("unmarshals the response JSON into the response result", func() {
			err := jsonClient.BuildAndDo(config)
			Expect(err).NotTo(HaveOccurred())

			Expect(unmarshaler.UnmarshalCallCount()).To(Equal(1))
		})

		Context("when the response status code is the wrong success type", func() {
			BeforeEach(func() {
				mockHttpClient := &fakes.HTTPClient{}
				mockHttpClient.DoReturns(&http.Response{
					StatusCode: http.StatusOK,
					Body:       &testsupport.BadReader{},
				}, nil)

				jsonClient.HttpClient = mockHttpClient
			})

			It("returns an error", func() {
				err := jsonClient.BuildAndDo(config)
				Expect(err).To(MatchError("unexpected status code on SomeAction: expected 201 but got 200"))
			})
		})

		Context("when the response status code is not success", func() {
			BeforeEach(func() {
				mockHttpClient := &fakes.HTTPClient{}
				mockHttpClient.DoReturns(&http.Response{
					StatusCode: http.StatusTeapot,
					Body:       &testsupport.BadReader{},
				}, nil)

				jsonClient.HttpClient = mockHttpClient
			})

			It("maps the status codes to meaningful error and does not unmarshal", func() {
				err := jsonClient.BuildAndDo(config)
				Expect(err).To(MatchError("TEAPOT!"))
				Expect(unmarshaler.UnmarshalCallCount()).To(Equal(0))
			})
		})

		Context("when the BaseURL has a trailing slash", func() {
			It("handles just fine", func() {
				jsonClient.BaseURL += "/"
				err := jsonClient.BuildAndDo(config)
				Expect(err).NotTo(HaveOccurred())
				requests := server.ReceivedRequests()
				Expect(requests).Should(HaveLen(1))
				Expect(requests[0].URL.Path).To(Equal("/some/path"))
			})
		})

		Context("when there are errors", func() {
			Context("the URL cannot be built", func() {
				BeforeEach(func() {
					jsonClient.BaseURL = "::/not-#%#%#-a-valid-base-url"
				})

				It("returns a meaningful error", func() {
					err := jsonClient.BuildAndDo(config)
					Expect(err).To(MatchError("build url: parse ::/not-: missing protocol scheme"))
				})
			})

			Context("the request body cannot be marshalled", func() {
				BeforeEach(func() {
					marshaler.MarshalReturns(nil, errors.New("something went wrong"))
				})

				It("should return an error", func() {
					err := jsonClient.BuildAndDo(config)
					Expect(err).To(MatchError("failed to marshal request: something went wrong"))
				})
			})

			Context("reading the response body read fails", func() {
				BeforeEach(func() {
					mockHttpClient := &fakes.HTTPClient{}
					mockHttpClient.DoReturns(&http.Response{
						StatusCode: 201,
						Body:       &testsupport.BadReader{},
					}, nil)

					jsonClient.HttpClient = mockHttpClient
				})

				It("should return an error", func() {
					err := jsonClient.BuildAndDo(config)
					Expect(err).To(MatchError("failed to read response body: banana"))
				})
			})

			Context("the response JSON cannot be unmarshaled", func() {
				BeforeEach(func() {
					unmarshaler.UnmarshalReturns(errors.New("something went wrong"))
				})

				It("should return an error", func() {
					err := jsonClient.BuildAndDo(config)
					Expect(err).To(MatchError("failed to unmarshal result: something went wrong"))
				})
			})
		})
	})
})
