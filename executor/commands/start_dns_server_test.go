package commands_test

import (
	"errors"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Start DNS Server", func() {
	var (
		ns               *fakes.Namespace
		context          *fakes.Context
		listenerFactory  *fakes.ListenerFactory
		linkFactory      *fakes.LinkFactory
		addressManager   *fakes.AddressManager
		dnsServerFactory *fakes.DNSServerFactory
		returnedListener *net.UDPConn

		sandboxRepo *fakes.SandboxRepository
		sbox        *fakes.Sandbox
		dnsServer   *fakes.Runner

		startDNS commands.StartDNSServer
	)

	BeforeEach(func() {
		listenerFactory = &fakes.ListenerFactory{}
		linkFactory = &fakes.LinkFactory{}
		addressManager = &fakes.AddressManager{}
		dnsServerFactory = &fakes.DNSServerFactory{}

		ns = &fakes.Namespace{}
		sandboxRepo = &fakes.SandboxRepository{}
		sbox = &fakes.Sandbox{}
		sbox.NamespaceReturns(ns)
		dnsServer = &fakes.Runner{}
		dnsServerFactory.NewReturns(dnsServer)

		context = &fakes.Context{}
		context.ListenerFactoryReturns(listenerFactory)
		context.LinkFactoryReturns(linkFactory)
		context.AddressManagerReturns(addressManager)
		context.DNSServerFactoryReturns(dnsServerFactory)
		context.SandboxRepositoryReturns(sandboxRepo)
		sandboxRepo.GetReturns(sbox, nil)

		returnedListener = &net.UDPConn{}
		listenerFactory.ListenUDPReturns(returnedListener, nil)

		ns.ExecuteStub = func(callback func(*os.File) error) error {
			return callback(nil)
		}

		startDNS = commands.StartDNSServer{
			ListenAddress: "10.10.10.10:53",
			SandboxName:   "some-sandbox-name",
		}
	})

	It("gets the namespace from the sandbox", func() {
		err := startDNS.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(sandboxRepo.GetCallCount()).To(Equal(1))
		Expect(sandboxRepo.GetArgsForCall(0)).To(Equal("some-sandbox-name"))

		Expect(sbox.NamespaceCallCount()).To(Equal(1))
	})

	It("locks and unlocks the sandbox", func() {
		err := startDNS.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(sbox.LockCallCount()).To(Equal(1))
		Expect(sbox.UnlockCallCount()).To(Equal(1))
	})

	It("adds a dummy link to the sandbox namespace", func() {
		ns.ExecuteStub = func(callback func(*os.File) error) error {
			Expect(linkFactory.CreateDummyCallCount()).To(Equal(0))
			err := callback(nil)
			Expect(linkFactory.CreateDummyCallCount()).To(Equal(1))
			return err
		}

		err := startDNS.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(ns.ExecuteCallCount()).To(Equal(1))

		Expect(linkFactory.CreateDummyCallCount()).To(Equal(1))
		linkName := linkFactory.CreateDummyArgsForCall(0)
		Expect(linkName).To(Equal("dns0"))
	})

	It("sets the address on the dummy device in the sandbox namespace", func() {
		ns.ExecuteStub = func(callback func(*os.File) error) error {
			Expect(addressManager.AddAddressCallCount()).To(Equal(0))
			err := callback(nil)
			Expect(addressManager.AddAddressCallCount()).To(Equal(1))
			return err
		}

		err := startDNS.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(addressManager.AddAddressCallCount()).To(Equal(1))
		ifName, address := addressManager.AddAddressArgsForCall(0)
		Expect(ifName).To(Equal("dns0"))
		Expect(address.IP.String()).To(Equal("10.10.10.10"))
	})

	It("ups the dummy device in the sandbox namespace", func() {
		ns.ExecuteStub = func(callback func(*os.File) error) error {
			Expect(linkFactory.SetUpCallCount()).To(Equal(0))
			err := callback(nil)
			Expect(linkFactory.SetUpCallCount()).To(Equal(1))
			return err
		}

		err := startDNS.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(linkFactory.SetUpCallCount()).To(Equal(1))
		Expect(linkFactory.SetUpArgsForCall(0)).To(Equal("dns0"))
	})

	It("creates a listener in the sandbox namespace", func() {
		ns.ExecuteStub = func(callback func(*os.File) error) error {
			Expect(listenerFactory.ListenUDPCallCount()).To(Equal(0))
			err := callback(nil)
			Expect(listenerFactory.ListenUDPCallCount()).To(Equal(1))

			return err
		}

		expectedAddress, err := net.ResolveUDPAddr("udp", "10.10.10.10:53")
		Expect(err).NotTo(HaveOccurred())

		err = startDNS.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(ns.ExecuteCallCount()).To(Equal(1))

		Expect(listenerFactory.ListenUDPCallCount()).To(Equal(1))
		network, addr := listenerFactory.ListenUDPArgsForCall(0)
		Expect(network).To(Equal("udp"))
		Expect(addr).To(Equal(expectedAddress))
	})

	It("uses the DNS Server Factory to create a DNS server with the listener", func() {
		err := startDNS.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(dnsServerFactory.NewCallCount()).To(Equal(1))
		packetConn := dnsServerFactory.NewArgsForCall(0)
		Expect(packetConn).To(BeIdenticalTo(returnedListener))
	})

	It("passes the dns server to the sandbox to be launched", func() {
		err := startDNS.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(sbox.LaunchDNSCallCount()).To(Equal(1))
		Expect(sbox.LaunchDNSArgsForCall(0)).To(Equal(dnsServer))
	})

	Context("when parsing the listen address fails", func() {
		BeforeEach(func() {
			startDNS.ListenAddress = "some-bogus-address"
		})

		It("returns a meaningful error", func() {
			err := startDNS.Execute(context)
			Expect(err).To(MatchError(MatchRegexp("resolve udp address:.*some-bogus-address")))
		})
	})

	Context("when creating the dummy link fails", func() {
		BeforeEach(func() {
			linkFactory.CreateDummyReturns(errors.New("pineapple"))
		})

		It("returns a meaningful error", func() {
			err := startDNS.Execute(context)
			Expect(err).To(MatchError("namespace execute: create dummy: pineapple"))
		})
	})

	Context("when adding the address fails", func() {
		BeforeEach(func() {
			addressManager.AddAddressReturns(errors.New("pineapple"))
		})

		It("returns a meaningful error", func() {
			err := startDNS.Execute(context)
			Expect(err).To(MatchError("namespace execute: add address: pineapple"))
		})
	})

	Context("when setting the link up fails", func() {
		BeforeEach(func() {
			linkFactory.SetUpReturns(errors.New("pineapple"))
		})

		It("returns a meaningful error", func() {
			err := startDNS.Execute(context)
			Expect(err).To(MatchError("namespace execute: set up: pineapple"))
		})
	})

	Context("when creating the listener fails", func() {
		BeforeEach(func() {
			listenerFactory.ListenUDPReturns(nil, errors.New("cantelope"))
		})

		It("returns a meaningful error", func() {
			err := startDNS.Execute(context)
			Expect(err).To(MatchError("namespace execute: listen udp: cantelope"))
		})
	})

	Context("when getting the sandbox from the sandbox repository fails", func() {
		BeforeEach(func() {
			sandboxRepo.GetReturns(nil, errors.New("lime"))
		})

		It("returns a meaningful error", func() {
			err := startDNS.Execute(context)
			Expect(err).To(MatchError("get sandbox: lime"))
		})
	})

	Context("when launching the DNS server on the sandbox returns an error", func() {
		BeforeEach(func() {
			sbox.LaunchDNSReturns(errors.New("bergamot"))
		})

		It("returns a meaningful error", func() {
			err := startDNS.Execute(context)
			Expect(err).To(MatchError("sandbox launch dns: bergamot"))
		})
	})

	Describe("String", func() {
		It("returns a human readable representation", func() {
			Expect(startDNS.String()).To(Equal("start dns server in sandbox some-sandbox-name"))
		})
	})
})
