package namespace_test

import (
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestNamespace(t *testing.T) {
	runtime.LockOSThread()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Namespace Suite")
}
