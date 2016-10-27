package main_test

import (
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"
	noop_debug "github.com/containernetworking/cni/plugins/test/noop/debug"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("CniWrapperPlugin", func() {
	var (
		cmd             *exec.Cmd
		debugFileName   string
		debug           *noop_debug.Debug
		expectedCmdArgs skel.CmdArgs
	)

	const input = `
{
  "name": "cni-wrapper",
  "type": "wrapper",
  "datastore": "/path/to/datastore",
	"delegate": {
			"name": "cni-noop",
			"type": "noop",
			"delegate":
			{"some":"stdin-json", "cniVersion": "0.2.0"}
   }
}
`

	BeforeEach(func() {

		debugFile, err := ioutil.TempFile("", "cni_debug")
		Expect(err).NotTo(HaveOccurred())
		Expect(debugFile.Close()).To(Succeed())
		debugFileName = debugFile.Name()

		Expect(debug.WriteDebug(debugFileName)).To(Succeed())

		// fmt.Println(debugFileName)
		cmd = exec.Command(paths.PathToPlugin)
		cmd.Env = []string{
			"CNI_COMMAND=ADD",
			"CNI_CONTAINERID=some-container-id",
			"CNI_ARGS=DEBUG=" + debugFileName,
			"CNI_NETNS=/some/netns/path",
			"CNI_IFNAME=some-eth0",
			"CNI_PATH=" + paths.CNIPath,
		}
		cmd.Stdin = strings.NewReader(input)
		expectedCmdArgs = skel.CmdArgs{
			ContainerID: "some-container-id",
			Netns:       "/some/netns/path",
			IfName:      "some-eth0",
			Args:        "DEBUG=" + debugFileName,
			Path:        "/some/bin/path",
			StdinData:   []byte(input),
		}
	})

	AfterEach(func() {
		//os.Remove(debugFileName)
	})

	It("responds to ADD using the ReportResult debug field", func() {
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))
		//Expect(session.Out.Contents()).To(MatchJSON("something"))
	})
})
