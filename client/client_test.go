package client_test

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/client"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/onsi/gomega/ghttp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		c           client.DaemonClient
		server      *ghttp.Server
		marshaler   *fakes.Marshaler
		unmarshaler *fakes.Unmarshaler
		container   models.Container

		roundTripper *fakes.RoundTripper
		httpClient   *http.Client
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
		marshaler = &fakes.Marshaler{}
		unmarshaler = &fakes.Unmarshaler{}

		roundTripper = &fakes.RoundTripper{}
		transport := http.DefaultTransport
		roundTripper.RoundTripStub = transport.RoundTrip

		httpClient = &http.Client{
			Transport: roundTripper,
		}

		c = client.DaemonClient{
			BaseURL:     server.URL(),
			Marshaler:   marshaler,
			Unmarshaler: unmarshaler,
			HttpClient:  httpClient,
		}

		marshaler.MarshalStub = json.Marshal

		container = models.Container{
			ID: "some-container-id",
		}
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("ContainerUp", func() {
		var cniPayload handlers.NetworksSetupContainerPayload

		BeforeEach(func() {
			cniPayload = handlers.NetworksSetupContainerPayload{
				Args:               "FOO=BAR;ABC=123",
				ContainerNamespace: "/some/namespace/path",
				InterfaceName:      "interface-name",
				VNI:                99,
				IPAM:               types.Result{},
			}

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/networks/some-network-id/some-container-id"),
				ghttp.VerifyJSONRepresenting(cniPayload),
				ghttp.VerifyHeaderKV("Content-type", "application/json"),
				ghttp.RespondWith(http.StatusCreated, nil),
			))
		})

		It("should POST to the /networks/:network_id/:container_id endpoint with a CNI payload", func() {
			Expect(c.ContainerUp("some-network-id", "some-container-id", cniPayload)).To(Succeed())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
			Expect(marshaler.MarshalCallCount()).To(Equal(1))
			Expect(marshaler.MarshalArgsForCall(0)).To(Equal(cniPayload))
		})

		It("uses the provided HTTP client", func() {
			Expect(c.ContainerUp("some-network-id", "some-container-id", cniPayload)).To(Succeed())

			Expect(roundTripper.RoundTripCallCount()).To(Equal(1))
			Expect(roundTripper.RoundTripArgsForCall(0).URL.Path).To(Equal("/networks/some-network-id/some-container-id"))
		})

		Context("when an error occurs", func() {
			Context("when the payload fails to marshal", func() {
				It("returns an error", func() {
					marshaler.MarshalReturns(nil, errors.New("explosion with marshal"))

					err := c.ContainerUp("", "", cniPayload)
					Expect(err).To(MatchError("failed to marshal cni payload: explosion with marshal"))
				})
			})

			Context("when the request cannot be performed", func() {
				It("returns an error", func() {
					c = client.DaemonClient{
						BaseURL:   "%%%%",
						Marshaler: marshaler,
					}

					err := c.ContainerUp("", "", cniPayload)
					Expect(err).To(MatchError(ContainSubstring("failed to perform request: parse")))
				})
			})

			Context("when the http request fails", func() {
				BeforeEach(func() {
					server.Reset()
					server.AppendHandlers(ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/networks/some-network-id/some-container-id"),
						ghttp.RespondWith(http.StatusInternalServerError, nil),
					))
				})

				It("should return an error", func() {
					err := c.ContainerUp("some-network-id", "some-container-id", cniPayload)
					Expect(err).To(MatchError(`unexpected status code on ContainerUp: expected 201 but got 500`))
				})
			})
		})
	})

	Describe("SaveContainer", func() {
		BeforeEach(func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/containers"),
				ghttp.VerifyJSON(`{"id":"some-container-id"}`),
				ghttp.VerifyHeaderKV("Content-type", "application/json"),
				ghttp.RespondWith(http.StatusCreated, nil),
			))

			marshaler.MarshalReturns([]byte(`{"id":"some-container-id"}`), nil)
		})

		It("should call the backend to save the container", func() {
			Expect(c.SaveContainer(container)).To(Succeed())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
			Expect(marshaler.MarshalCallCount()).To(Equal(1))
			Expect(marshaler.MarshalArgsForCall(0)).To(Equal(container))
		})

		It("uses the provided HTTP client", func() {
			Expect(c.SaveContainer(container)).To(Succeed())

			Expect(roundTripper.RoundTripCallCount()).To(Equal(1))
			Expect(roundTripper.RoundTripArgsForCall(0).URL.Path).To(Equal("/containers"))
		})

		Context("when an error occurs", func() {
			Context("when the container fails to marshal", func() {
				It("should return an error", func() {
					marshaler.MarshalReturns(nil, errors.New("explosion with marshal"))

					err := c.SaveContainer(container)
					Expect(err).To(MatchError("failed to marshal container: explosion with marshal"))
				})
			})

			Context("when the request cannot be performed", func() {
				It("should return an error", func() {
					c = client.DaemonClient{
						BaseURL:   "%%%%",
						Marshaler: marshaler,
					}

					err := c.SaveContainer(container)
					Expect(err).To(MatchError(ContainSubstring("failed to perform request: parse")))
				})
			})

			Context("when the http request fails", func() {
				BeforeEach(func() {
					server.Reset()
					server.AppendHandlers(ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/containers"),
						ghttp.RespondWith(http.StatusInternalServerError, nil),
					))
				})

				It("should return an error", func() {
					err := c.SaveContainer(container)
					Expect(err).To(MatchError(`unexpected status code on SaveContainer: expected 201 but got 500`))
				})
			})
		})
	})

	Describe("ListContainers", func() {
		var expectedContainer models.Container

		BeforeEach(func() {
			expectedContainer = models.Container{
				ID:     "some-id",
				IP:     "192.168.1.9",
				MAC:    "HH:HH:HH:HH:HH",
				HostIP: "10.0.0.0",
			}

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/containers"),
				ghttp.RespondWithJSONEncoded(http.StatusOK, []models.Container{expectedContainer}),
			))
		})

		It("should call the backend to get all the containers", func() {
			unmarshaler.UnmarshalStub = json.Unmarshal

			container, err := c.ListContainers()
			Expect(err).NotTo(HaveOccurred())

			Expect(container).To(ConsistOf(expectedContainer))
		})

		Context("when an error occurs", func() {
			Context("when the json cannot be unmarshaled", func() {
				BeforeEach(func() {
					unmarshaler.UnmarshalReturns(errors.New("something went wrong"))
				})

				It("should return an error", func() {
					_, err := c.ListContainers()
					Expect(err).To(MatchError("failed to unmarshal containers: something went wrong"))
				})
			})

			Context("when the endpoint responds with the wrong status", func() {
				BeforeEach(func() {
					server.SetHandler(0, ghttp.RespondWith(http.StatusTeapot, nil))
				})

				It("should return and error", func() {
					_, err := c.ListContainers()
					Expect(err).To(MatchError(`unexpected status code on ListContainers: expected 200 but got 418`))
				})
			})
		})
	})

	Describe("RemoveContainer", func() {
		BeforeEach(func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("DELETE", "/containers/some-container"),
				ghttp.RespondWith(http.StatusNoContent, nil),
			))
		})

		It("should call the backend to remove the container", func() {
			Expect(c.RemoveContainer("some-container")).To(Succeed())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
		})

		It("uses the provided HTTP client", func() {
			Expect(c.RemoveContainer("some-container")).To(Succeed())

			Expect(roundTripper.RoundTripCallCount()).To(Equal(1))
			Expect(roundTripper.RoundTripArgsForCall(0).URL.Path).To(Equal("/containers/some-container"))
		})

		Context("when an error occurs", func() {
			Context("when the container does not exist", func() {
				BeforeEach(func() {
					server.Reset()
					server.AppendHandlers(ghttp.CombineHandlers(
						ghttp.VerifyRequest("DELETE", "/containers/non-existent-container-id"),
						ghttp.RespondWith(http.StatusNotFound, nil),
					))
				})

				It("it should return a RecordNotFound error", func() {
					err := c.RemoveContainer("non-existent-container-id")
					Expect(err).To(Equal(client.RecordNotFoundError))
				})
			})

			Context("when the request cannot be constructed", func() {
				It("should return an error", func() {
					c = client.DaemonClient{
						BaseURL:   "%%%%",
						Marshaler: marshaler,
					}

					err := c.RemoveContainer("whatever")
					Expect(err).To(MatchError(ContainSubstring("failed to construct request: parse")))
				})
			})

			Context("when the request fails", func() {
				BeforeEach(func() {
					c.BaseURL = "http://0.0.0.0:1"
				})

				It("returns a error", func() {
					err := c.RemoveContainer("whatever")
					Expect(err).To(MatchError(ContainSubstring("failed to perform request")))
					Expect(err).To(MatchError(ContainSubstring("connection refused")))
				})
			})

			Context("when the response status code is unexpected", func() {
				BeforeEach(func() {
					server.Reset()
					server.AppendHandlers(ghttp.CombineHandlers(
						ghttp.VerifyRequest("DELETE", "/containers/whatever"),
						ghttp.RespondWith(http.StatusInternalServerError, nil),
					))
				})

				It("should return an error", func() {
					err := c.RemoveContainer("whatever")
					Expect(err).To(MatchError(`unexpected status code on RemoveContainer: expected 204 but got 500`))
				})
			})
		})
	})

	Describe("AllocateIP", func() {
		var returnedResult types.Result

		BeforeEach(func() {
			returnedResult = types.Result{
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

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/ipam/some-network-id/some-container-id"),
				ghttp.VerifyHeaderKV("Content-type", "application/json"),
				ghttp.RespondWithJSONEncoded(http.StatusCreated, returnedResult),
			))
		})

		It("should call the backend to allocate an IP", func() {
			unmarshaler.UnmarshalStub = json.Unmarshal

			receivedResult, err := c.AllocateIP("some-network-id", "some-container-id")
			Expect(err).NotTo(HaveOccurred())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))

			Expect(receivedResult).To(Equal(returnedResult))
		})

		It("uses the provided HTTP client", func() {
			_, err := c.AllocateIP("some-network-id", "some-container-id")
			Expect(err).NotTo(HaveOccurred())

			Expect(roundTripper.RoundTripCallCount()).To(Equal(1))
			Expect(roundTripper.RoundTripArgsForCall(0).URL.Path).To(Equal("/ipam/some-network-id/some-container-id"))
		})

		Context("when an error occurs", func() {
			Context("when the request cannot be performed", func() {
				It("should return an error", func() {
					c.BaseURL = "%%%%"

					_, err := c.AllocateIP("some-network-id", "some-container-id")
					Expect(err).To(MatchError(ContainSubstring("failed to perform request: parse")))
				})
			})

			Context("when the http response code is a 409 Conflict", func() {
				It("should return an ipam.NoMoreAddressesError", func() {
					server.SetHandler(0, ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/ipam/some-network-id/some-container-id"),
						ghttp.RespondWith(http.StatusConflict, `{ "error": "boom" }`),
					))

					_, err := c.AllocateIP("some-network-id", "some-container-id")
					Expect(err).To(Equal(ipam.NoMoreAddressesError))
				})
			})

			Context("when the http response code is unexpected", func() {
				It("should return an error", func() {
					server.SetHandler(0, ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/ipam/some-network-id/some-container-id"),
						ghttp.RespondWith(http.StatusTeapot, `{{{`),
					))

					_, err := c.AllocateIP("some-network-id", "some-container-id")
					Expect(err).To(MatchError(`unexpected status code on AllocateIP: expected 201 but got 418`))
				})
			})

			Context("when the container fails to marshal", func() {
				It("should return an error", func() {
					unmarshaler.UnmarshalReturns(errors.New("explosion with marshal"))

					_, err := c.AllocateIP("some-network-id", "some-container-id")
					Expect(err).To(MatchError("failed to unmarshal IPAM result: explosion with marshal"))
				})
			})
		})
	})

	Describe("ReleaseIP", func() {
		BeforeEach(func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("DELETE", "/ipam/some-network-id/some-container-id"),
				ghttp.RespondWithJSONEncoded(http.StatusNoContent, nil),
			))
		})

		It("should call the backend to release an IP", func() {
			err := c.ReleaseIP("some-network-id", "some-container-id")
			Expect(err).NotTo(HaveOccurred())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
		})

		It("uses the provided HTTP client", func() {
			err := c.ReleaseIP("some-network-id", "some-container-id")
			Expect(err).NotTo(HaveOccurred())

			Expect(roundTripper.RoundTripCallCount()).To(Equal(1))
			Expect(roundTripper.RoundTripArgsForCall(0).URL.Path).To(Equal("/ipam/some-network-id/some-container-id"))
		})

		Context("when the request cannot be constructed", func() {
			It("should return an error", func() {
				c.BaseURL = "%%%%"

				err := c.ReleaseIP("some-network-id", "some-container-id")
				Expect(err).To(MatchError(ContainSubstring("failed to construct request: parse")))
			})
		})

		Context("when the http response code is unexpected", func() {
			It("should return an error", func() {
				server.SetHandler(0, ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/ipam/some-network-id/some-container-id"),
					ghttp.RespondWithJSONEncoded(http.StatusTeapot, nil),
				))

				err := c.ReleaseIP("some-network-id", "some-container-id")
				Expect(err).To(MatchError(`unexpected status code on ReleaseIP: expected 204 but got 418`))
			})
		})

		Context("when the request fails", func() {
			BeforeEach(func() {
				c.BaseURL = "http://0.0.0.0:1"
			})

			It("returns a error", func() {
				err := c.ReleaseIP("some-network-id", "some-container-id")
				Expect(err).To(MatchError(ContainSubstring("connection refused")))
			})
		})
	})
})
