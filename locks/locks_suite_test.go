package locks_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLocks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Locks Suite")
}
