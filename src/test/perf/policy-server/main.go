package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"lib/models"
	"lib/policy_client"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"

	"code.cloudfoundry.org/cf-networking-helpers/mutualtls"
	"code.cloudfoundry.org/lager"
)

var (
	config               Config
	testDuration         time.Duration
	pollInterval         time.Duration
	externalPolicyClient *policy_client.ExternalClient
	internalPolicyClient *policy_client.InternalClient
)

type Config struct {
	Api                 string `json:"api"`
	Apps                int    `json:"apps"`
	CreateNewPolicies   bool   `json:"create_new_policies"`
	TestDurationMinutes int    `json:"test_duration_minutes"`
	Logs                string `json:"logs"`
	NumCells            int    `json:"num_cells"`
	PoliciesPerApp      int    `json:"policies_per_app"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`

	ServerCACertFile            string `json:"ca_cert_file" validate:"nonzero"`
	ClientCertFile              string `json:"client_cert_file" validate:"nonzero"`
	ClientKeyFile               string `json:"client_key_file" validate:"nonzero"`
	PolicyServerInternalBaseURL string `json:"policy_server_internal_base_url"`
}

func loadTestConfig(logger lager.Logger) {
	configPath := helpers.ConfigPath()
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		logger.Fatal("reading-config", err)
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		logger.Fatal("unmarshalling-config", err)
	}

	if config.Api == "" {
		logger.Fatal("reading-api-from-config", errors.New("API not specified in config"))
	}

	testDuration = time.Duration(config.TestDurationMinutes) * time.Minute
	pollInterval = time.Duration(config.PollIntervalSeconds) * time.Second
}

func getInternalPolicyClient(logger lager.Logger) *policy_client.InternalClient {
	clientTLSConfig, err := mutualtls.NewClientTLSConfig(config.ClientCertFile, config.ClientKeyFile, config.ServerCACertFile)
	if err != nil {
		logger.Fatal("mutual-tls", err)
	}
	clientTLSConfig.InsecureSkipVerify = true

	httpClient := &http.Client{
		Timeout: pollInterval,
		Transport: &http.Transport{
			TLSClientConfig: clientTLSConfig,
		},
	}

	return policy_client.NewInternal(logger.Session("internal-policy-client"), httpClient, config.PolicyServerInternalBaseURL)
}

func getExternalPolicyClient(logger lager.Logger) *policy_client.ExternalClient {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	policyServerAPI := fmt.Sprintf("https://%s", config.Api)
	return policy_client.NewExternal(logger.Session("external-policy-client"), httpClient, policyServerAPI)
}

