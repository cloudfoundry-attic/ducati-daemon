package sandbox_test

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sandbox", func() {
	var (
		sb          sandbox.Sandbox
		sbNamespace *fakes.Namespace
	)

	BeforeEach(func() {
		sbNamespace = &fakes.Namespace{}
		sb = sandbox.New(sbNamespace)
	})

	Describe("Namespace", func() {
		It("returns the sandbox namespace", func() {
			ns := sb.Namespace()
			Expect(ns).To(Equal(sbNamespace))
		})
	})
})
