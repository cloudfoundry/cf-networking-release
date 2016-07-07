package flannel_test

import (
	"io/ioutil"
	"lib/flannel"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Discovering the local subnet", func() {
	var (
		localSubnet flannel.LocalSubnet
	)

	BeforeEach(func() {
		contents := `FLANNEL_NETWORK=10.255.0.0/16
FLANNEL_SUBNET=10.255.19.1/24
FLANNEL_MTU=1450
FLANNEL_IPMASQ=false
`
		tempFile, err := ioutil.TempFile("", "subnet.env")
		Expect(err).NotTo(HaveOccurred())

		_, err = tempFile.WriteString(contents)
		Expect(err).NotTo(HaveOccurred())
		Expect(tempFile.Close()).To(Succeed())

		localSubnet = flannel.LocalSubnet{
			FlannelSubnetFilePath: tempFile.Name(),
		}
	})

	It("returns the subnet in the flannel subnet env", func() {
		subnet, err := localSubnet.DiscoverLocalSubnet()
		Expect(err).NotTo(HaveOccurred())
		Expect(subnet).To(Equal("10.255.19.1/24"))
	})

	Context("when there is a problem opening the file", func() {
		It("returns a helpful error", func() {
			localSubnet = flannel.LocalSubnet{
				FlannelSubnetFilePath: "bad-path",
			}
			_, err := localSubnet.DiscoverLocalSubnet()
			Expect(err).To(MatchError("open bad-path: no such file or directory"))
		})
	})

	Context("when the file is malformed", func() {
		It("returns a helpful error", func() {
			Expect(ioutil.WriteFile(localSubnet.FlannelSubnetFilePath, []byte("boo"), 0600)).To(Succeed())

			_, err := localSubnet.DiscoverLocalSubnet()
			Expect(err).To(MatchError("unable to parse flannel subnet file"))
		})

	})
})
