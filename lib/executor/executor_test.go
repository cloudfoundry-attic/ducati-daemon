package executor_test

import (
	"errors"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/conditions"
	exec_fakes "github.com/cloudfoundry-incubator/ducati-daemon/executor/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/executor/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"

	"github.com/appc/cni/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Executor", func() {
	var (
		commandExecutor *exec_fakes.Executor
		linkFactory     *fakes.LinkFactory

		ipamResult types.Result
		ex         executor.Executor
	)

	BeforeEach(func() {
		linkFactory = &fakes.LinkFactory{}
		commandExecutor = &exec_fakes.Executor{}

		ex = executor.Executor{
			Executor:    commandExecutor,
			LinkFactory: linkFactory,
		}

		ipamResult = types.Result{
			IP4: &types.IPConfig{
				IP: net.IPNet{
					IP:   net.ParseIP("192.168.100.1"),
					Mask: net.CIDRMask(24, 32),
				},
				Gateway: net.ParseIP("192.168.100.1"),
				Routes: []types.Route{{
					Dst: net.IPNet{
						IP:   net.ParseIP("192.168.1.5"),
						Mask: net.CIDRMask(24, 32),
					},
					GW: net.ParseIP("192.168.1.1"),
				}},
			},
		}
	})

	Describe("EnsureVxlanDeviceExists", func() {
		It("executes the setup actions", func() {
			_, err := ex.EnsureVxlanDeviceExists(99, namespace.NewNamespace("/some/namespace"))
			Expect(err).NotTo(HaveOccurred())
			Expect(commandExecutor.ExecuteCallCount()).To(Equal(1))

			command := commandExecutor.ExecuteArgsForCall(0)
			Expect(command).To(Equal(
				commands.InNamespace{
					Namespace: namespace.NewNamespace("/some/namespace"),
					Command: commands.Unless{
						Condition: conditions.LinkExists{
							LinkFinder: linkFactory,
							Name:       "vxlan99",
						},
						Command: commands.InNamespace{
							Namespace: namespace.NewNamespace("/proc/self/ns/net"),
							Command: commands.All(
								commands.CreateVxlan{
									Name: "vxlan99",
									VNI:  99,
								},
								commands.SetLinkNamespace{
									Namespace: "/some/namespace",
									Name:      "vxlan99",
								},
							),
						},
					},
				},
			))
		})

		It("returns the vxlan link name", func() {
			name, err := ex.EnsureVxlanDeviceExists(99, namespace.NewNamespace("/some/namespace"))
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("vxlan99"))
		})

		Context("when the setup commands fail", func() {
			BeforeEach(func() {
				commandExecutor.ExecuteReturns(errors.New("boom"))
			})

			It("returns a meaningful error", func() {
				_, err := ex.EnsureVxlanDeviceExists(99, namespace.NewNamespace("/some/namespace"))
				Expect(err).To(MatchError("failed to setup vxlan device: boom"))
			})
		})
	})

	Describe("SetupSandboxNS", func() {
		It("executes the setup actions", func() {
			err := ex.SetupSandboxNS(
				"vxlan-name", "bridge-name",
				namespace.NewNamespace("/sandbox/namespace"),
				"sandbox-link",
				ipamResult,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(commandExecutor.ExecuteCallCount()).To(Equal(1))

			command := commandExecutor.ExecuteArgsForCall(0)
			Expect(command).To(Equal(
				commands.InNamespace{
					Namespace: namespace.NewNamespace("/sandbox/namespace"),
					Command: commands.All(
						commands.SetLinkUp{
							LinkName: "vxlan-name",
						},
						commands.SetLinkUp{
							LinkName: "sandbox-link",
						},
						commands.Unless{
							Condition: conditions.LinkExists{
								LinkFinder: linkFactory,
								Name:       "bridge-name",
							},
							Command: commands.All(
								commands.CreateBridge{
									Name: "bridge-name",
								},
								commands.AddAddress{
									InterfaceName: "bridge-name",
									Address: net.IPNet{
										IP:   ipamResult.IP4.Gateway,
										Mask: ipamResult.IP4.IP.Mask,
									},
								},
								commands.SetLinkUp{
									LinkName: "bridge-name",
								},
							),
						},
						commands.SetLinkMaster{
							Master: "bridge-name",
							Slave:  "vxlan-name",
						},
						commands.SetLinkMaster{
							Master: "bridge-name",
							Slave:  "sandbox-link",
						},
					),
				},
			))
		})
	})

	Describe("SetupContainerNS", func() {
		It("executes the setup actions", func() {
			macAddress := "01:02:03:04:05:06"
			containerMAC, err := net.ParseMAC(macAddress)
			Expect(err).NotTo(HaveOccurred())

			commandExecutor.ExecuteStub = func(command commands.Command) error {
				if commandExecutor.ExecuteCallCount() == 2 {
					Expect(command).To(Equal(
						commands.InNamespace{
							Namespace: namespace.NewNamespace("/var/some/container/namespace"),
							Command: &commands.GetHardwareAddress{
								LinkName: "some-eth0",
								Result:   nil,
							},
						},
					))

					inNamespace := command.(commands.InNamespace)
					inNamespace.Command.(*commands.GetHardwareAddress).Result = containerMAC
				}
				return nil
			}

			sandboxLinkName, containerLinkMAC, err := ex.SetupContainerNS(
				"/var/some/sandbox/namespace",
				"/var/some/container/namespace",
				"some-container-id",
				"some-eth0",
				ipamResult,
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(sandboxLinkName).To(Equal("some-contai"))
			Expect(containerLinkMAC).To(Equal(macAddress))
			Expect(commandExecutor.ExecuteCallCount()).To(Equal(2))

			Expect(commandExecutor.ExecuteArgsForCall(0)).To(Equal(
				commands.InNamespace{
					Namespace: namespace.NewNamespace("/var/some/container/namespace"),
					Command: commands.All(
						commands.CreateVeth{
							Name:     "some-eth0",
							PeerName: "some-contai",
							MTU:      1450,
						},
						commands.SetLinkNamespace{
							Name:      "some-contai",
							Namespace: "/var/some/sandbox/namespace",
						},
						commands.AddAddress{
							InterfaceName: "some-eth0",
							Address:       ipamResult.IP4.IP,
						},
						commands.SetLinkUp{
							LinkName: "some-eth0",
						},
						commands.AddRoute{
							Interface: "some-eth0",
							Destination: net.IPNet{
								IP:   net.ParseIP("192.168.1.5"),
								Mask: net.CIDRMask(24, 32),
							},
							Gateway: net.ParseIP("192.168.1.1"),
						},
					),
				},
			))
		})

		Context("when no routes are specified", func() {
			BeforeEach(func() {
				ipamResult.IP4.Routes = []types.Route{}
			})

			It("does not attempt to add routes", func() {
				ex.SetupContainerNS(
					"/var/some/sandbox/namespace",
					"/var/some/container/namespace",
					"some-container-id",
					"some-eth0",
					ipamResult,
				)

				Expect(commandExecutor.ExecuteCallCount()).To(Equal(2))
				Expect(commandExecutor.ExecuteArgsForCall(0)).To(Equal(
					commands.InNamespace{
						Namespace: namespace.NewNamespace("/var/some/container/namespace"),
						Command: commands.All(
							commands.CreateVeth{
								Name:     "some-eth0",
								PeerName: "some-contai",
								MTU:      1450,
							},
							commands.SetLinkNamespace{
								Name:      "some-contai",
								Namespace: "/var/some/sandbox/namespace",
							},
							commands.AddAddress{
								InterfaceName: "some-eth0",
								Address:       ipamResult.IP4.IP,
							},
							commands.SetLinkUp{
								LinkName: "some-eth0",
							},
						),
					},
				))
			})
		})

		Context("when multiple routes are specified", func() {
			BeforeEach(func() {
				ipamResult.IP4.Routes = append(ipamResult.IP4.Routes, types.Route{
					Dst: net.IPNet{
						IP:   net.ParseIP("10.10.10.10"),
						Mask: net.CIDRMask(8, 32),
					},
					GW: net.ParseIP("10.10.10.1"),
				})
			})

			It("adds all routes", func() {
				ex.SetupContainerNS(
					"/var/some/sandbox/namespace",
					"/var/some/container/namespace",
					"some-container-id",
					"some-eth0",
					ipamResult,
				)

				Expect(commandExecutor.ExecuteCallCount()).To(Equal(2))
				Expect(commandExecutor.ExecuteArgsForCall(0)).To(Equal(
					commands.InNamespace{
						Namespace: namespace.NewNamespace("/var/some/container/namespace"),
						Command: commands.All(
							commands.CreateVeth{
								Name:     "some-eth0",
								PeerName: "some-contai",
								MTU:      1450,
							},
							commands.SetLinkNamespace{
								Name:      "some-contai",
								Namespace: "/var/some/sandbox/namespace",
							},
							commands.AddAddress{
								InterfaceName: "some-eth0",
								Address:       ipamResult.IP4.IP,
							},
							commands.SetLinkUp{
								LinkName: "some-eth0",
							},
							commands.AddRoute{
								Interface: "some-eth0",
								Destination: net.IPNet{
									IP:   net.ParseIP("192.168.1.5"),
									Mask: net.CIDRMask(24, 32),
								},
								Gateway: net.ParseIP("192.168.1.1"),
							},
							commands.AddRoute{
								Interface: "some-eth0",
								Destination: net.IPNet{
									IP:   net.ParseIP("10.10.10.10"),
									Mask: net.CIDRMask(8, 32),
								},
								Gateway: net.ParseIP("10.10.10.1"),
							},
						),
					},
				))
			})
		})

		Context("When a gateway is missing from the the route", func() {
			BeforeEach(func() {
				ipamResult.IP4.Routes[0].GW = nil
				ipamResult.IP4.Gateway = net.ParseIP("192.168.100.1")
			})

			It("uses the default gateway for the route", func() {
				ex.SetupContainerNS(
					"/var/some/sandbox/namespace",
					"/var/some/container/namespace",
					"some-container-id",
					"some-eth0",
					ipamResult,
				)

				Expect(commandExecutor.ExecuteCallCount()).To(Equal(2))
				Expect(commandExecutor.ExecuteArgsForCall(0)).To(Equal(
					commands.InNamespace{
						Namespace: namespace.NewNamespace("/var/some/container/namespace"),
						Command: commands.All(
							commands.CreateVeth{
								Name:     "some-eth0",
								PeerName: "some-contai",
								MTU:      1450,
							},
							commands.SetLinkNamespace{
								Name:      "some-contai",
								Namespace: "/var/some/sandbox/namespace",
							},
							commands.AddAddress{
								InterfaceName: "some-eth0",
								Address:       ipamResult.IP4.IP,
							},
							commands.SetLinkUp{
								LinkName: "some-eth0",
							},
							commands.AddRoute{
								Interface: "some-eth0",
								Destination: net.IPNet{
									IP:   net.ParseIP("192.168.1.5"),
									Mask: net.CIDRMask(24, 32),
								},
								Gateway: net.ParseIP("192.168.100.1"),
							},
						),
					},
				))
			})
		})

		Context("when executing the setup commands fails", func() {
			BeforeEach(func() {
				commandExecutor.ExecuteReturns(errors.New("boom"))
			})

			It("returns a meaningful error", func() {
				_, _, err := ex.SetupContainerNS(
					"/var/some/sandbox/namespace",
					"/var/some/container/namespace",
					"some-container-id",
					"some-eth0",
					ipamResult,
				)

				Expect(commandExecutor.ExecuteCallCount()).To(Equal(1))
				Expect(err).To(MatchError("container namespace setup failed: boom"))
			})
		})

		Context("when getting the hardware address fails", func() {
			BeforeEach(func() {
				commandExecutor.ExecuteStub = func(command commands.Command) error {
					if commandExecutor.ExecuteCallCount() == 2 {
						return errors.New("boom")
					}
					return nil
				}
			})

			It("returns a meaningful error", func() {
				_, _, err := ex.SetupContainerNS(
					"/var/some/sandbox/namespace",
					"/var/some/container/namespace",
					"some-container-id",
					"some-eth0",
					ipamResult,
				)

				Expect(commandExecutor.ExecuteCallCount()).To(Equal(2))
				Expect(err).To(MatchError("failed to get container hardware address: boom"))
			})
		})
	})
})
