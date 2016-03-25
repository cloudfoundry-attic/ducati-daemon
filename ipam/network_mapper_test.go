package ipam_test

import (
	"fmt"
	"math/rand"

	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkMapper", func() {
	var networkMapper *ipam.FixedNetworkMapper

	Describe("GetVNI using a real digester", func() {
		BeforeEach(func() {
			networkMapper = &ipam.FixedNetworkMapper{}
			rand.Seed(config.GinkgoConfig.RandomSeed)
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
})
