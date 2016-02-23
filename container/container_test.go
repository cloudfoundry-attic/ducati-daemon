package container_test

import (
	"errors"
	"net"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/conditions"
	cond_fakes "github.com/cloudfoundry-incubator/ducati-daemon/conditions/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Setup", func() {
	var (
		creator      container.Creator
		executor     *fakes.Executor
		linkFinder   *cond_fakes.LinkFinder
		containerMAC net.HardwareAddr
		ipamResult   types.Result
		config       container.CreatorConfig
	)

	BeforeEach(func() {
		executor = &fakes.Executor{}
		linkFinder = &cond_fakes.LinkFinder{}
		creator = container.Creator{
			Executor:   executor,
			LinkFinder: linkFinder,
		}

		var err error
		macAddress := "01:02:03:04:05:06"
		containerMAC, err = net.ParseMAC(macAddress)
		Expect(err).NotTo(HaveOccurred())

		ipamResult = types.Result{
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

		executor.ExecuteStub = func(command commands.Command) error {
			if executor.ExecuteCallCount() == 2 {
				nsCommand := command.(commands.InNamespace)
				getCommand := nsCommand.Command.(*commands.GetHardwareAddress)
				getCommand.Result = containerMAC
			}
			return nil
		}

		config = container.CreatorConfig{
			SandboxNsPath:   "/some/sandbox/namespace",
			ContainerNsPath: "/some/container/namespace",
			ContainerID:     "sandbox-link",
			InterfaceName:   "container-link",
			BridgeName:      "vxlan-br0",
			VNI:             99,
			IPAMResult:      ipamResult,
		}
	})

	It("should return a container that has been setup", func() {
		container, err := creator.Setup(config)
		Expect(err).NotTo(HaveOccurred())
		Expect(container).To(Equal(models.Container{
			ID:  "sandbox-link",
			MAC: "01:02:03:04:05:06",
			IP:  "192.168.100.2",
		}))

		Expect(executor.ExecuteCallCount()).To(Equal(2))
		Expect(executor.ExecuteArgsForCall(0)).To(BeEquivalentTo(
			commands.All(
				commands.InNamespace{
					Namespace: namespace.NewNamespace("/some/sandbox/namespace"),
					Command: commands.Unless{
						Condition: conditions.LinkExists{
							LinkFinder: linkFinder,
							Name:       "vxlan99",
						},
						Command: commands.All(
							commands.InNamespace{
								Namespace: namespace.NewNamespace("/proc/self/ns/net"),
								Command: commands.All(
									commands.CreateVxlan{
										Name: "vxlan99",
										VNI:  99,
									},
									commands.SetLinkNamespace{
										Namespace: "/some/sandbox/namespace",
										Name:      "vxlan99",
									},
								),
							},
							commands.SetLinkUp{
								LinkName: "vxlan99",
							},
						),
					},
				},
				commands.InNamespace{
					Namespace: namespace.NewNamespace("/some/container/namespace"),
					Command: commands.Group{commands.CreateVeth{
						Name:     "container-link",
						PeerName: "sandbox-link",
						MTU:      1450,
					},
						commands.SetLinkNamespace{
							Name:      "sandbox-link",
							Namespace: "/some/sandbox/namespace",
						},
						commands.AddAddress{
							InterfaceName: "container-link",
							Address: net.IPNet{
								IP:   net.ParseIP("192.168.100.2"),
								Mask: net.CIDRMask(24, 32),
							},
						},
						commands.SetLinkUp{
							LinkName: "container-link",
						},
						commands.AddRoute{
							Interface: "container-link",
							Destination: net.IPNet{
								IP:   net.ParseIP("192.168.1.5"),
								Mask: net.CIDRMask(24, 32),
							},
							Gateway: net.ParseIP("192.168.1.1"),
						},
						commands.AddRoute{
							Interface: "container-link",
							Destination: net.IPNet{
								IP:   net.ParseIP("192.168.2.5"),
								Mask: net.CIDRMask(24, 32),
							},
							Gateway: net.ParseIP("192.168.1.99"),
						},
					},
				},
				commands.InNamespace{
					Namespace: namespace.NewNamespace("/some/sandbox/namespace"),
					Command: commands.All(
						commands.SetLinkUp{
							LinkName: "sandbox-link",
						},
						commands.Unless{
							Condition: conditions.LinkExists{
								LinkFinder: linkFinder,
								Name:       "vxlan-br0",
							},
							Command: commands.All(
								commands.CreateBridge{
									Name: "vxlan-br0",
								},
								commands.AddAddress{
									InterfaceName: "vxlan-br0",
									Address: net.IPNet{
										IP:   net.ParseIP("192.168.100.1"),
										Mask: net.CIDRMask(24, 32),
									},
								},
								commands.SetLinkUp{
									LinkName: "vxlan-br0",
								},
							),
						},
						commands.SetLinkMaster{
							Master: "vxlan-br0",
							Slave:  "vxlan99",
						},
						commands.SetLinkMaster{
							Master: "vxlan-br0",
							Slave:  "sandbox-link",
						},
					),
				},
			),
		))
	})

	Context("when an error occurs", func() {
		Context("when setting up the container fails", func() {
			BeforeEach(func() {
				executor.ExecuteStub = func(command commands.Command) error {
					switch executor.ExecuteCallCount() {
					case 1:
						return errors.New("some setup error")
					default:
						return nil
					}

				}
			})

			It("should return an error", func() {
				_, err := creator.Setup(config)
				Expect(err).To(MatchError("some setup error"))
			})
		})

		Context("when setting the hardware address fails", func() {
			BeforeEach(func() {
				executor.ExecuteStub = func(command commands.Command) error {
					switch executor.ExecuteCallCount() {
					case 2:
						return errors.New("some hardware error")
					default:
						return nil
					}

				}
			})

			It("should return an error", func() {
				_, err := creator.Setup(config)
				Expect(err).To(MatchError("some hardware error"))
			})
		})
	})
})
