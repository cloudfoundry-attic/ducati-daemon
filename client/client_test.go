package client_test

import (
	"errors"
	"net"
	"net/http"

	"github.com/appc/cni/pkg/skel"
	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/client"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/onsi/gomega/ghttp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		daemonClient *client.DaemonClient
		server       *ghttp.Server

		roundTripper *fakes.RoundTripper
		httpClient   *http.Client
	)

	BeforeEach(func() {
		server = ghttp.NewServer()

		roundTripper = &fakes.RoundTripper{}
		roundTripper.RoundTripStub = http.DefaultTransport.RoundTrip

		httpClient = &http.Client{
			Transport: roundTripper,
		}

		daemonClient = client.New(server.URL(), httpClient)
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("CNIAdd", func() {
		var expectedCNIPayload models.CNIAddPayload
		var expectedResult types.Result

		Context("when network is set", func() {
			BeforeEach(func() {
				expectedCNIPayload = models.CNIAddPayload{
					Args:               "FOO=BAR;ABC=123",
					ContainerNamespace: "/some/namespace/path",
					InterfaceName:      "interface-name",
					ContainerID:        "some-container-id",
					Network: models.NetworkPayload{
						Properties: models.Properties{
							AppID:   "some-app-id",
							SpaceID: "some-space-id",
						},
					},
				}

				expectedResult = types.Result{
					IP4: &types.IPConfig{
						IP: net.IPNet{
							IP:   net.ParseIP("5.6.7.8"),
							Mask: net.CIDRMask(24, 32),
						},
						Gateway: net.ParseIP("1.2.3.4"),
						Routes:  nil,
					},
					DNS: types.DNS{Domain: "potato"},
				}

				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/cni/add"),
					ghttp.VerifyJSONRepresenting(expectedCNIPayload),
					ghttp.VerifyHeaderKV("Content-type", "application/json"),
					ghttp.RespondWithJSONEncoded(http.StatusCreated, expectedResult),
				))
			})

			It("sends the correct payload and returns the ipam result", func() {
				result, err := daemonClient.CNIAdd(&skel.CmdArgs{
					ContainerID: "some-container-id",
					Netns:       "/some/namespace/path",
					IfName:      "interface-name",
					Args:        "FOO=BAR;ABC=123",
					StdinData: []byte(`{
						"network": {
							"network_id": "",
							"properties": {
								"app_id": "some-app-id",
								"space_id": "some-space-id"
							}
						}
					}`),
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(server.ReceivedRequests()).To(HaveLen(1))
				Expect(result).To(Equal(expectedResult))
			})
		})

		Context("when network is omitted", func() {
			BeforeEach(func() {
				expectedCNIPayload = models.CNIAddPayload{
					Args:               "FOO=BAR;ABC=123",
					ContainerNamespace: "/some/namespace/path",
					InterfaceName:      "interface-name",
					ContainerID:        "some-container-id",
				}
			})

			It("returns an invalid network payload message", func() {
				_, err := daemonClient.CNIAdd(&skel.CmdArgs{
					ContainerID: "some-container-id",
					Netns:       "/some/namespace/path",
					IfName:      "interface-name",
					Args:        "FOO=BAR;ABC=123",
					StdinData:   []byte{},
				})
				Expect(err).To(MatchError("invalid network spec: unexpected end of JSON input"))
				Expect(server.ReceivedRequests()).To(HaveLen(0))
			})
		})
	})

	Describe("CNIDel", func() {
		var expectedCNIPayload models.CNIDelPayload

		BeforeEach(func() {
			expectedCNIPayload = models.CNIDelPayload{
				ContainerID:        "some-container-id",
				ContainerNamespace: "/some/namespace/path",
				InterfaceName:      "interface-name",
			}

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/cni/del"),
				ghttp.VerifyJSONRepresenting(expectedCNIPayload),
				ghttp.VerifyHeaderKV("Content-type", "application/json"),
				ghttp.RespondWithJSONEncoded(http.StatusNoContent, nil),
			))
		})

		It("sends the correct payload and succeeds", func() {
			err := daemonClient.CNIDel(&skel.CmdArgs{
				ContainerID: "some-container-id",
				Netns:       "/some/namespace/path",
				IfName:      "interface-name",
			})

			Expect(server.ReceivedRequests()).To(HaveLen(1))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ContainerUp", func() {
		var (
			requestPayload     models.CNIAddPayload
			expectedResult     types.Result
			responseStatusCode int
		)

		BeforeEach(func() {
			requestPayload = models.CNIAddPayload{
				Args:               "FOO=BAR;ABC=123",
				ContainerNamespace: "/some/namespace/path",
				InterfaceName:      "interface-name",
				ContainerID:        "some-container-id",
				Network: models.NetworkPayload{
					Properties: models.Properties{
						AppID:   "some-app-id",
						SpaceID: "some-space-id",
					},
				},
			}

			responseStatusCode = http.StatusCreated

			expectedResult = types.Result{
				IP4: &types.IPConfig{
					IP: net.IPNet{
						IP:   net.ParseIP("5.6.7.8"),
						Mask: net.CIDRMask(24, 32),
					},
					Gateway: net.ParseIP("1.2.3.4"),
					Routes:  nil,
				},
				DNS: types.DNS{Domain: "potato"},
			}

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/cni/add"),
				ghttp.VerifyJSONRepresenting(requestPayload),
				ghttp.VerifyHeaderKV("Content-type", "application/json"),
				ghttp.VerifyHeaderKV("Accept", "application/json"),
				ghttp.RespondWithJSONEncodedPtr(&responseStatusCode, &expectedResult),
			))
		})

		It("returns the ipam result", func() {
			result, err := daemonClient.ContainerUp(requestPayload)
			Expect(err).NotTo(HaveOccurred())

			Expect(server.ReceivedRequests()).To(HaveLen(1))
			Expect(result).To(Equal(expectedResult))
		})

		Context("when the request fails", func() {
			BeforeEach(func() {
				roundTripper.RoundTripReturns(nil, errors.New("mango"))
			})

			It("returns a meaningful error", func() {
				_, err := daemonClient.ContainerUp(requestPayload)
				Expect(err).To(MatchError(MatchRegexp("container up:.*mango")))
			})
		})

		Context("when the server responds with 400 http.StatusBadRequest", func() {
			BeforeEach(func() {
				responseStatusCode = http.StatusBadRequest
			})

			It("returns an ipam.AlreadyOnNetworkError", func() {
				_, err := daemonClient.ContainerUp(requestPayload)
				Expect(err).To(Equal(ipam.AlreadyOnNetworkError))
			})
		})

		Context("when the server responds with 409 http.StatusConflict", func() {
			BeforeEach(func() {
				responseStatusCode = http.StatusConflict
			})

			It("returns an ipam.NoMoreAddressesError", func() {
				_, err := daemonClient.ContainerUp(requestPayload)
				Expect(err).To(Equal(ipam.NoMoreAddressesError))
			})
		})

		Context("when the server responds with an unexpected status code", func() {
			BeforeEach(func() {
				responseStatusCode = http.StatusTeapot
			})

			It("returns a meaningful error", func() {
				_, err := daemonClient.ContainerUp(requestPayload)
				Expect(err).To(MatchError("unexpected status code on ContainerUp: expected 201 but got 418"))
			})
		})
	})

	Describe("ContainerDown", func() {
		var (
			requestPayload     models.CNIDelPayload
			responseStatusCode int
		)

		BeforeEach(func() {
			requestPayload = models.CNIDelPayload{
				ContainerID:        "some-container-id",
				ContainerNamespace: "/some/namespace/path",
				InterfaceName:      "interface-name",
			}

			responseStatusCode = http.StatusNoContent

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/cni/del"),
				ghttp.VerifyJSONRepresenting(requestPayload),
				ghttp.VerifyHeaderKV("Content-type", "application/json"),
				ghttp.RespondWithJSONEncodedPtr(&responseStatusCode, nil),
			))
		})

		It("sends the correct payload and succeeds", func() {
			err := daemonClient.ContainerDown(requestPayload)
			Expect(err).NotTo(HaveOccurred())

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		Context("when the request fails", func() {
			BeforeEach(func() {
				roundTripper.RoundTripReturns(nil, errors.New("mango"))
			})

			It("returns a meaningful error", func() {
				err := daemonClient.ContainerDown(requestPayload)
				Expect(err).To(MatchError(MatchRegexp("container down:.*mango")))
			})
		})

		Context("when the server responds with an unexpected status code", func() {
			BeforeEach(func() {
				responseStatusCode = http.StatusTeapot
			})

			It("returns a meaningful error", func() {
				err := daemonClient.ContainerDown(requestPayload)
				Expect(err).To(MatchError("unexpected status code on ContainerDown: expected 204 but got 418"))
			})
		})
	})

	Describe("GetContainer", func() {
		var responseStatusCode int
		var expectedContainer models.Container

		BeforeEach(func() {
			responseStatusCode = http.StatusOK
			expectedContainer = models.Container{
				ID:          "some-container-id",
				IP:          "1.2.3.4",
				MAC:         "00:11:22:33:44:55:66",
				HostIP:      "5.6.7.8",
				NetworkID:   "some-network-id",
				SandboxName: "some-sandbox-name",
				App:         "some-app-id",
			}

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/containers/container-id"),
				ghttp.VerifyHeaderKV("Accept", "application/json"),
				ghttp.RespondWithJSONEncodedPtr(&responseStatusCode, &expectedContainer),
			))
		})

		It("sends the correct payload and succeeds", func() {
			response, err := daemonClient.GetContainer("container-id")
			Expect(err).NotTo(HaveOccurred())

			Expect(server.ReceivedRequests()).To(HaveLen(1))
			Expect(response).To(Equal(expectedContainer))
		})

		Context("when the server responds with 404 http.StatusNotFound", func() {
			BeforeEach(func() {
				responseStatusCode = http.StatusNotFound
			})

			It("returns a RecordNotFoundError", func() {
				_, err := daemonClient.GetContainer("container-id")
				Expect(err).To(Equal(client.RecordNotFoundError))
			})
		})

		Context("when the server responds with an unexpected status code", func() {
			BeforeEach(func() {
				responseStatusCode = http.StatusTeapot
			})

			It("returns a meaningful error", func() {
				_, err := daemonClient.GetContainer("container-id")
				Expect(err).To(MatchError("unexpected status code on GetContainer: expected 200 but got 418"))
			})
		})
	})

	Describe("ListNetworkContainers", func() {
		var responseStatusCode int
		var expectedContainers []models.Container

		BeforeEach(func() {
			responseStatusCode = http.StatusOK
			expectedContainers = []models.Container{{
				ID:          "some-container-id",
				IP:          "1.2.3.4",
				MAC:         "00:11:22:33:44:55:66",
				HostIP:      "5.6.7.8",
				NetworkID:   "some-network-id",
				SandboxName: "some-sandbox-name",
				App:         "some-app-id",
			}}

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/networks/network-id"),
				ghttp.VerifyHeaderKV("Accept", "application/json"),
				ghttp.RespondWithJSONEncodedPtr(&responseStatusCode, expectedContainers),
			))
		})

		It("sends the correct payload and succeeds", func() {
			response, err := daemonClient.ListNetworkContainers("network-id")
			Expect(err).NotTo(HaveOccurred())

			Expect(server.ReceivedRequests()).To(HaveLen(1))
			Expect(response).To(Equal(expectedContainers))
		})

		Context("when the server responds with an unexpected status code", func() {
			BeforeEach(func() {
				responseStatusCode = http.StatusTeapot
			})

			It("returns a meaningful error", func() {
				_, err := daemonClient.ListNetworkContainers("network-id")
				Expect(err).To(MatchError("unexpected status code on ListNetworkContainers: expected 200 but got 418"))
			})
		})
	})

	Describe("ListContainers", func() {
		var responseStatusCode int
		var expectedContainers []models.Container

		BeforeEach(func() {
			responseStatusCode = http.StatusOK
			expectedContainers = []models.Container{{
				ID:          "some-container-id",
				IP:          "1.2.3.4",
				MAC:         "00:11:22:33:44:55:66",
				HostIP:      "5.6.7.8",
				NetworkID:   "some-network-id",
				SandboxName: "some-sandbox-name",
				App:         "some-app-id",
			}, {
				ID:          "some-other-container-id",
				IP:          "2.3.4.5",
				MAC:         "11:22:33:44:55:66:77",
				HostIP:      "6.7.8.9",
				NetworkID:   "some-other-network-id",
				SandboxName: "some-other-sandbox-name",
				App:         "some-other-app-id",
			}}

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/containers"),
				ghttp.VerifyHeaderKV("Accept", "application/json"),
				ghttp.RespondWithJSONEncodedPtr(&responseStatusCode, expectedContainers),
			))
		})

		It("sends the correct payload and succeeds", func() {
			response, err := daemonClient.ListContainers()
			Expect(err).NotTo(HaveOccurred())

			Expect(server.ReceivedRequests()).To(HaveLen(1))
			Expect(response).To(Equal(expectedContainers))
		})

		Context("when the server responds with an unexpected status code", func() {
			BeforeEach(func() {
				responseStatusCode = http.StatusTeapot
			})

			It("returns a meaningful error", func() {
				_, err := daemonClient.ListContainers()
				Expect(err).To(MatchError("unexpected status code on ListContainers: expected 200 but got 418"))
			})
		})
	})
})
