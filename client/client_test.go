package client_test

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/client"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/testsupport"
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

	Describe("ListNetworkContainers", func() {
		var expectedContainers []models.Container

		BeforeEach(func() {
			expectedContainers = []models.Container{
				models.Container{
					ID:     "some-id",
					IP:     "192.168.1.9",
					MAC:    "HH:HH:HH:HH:HH",
					HostIP: "10.0.0.0",
				},
				models.Container{
					ID:     "some-other-id",
					IP:     "192.168.1.10",
					MAC:    "HH:HH:HH:HH:HA",
					HostIP: "10.0.0.0",
				},
			}
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/networks/some-network-id"),
				ghttp.RespondWithJSONEncoded(http.StatusOK, expectedContainers),
			))

			unmarshaler.UnmarshalStub = json.Unmarshal
		})

		It("should GET /networks/:network_id", func() {
			containers, err := c.ListNetworkContainers("some-network-id")
			Expect(err).NotTo(HaveOccurred())

			Expect(server.ReceivedRequests()).Should(HaveLen(1))
			Expect(unmarshaler.UnmarshalCallCount()).To(Equal(1))
			Expect(containers).To(ConsistOf(expectedContainers))
		})

		It("uses the provided HTTP client", func() {
			_, err := c.ListNetworkContainers("some-network-id")
			Expect(err).NotTo(HaveOccurred())

			Expect(roundTripper.RoundTripCallCount()).To(Equal(1))
			Expect(roundTripper.RoundTripArgsForCall(0).URL.Path).To(Equal("/networks/some-network-id"))
		})

		Context("when an error occurs", func() {
			Context("when the request cannot be performed", func() {
				It("returns an error", func() {
					c = client.DaemonClient{
						BaseURL:   "%%%%",
						Marshaler: marshaler,
					}

					_, err := c.ListNetworkContainers("some-network-id")
					Expect(err).To(MatchError(ContainSubstring("failed to perform request: parse")))
				})
			})

			Context("when the endpoint responds with the wrong status", func() {
				BeforeEach(func() {
					server.SetHandler(0, ghttp.RespondWith(http.StatusTeapot, nil))
				})

				It("should return and error", func() {
					_, err := c.ListNetworkContainers("some-network-id")
					Expect(err).To(MatchError(`unexpected status code on ListNetworkContainers: expected 200 but got 418`))
				})
			})

			Context("when the response JSON cannot be unmarshaled", func() {
				BeforeEach(func() {
					unmarshaler.UnmarshalReturns(errors.New("something went wrong"))
				})

				It("should return an error", func() {
					_, err := c.ListNetworkContainers("some-network-id")
					Expect(err).To(MatchError("failed to unmarshal containers: something went wrong"))
				})
			})
		})
	})

	Describe("ContainerUp", func() {
		var cniPayload models.NetworksSetupContainerPayload
		var returnedResult types.Result

		BeforeEach(func() {
			cniPayload = models.NetworksSetupContainerPayload{
				Args:               "FOO=BAR;ABC=123",
				ContainerNamespace: "/some/namespace/path",
				InterfaceName:      "interface-name",
				VNI:                99,
			}

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
				ghttp.VerifyRequest("POST", "/networks/some-network-id/some-container-id"),
				ghttp.VerifyJSONRepresenting(cniPayload),
				ghttp.VerifyHeaderKV("Content-type", "application/json"),
				ghttp.RespondWithJSONEncoded(http.StatusCreated, returnedResult),
			))
		})

		It("should POST to the /networks/:network_id/:container_id endpoint with a CNI payload", func() {
			unmarshaler.UnmarshalStub = json.Unmarshal

			receivedResult, err := c.ContainerUp("some-network-id", "some-container-id", cniPayload)
			Expect(err).NotTo(HaveOccurred())

			Expect(receivedResult).To(Equal(returnedResult))
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
			Expect(marshaler.MarshalCallCount()).To(Equal(1))
			Expect(marshaler.MarshalArgsForCall(0)).To(Equal(cniPayload))
		})

		It("uses the provided HTTP client", func() {
			_, err := c.ContainerUp("some-network-id", "some-container-id", cniPayload)
			Expect(err).NotTo(HaveOccurred())
			Expect(roundTripper.RoundTripCallCount()).To(Equal(1))
			Expect(roundTripper.RoundTripArgsForCall(0).URL.Path).To(Equal("/networks/some-network-id/some-container-id"))
		})

		Context("when an error occurs", func() {
			Context("when the payload fails to marshal", func() {
				It("returns an error", func() {
					marshaler.MarshalReturns(nil, errors.New("explosion with marshal"))

					_, err := c.ContainerUp("", "", cniPayload)
					Expect(err).To(MatchError("failed to marshal cni payload: explosion with marshal"))
				})
			})

			Context("when the request cannot be performed", func() {
				It("returns an error", func() {
					c = client.DaemonClient{
						BaseURL:   "%%%%",
						Marshaler: marshaler,
					}

					_, err := c.ContainerUp("", "", cniPayload)
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
					_, err := c.ContainerUp("some-network-id", "some-container-id", cniPayload)
					Expect(err).To(MatchError(`unexpected status code on ContainerUp: expected 201 but got 500`))
				})
			})

			Context("when the response body cannot be read", func() {
				BeforeEach(func() {
					badReader := &testsupport.BadReader{
						Error: errors.New("potato"),
					}
					badResponse := &http.Response{
						StatusCode: http.StatusCreated,
						Body:       badReader,
					}
					roundTripper.RoundTripStub = nil
					roundTripper.RoundTripReturns(badResponse, nil)
				})

				It("should return a wrapped error", func() {
					_, err := c.ContainerUp("some-network-id", "some-container-id", cniPayload)
					Expect(err).To(MatchError("reading response body: potato"))
				})

			})

			Context("when the http response code is a 409 Conflict", func() {
				It("should return an ipam.NoMoreAddressesError", func() {
					server.SetHandler(0, ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/networks/some-network-id/some-container-id"),
						ghttp.RespondWith(http.StatusConflict, `{ "error": "boom" }`),
					))

					_, err := c.ContainerUp("some-network-id", "some-container-id", cniPayload)
					Expect(err).To(Equal(ipam.NoMoreAddressesError))
				})
			})

			Context("when the http response code is unexpected", func() {
				It("should return an error", func() {
					server.SetHandler(0, ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/networks/some-network-id/some-container-id"),
						ghttp.RespondWith(http.StatusTeapot, `{{{`),
					))

					_, err := c.ContainerUp("some-network-id", "some-container-id", cniPayload)
					Expect(err).To(MatchError(`unexpected status code on ContainerUp: expected 201 but got 418`))
				})
			})

			Context("when the IPAM result fails to unmarshal", func() {
				It("should return an error", func() {
					unmarshaler.UnmarshalReturns(errors.New("explosion with unmarshal"))

					_, err := c.ContainerUp("some-network-id", "some-container-id", cniPayload)
					Expect(err).To(MatchError("failed to unmarshal IPAM result: explosion with unmarshal"))
				})
			})
		})
	})

	Describe("ContainerDown", func() {
		var cniPayload models.NetworksDeleteContainerPayload

		BeforeEach(func() {
			cniPayload = models.NetworksDeleteContainerPayload{
				ContainerNamespace: "/some/namespace/path",
				InterfaceName:      "some-interface-name",
				VNI:                42,
			}

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("DELETE", "/networks/some-network-id/some-container-id"),
				ghttp.VerifyJSONRepresenting(cniPayload),
				ghttp.VerifyHeaderKV("Content-type", "application/json"),
				ghttp.RespondWith(http.StatusNoContent, nil),
			))
		})

		It("should DELETE the /networks/:network_id/:container_id endpoint with a CNI payload", func() {
			Expect(c.ContainerDown("some-network-id", "some-container-id", cniPayload)).To(Succeed())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
			Expect(marshaler.MarshalCallCount()).To(Equal(1))
			Expect(marshaler.MarshalArgsForCall(0)).To(Equal(cniPayload))
		})

		It("uses the provided HTTP client", func() {
			Expect(c.ContainerDown("some-network-id", "some-container-id", cniPayload)).To(Succeed())

			Expect(roundTripper.RoundTripCallCount()).To(Equal(1))
			Expect(roundTripper.RoundTripArgsForCall(0).URL.Path).To(Equal("/networks/some-network-id/some-container-id"))
		})

		Context("when an error occurs", func() {
			Context("when the payload fails to marshal", func() {
				It("returns an error", func() {
					marshaler.MarshalReturns(nil, errors.New("explosion with marshal"))

					err := c.ContainerDown("", "", cniPayload)
					Expect(err).To(MatchError("failed to marshal cni payload: explosion with marshal"))
				})
			})

			Context("when the request cannot be constructed", func() {
				It("returns an error", func() {
					c = client.DaemonClient{
						BaseURL:   "%%%%",
						Marshaler: marshaler,
					}

					err := c.ContainerDown("", "", cniPayload)
					Expect(err).To(MatchError(ContainSubstring("failed to construct request: parse")))
				})
			})

			Context("when the request fails to connect", func() {
				BeforeEach(func() {
					c.BaseURL = "http://0.0.0.0:12345"
				})
				It("should return an error", func() {
					err := c.ContainerDown("", "", cniPayload)
					Expect(err).To(MatchError(ContainSubstring("request failed: dial")))
				})
			})

			Context("when the http response code is unexpected", func() {
				BeforeEach(func() {
					server.Reset()
					server.AppendHandlers(ghttp.CombineHandlers(
						ghttp.VerifyRequest("DELETE", "/networks/some-network-id/some-container-id"),
						ghttp.RespondWith(http.StatusInternalServerError, nil),
					))
				})

				It("should return an error", func() {
					err := c.ContainerDown("some-network-id", "some-container-id", cniPayload)
					Expect(err).To(MatchError(`unexpected status code on ContainerDown: expected 204 but got 500`))
				})
			})
		})
	})

})
