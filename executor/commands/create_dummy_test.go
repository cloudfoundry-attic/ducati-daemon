package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateDummy", func() {
	var (
		context     *fakes.Context
		linkFactory *fakes.LinkFactory
		createDummy commands.CreateDummy
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		linkFactory = &fakes.LinkFactory{}
		context.LinkFactoryReturns(linkFactory)

		createDummy = commands.CreateDummy{
			Name: "my-dummy",
		}
	})

	It("uses the factory to create the adapter", func() {
		err := createDummy.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(linkFactory.CreateDummyCallCount()).To(Equal(1))
		name := linkFactory.CreateDummyArgsForCall(0)
		Expect(name).To(Equal("my-dummy"))
	})

	Context("when creating the dummy link fails", func() {
		BeforeEach(func() {
			linkFactory.CreateDummyReturns(errors.New("no dummy for you, dummy"))
		})

		It("wraps and propagates the error", func() {
			err := createDummy.Execute(context)
			Expect(err).To(MatchError("create dummy: no dummy for you, dummy"))
		})
	})

	Describe("String", func() {
		It("is self describing", func() {
			Expect(createDummy.String()).To(Equal("ip link add my-dummy type dummy"))
		})
	})
})
