package client_test

import (
	"encoding/json"
	"net/http"

	"github.com/appc/cni/pkg/skel"
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

		roundTripper *fakes.RoundTripper
		httpClient   *http.Client
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
		marshaler = &fakes.Marshaler{}
		unmarshaler = &fakes.Unmarshaler{}

		roundTripper = &fakes.RoundTripper{}
		roundTripper.RoundTripStub = http.DefaultTransport.RoundTrip

		httpClient = &http.Client{
			Transport: roundTripper,
		}

		c = client.DaemonClient{
			JSONClient: client.JSONClient{
				BaseURL:     server.URL(),
				Marshaler:   marshaler,
				Unmarshaler: unmarshaler,
				HttpClient:  httpClient,
			},
		}

		marshaler.MarshalStub = json.Marshal
		unmarshaler.UnmarshalStub = json.Unmarshal
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("CNIAdd", func() {
		var cniPayload models.CNIAddPayload

		BeforeEach(func() {
			cniPayload = models.CNIAddPayload{
				Args:               "FOO=BAR;ABC=123",
				ContainerNamespace: "/some/namespace/path",
				InterfaceName:      "interface-name",
				Network:            models.NetworkPayload{ID: "legacy", App: "some-app-guid"},
				ContainerID:        "some-container-id",
			}

			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/cni/add"),
				ghttp.VerifyJSONRepresenting(cniPayload),
				ghttp.VerifyHeaderKV("Content-type", "application/json"),
				ghttp.RespondWithJSONEncoded(http.StatusCreated, types.Result{}),
			))
		})

		Context("when network spec is not provided", func() {
			It("sets the Network.ID to 'legacy'", func() {
				_, err := c.CNIAdd(&skel.CmdArgs{
					ContainerID: "some-container-id",
					Netns:       "/some/namespace/path",
					IfName:      "interface-name",
					Args:        "FOO=BAR;ABC=123",
					StdinData:   []byte(`{"network": {"network_id": "", "app": "some-app-guid"}}`),
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(marshaler.MarshalCallCount()).To(Equal(1))
				Expect(marshaler.MarshalArgsForCall(0)).To(Equal(cniPayload))
			})
		})
	})
})