func randomAppGUID(index int) string {
	return fmt.Sprintf("%08x", rand.Int63())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func makeChunks(policies []models.Policy) [][]models.Policy {
	tokenRenewalChunkSize := 1000
	var chunks [][]models.Policy
	for i := 0; i < len(policies); i += tokenRenewalChunkSize {
		chunks = append(chunks, policies[i:i+min(tokenRenewalChunkSize, len(policies))])
	}
	return chunks
}

func addNewPolicies(logger lager.Logger, appGuids []string, token string) {
	logger.Info("creating-policies-for-each-application-guid")
	policies := []models.Policy{}
	for _ = range appGuids {
		for i := 0; i < config.PoliciesPerApp; i++ {
			dstGuid := appGuids[rand.Intn(len(appGuids))]
			srcGuid := appGuids[rand.Intn(len(appGuids))]

			policy := models.Policy{
				Source: models.Source{
					ID: srcGuid,
				},
				Destination: models.Destination{
					ID:       dstGuid,
					Protocol: "tcp",
					Port:     10000 + rand.Intn(10000),
				},
			}
			policies = append(policies, policy)
		}
	}

	logger.Info("adding-policies")
	for _, chunk := range makeChunks(policies) {
		logger.Info("adding-policies-chunk")
		err := externalPolicyClient.AddPolicies(token, chunk)
		if err != nil {
			logger.Fatal("adding-policies", err)
		}
		token = getCurrentToken(logger)
	}
	logger.Info("finished-adding-policies-to-policy-server")
}

func getPoliciesForCell(logger lager.Logger, ids []string, index, numCalls int) {
	logger.Info("getting-policies-by-id", lager.Data{
		"index":    index,
		"numCalls": numCalls,
	})

	_, err := internalPolicyClient.GetPoliciesByID(ids...)
	if err != nil {
		logger.Fatal("getting-policies-by-id", err)
	} else {
		logger.Info(fmt.Sprintf("finished-request-from-cell-#%d-on-call-#%d", index, numCalls))
	}
}

func deleteOldPolicies(logger lager.Logger, token string) {
	logger.Info("getting-existing-policies")
	policies, err := externalPolicyClient.GetPolicies(token)
	if err != nil {
		logger.Fatal("get-policies", err)
	}
	logger.Info("number-of-existing-policies", lager.Data{"num-existing-policies": len(policies)})

	logger.Info("deleting-existing-policies")
	for _, chunk := range makeChunks(policies) {
		logger.Info("deleting-policies-chunk")
		err := externalPolicyClient.DeletePolicies(token, chunk)
		if err != nil {
			logger.Fatal("deleting-policies", err)
		}
		token = getCurrentToken(logger)
	}
	logger.Info("deleted-existing-policies")
}

func jitter(baseTime time.Duration, jitterAmount time.Duration) time.Duration {
	x := rand.Int63n(int64(jitterAmount)*2) - int64(jitterAmount)
	return baseTime + time.Duration(x)
}

func pollPolicyServer(logger lager.Logger, ids []string, index int) {
	numCalls := 0
	for {
		select {
		case <-time.After(jitter(pollInterval, 1*time.Second)):
			go getPoliciesForCell(logger, ids, index, numCalls)
			numCalls = numCalls + 1
			continue
		}
	}
}

func getCurrentToken(logger lager.Logger) string {
	cmd := exec.Command("cf", "oauth-token")

	tokenBytes, err := cmd.Output()
	if err != nil {
		logger.Fatal("running-command-cf-oauth-token`", err)
	}

	token := string(tokenBytes[0 : len(tokenBytes)-1]) // remove trailing \n
	logger.Info("parsed-cf-oauth-token", lager.Data{"token": token})

	return token
}

func main() {
	logger := lager.NewLogger("cf-networking.policy-server-test")

	loadTestConfig(logger)

	file, err := os.OpenFile(config.Logs, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		logger.Fatal("writing-to-log-file", err)
	}
	logger.RegisterSink(lager.NewWriterSink(file, lager.INFO))
	logger.Info("started")
	defer logger.Info("exited")

	token := getCurrentToken(logger)

	logger.Info("creating-application-guids")
	rand.Seed(1) // always use the same random sequence
	var guids []string
	for i := 0; i < config.Apps; i++ {
		guids = append(guids, randomAppGUID(i))
	}
	logger.Info(fmt.Sprintf("finished-creating-%d-application-guids", config.Apps))

	internalPolicyClient = getInternalPolicyClient(logger)
	externalPolicyClient = getExternalPolicyClient(logger)

	if config.CreateNewPolicies {
		deleteOldPolicies(logger, token)
		token = getCurrentToken(logger)
		addNewPolicies(logger, guids, token)
	} else {
		logger.Info("skipped-creating-policies")
	}

	appsPerCell := config.Apps / config.NumCells
	var cells [][]string
	for i := 0; i < config.NumCells; i++ {
		cells = append(cells, guids[i*appsPerCell:(i+1)*appsPerCell])
	}

	for i := 0; i < len(cells); i++ {
		go func(i int) {
			logger.Info(fmt.Sprintf("cell-%d-polling-policy-server", i))
			pollPolicyServer(logger, cells[i], i)
		}(i)
	}

	fmt.Println("Press CTRL-C to exit")
	select {
	case <-time.After(testDuration):
		logger.Info(fmt.Sprintf("exiting"))
		os.Exit(0)
	}
}
