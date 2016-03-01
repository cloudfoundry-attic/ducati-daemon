package threading_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestThreading(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Threading Suite")
}
