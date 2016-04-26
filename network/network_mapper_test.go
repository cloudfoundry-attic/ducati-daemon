package network_test

import (
	"fmt"
	"math/rand"

	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/network"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkMapper", func() {
	var networkMapper *network.FixedNetworkMapper

	Describe("GetVNI using a real digester", func() {
		BeforeEach(func() {
			networkMapper = &network.FixedNetworkMapper{}
		})

		It("generates VNIs less than 2^24", func() {
			for i := 0; i < 1000; i++ {
				j := rand.Int63()
				networkID := fmt.Sprintf("some-test-network-%x", j)
				vni, err := networkMapper.GetVNI(networkID)
				Expect(err).NotTo(HaveOccurred())
				Expect(vni).To(BeNumerically("<", 1<<24))
			}
		})

		It("usually does not collide (this test will fail on 3% of random seeds)", func() {
			vniCounts := make(map[int]int)
			for i := 0; i < 1000; i++ {
				j := rand.Int()
				networkID := fmt.Sprintf("network-%x", j)
				vni, err := networkMapper.GetVNI(networkID)
				Expect(err).NotTo(HaveOccurred())
				vniCounts[vni] += 1
			}

			for _, v := range vniCounts {
				Expect(v).To(BeNumerically("<=", 1))
			}
		})
	})

	Describe("GetNetworkID", func() {
		var networkPayload models.NetworkPayload

		BeforeEach(func() {
			networkMapper = &network.FixedNetworkMapper{
				DefaultNetworkID: fmt.Sprintf("some-network-%x", rand.Int()),
			}
			networkPayload = models.NetworkPayload{
				Properties: models.Properties{
					AppID:   "some-app-guid",
					SpaceID: fmt.Sprintf("some-space-%x", rand.Int()),
				},
			}
		})

		It("sets the network ID equal to the space ID", func() {
			networkID, err := networkMapper.GetNetworkID(networkPayload)
			Expect(err).NotTo(HaveOccurred())
			Expect(networkID).To(Equal(networkPayload.Properties.SpaceID))
		})

		Context("when the space ID is not set", func() {
			BeforeEach(func() {
				networkPayload.Properties.SpaceID = ""
			})
			It("sets the network ID to be the constant", func() {
				networkID, err := networkMapper.GetNetworkID(networkPayload)
				Expect(err).NotTo(HaveOccurred())
				Expect(networkID).To(Equal(networkMapper.DefaultNetworkID))
			})
		})
	})
})
