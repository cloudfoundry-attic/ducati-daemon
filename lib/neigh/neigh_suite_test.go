package neigh_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestNeigh(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Neigh Suite")
}
