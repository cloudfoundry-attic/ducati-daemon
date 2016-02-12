package ns_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestNs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ns Suite")
}
