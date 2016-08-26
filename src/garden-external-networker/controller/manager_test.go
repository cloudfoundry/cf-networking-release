package controller_test

import (
	"errors"
	"net"

	"code.cloudfoundry.org/lager/lagertest"

	"garden-external-networker/controller"
	"garden-external-networker/fakes"

	lib_fakes "lib/fakes"

	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", func() {
	var (
		manager                 *controller.Manager
		cniController           *fakes.CNIController
		mounter                 *fakes.Mounter
		encodedGardenProperties string
		expectedExtraProperties map[string]string
		ipTables                *lib_fakes.IPTables
		logger                  *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		mounter = &fakes.Mounter{}
		cniController = &fakes.CNIController{}
		ipTables = &lib_fakes.IPTables{}
		cniController.UpReturns(&types.Result{
			IP4: &types.IPConfig{
				IP: net.IPNet{
					IP:   net.ParseIP("169.254.1.2"),
					Mask: net.IPv4Mask(255, 255, 255, 0),
				},
			},
		}, nil)
		manager = &controller.Manager{
			Logger:         logger,
			CNIController:  cniController,
			Mounter:        mounter,
			BindMountRoot:  "/some/fake/path",
			IPTables:       ipTables,
			OverlayNetwork: "10.255.0.0/16",
		}
		encodedGardenProperties = `{ "app_id": "some-group-id" }`
		expectedExtraProperties = map[string]string{"app_id": "some-group-id"}
	})

	Describe("Up", func() {
		It("should ensure that the netNS is mounted to the provided path", func() {
			_, err := manager.Up(42, "some-container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(mounter.IdempotentlyMountCallCount()).To(Equal(1))
			source, target := mounter.IdempotentlyMountArgsForCall(0)
			Expect(source).To(Equal("/proc/42/ns/net"))
			Expect(target).To(Equal("/some/fake/path/some-container-handle"))
		})

		It("should return the IP address in the CNI result as a property", func() {
			properties, err := manager.Up(42, "some-container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(properties.ContainerIP).To(Equal(net.ParseIP("169.254.1.2")))
			Expect(properties.DeprecatedHostIP).To(Equal(net.ParseIP("255.255.255.255")))
		})

		It("should call CNI Up, passing in the bind-mounted path to the net ns", func() {
			_, err := manager.Up(42, "some-container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(cniController.UpCallCount()).To(Equal(1))
			namespacePath, handle, properties := cniController.UpArgsForCall(0)
			Expect(namespacePath).To(Equal("/some/fake/path/some-container-handle"))
			Expect(handle).To(Equal("some-container-handle"))
			Expect(properties).To(Equal(expectedExtraProperties))
		})

		Context("when the chain name is longer than 28 characters", func() {
			It("truncates the name", func() {
				_, err := manager.Up(42, "some-very-long-container-handle", encodedGardenProperties)
				Expect(err).NotTo(HaveOccurred())

				Expect(ipTables.NewChainCallCount()).To(Equal(1))
				_, chain := ipTables.NewChainArgsForCall(0)
				Expect(chain).To(Equal("netout--some-very-long-conta"))
			})
		})

		It("should create the container's chain by prepending netout to the handle", func() {
			_, err := manager.Up(42, "container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.NewChainCallCount()).To(Equal(1))
			table, chain := ipTables.NewChainArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netout--container-handle"))

			Expect(ipTables.InsertCallCount()).To(Equal(1))
			table, chain, pos, rulespec := ipTables.InsertArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("FORWARD"))
			Expect(pos).To(Equal(1))
			Expect(rulespec).To(Equal([]string{"--jump", "netout--container-handle"}))
		})

		It("should write the default NetOut rules", func() {
			_, err := manager.Up(42, "container-handle", encodedGardenProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.AppendUniqueCallCount()).To(Equal(2))
			table, chain, rulespec := ipTables.AppendUniqueArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netout--container-handle"))
			Expect(rulespec).To(Equal([]string{"-s", "169.254.1.2",
				"!", "-d", "10.255.0.0/16",
				"-m", "state", "--state", "RELATED,ESTABLISHED",
				"--jump", "RETURN"}))

			table, chain, rulespec = ipTables.AppendUniqueArgsForCall(1)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal("netout--container-handle"))
			Expect(rulespec).To(Equal([]string{"-s", "169.254.1.2",
				"!", "-d", "10.255.0.0/16",
				"--jump", "REJECT",
				"--reject-with", "icmp-port-unreachable"}))
		})

		Context("when inserting fails", func() {
			BeforeEach(func() {
				ipTables.InsertReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := manager.Up(42, "container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("inserting rule: banana"))
			})
		})

		Context("when creating the chain fails", func() {
			BeforeEach(func() {
				ipTables.NewChainReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := manager.Up(42, "container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("creating chain: banana"))
			})
		})

		Context("when appending a rule fails", func() {
			BeforeEach(func() {
				ipTables.AppendUniqueReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := manager.Up(42, "container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("appending rule: banana"))
			})
		})

		Context("when missing args", func() {
			It("should return a friendly error", func() {
				_, err := manager.Up(0, "some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("up missing pid"))

				_, err = manager.Up(42, "", encodedGardenProperties)
				Expect(err).To(MatchError("up missing container handle"))
			})
		})

		Context("when missing the encoded garden properties", func() {
			It("should not complain", func() {
				_, err := manager.Up(42, "some-container-handle", "")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the encoded garden properties is an empty hash", func() {
			It("should still call CNI and the netman agent", func() {
				_, err := manager.Up(42, "some-container-handle", "{}")
				Expect(err).NotTo(HaveOccurred())

				Expect(cniController.UpCallCount()).To(Equal(1))
				Expect(mounter.IdempotentlyMountCallCount()).To(Equal(1))
			})
		})

		Context("when unmarshaling the encoded garden properties fails", func() {
			It("returns the error", func() {
				_, err := manager.Up(42, "some-container-handle", "%%%%")
				Expect(err).To(MatchError(ContainSubstring("unmarshal garden properties: invalid character")))
			})
		})

		Context("when the mounter fails", func() {
			It("should return the error", func() {
				mounter.IdempotentlyMountReturns(errors.New("boom"))
				_, err := manager.Up(42, "some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("failed mounting /proc/42/ns/net to /some/fake/path/some-container-handle: boom"))
			})
		})

		Context("when the cni Up fails", func() {
			It("should return the error", func() {
				cniController.UpReturns(nil, errors.New("bang"))
				_, err := manager.Up(42, "some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("cni up failed: bang"))
			})
		})
	})

	Describe("Down", func() {
		It("should ensure that the netNS is unmounted", func() {
			Expect(manager.Down("some-container-handle", encodedGardenProperties)).To(Succeed())
			Expect(mounter.RemoveMountCallCount()).To(Equal(1))

			Expect(mounter.RemoveMountArgsForCall(0)).To(Equal("/some/fake/path/some-container-handle"))
		})

		It("should call CNI Down, passing in the bind-mounted path to the net ns", func() {
			Expect(manager.Down("some-container-handle", encodedGardenProperties)).To(Succeed())
			Expect(cniController.DownCallCount()).To(Equal(1))
			namespacePath, handle, spec := cniController.DownArgsForCall(0)
			Expect(namespacePath).To(Equal("/some/fake/path/some-container-handle"))
			Expect(handle).To(Equal("some-container-handle"))
			Expect(spec).To(Equal(expectedExtraProperties))
		})

		Context("when encodedGardenProperties is empty", func() {
			It("should call CNI", func() {
				err := manager.Down("some-container-handle", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(cniController.DownCallCount()).To(Equal(1))
				Expect(mounter.RemoveMountCallCount()).To(Equal(1))
			})
		})

		Context("when missing args", func() {
			It("should return a friendly error", func() {
				err := manager.Down("", "")
				Expect(err).To(MatchError("down missing container handle"))
			})
		})

		Context("when the mounter fails", func() {
			It("should return the error", func() {
				mounter.RemoveMountReturns(errors.New("boom"))
				err := manager.Down("some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("failed removing mount /some/fake/path/some-container-handle: boom"))
			})
		})

		Context("when the cni Down fails", func() {
			It("should return the error", func() {
				cniController.DownReturns(errors.New("bang"))
				err := manager.Down("some-container-handle", encodedGardenProperties)
				Expect(err).To(MatchError("cni down failed: bang"))
			})
		})
	})

	Describe("NetOut", func() {
		var netOutProperties string
		BeforeEach(func() {
			netOutProperties = `{
				"container_ip":"1.2.3.4",
				"netout_rule":{
					"protocol":1,
					"networks":[{"start":"1.1.1.1","end":"2.2.2.2"},{"start":"3.3.3.3","end":"4.4.4.4"}],
					"ports":[{"start":9000,"end":9999},{"start":1111,"end":2222}]
				}
			}`
		})
		It("prepends allow rules to the container's netout chain", func() {
			err := manager.NetOut("some-handle", netOutProperties)
			Expect(err).NotTo(HaveOccurred())

			Expect(ipTables.InsertCallCount()).To(Equal(4))
			writtenRules := [][]string{}
			for i := 0; i < 4; i++ {
				table, chain, pos, rulespec := ipTables.InsertArgsForCall(i)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal("netout--some-handle"))
				Expect(pos).To(Equal(1))
				writtenRules = append(writtenRules, rulespec)
			}
			Expect(writtenRules).To(ConsistOf(
				[]string{"--source", "1.2.3.4",
					"-m", "iprange", "-p", "tcp",
					"--dst-range", "1.1.1.1-2.2.2.2",
					"-m", "tcp", "--destination-port", "9000:9999",
					"--jump", "RETURN"},
				[]string{"--source", "1.2.3.4",
					"-m", "iprange", "-p", "tcp",
					"--dst-range", "1.1.1.1-2.2.2.2",
					"-m", "tcp", "--destination-port", "1111:2222",
					"--jump", "RETURN"},
				[]string{"--source", "1.2.3.4",
					"-m", "iprange", "-p", "tcp",
					"--dst-range", "3.3.3.3-4.4.4.4",
					"-m", "tcp", "--destination-port", "9000:9999",
					"--jump", "RETURN"},
				[]string{"--source", "1.2.3.4",
					"-m", "iprange", "-p", "tcp",
					"--dst-range", "3.3.3.3-4.4.4.4",
					"-m", "tcp", "--destination-port", "1111:2222",
					"--jump", "RETURN"},
			))
		})
		Context("when the handle is over 28 characters", func() {
			It("truncates the handle", func() {
				err := manager.NetOut("a-very-long-container-handle", netOutProperties)
				Expect(err).NotTo(HaveOccurred())
				Expect(ipTables.InsertCallCount()).To(Equal(4))
				for i := 0; i < 4; i++ {
					_, chain, _, _ := ipTables.InsertArgsForCall(i)
					Expect(chain).To(Equal("netout--a-very-long-containe"))
				}
			})
		})
		Context("when inserting the rule fails", func() {
			BeforeEach(func() {
				ipTables.InsertReturns(errors.New("banana"))
			})
			It("returns the error", func() {
				err := manager.NetOut("some-handle", netOutProperties)
				Expect(err).To(MatchError("inserting net-out rule: banana"))
			})
		})
		Context("when unmarshaling json fails", func() {
			BeforeEach(func() {
				netOutProperties = `%%%%%%%`
			})
			It("returns the error", func() {
				err := manager.NetOut("some-handle", netOutProperties)
				Expect(err).To(MatchError(ContainSubstring("unmarshaling net-out properties: invalid character")))
			})
		})
	})
})
