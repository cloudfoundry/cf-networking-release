package manager_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"code.cloudfoundry.org/garden"
	types040 "github.com/containernetworking/cni/pkg/types/040"

	"code.cloudfoundry.org/garden-external-networker/fakes"
	"code.cloudfoundry.org/garden-external-networker/manager"

	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", func() {
	var (
		mgr                   *manager.Manager
		upInputs              manager.UpInputs
		cniController         *fakes.CNIController
		proxyRedirect         *fakes.ProxyRedirect
		mounter               *fakes.Mounter
		gardenProperties      map[string]interface{}
		expectedMetadata      map[string]interface{}
		expectedLegacyNetConf map[string]interface{}
		portAllocator         *fakes.PortAllocator
		netInRules            []garden.NetIn
		netOutRules           []garden.NetOutRule
		logger                *bytes.Buffer
		containerHandle       string
		cniUpResult           *types040.Result
	)

	BeforeEach(func() {
		logger = &bytes.Buffer{}
		containerHandle = "some-container-handle"
		interfaceIndex := 1
		mounter = &fakes.Mounter{}
		cniController = &fakes.CNIController{}
		portAllocator = &fakes.PortAllocator{}
		proxyRedirect = &fakes.ProxyRedirect{}
		cniUpResult = &types040.Result{
			Interfaces: []*types040.Interface{
				{
					Name: "s-010133166033",
					Mac:  "aa:aa:0a:85:a6:21",
				},
				{
					Name:    "eth0",
					Mac:     "aa:aa:0a:85:a6:21",
					Sandbox: "/var/vcap/data/garden-cni/container-netns/check-341ecc13-9e29-4845-6402-f59e8b13603b",
				},
			},
			IPs: []*types040.IPConfig{
				{
					Version:   "4",
					Interface: &interfaceIndex,
					Address: net.IPNet{
						IP:   net.ParseIP("169.254.1.2"),
						Mask: net.IPv4Mask(255, 255, 255, 0),
					},
				},
			},
			DNS: types.DNS{
				Nameservers: []string{"8.8.8.8"},
			},
		}

		cniController.UpReturns(cniUpResult, nil)

		mgr = &manager.Manager{
			Logger:        logger,
			CNIController: cniController,
			Mounter:       mounter,
			BindMountRoot: "some/fake/path",
			PortAllocator: portAllocator,
			SearchDomains: []string{"pivotal.io", "foo.bar", "baz.me"},
			ProxyRedirect: proxyRedirect,
		}

		netInRules = []garden.NetIn{
			{
				HostPort:      12345,
				ContainerPort: 7000,
			},
			{
				HostPort:      23456,
				ContainerPort: 7001,
			},
		}
		netOutRules = []garden.NetOutRule{
			garden.NetOutRule{
				Protocol: garden.ProtocolTCP,
				Networks: []garden.IPRange{
					{Start: net.ParseIP("1.1.1.1"), End: net.ParseIP("2.2.2.2")},
					{Start: net.ParseIP("3.3.3.3"), End: net.ParseIP("4.4.4.4")},
				},
				Ports: []garden.PortRange{
					{Start: 9000, End: 9999},
					{Start: 1111, End: 2222},
				},
			},
		}
		gardenProperties = map[string]interface{}{"policy_group_id": "some-group-id"}

		upInputs = manager.UpInputs{
			Pid:        42,
			Properties: gardenProperties,
			NetOut:     netOutRules,
			NetIn:      netInRules,
		}

		expectedMetadata = map[string]interface{}{"policy_group_id": "some-group-id"}
		expectedLegacyNetConf = map[string]interface{}{
			"portMappings": netInRules,
			"netOutRules":  netOutRules,
		}
	})

	Describe("Up", func() {
		It("ensures that the netNS is mounted via /proc/pid/ns/net to the provided path", func() {
			_, err := mgr.Up(containerHandle, upInputs)
			Expect(err).NotTo(HaveOccurred())

			Expect(mounter.IdempotentlyMountCallCount()).To(Equal(1))
			source, target := mounter.IdempotentlyMountArgsForCall(0)
			Expect(source).To(Equal("/proc/42/ns/net"))
			Expect(target).To(Equal(filepath.Join("some", "fake", "path", containerHandle)))
		})

		Context("when the pid is 0", func() {
			BeforeEach(func() {
				upInputs.Pid = 0
			})

			It("ensures that the netNS is mounted via /proc/self/fd/3 to the provided path", func() {
				_, err := mgr.Up(containerHandle, upInputs)
				Expect(err).NotTo(HaveOccurred())

				Expect(mounter.IdempotentlyMountCallCount()).To(Equal(1))
				source, target := mounter.IdempotentlyMountArgsForCall(0)
				Expect(source).To(Equal("/proc/self/fd/3"))
				Expect(target).To(Equal(filepath.Join("some", "fake", "path", containerHandle)))
			})
		})

		It("should create proxy redirect rules in the container namespace", func() {
			_, err := mgr.Up(containerHandle, upInputs)
			Expect(err).NotTo(HaveOccurred())

			Expect(proxyRedirect.ApplyCallCount()).To(Equal(1))
			actualContainerNamespace := proxyRedirect.ApplyArgsForCall(0)
			Expect(actualContainerNamespace).To(Equal(filepath.Join("some", "fake", "path", containerHandle)))
		})

		It("should return the IP address in the CNI result as a property", func() {
			out, err := mgr.Up(containerHandle, upInputs)
			Expect(err).NotTo(HaveOccurred())

			Expect(out.Properties.ContainerIP).To(Equal("169.254.1.2"))
			Expect(out.Properties.DeprecatedHostIP).To(Equal("255.255.255.255"))
		})

		It("should return the interface in the CNI result as a property", func() {
			out, err := mgr.Up(containerHandle, upInputs)
			Expect(err).NotTo(HaveOccurred())

			Expect(out.Properties.Interface).To(Equal("eth0"))
		})

		Context("when there is no IPv4 address in the CNI result", func() {
			BeforeEach(func() {
				cniUpResult.IPs = []*types040.IPConfig{
					{
						Version: "6",
						Address: net.IPNet{
							IP: net.ParseIP("2001:db8::68"),
						},
					},
				}
			})

			It("should return an error", func() {
				_, err := mgr.Up(containerHandle, upInputs)
				Expect(err).To(MatchError("expected an IPv4 address in the CNI result"))
			})
		})

		Context("when there is no pointer to a container interface", func() {
			BeforeEach(func() {
				cniUpResult.IPs[0].Interface = nil
			})

			It("should return empty interface in the CNI result which must be omitted after marshalling", func() {
				out, err := mgr.Up(containerHandle, upInputs)

				Expect(err).NotTo(HaveOccurred())
				Expect(out.Properties.Interface).To(BeEmpty())

				outMarshalled, err := json.Marshal(out)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(outMarshalled)).NotTo(ContainSubstring("garden.network.interface"))
			})
		})

		Context("when the interface reference is invalid", func() {
			BeforeEach(func() {
				interfaceIndexOutOfRange := 2
				cniUpResult.IPs[0].Interface = &interfaceIndexOutOfRange
			})

			It("should return an error", func() {
				_, err := mgr.Up(containerHandle, upInputs)
				Expect(err).To(MatchError("no corresponding interface found, interface index: 2, number of interfaces: 2"))
			})
		})

		It("should return the DNS nameservers info as a separate key in the up output", func() {
			out, err := mgr.Up(containerHandle, upInputs)
			Expect(err).NotTo(HaveOccurred())

			Expect(out.DNSServers).To(Equal([]string{"8.8.8.8"}))
		})

		It("should return the search domains info as a separate key in the up output", func() {
			out, err := mgr.Up(containerHandle, upInputs)
			Expect(err).NotTo(HaveOccurred())

			Expect(out.SearchDomains).To(Equal([]string{"pivotal.io", "foo.bar", "baz.me"}))
		})

		It("should call CNI Up, passing in the bind-mounted path to the net ns", func() {
			_, err := mgr.Up(containerHandle, upInputs)
			Expect(err).NotTo(HaveOccurred())

			Expect(cniController.UpCallCount()).To(Equal(1))
			namespacePath, handle, metadata, legacyNetConf := cniController.UpArgsForCall(0)
			Expect(namespacePath).To(Equal(filepath.Join("some", "fake", "path", containerHandle)))
			Expect(handle).To(Equal(containerHandle))
			Expect(metadata).To(Equal(expectedMetadata))
			Expect(legacyNetConf).To(Equal(expectedLegacyNetConf))
		})

		It("returns the mapped ports", func() {
			out, err := mgr.Up(containerHandle, upInputs)
			Expect(err).NotTo(HaveOccurred())

			Expect(out.Properties.MappedPorts).To(MatchJSON(`[
				{"HostPort": 12345, "ContainerPort": 7000},
				{"HostPort": 23456, "ContainerPort": 7001}
			]`))
		})

		Context("when the host port is 0", func() {
			BeforeEach(func() {
				netInRules = []garden.NetIn{
					{
						HostPort:      0,
						ContainerPort: 7000,
					},
				}
				upInputs.NetIn = netInRules
				portAllocator.AllocatePortReturns(1234, nil)
			})
			It("allocates a port", func() {
				out, err := mgr.Up(containerHandle, upInputs)

				Expect(err).NotTo(HaveOccurred())

				Expect(portAllocator.AllocatePortCallCount()).To(Equal(1))
				handle, port := portAllocator.AllocatePortArgsForCall(0)
				Expect(handle).To(Equal("some-container-handle"))
				Expect(port).To(Equal(0))

				Expect(cniController.UpCallCount()).To(Equal(1))
				_, handle, _, legacyNetConf := cniController.UpArgsForCall(0)
				Expect(handle).To(Equal(containerHandle))
				Expect(legacyNetConf).To(HaveKeyWithValue("portMappings", []garden.NetIn{
					{
						HostPort:      1234,
						ContainerPort: 7000,
					},
				}))

				Expect(out.Properties.MappedPorts).To(MatchJSON(`[{"HostPort": 1234, "ContainerPort": 7000}]`))
			})
		})

		Context("when the port allocation fails", func() {
			BeforeEach(func() {
				netInRules = []garden.NetIn{
					{
						HostPort:      0,
						ContainerPort: 7000,
					},
				}
				upInputs.NetIn = netInRules
				portAllocator.AllocatePortReturns(0, errors.New("banana"))
			})
			It("returns an error", func() {
				_, err := mgr.Up(containerHandle, upInputs)

				Expect(err).To(MatchError("allocating port: banana"))
			})
		})

		Context("when CNI up returns a nil result", func() {
			BeforeEach(func() {
				cniController.UpReturns(nil, nil)
			})
			It("returns an error", func() {
				_, err := mgr.Up("container-handle", upInputs)
				Expect(err).To(MatchError("cni up failed: no ip allocated"))
			})
		})

		Context("when missing args", func() {
			It("should return a friendly error", func() {
				_, err := mgr.Up(containerHandle, manager.UpInputs{
					Properties: gardenProperties,
					NetOut:     netOutRules,
					NetIn:      netInRules,
				})
				Expect(err).To(MatchError("up missing container handle"))

				_, err = mgr.Up("", upInputs)
				Expect(err).To(MatchError("up missing container handle"))
			})
		})

		Context("when missing the garden properties are nil", func() {
			It("should not complain", func() {
				_, err := mgr.Up(containerHandle, manager.UpInputs{Pid: 42, Properties: nil})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the encoded garden properties is an empty hash", func() {
			It("should still call CNI and the netman agent", func() {
				props := make(map[string]interface{})
				_, err := mgr.Up(containerHandle, manager.UpInputs{Pid: 42, Properties: props})
				Expect(err).NotTo(HaveOccurred())

				Expect(cniController.UpCallCount()).To(Equal(1))
				Expect(mounter.IdempotentlyMountCallCount()).To(Equal(1))
			})
		})

		Context("when the mounter fails", func() {
			It("should return the error", func() {
				mounter.IdempotentlyMountReturns(errors.New("boom"))
				_, err := mgr.Up(containerHandle, upInputs)
				Expect(err).To(MatchError(fmt.Sprintf("failed mounting /proc/42/ns/net to %s: boom",
					filepath.Join("some", "fake", "path", containerHandle))))
			})
		})

		Context("when the cni Up fails", func() {
			It("should return the error", func() {
				cniController.UpReturns(nil, errors.New("bang"))
				_, err := mgr.Up(containerHandle, upInputs)
				Expect(err).To(MatchError("cni up failed: bang"))
			})
		})

		Context("when the proxy redirect fails", func() {
			It("should return the error", func() {
				proxyRedirect.ApplyReturns(errors.New("bang"))
				_, err := mgr.Up(containerHandle, upInputs)
				Expect(err).To(MatchError("proxy redirect apply: bang"))
			})
		})
	})

	Describe("Down", func() {
		It("should ensure that the netNS is unmounted", func() {
			Expect(mgr.Down(containerHandle)).To(Succeed())
			Expect(mounter.RemoveMountCallCount()).To(Equal(1))

			Expect(mounter.RemoveMountArgsForCall(0)).To(Equal(filepath.Join("some", "fake", "path", containerHandle)))
		})

		It("should call CNI Down, passing in the bind-mounted path to the net ns", func() {
			Expect(mgr.Down(containerHandle)).To(Succeed())
			Expect(cniController.DownCallCount()).To(Equal(1))
			namespacePath, handle := cniController.DownArgsForCall(0)
			Expect(namespacePath).To(Equal(filepath.Join("some", "fake", "path", containerHandle)))
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
				Expect(logger.String()).To(ContainSubstring(fmt.Sprintf("removing bind mount %s: boom\n", filepath.Join("some", "fake", "path", containerHandle))))

				Expect(portAllocator.ReleaseAllPortsCallCount()).To(Equal(1))
			})
		})

		Context("when releasing all ports fails", func() {
			It("logs the error and succeeds", func() {
				portAllocator.ReleaseAllPortsReturns(errors.New("potato"))
				err := mgr.Down(containerHandle)
				Expect(err).NotTo(HaveOccurred())
				Expect(logger.String()).To(ContainSubstring("releasing ports: potato\n"))
			})
		})

	})
})
