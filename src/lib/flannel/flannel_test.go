package flannel_test

import (
	"io/ioutil"
	"lib/flannel"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Discovering the local subnet and flannel network", func() {
	var (
		networkInfo flannel.NetworkInfo
	)

	BeforeEach(func() {
		contents := `FLANNEL_NETWORK=10.240.0.0/12
FLANNEL_SUBNET=10.255.19.1/24
FLANNEL_MTU=1450
FLANNEL_IPMASQ=false
`
		tempFile, err := ioutil.TempFile("", "subnet.env")
		Expect(err).NotTo(HaveOccurred())

		_, err = tempFile.WriteString(contents)
		Expect(err).NotTo(HaveOccurred())
		Expect(tempFile.Close()).To(Succeed())

		networkInfo = flannel.NetworkInfo{
			FlannelSubnetFilePath: tempFile.Name(),
		}
	})

	It("returns the subnet and network in the flannel subnet env", func() {
		subnet, network, err := networkInfo.DiscoverNetworkInfo()
		Expect(err).NotTo(HaveOccurred())
		Expect(subnet).To(Equal("10.255.19.1/24"))
		Expect(network).To(Equal("10.240.0.0/12"))
	})

	Context("when there is a problem opening the file", func() {
		It("returns a helpful error", func() {
			networkInfo = flannel.NetworkInfo{
				FlannelSubnetFilePath: "bad-path",
			}
			_, _, err := networkInfo.DiscoverNetworkInfo()
			Expect(err).To(MatchError("open bad-path: no such file or directory"))
		})
	})

	Context("when the file is malformed", func() {
		It("returns a helpful error", func() {
			Expect(ioutil.WriteFile(networkInfo.FlannelSubnetFilePath, []byte("boo"), 0600)).To(Succeed())

			_, _, err := networkInfo.DiscoverNetworkInfo()
			Expect(err).To(MatchError("unable to parse flannel subnet file"))
		})
	})

	Context("when the file doesn't have a valid subnet entry", func() {
		It("returns a helpful error", func() {
			Expect(ioutil.WriteFile(networkInfo.FlannelSubnetFilePath, []byte(`FLANNEL_NETWORK=10.255.0.0/16
FLANNEL_SUBNET=banana
FLANNEL_MTU=1450
FLANNEL_IPMASQ=false
`), 0600)).To(Succeed())
			_, _, err := networkInfo.DiscoverNetworkInfo()
			Expect(err).To(MatchError("unable to parse flannel subnet file"))
		})
	})

	Context("when the file doesn't have a valid network entry", func() {
		It("returns a helpful error", func() {
			Expect(ioutil.WriteFile(networkInfo.FlannelSubnetFilePath, []byte(`FLANNEL_NETWORK=banana
FLANNEL_SUBNET=10.255.19.1/24
FLANNEL_MTU=1450
FLANNEL_IPMASQ=false
`), 0600)).To(Succeed())
			_, _, err := networkInfo.DiscoverNetworkInfo()
			Expect(err).To(MatchError("unable to parse flannel network from subnet file"))
		})
	})
})
