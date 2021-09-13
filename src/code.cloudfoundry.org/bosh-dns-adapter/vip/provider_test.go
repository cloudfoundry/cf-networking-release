package vip_test

import (
	"fmt"
	"net"

	"code.cloudfoundry.org/bosh-dns-adapter/vip"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provider", func() {

	var (
		provider *vip.Provider
		cidr     *net.IPNet
	)

	BeforeEach(func() {
		var err error
		_, cidr, err = net.ParseCIDR("127.128.0.0/9")
		Expect(err).NotTo(HaveOccurred())

		provider = &vip.Provider{
			CIDR: cidr,
		}
	})

	It("returns a parsable IP", func() {
		Expect(net.ParseIP(provider.Get("a-hostname.apps.internal"))).ToNot(BeNil())
	})

	Specify("the same hostname always returns the same VIP", func() {
		vip1 := provider.Get("potato")
		vip2 := provider.Get("potato")
		vip3 := provider.Get("potato")

		Expect(vip1).To(Equal(vip2))
		Expect(vip1).To(Equal(vip3))
		Expect(vip2).To(Equal(vip3))
	})

	Specify("different hostnames return different vips", func() {
		vip1 := provider.Get("potato")
		vip2 := provider.Get("banana")
		vip3 := provider.Get("fruitcake")

		Expect(vip1).NotTo(Equal(vip2))
		Expect(vip1).NotTo(Equal(vip3))
		Expect(vip2).NotTo(Equal(vip3))
	})

	It("uses the full range of 127.128.0.0/9", func() {
		foundPrefixes0 := map[byte]interface{}{}
		foundPrefixes1 := map[byte]interface{}{}
		foundPrefixes2 := map[byte]interface{}{}
		foundPrefixes3 := map[byte]interface{}{}
		for i := 0; i < 10000; i++ {
			vipStr := provider.Get(fmt.Sprintf("%d", i))
			vip := net.ParseIP(vipStr)
			foundPrefixes0[vip.To4()[0]] = true
			foundPrefixes1[vip.To4()[1]] = true
			foundPrefixes2[vip.To4()[2]] = true
			foundPrefixes3[vip.To4()[3]] = true
		}
		Expect(foundPrefixes0).To(HaveLen(1))
		Expect(foundPrefixes1).To(HaveLen(128))
		Expect(foundPrefixes2).To(HaveLen(256))
		Expect(foundPrefixes3).To(HaveLen(256))
	})

	It("returns ips from within the specified range", func() {
		for i := 0; i < 10000; i++ {
			vipStr := provider.Get(fmt.Sprintf("%d", i))
			vip := net.ParseIP(vipStr)
			Expect(cidr.Contains(vip)).To(BeTrue(), fmt.Sprintf("%s is not within %s", vipStr, cidr))
		}
	})
})
