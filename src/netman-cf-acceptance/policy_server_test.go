package acceptance_test

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("policy server tests", func() {
	It("makes the policy server available at an external route", func() {
		cmd := exec.Command("curl", "-v", fmt.Sprintf("http://%s/networking", config.ApiEndpoint))

		sess, err := gexec.Start(cmd, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess.Wait(Timeout_Short)).Should(gexec.Exit(0))

		curlOutput := sess.Out.Contents()
		Expect(curlOutput).To(ContainSubstring("Network policy server, up for"))
	})
})
