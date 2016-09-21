package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"policy-server/config"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const DEFAULT_TIMEOUT = "5s"

var policyServerPath string

var mockUAAServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/check_token" {
		if r.Header["Authorization"][0] == "Basic dGVzdDp0ZXN0" {
			bodyBytes, _ := ioutil.ReadAll(r.Body)
			token := strings.Split(string(bodyBytes), "=")[1]
			Expect(token).NotTo(BeEmpty())

			if string(token) == "valid-token" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"scope":["network.admin"], "user_name":"some-user"}`))
			} else {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error_description":"Some requested scopes are missing: network.admin"}`))
			}
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
		return
	}
	w.WriteHeader(http.StatusNotFound)
}))

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Acceptance Suite")
}

var _ = BeforeSuite(func() {
	// only run on node 1
	fmt.Fprintf(GinkgoWriter, "building binary...")
	var err error
	policyServerPath, err = gexec.Build("policy-server/cmd/policy-server", "-race")
	fmt.Fprintf(GinkgoWriter, "done")
	Expect(err).NotTo(HaveOccurred())

	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func VerifyTCPConnection(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func WriteConfigFile(policyServerConfig config.Config) string {
	configFile, err := ioutil.TempFile("", "test-config")
	Expect(err).NotTo(HaveOccurred())

	configBytes, err := json.Marshal(policyServerConfig)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(configFile.Name(), configBytes, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	return configFile.Name()
}

const NUM_PARALLEL_WORKERS = 4

func workPoolRunOnChannel(work chan interface{}, workFunc func(item interface{})) {
	var wg sync.WaitGroup

	for workerID := 0; workerID < NUM_PARALLEL_WORKERS; workerID++ {
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			for item := range work {
				workFunc(item)
			}
			wg.Done()
		}()
	}

	// wait for all work to complete
	wg.Wait()
}

func workPoolRun(items []interface{}, workFunc func(item interface{})) {
	work := make(chan interface{})

	go func() {
		// queue the work
		for _, item := range items {
			work <- item
		}
		close(work)
	}()

	workPoolRunOnChannel(work, workFunc)
}
