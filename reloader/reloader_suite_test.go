package reloader_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestReloader(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reloader Suite")
}
