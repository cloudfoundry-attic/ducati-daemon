package ipam_test

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkMapper", func() {
	var networkMapper ipam.NetworkMapper

	BeforeEach(func() {
		networkMapper = &ipam.FixedNetworkMapper{}
	})

	Describe("GetVNI", func() {
		It("consistently digests the same input string into a <15 digit int", func() {
			vni, err := networkMapper.GetVNI("network-id-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(vni).To(Equal(13936832))

			vni, err = networkMapper.GetVNI("network-id-2")
			Expect(err).NotTo(HaveOccurred())
			Expect(vni).To(Equal(12823693))

			vni, err = networkMapper.GetVNI("network-id-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(vni).To(Equal(13936832))
		})
	})
})
