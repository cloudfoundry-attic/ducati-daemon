package acceptance_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var ducatidPath string

func TestDucatid(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ducati Daemon Acceptance Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	ducatidPath, err := gexec.Build("github.com/cloudfoundry-incubator/ducati-daemon/cmd/ducatid", "-race")
	Expect(err).NotTo(HaveOccurred())

	return []byte(ducatidPath)
}, func(pathsByte []byte) {
	ducatidPath = string(pathsByte)
})
