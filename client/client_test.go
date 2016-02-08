package client_test

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/client"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
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
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
		marshaler = &fakes.Marshaler{}
		unmarshaler = &fakes.Unmarshaler{}
		c = client.DaemonClient{
			BaseURL:     server.URL(),
			Marshaler:   marshaler,
			Unmarshaler: unmarshaler,
		}

		marshaler.MarshalReturns([]byte(`{"id":"some-container"}`), nil)

		container = models.Container{
			ID: "some-container",
		}
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("SaveContainer", func() {
		It("should call the backend to save the container", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/containers"),
				ghttp.VerifyJSON(`{"id":"some-container"}`),
				ghttp.RespondWith(http.StatusCreated, nil),
			))

			Expect(c.SaveContainer(container)).To(Succeed())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
			Expect(marshaler.MarshalCallCount()).To(Equal(1))
			Expect(marshaler.MarshalArgsForCall(0)).To(Equal(container))
		})

		Context("when an error occurs", func() {
			Context("when the container fails to marshal", func() {
				It("should return an error", func() {
					marshaler.MarshalReturns(nil, errors.New("explosion with marshal"))

					err := c.SaveContainer(container)
					Expect(err).To(MatchError("failed to marshal container: explosion with marshal"))
				})
			})

			Context("when the request cannot be constructed", func() {
				It("should return an error", func() {
					c = client.DaemonClient{
						BaseURL:   "%%%%",
						Marshaler: marshaler,
					}

					err := c.SaveContainer(container)
					Expect(err).To(MatchError(ContainSubstring("failed to construct request: parse")))
				})
			})

			Context("when the http request fails", func() {
				It("should return an error", func() {
					server.AppendHandlers(ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/containers"),
						ghttp.RespondWith(http.StatusInternalServerError, nil),
					))

					err := c.SaveContainer(container)
					Expect(err).To(MatchError(`unexpected status code on SaveContainer: expected 201 but got 500`))
				})
			})
		})
	})

	Describe("RemoveContainer", func() {
		It("should call the backend to remove the container", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("DELETE", "/containers/some-container"),
				ghttp.RespondWith(http.StatusNoContent, nil),
			))

			Expect(c.RemoveContainer("some-container")).To(Succeed())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
		})

		Context("when an error occurs", func() {
			Context("when the container does not exist", func() {
				It("it should return a RecordNotFound error", func() {
					server.AppendHandlers(ghttp.CombineHandlers(
						ghttp.VerifyRequest("DELETE", "/containers/non-existent-container-id"),
						ghttp.RespondWith(http.StatusNotFound, nil),
					))

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

			Context("when the http request fails", func() {
				It("should return an error", func() {
					server.AppendHandlers(ghttp.CombineHandlers(
						ghttp.VerifyRequest("DELETE", "/containers/whatever"),
						ghttp.RespondWith(http.StatusInternalServerError, nil),
					))

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
				ghttp.VerifyRequest("POST", "/ipam/some-network-name"),
				ghttp.RespondWithJSONEncoded(http.StatusCreated, returnedResult),
			))
		})

		It("should call the backend to allocate an IP", func() {
			unmarshaler.UnmarshalStub = json.Unmarshal

			receivedResult, err := c.AllocateIP("some-network-name")
			Expect(err).NotTo(HaveOccurred())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))

			Expect(receivedResult).To(Equal(returnedResult))
		})

		Context("when an error occurs", func() {
			Context("when the request cannot be constructed", func() {
				It("should return an error", func() {
					c.BaseURL = "%%%%"

					_, err := c.AllocateIP("some-network-name")
					Expect(err).To(MatchError(ContainSubstring("failed to construct request: parse")))
				})
			})

			Context("when the http response code is unexpected", func() {
				It("should return an error", func() {
					server.SetHandler(0, ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/ipam/some-network-name"),
						ghttp.RespondWith(http.StatusTeapot, `{{{`),
					))

					_, err := c.AllocateIP("some-network-name")
					Expect(err).To(MatchError(`unexpected status code on AllocateIP: expected 201 but got 418`))
				})
			})

			Context("when the container fails to marshal", func() {
				It("should return an error", func() {
					unmarshaler.UnmarshalReturns(errors.New("explosion with marshal"))

					_, err := c.AllocateIP("some-network-name")
					Expect(err).To(MatchError("failed to unmarshal IPAM result: explosion with marshal"))
				})
			})
		})
	})
})
