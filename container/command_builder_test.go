package container_test

import (
	"net"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/conditions"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CommandBuilder", func() {
	Describe("IdempotentlyCreateSandbox", func() {
		var b container.CommandBuilder

		BeforeEach(func() {
			b = container.CommandBuilder{}
		})

		It("should return a command group that idempotently creates the sandbox", func() {
			cmd := b.IdempotentlyCreateSandbox("some-sandbox-name", "some-vxlan-name")

			Expect(cmd).To(Equal(
				commands.Unless{
					Condition: conditions.SandboxNamespaceExists{
						Name: "some-sandbox-name",
					},
					Command: commands.All(
						commands.CreateSandboxNamespace{
							Name: "some-sandbox-name",
						},
					),
				}))
		})
	})

	Describe("IdempotentlyCreateVxlan", func() {
		var sandboxNS *fakes.Namespace

		It("should return a command group that idempotently creates the vxlan device", func() {
			hostNamespace := &fakes.Namespace{NameStub: func() string { return "host namespace sentinel" }}

			sandboxNS = &fakes.Namespace{}
			sandboxNS.NameReturns("/sandbox/repo/some-sandbox-name")

			missWatcher := &fakes.MissWatcher{}
			b := container.CommandBuilder{
				HostNamespace: hostNamespace,
				MissWatcher:   missWatcher,
			}

			cmd := b.IdempotentlyCreateVxlan("some-vxlan-name", 1234, "some-sandbox-name", sandboxNS)

			Expect(cmd).To(Equal(
				commands.InNamespace{
					Namespace: sandboxNS,
					Command: commands.Unless{
						Condition: conditions.LinkExists{
							Name: "some-vxlan-name",
						},
						Command: commands.All(
							commands.InNamespace{
								Namespace: hostNamespace,
								Command: commands.All(
									commands.CreateVxlan{
										Name: "some-vxlan-name",
										VNI:  1234,
									},
									commands.MoveLink{
										Namespace: sandboxNS,
										Name:      "some-vxlan-name",
									},
								),
							},
							commands.InNamespace{
								Namespace: sandboxNS,
								Command: commands.SetLinkUp{
									LinkName: "some-vxlan-name",
								},
							},
							commands.StartMonitor{
								HostNamespace: hostNamespace,
								Watcher:       missWatcher,
								SandboxName:   "some-sandbox-name",
								VxlanLinkName: "some-vxlan-name",
							},
						),
					},
				},
			))
		})
	})

	Describe("AddRoutes", func() {
		var ipConfig *types.IPConfig

		BeforeEach(func() {
			ipConfig = &types.IPConfig{
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
			}
		})

		It("should return a command group that adds routes to an interface", func() {
			b := container.CommandBuilder{}
			cmd := b.AddRoutes("some-interface-name", ipConfig)

			Expect(cmd).To(Equal(
				commands.All(
					commands.AddRoute{
						Interface: "some-interface-name",
						Destination: net.IPNet{
							IP:   net.ParseIP("192.168.1.5"),
							Mask: net.CIDRMask(24, 32),
						},
						Gateway: net.ParseIP("192.168.1.1"),
					},
					commands.AddRoute{
						Interface: "some-interface-name",
						Destination: net.IPNet{
							IP:   net.ParseIP("192.168.2.5"),
							Mask: net.CIDRMask(24, 32),
						},
						Gateway: net.ParseIP("192.168.1.99"),
					},
				),
			))
		})

		Context("when a route is missing a gateway", func() {
			BeforeEach(func() {
				ipConfig.Routes[1].GW = nil
			})

			It("should set the gateway to be the default gateway from the IPConfig", func() {
				b := container.CommandBuilder{}
				cmd := b.AddRoutes("some-interface-name", ipConfig)

				Expect(cmd.(commands.Group)[1].(commands.AddRoute).Gateway).To(Equal(
					net.ParseIP("192.168.100.1"),
				))
			})
		})
	})

	Describe("SetupVeth", func() {
		var (
			b            container.CommandBuilder
			routeCommand executor.Command
			containerNS  namespace.Namespace
			sandboxNS    *fakes.Namespace
		)

		BeforeEach(func() {
			sandboxNS = &fakes.Namespace{}
			sandboxNS.NameReturns("/some/sb/path")
			routeCommand = commands.AddRoute{Interface: "something"}

			b = container.CommandBuilder{}

			containerNS = &fakes.Namespace{NameStub: func() string { return "container ns sentinel" }}
		})

		It("should return a command group that sets up veth in container", func() {
			address := net.IPNet{
				IP:   net.ParseIP("192.168.2.5"),
				Mask: net.CIDRMask(24, 32),
			}

			cmd := b.SetupVeth(
				containerNS,
				"sandbox-veth",
				"container-veth",
				address,
				sandboxNS,
				routeCommand,
			)

			Expect(cmd).To(Equal(
				commands.InNamespace{
					Namespace: containerNS,
					Command: commands.Group(
						append(
							[]executor.Command{
								commands.CreateVeth{
									Name:     "container-veth",
									PeerName: "sandbox-veth",
									MTU:      1450,
								},
								commands.MoveLink{
									Name:      "sandbox-veth",
									Namespace: sandboxNS,
								},
								commands.AddAddress{
									InterfaceName: "container-veth",
									Address:       address,
								},
								commands.SetLinkUp{
									LinkName: "container-veth",
								},
							},
							routeCommand,
						),
					),
				},
			))
		})
	})

	Describe("IdempotentlySetupBridge", func() {
		It("returns a command group that sets up the bridge", func() {
			sandboxNS := &fakes.Namespace{NameStub: func() string { return "sandbox ns sentinel" }}
			b := container.CommandBuilder{}

			ipamResult := &types.Result{
				IP4: &types.IPConfig{
					IP: net.IPNet{
						IP:   net.ParseIP("192.168.100.2"),
						Mask: net.CIDRMask(24, 32),
					},
					Gateway: net.ParseIP("192.168.100.1"),
					Routes: []types.Route{
						{
							Dst: net.IPNet{
								IP:   net.ParseIP("192.168.1.5"),
								Mask: net.CIDRMask(24, 32),
							},
							GW: net.ParseIP("192.168.1.1"),
						},
					},
				},
			}

			cmd := b.IdempotentlySetupBridge("some-vxlan-name", "some-link-name", "some-bridge-name", sandboxNS, ipamResult)

			Expect(cmd).To(Equal(
				commands.InNamespace{
					Namespace: sandboxNS,
					Command: commands.All(
						commands.SetLinkUp{
							LinkName: "some-link-name",
						},
						commands.Unless{
							Condition: conditions.LinkExists{
								Name: "some-bridge-name",
							},
							Command: commands.All(
								commands.CreateBridge{
									Name: "some-bridge-name",
								},
								commands.AddAddress{
									InterfaceName: "some-bridge-name",
									Address: net.IPNet{
										IP:   net.ParseIP("192.168.100.1"),
										Mask: net.CIDRMask(24, 32),
									},
								},
								commands.SetLinkUp{
									LinkName: "some-bridge-name",
								},
							),
						},
						commands.SetLinkMaster{
							Master: "some-bridge-name",
							Slave:  "some-vxlan-name",
						},
						commands.SetLinkMaster{
							Master: "some-bridge-name",
							Slave:  "some-link-name",
						},
					),
				},
			))
		})
	})
})
