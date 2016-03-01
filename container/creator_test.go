package container_test

import (
	"errors"
	"fmt"
	"net"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	comm_fakes "github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/conditions"
	cond_fakes "github.com/cloudfoundry-incubator/ducati-daemon/conditions/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	exec_fakes "github.com/cloudfoundry-incubator/ducati-daemon/executor/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Setup", func() {
	var (
		creator           container.Creator
		executor          *exec_fakes.Executor
		linkFinder        *cond_fakes.LinkFinder
		containerMAC      net.HardwareAddr
		ipamResult        types.Result
		config            container.CreatorConfig
		sandboxRepository *comm_fakes.Repository
		sandboxNS         namespace.Namespace
		locker            *comm_fakes.Locker
	)

	BeforeEach(func() {
		executor = &exec_fakes.Executor{}
		linkFinder = &cond_fakes.LinkFinder{}
		sandboxRepository = &comm_fakes.Repository{}
		locker = &comm_fakes.Locker{}
		creator = container.Creator{
			Executor:    executor,
			LinkFinder:  linkFinder,
			SandboxRepo: sandboxRepository,
			Locker:      locker,
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

		sandboxNS = namespace.NewNamespace("/some/sandbox/namespace")
		executor.ExecuteStub = func(command commands.Command) error {
			switch executor.ExecuteCallCount() {
			case 1:
				cnsUnless := command.(commands.Unless)
				cnsCommand := cnsUnless.Command.(*commands.CreateNamespace)
				cnsCommand.Result = sandboxNS
			case 3:
				nsCommand := command.(commands.InNamespace)
				getCommand := nsCommand.Command.(*commands.GetHardwareAddress)
				getCommand.Result = containerMAC
			}
			return nil
		}

		config = container.CreatorConfig{
			NetworkID:       "some-crazy-network-id",
			ContainerNsPath: "/some/container/namespace",
			ContainerID:     "sandbox-link",
			InterfaceName:   "container-link",
			BridgeName:      "vxlan-br0",
			VNI:             99,
			HostIP:          "10.11.12.13",
			IPAMResult:      ipamResult,
		}
	})

	It("should synchronize all operations by locking on the sandbox", func() {
		_, err := creator.Setup(config)
		Expect(err).NotTo(HaveOccurred())

		Expect(locker.LockCallCount()).To(Equal(1))
		Expect(locker.UnlockCallCount()).To(Equal(1))
		Expect(locker.LockArgsForCall(0)).To(Equal("vni-99"))
		Expect(locker.UnlockArgsForCall(0)).To(Equal("vni-99"))
	})

	It("should return a container that has been setup", func() {
		container, err := creator.Setup(config)
		Expect(err).NotTo(HaveOccurred())
		Expect(container).To(Equal(models.Container{
			NetworkID: "some-crazy-network-id",
			ID:        "sandbox-link",
			MAC:       "01:02:03:04:05:06",
			IP:        "192.168.100.2",
			HostIP:    "10.11.12.13",
		}))

		Expect(executor.ExecuteCallCount()).To(Equal(3))

		Expect(executor.ExecuteArgsForCall(0)).To(Equal(
			commands.Unless{
				Condition: conditions.NamespaceExists{
					Name:       fmt.Sprintf("vni-%d", config.VNI),
					Repository: sandboxRepository,
				},
				Command: &commands.CreateNamespace{
					Name:       fmt.Sprintf("vni-%d", config.VNI),
					Repository: sandboxRepository,
					Result:     sandboxNS,
				},
			},
		))

		Expect(executor.ExecuteArgsForCall(1)).To(BeEquivalentTo(
			commands.All(
				commands.InNamespace{
					Namespace: sandboxNS,
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
									commands.MoveLink{
										Namespace: "/some/sandbox/namespace",
										Name:      "vxlan99",
									},
								),
							},
							commands.InNamespace{
								Namespace: namespace.NewNamespace("/some/sandbox/namespace"),
								Command: commands.SetLinkUp{
									LinkName: "vxlan99",
								},
							},
						),
					},
				},
				commands.InNamespace{
					Namespace: namespace.NewNamespace("/some/container/namespace"),
					Command: commands.Group{
						commands.CreateVeth{
							Name:     "container-link",
							PeerName: "sandbox-link",
							MTU:      1450,
						},
						commands.MoveLink{
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

	Context("when the container ID is longer than 15 characters", func() {
		BeforeEach(func() {
			config = container.CreatorConfig{
				NetworkID:       "some-other-network-id",
				ContainerNsPath: "/some/container/namespace",
				ContainerID:     "1234567890123456",
				InterfaceName:   "container-link",
				BridgeName:      "vxlan-br0",
				VNI:             99,
				HostIP:          "10.11.12.13",
				IPAMResult:      ipamResult,
			}
		})

		It("truncates the sandbox link name", func() {
			_, err := creator.Setup(config)
			Expect(err).NotTo(HaveOccurred())

			Expect(executor.ExecuteCallCount()).To(Equal(3))
			Expect(executor.ExecuteArgsForCall(1)).To(BeEquivalentTo(
				commands.All(
					commands.InNamespace{
						Namespace: sandboxNS,
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
										commands.MoveLink{
											Namespace: "/some/sandbox/namespace",
											Name:      "vxlan99",
										},
									),
								},
								commands.InNamespace{
									Namespace: namespace.NewNamespace("/some/sandbox/namespace"),
									Command: commands.SetLinkUp{
										LinkName: "vxlan99",
									},
								},
							),
						},
					},
					commands.InNamespace{
						Namespace: namespace.NewNamespace("/some/container/namespace"),
						Command: commands.Group{
							commands.CreateVeth{
								Name:     "container-link",
								PeerName: "123456789012345",
								MTU:      1450,
							},
							commands.MoveLink{
								Name:      "123456789012345",
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
						Namespace: sandboxNS,
						Command: commands.All(
							commands.SetLinkUp{
								LinkName: "123456789012345",
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
								Slave:  "123456789012345",
							},
						),
					},
				),
			))
		})
	})

	Context("when an error occurs", func() {
		Context("when creating the sandbox namespace fails", func() {
			BeforeEach(func() {
				executor.ExecuteStub = func(command commands.Command) error {
					switch executor.ExecuteCallCount() {
					case 1:
						return errors.New("some sandbox create error")
					}

					return nil
				}
			})

			It("should return an error", func() {
				_, err := creator.Setup(config)
				Expect(err).To(MatchError("some sandbox create error"))
			})
		})

		Context("when setting up the container fails", func() {
			BeforeEach(func() {
				executor.ExecuteStub = func(command commands.Command) error {
					switch executor.ExecuteCallCount() {
					case 1:
						cnsUnless := command.(commands.Unless)
						cnsCommand := cnsUnless.Command.(*commands.CreateNamespace)
						cnsCommand.Result = sandboxNS
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
				executor.ExecuteStub = func(command commands.Command) error {
					switch executor.ExecuteCallCount() {
					case 1:
						cnsUnless := command.(commands.Unless)
						cnsCommand := cnsUnless.Command.(*commands.CreateNamespace)
						cnsCommand.Result = sandboxNS
					case 3:
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
