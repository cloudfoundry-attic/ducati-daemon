package container_test

import (
	"errors"
	"net"
	"regexp"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Creator", func() {
	var (
		creator         container.Creator
		ex              *fakes.Executor
		containerMAC    net.HardwareAddr
		containerNS     *fakes.Namespace
		ipamResult      *types.Result
		config          container.CreatorConfig
		sandbox         *fakes.Sandbox
		sandboxRepo     *fakes.SandboxRepository
		sandboxNS       *fakes.Namespace
		missWatcher     watcher.MissWatcher
		commandBuilder  *fakes.CommandBuilder
		namespaceOpener *fakes.Opener
	)

	BeforeEach(func() {
		ex = &fakes.Executor{}
		sandbox = &fakes.Sandbox{}
		sandboxRepo = &fakes.SandboxRepository{}
		missWatcher = &fakes.MissWatcher{}
		commandBuilder = &fakes.CommandBuilder{}
		containerNS = &fakes.Namespace{NameStub: func() string { return "container ns sentinel" }}
		namespaceOpener = &fakes.Opener{}
		namespaceOpener.OpenPathReturns(containerNS, nil)
		creator = container.Creator{
			Executor:        ex,
			SandboxRepo:     sandboxRepo,
			Watcher:         missWatcher,
			CommandBuilder:  commandBuilder,
			NamespaceOpener: namespaceOpener,
			DNSAddress:      "some-dns-address",
			HostIP:          net.ParseIP("10.11.12.13"),
		}

		macAddress := "01:02:03:04:05:06"
		var err error
		containerMAC, err = net.ParseMAC(macAddress)
		Expect(err).NotTo(HaveOccurred())

		ipamResult = &types.Result{
			IP4: &types.IPConfig{
				IP: net.IPNet{
					IP:   net.ParseIP("192.168.100.2"),
					Mask: net.CIDRMask(24, 32),
				},
				Gateway: net.ParseIP("192.168.100.1"),
				Routes: []types.Route{{
					Dst: net.IPNet{
						IP:   net.ParseIP("192.168.1.5"),
						Mask: net.CIDRMask(24, 32),
					},
					GW: net.ParseIP("192.168.1.1"),
				}, {
					Dst: net.IPNet{
						IP:   net.ParseIP("192.168.2.5"),
						Mask: net.CIDRMask(24, 32),
					},
					GW: net.ParseIP("192.168.1.99"),
				}},
			},
		}

		sandboxNS = &fakes.Namespace{NameStub: func() string { return "sandbox ns sentinel" }}
		sandbox.NamespaceReturns(sandboxNS)

		sandboxRepo.GetReturns(sandbox, nil)

		ex.ExecuteStub = func(command executor.Command) error {
			switch ex.ExecuteCallCount() {
			case 3:
				nsCommand := command.(commands.InNamespace)
				getCommand := nsCommand.Command.(*commands.GetHardwareAddress)
				getCommand.Result = containerMAC
			}
			return nil
		}

		config = container.CreatorConfig{
			NetworkID:       "some-crazy-network-id",
			ContainerNsPath: "/some/container/ns/path",
			ContainerID:     "123456789012345",
			InterfaceName:   "container-link",
			VNI:             99,
			IPAMResult:      ipamResult,
			App:             "some-app-guid",
		}
	})

	It("should open the container NS", func() {
		_, err := creator.Setup(config)
		Expect(err).NotTo(HaveOccurred())

		Expect(namespaceOpener.OpenPathCallCount()).To(Equal(1))
		Expect(namespaceOpener.OpenPathArgsForCall(0)).To(Equal("/some/container/ns/path"))
	})

	Context("when opening the container NS fails", func() {
		It("should return a meaningful error", func() {
			namespaceOpener.OpenPathReturns(nil, errors.New("turnip"))

			_, err := creator.Setup(config)
			Expect(err).To(MatchError("open container netns: turnip"))
		})
	})

	It("should return the info about the container", func() {
		container, err := creator.Setup(config)
		Expect(err).NotTo(HaveOccurred())
		Expect(container).To(Equal(models.Container{
			NetworkID:   "some-crazy-network-id",
			ID:          "123456789012345",
			MAC:         "01:02:03:04:05:06",
			IP:          "192.168.100.2",
			HostIP:      "10.11.12.13",
			SandboxName: "vni-99",
			App:         "some-app-guid",
		}))
	})

	It("should synchronize all operations by locking on the sandbox", func() {
		_, err := creator.Setup(config)
		Expect(err).NotTo(HaveOccurred())

		Expect(sandbox.LockCallCount()).To(Equal(1))
		Expect(sandbox.UnlockCallCount()).To(Equal(1))
	})

	It("should execute the IdempotentlyCreateSandbox command group", func() {
		createSandboxResult := &fakes.Command{}
		commandBuilder.IdempotentlyCreateSandboxReturns(createSandboxResult)

		_, err := creator.Setup(config)
		Expect(err).NotTo(HaveOccurred())
		Expect(ex.ExecuteCallCount()).To(Equal(3))

		Expect(ex.ExecuteArgsForCall(0)).To(Equal(createSandboxResult))

		sandboxName, vxlanDeviceName, dnsAddress := commandBuilder.IdempotentlyCreateSandboxArgsForCall(0)
		Expect(sandboxName).To(Equal("vni-99"))
		Expect(vxlanDeviceName).To(Equal("vxlan99"))
		Expect(dnsAddress).To(Equal("some-dns-address"))
	})

	Context("when creating the sandbox errors", func() {
		It("should return a meaningful error", func() {
			ex.ExecuteReturns(errors.New("potato"))

			_, err := creator.Setup(config)
			Expect(err).To(MatchError("executing command: create sandbox: potato"))
		})
	})

	It("should get the sandbox ns from the sandbox repo", func() {
		_, err := creator.Setup(config)
		Expect(err).NotTo(HaveOccurred())

		Expect(sandboxRepo.GetCallCount()).To(Equal(1))
		Expect(sandboxRepo.GetArgsForCall(0)).To(Equal("vni-99"))
	})

	Context("when getting the sandbox ns from the sandbox repo fails", func() {
		It("should return a meaningful error", func() {
			sandboxRepo.GetReturns(nil, errors.New("daikon"))

			_, err := creator.Setup(config)
			Expect(err).To(MatchError("get sandbox: daikon"))
		})
	})

	It("should execute the IdempotentlyCreateVxlan command group", func() {
		createVxlanResult := &fakes.Command{}
		commandBuilder.IdempotentlyCreateVxlanReturns(createVxlanResult)

		_, err := creator.Setup(config)
		Expect(err).NotTo(HaveOccurred())
		Expect(ex.ExecuteCallCount()).To(Equal(3))

		commandGroup := (ex.ExecuteArgsForCall(1)).(commands.Group)
		Expect(commandGroup[0]).To(Equal(createVxlanResult))

		vxlanName, vni, sandboxName, sbNS := commandBuilder.IdempotentlyCreateVxlanArgsForCall(0)
		Expect(vxlanName).To(Equal("vxlan99"))
		Expect(vni).To(Equal(99))
		Expect(sandboxName).To(Equal("vni-99"))
		Expect(sbNS).To(Equal(sandboxNS))
	})

	It("should execute the SetupVeth command group, including the route commands", func() {
		setupContainerResult := &fakes.Command{}
		fakeRouteCommands := &fakes.Command{}

		commandBuilder.SetupVethReturns(setupContainerResult)
		commandBuilder.AddRoutesReturns(fakeRouteCommands)

		_, err := creator.Setup(config)
		Expect(err).NotTo(HaveOccurred())

		commandGroup := (ex.ExecuteArgsForCall(1)).(commands.Group)
		Expect(commandGroup[1]).To(Equal(setupContainerResult))

		contNS, sandboxLinkName, containerLinkName, address, sbNS, routeCommands := commandBuilder.SetupVethArgsForCall(0)
		Expect(contNS).To(Equal(containerNS))
		Expect(sandboxLinkName).To(Equal("MXGEYC3M7HCW4KR"))
		Expect(containerLinkName).To(Equal("container-link"))
		Expect(address).To(Equal(ipamResult.IP4.IP))
		Expect(sbNS).To(Equal(sandboxNS))
		Expect(routeCommands).To(BeIdenticalTo(fakeRouteCommands))
	})

	It("should execute the IdempotentlySetupBridge command group", func() {
		setupBridgeResult := &fakes.Command{}

		commandBuilder.IdempotentlySetupBridgeReturns(setupBridgeResult)

		_, err := creator.Setup(config)
		Expect(err).NotTo(HaveOccurred())

		commandGroup := (ex.ExecuteArgsForCall(1)).(commands.Group)
		Expect(commandGroup[2]).To(Equal(setupBridgeResult))

		vxlanName, sandboxLinkName, bridgeName, sbNS, ipamResult := commandBuilder.IdempotentlySetupBridgeArgsForCall(0)
		Expect(vxlanName).To(Equal("vxlan99"))
		Expect(sandboxLinkName).To(Equal("MXGEYC3M7HCW4KR"))
		Expect(bridgeName).To(Equal("vxlanbr99"))
		Expect(sbNS).To(Equal(sandboxNS))
		Expect(ipamResult).To(Equal(&types.Result{
			IP4: &types.IPConfig{
				IP: net.IPNet{
					IP:   net.ParseIP("192.168.100.2"),
					Mask: net.CIDRMask(24, 32),
				},
				Gateway: net.ParseIP("192.168.100.1"),
				Routes: []types.Route{{
					Dst: net.IPNet{
						IP:   net.ParseIP("192.168.1.5"),
						Mask: net.CIDRMask(24, 32),
					},
					GW: net.ParseIP("192.168.1.1"),
				}, {
					Dst: net.IPNet{
						IP:   net.ParseIP("192.168.2.5"),
						Mask: net.CIDRMask(24, 32),
					},
					GW: net.ParseIP("192.168.1.99"),
				}},
			},
		}))
	})

	Context("when the container ID is very long", func() {
		It("keeps the sandbox link name short", func() {
			config.ContainerID = "1234567890123456789"

			_, err := creator.Setup(config)
			Expect(err).NotTo(HaveOccurred())

			_, sandboxLinkName, _, _, _, _ := commandBuilder.SetupVethArgsForCall(0)
			Expect(sandboxLinkName).To(HaveLen(15))

			_, sandboxLinkName, _, _, _ = commandBuilder.IdempotentlySetupBridgeArgsForCall(0)
			Expect(sandboxLinkName).To(HaveLen(15))
		})
	})

	It("sets the sandbox link name using only alphanumeric characters", func() {
		config.ContainerID = "1234567890123456789"

		_, err := creator.Setup(config)
		Expect(err).NotTo(HaveOccurred())

		_, sandboxLinkName, _, _, _, _ := commandBuilder.SetupVethArgsForCall(0)

		matches, err := regexp.MatchString("^[a-zA-Z0-9]*$", sandboxLinkName)
		Expect(err).NotTo(HaveOccurred())
		Expect(matches).To(BeTrue())
	})

	Context("when there are two container IDs with the same prefix", func() {
		It("gives them different sandbox link names", func() {
			config.ContainerID = "1234567890123456789"

			_, err := creator.Setup(config)
			Expect(err).NotTo(HaveOccurred())

			_, sandboxLinkName1, _, _, _ := commandBuilder.IdempotentlySetupBridgeArgsForCall(0)
			_, sandboxLinkName1, _, _, _, _ = commandBuilder.SetupVethArgsForCall(0)

			config.ContainerID = "1234567890123456798"

			_, err = creator.Setup(config)
			Expect(err).NotTo(HaveOccurred())

			_, sandboxLinkName2, _, _, _ := commandBuilder.IdempotentlySetupBridgeArgsForCall(1)
			_, sandboxLinkName2, _, _, _, _ = commandBuilder.SetupVethArgsForCall(1)

			Expect(sandboxLinkName1).NotTo(Equal(sandboxLinkName2))
		})
	})

	Context("when an error occurs", func() {
		Context("when setting up the container fails", func() {
			BeforeEach(func() {
				ex.ExecuteStub = func(command executor.Command) error {
					switch ex.ExecuteCallCount() {
					case 2:
						return errors.New("some setup error")
					}

					return nil
				}
			})

			It("should return an error", func() {
				_, err := creator.Setup(config)
				Expect(err).To(MatchError("some setup error"))
			})
		})

		Context("when setting the hardware address fails", func() {
			BeforeEach(func() {
				ex.ExecuteStub = func(command executor.Command) error {
					switch ex.ExecuteCallCount() {
					case 2:
						return errors.New("some hardware error")
					}

					return nil
				}
			})

			It("should return an error", func() {
				_, err := creator.Setup(config)
				Expect(err).To(MatchError("some hardware error"))
			})
		})
	})
})
