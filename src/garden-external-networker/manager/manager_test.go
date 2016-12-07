package manager_test

import (
	"errors"
	"fmt"
	"net"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager/lagertest"

	"garden-external-networker/fakes"
	"garden-external-networker/manager"

	lib_fakes "lib/fakes"

	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Manager", func() {
	var (
		mgr                     *manager.Manager
		cniController           *fakes.CNIController
		mounter                 *fakes.Mounter
		gardenProperties        map[string]string
		expectedExtraProperties map[string]string
		portAllocator           *fakes.PortAllocator
		netInProvider           *fakes.NetInProvider
		netOutProvider          *fakes.NetOutProvider
		ipTables                *lib_fakes.IPTables
		logger                  *lagertest.TestLogger
		containerHandle         string
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		containerHandle = "some-container-handle"
		mounter = &fakes.Mounter{}
		cniController = &fakes.CNIController{}
		ipTables = &lib_fakes.IPTables{}
		portAllocator = &fakes.PortAllocator{}

		netInProvider = &fakes.NetInProvider{}
		netOutProvider = &fakes.NetOutProvider{}

		cniController.UpReturns(&types.Result{
			IP4: &types.IPConfig{
				IP: net.IPNet{
					IP:   net.ParseIP("169.254.1.2"),
					Mask: net.IPv4Mask(255, 255, 255, 0),
				},
			},
		}, nil)
		mgr = &manager.Manager{
			Logger:         logger,
			CNIController:  cniController,
			Mounter:        mounter,
			BindMountRoot:  "/some/fake/path",
			OverlayNetwork: "10.255.0.0/16",
			PortAllocator:  portAllocator,
			NetInProvider:  netInProvider,
			NetOutProvider: netOutProvider,
		}
		gardenProperties = map[string]string{"policy_group_id": "some-group-id"}
		expectedExtraProperties = map[string]string{"policy_group_id": "some-group-id"}
	})

	Describe("Up", func() {
		It("should ensure that the netNS is mounted to the provided path", func() {
			_, err := mgr.Up(containerHandle, manager.UpInputs{
				Pid:        42,
				Properties: gardenProperties,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(mounter.IdempotentlyMountCallCount()).To(Equal(1))
			source, target := mounter.IdempotentlyMountArgsForCall(0)
			Expect(source).To(Equal("/proc/42/ns/net"))
			Expect(target).To(Equal(fmt.Sprintf("/some/fake/path/%s", containerHandle)))
		})

		It("should return the IP address in the CNI result as a property", func() {
			out, err := mgr.Up(containerHandle, manager.UpInputs{Pid: 42, Properties: gardenProperties})
			Expect(err).NotTo(HaveOccurred())

			Expect(out.Properties.ContainerIP).To(Equal("169.254.1.2"))
			Expect(out.Properties.DeprecatedHostIP).To(Equal("255.255.255.255"))
		})

		It("should call CNI Up, passing in the bind-mounted path to the net ns", func() {
			_, err := mgr.Up(containerHandle, manager.UpInputs{Pid: 42, Properties: gardenProperties})
			Expect(err).NotTo(HaveOccurred())

			Expect(cniController.UpCallCount()).To(Equal(1))
			namespacePath, handle, properties := cniController.UpArgsForCall(0)
			Expect(namespacePath).To(Equal(fmt.Sprintf("/some/fake/path/%s", containerHandle)))
			Expect(handle).To(Equal(containerHandle))
			Expect(properties).To(Equal(expectedExtraProperties))
		})

		Context("when CNI up returns a nil result", func() {
			BeforeEach(func() {
				cniController.UpReturns(nil, nil)
			})
			It("returns an error", func() {
				_, err := mgr.Up("container-handle", manager.UpInputs{Pid: 42, Properties: gardenProperties})
				Expect(err).To(MatchError("cni up failed: no ip allocated"))
			})
		})

		Context("when initializing netout fails", func() {
			BeforeEach(func() {
				netInProvider.InitializeReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := mgr.Up("container-handle", manager.UpInputs{Pid: 42, Properties: gardenProperties})
				Expect(err).To(MatchError("initialize iptables for netin: banana"))
			})
		})

		Context("when creating the chain fails", func() {
			BeforeEach(func() {
				netOutProvider.InitializeReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := mgr.Up("container-handle", manager.UpInputs{Pid: 42, Properties: gardenProperties})
				Expect(err).To(MatchError("initialize net out: banana"))
			})
		})

		Context("when missing args", func() {
			It("should return a friendly error", func() {
				_, err := mgr.Up(containerHandle, manager.UpInputs{Pid: 0, Properties: gardenProperties})
				Expect(err).To(MatchError("up missing pid"))

				_, err = mgr.Up("", manager.UpInputs{Pid: 42, Properties: gardenProperties})
				Expect(err).To(MatchError("up missing container handle"))
			})
		})

		Context("when missing the garden properties are nil", func() {
			It("should not complain", func() {
				var props map[string]string
				_, err := mgr.Up(containerHandle, manager.UpInputs{Pid: 42, Properties: props})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the encoded garden properties is an empty hash", func() {
			It("should still call CNI and the netman agent", func() {
				props := make(map[string]string)
				_, err := mgr.Up(containerHandle, manager.UpInputs{Pid: 42, Properties: props})
				Expect(err).NotTo(HaveOccurred())

				Expect(cniController.UpCallCount()).To(Equal(1))
				Expect(mounter.IdempotentlyMountCallCount()).To(Equal(1))
			})
		})

		Context("when the mounter fails", func() {
			It("should return the error", func() {
				mounter.IdempotentlyMountReturns(errors.New("boom"))
				_, err := mgr.Up(containerHandle, manager.UpInputs{Pid: 42, Properties: gardenProperties})
				Expect(err).To(MatchError("failed mounting /proc/42/ns/net to /some/fake/path/some-container-handle: boom"))
			})
		})

		Context("when the cni Up fails", func() {
			It("should return the error", func() {
				cniController.UpReturns(nil, errors.New("bang"))
				_, err := mgr.Up(containerHandle, manager.UpInputs{Pid: 42, Properties: gardenProperties})
				Expect(err).To(MatchError("cni up failed: bang"))
			})
		})
	})

	Describe("Down", func() {
		It("should ensure that the netNS is unmounted", func() {
			Expect(mgr.Down(containerHandle)).To(Succeed())
			Expect(mounter.RemoveMountCallCount()).To(Equal(1))

			Expect(mounter.RemoveMountArgsForCall(0)).To(Equal("/some/fake/path/some-container-handle"))
		})

		It("should call CNI Down, passing in the bind-mounted path to the net ns", func() {
			Expect(mgr.Down(containerHandle)).To(Succeed())
			Expect(cniController.DownCallCount()).To(Equal(1))
			namespacePath, handle := cniController.DownArgsForCall(0)
			Expect(namespacePath).To(Equal("/some/fake/path/some-container-handle"))
			Expect(handle).To(Equal(containerHandle))
		})

		It("should release all ports which were allocated for the container", func() {
			Expect(mgr.Down(containerHandle)).To(Succeed())
			Expect(portAllocator.ReleaseAllPortsCallCount()).To(Equal(1))
			Expect(portAllocator.ReleaseAllPortsArgsForCall(0)).To(Equal(containerHandle))
		})

		Context("when encodedGardenProperties is empty", func() {
			It("should call CNI", func() {
				err := mgr.Down(containerHandle)
				Expect(err).NotTo(HaveOccurred())
				Expect(cniController.DownCallCount()).To(Equal(1))
				Expect(mounter.RemoveMountCallCount()).To(Equal(1))
			})
		})

		Context("when missing args", func() {
			It("should return a friendly error", func() {
				err := mgr.Down("")
				Expect(err).To(MatchError("down missing container handle"))
			})
		})

		Context("when the cni Down fails", func() {
			It("should return the error", func() {
				cniController.DownReturns(errors.New("bang"))
				err := mgr.Down(containerHandle)
				Expect(err).To(MatchError("cni down: bang"))
			})
		})

		Context("when the mounter fails", func() {
			It("logs the error and continues cleanup", func() {
				mounter.RemoveMountReturns(errors.New("boom"))
				err := mgr.Down(containerHandle)
				Expect(err).NotTo(HaveOccurred())
				Expect(logger).To(gbytes.Say(`removing mount.*bind mount path.*/some/fake/path/some-container-handle.*boom`))

				Expect(portAllocator.ReleaseAllPortsCallCount()).To(Equal(1))
			})
		})

		Context("when the net out cleanup fails", func() {
			It("logs the error and continues cleanup", func() {
				netOutProvider.CleanupReturns(errors.New("potato"))
				err := mgr.Down(containerHandle)
				Expect(err).NotTo(HaveOccurred())
				Expect(logger).To(gbytes.Say(`net out cleanup.*potato`))

				Expect(portAllocator.ReleaseAllPortsCallCount()).To(Equal(1))
			})
		})

		Context("when the net in cleanup fails", func() {
			It("logs the error and continues cleanup", func() {
				netInProvider.CleanupReturns(errors.New("potato"))
				err := mgr.Down(containerHandle)
				Expect(err).NotTo(HaveOccurred())
				Expect(logger).To(gbytes.Say(`net in cleanup.*potato`))

				Expect(portAllocator.ReleaseAllPortsCallCount()).To(Equal(1))
			})
		})

		Context("when releasing all ports fails", func() {
			It("logs the error and succeeds", func() {
				portAllocator.ReleaseAllPortsReturns(errors.New("potato"))
				err := mgr.Down(containerHandle)
				Expect(err).NotTo(HaveOccurred())
				Expect(logger).To(gbytes.Say(`releasing ports.*potato`))
			})
		})

	})

	Describe("NetOut", func() {
		var (
			netOutInputs manager.NetOutInputs
		)
		BeforeEach(func() {
			netOutRule := garden.NetOutRule{
				Protocol: garden.ProtocolTCP,
				Networks: []garden.IPRange{
					{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
					{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
				},
				Ports: []garden.PortRange{
					{Start: 9000, End: 9999},
					{Start: 1111, End: 2222},
				},
			}
			netOutInputs = manager.NetOutInputs{
				ContainerIP: "1.2.3.4",
				NetOutRule:  netOutRule,
			}
		})

		It("delegates to netout provider for netout calls", func() {
			err := mgr.NetOut("some-handle", netOutInputs)
			Expect(err).NotTo(HaveOccurred())
			Expect(netOutProvider.InsertRuleCallCount()).To(Equal(1))
			handle, rule, containerIP := netOutProvider.InsertRuleArgsForCall(0)
			Expect(handle).To(Equal("some-handle"))
			Expect(rule).To(Equal(netOutInputs.NetOutRule))
			Expect(containerIP).To(Equal(netOutInputs.ContainerIP))
		})

		Context("when inserting the rule fails", func() {
			BeforeEach(func() {
				netOutProvider.InsertRuleReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				err := mgr.NetOut("some-handle", netOutInputs)
				Expect(err).To(MatchError("banana"))
			})
		})
	})

	Describe("NetIn", func() {
		var input manager.NetInInputs
		BeforeEach(func() {
			input = manager.NetInInputs{
				HostIP:        "1.2.3.4",
				HostPort:      0,
				ContainerIP:   "10.0.0.2",
				ContainerPort: 8888,
			}
			portAllocator.AllocatePortReturns(1234, nil)
		})

		It("allocates a port and calls netin provider", func() {
			_, err := mgr.NetIn(containerHandle, input)
			Expect(err).NotTo(HaveOccurred())

			Expect(netInProvider.AddRuleCallCount()).To(Equal(1))
			handle, hostPort, containerPort, hostIP, containerIP := netInProvider.AddRuleArgsForCall(0)
			Expect(handle).To(Equal(containerHandle))
			Expect(hostPort).To(Equal(1234))
			Expect(containerPort).To(Equal(input.ContainerPort))
			Expect(hostIP).To(Equal(input.HostIP))
			Expect(containerIP).To(Equal(input.ContainerIP))
		})

		BeforeEach(func() {
			input = manager.NetInInputs{
				HostIP:        "1.2.3.4",
				HostPort:      1111,
				ContainerIP:   "10.0.0.2",
				ContainerPort: 8888,
			}
		})

		It("uses the specified port", func() {
			output, err := mgr.NetIn(containerHandle, input)
			Expect(err).NotTo(HaveOccurred())

			Expect(output).To(Equal(&manager.NetInOutputs{
				HostPort:      1234,
				ContainerPort: 8888,
			}))
		})

		Context("when no container port is specified", func() {
			BeforeEach(func() {
				input = manager.NetInInputs{
					HostIP:        "1.2.3.4",
					HostPort:      1111,
					ContainerIP:   "10.0.0.2",
					ContainerPort: 0,
				}
			})

			It("uses the specified external port", func() {
				output, err := mgr.NetIn(containerHandle, input)
				Expect(err).NotTo(HaveOccurred())

				Expect(output).To(Equal(&manager.NetInOutputs{
					HostPort:      1234,
					ContainerPort: 1234,
				}))
			})
		})

		Context("when the port allocator errors", func() {
			BeforeEach(func() {
				portAllocator.AllocatePortReturns(0, errors.New("potato"))
			})
			It("wraps and returns the error", func() {
				_, err := mgr.NetIn(containerHandle, input)
				Expect(err).To(MatchError("allocate port: potato"))
			})
		})

		Context("when the add rule errors", func() {
			BeforeEach(func() {
				netInProvider.AddRuleReturns(errors.New("potato"))
			})
			It("wraps and returns the error", func() {
				_, err := mgr.NetIn(containerHandle, input)
				Expect(err).To(MatchError("add rule: potato"))
			})
		})
	})

	Describe("BulkNetOut", func() {
		var (
			bulkNetOutInputs manager.BulkNetOutInputs
		)
		BeforeEach(func() {
			netOutRules := []garden.NetOutRule{
				garden.NetOutRule{
					Protocol: garden.ProtocolTCP,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
					},
					Ports: []garden.PortRange{
						{Start: 9000, End: 9999},
					},
				},
				garden.NetOutRule{
					Protocol: garden.ProtocolTCP,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("2.2.2.2"), End: net.ParseIP("3.3.3.3")},
					},
					Ports: []garden.PortRange{
						{Start: 9000, End: 9999},
					},
				},
			}
			bulkNetOutInputs = manager.BulkNetOutInputs{
				ContainerIP: "1.2.3.4",
				NetOutRules: netOutRules,
			}
		})

		It("delegates to netout provider for bulk netout calls", func() {
			err := mgr.BulkNetOut("some-handle", bulkNetOutInputs)
			Expect(err).NotTo(HaveOccurred())
			Expect(netOutProvider.BulkInsertRulesCallCount()).To(Equal(1))

			handle, rules, containerIP := netOutProvider.BulkInsertRulesArgsForCall(0)
			Expect(handle).To(Equal("some-handle"))
			Expect(rules).To(Equal(bulkNetOutInputs.NetOutRules))
			Expect(containerIP).To(Equal(bulkNetOutInputs.ContainerIP))

		})

		Context("when inserting the rule fails", func() {
			BeforeEach(func() {
				netOutProvider.BulkInsertRulesReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				err := mgr.BulkNetOut("some-handle", bulkNetOutInputs)
				Expect(err).To(MatchError("insert rule: banana"))
			})
		})
	})

})
