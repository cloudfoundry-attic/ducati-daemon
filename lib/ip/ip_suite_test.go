package ip_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestIp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ip Suite")
}
