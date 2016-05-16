package acceptance_test

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/garden/client/connection"

	garden_client "github.com/cloudfoundry-incubator/garden/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Container networking", func() {
	const ExternalNetworkSpecKey = "garden.external.network-spec"
	const cmdGetContainerIP = `ifconfig eth0 | grep "inet addr" | awk -F: '{print $2}' | awk '{print $1}' | tr -d '\n'`

	var (
		gardenClient1 garden.Client
		gardenClient2 garden.Client

		gardenContainer1 garden.Container
		gardenContainer2 garden.Container

		containerAddr1, containerAddr2 string

		appID, spaceID string
	)

	BeforeEach(func() {
		gardenClient1 = garden_client.New(connection.New("tcp", fmt.Sprintf("%s:7777", gardenServer1)))
		gardenClient2 = garden_client.New(connection.New("tcp", fmt.Sprintf("%s:7777", gardenServer2)))

		appID = fmt.Sprintf("some-app-%x", rand.Int())
		spaceID = fmt.Sprintf("some-space-%x", rand.Int())

		var err error
		gardenContainer1, err = gardenClient1.Create(garden.ContainerSpec{
			Properties: garden.Properties{
				"network.app_id":   appID,
				"network.space_id": spaceID,
			},
		})
		Expect(err).NotTo(HaveOccurred())
		containerAddr1 = runInContainer(gardenContainer1, cmdGetContainerIP)

		gardenContainer2, err = gardenClient2.Create(garden.ContainerSpec{
			Properties: garden.Properties{
				"network.app_id":   appID,
				"network.space_id": spaceID,
			},
		})
		Expect(err).NotTo(HaveOccurred())
		containerAddr2 = runInContainer(gardenContainer2, cmdGetContainerIP)
	})

	AfterEach(func() {
		err := gardenClient1.Destroy(gardenContainer1.Handle())
		Expect(err).NotTo(HaveOccurred())

		err = gardenClient2.Destroy(gardenContainer2.Handle())
		Expect(err).NotTo(HaveOccurred())
	})

	It("connects the containers", func() {
		By("pinging from container 1 to container 2")
		runInContainer(gardenContainer1, `ping -c3 `+containerAddr2)

		By("running a shasum server on container 2")
		go func() {
			defer GinkgoRecover()
			runInContainer(gardenContainer2, fmt.Sprintf(`nc -l -p 9999 %s -e sha1sum`, containerAddr2))
		}()

		time.Sleep(500 * time.Millisecond)

		By("using container 1 as a client, requesting a shasum for some random data")
		expectedSha := runInContainer(gardenContainer1, `cat /dev/urandom | head -c 1000 | tee testdata | sha1sum`)
		computedSha := runInContainer(gardenContainer1, fmt.Sprintf(`cat testdata | nc %s 9999`, containerAddr2))
		Expect(computedSha).To(Equal(expectedSha))
	})
})

func runInContainer(container garden.Container, shellCmd string) string {
	GinkgoWriter.Write([]byte(container.Handle() + ": " + shellCmd + "\n"))
	procSpec := garden.ProcessSpec{
		Path: "/bin/sh",
		Args: []string{"-c", shellCmd},
		User: "root",
	}

	stdout := &bytes.Buffer{}

	procIO := garden.ProcessIO{
		Stdin:  &bytes.Buffer{},
		Stdout: io.MultiWriter(stdout, GinkgoWriter),
		Stderr: GinkgoWriter,
	}
	process, err := container.Run(procSpec, procIO)
	Expect(err).NotTo(HaveOccurred())
	Eventually(process.Wait).Should(Equal(0))

	GinkgoWriter.Write([]byte("\n" + container.Handle() + ": done \n"))
	return stdout.String()
}
