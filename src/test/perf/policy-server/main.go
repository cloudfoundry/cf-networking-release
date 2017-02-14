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
	"os/user"
	"path/filepath"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"

	"code.cloudfoundry.org/lager"
)

var (
	api               string
	apps              int
	cfUser            string
	cfPassword        string
	config            Config
	createNewPolicies bool
	expiration        time.Duration
	logs              string
	numCells          int
	policiesPerApp    int
	pollInterval      time.Duration
	policyClient      *policy_client.ExternalClient
)

type Config struct {
	AdminUser           string `json:"admin_user"`
	AdminPassword       string `json:"admin_password"`
	Api                 string `json:"api"`
	Apps                int    `json:"apps"`
	CreateNewPolicies   bool   `json:"create_new_policies"`
	ExpirationMinutes   int    `json:"expiration"`
	Logs                string `json:"logs"`
	NumCells            int    `json:"num_cells"`
	PoliciesPerApp      int    `json:"policies_per_app"`
	PollIntervalSeconds int    `json:"poll_interval"`
	SkipSslValidation   bool   `json:"skip_ssl_validation"`
}

func main() {
	logger := lager.NewLogger("cf-networking.policy-server-test")

	loadTestConfig(logger)

	file, err := os.OpenFile(logs, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		logger.Fatal("writing-to-log-file", err)
	}
	logger.RegisterSink(lager.NewWriterSink(file, lager.INFO))
	logger.Info("started")
	defer logger.Info("exited")

	if api == "" {
		logger.Fatal("reading-api-from-config", errors.New("API not specified in config"))
	}

	user, err := user.Current()
	if err != nil {
		logger.Fatal("get-current-user", err)
	}

	userDir := user.HomeDir
	if _, err = os.Stat(filepath.Join(userDir, ".cf")); os.IsNotExist(err) {
		logger.Fatal("get-user-home-dir", err)
	}

	defaultCfDir := filepath.Join(userDir, ".cf")
	cfDirs := createTempCfDirs(logger, numCells, defaultCfDir)

	homeToken := getCurrentToken(logger, defaultCfDir)

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.SkipSslValidation,
			},
		},
	}

	policyServerAPI := fmt.Sprintf("https://%s", api)
	policyClient = policy_client.NewExternal(logger, httpClient, policyServerAPI)

	logger.Info("creating-application-guids")
	var guids []string
	for i := 0; i < apps; i++ {
		guid := fmt.Sprintf("9cb281b-e272-4df7-b398-b6663ca-%04d", i) // TODO: improve guid creation
		guids = append(guids, guid)
	}
	logger.Info(fmt.Sprintf("finished-creating-%d-application-guids", apps))

	if createNewPolicies {
		deleteOldPolicies(logger, homeToken)
		addNewPolicies(logger, guids, homeToken)
	} else {
		logger.Info("skipped-creating-policies")
	}

	appsPerCell := apps / numCells
	var cells [][]string
	for i := 0; i < numCells; i++ {
		cells = append(cells, guids[i*appsPerCell:(i+1)*appsPerCell])
	}

	for i := 0; i < len(cells); i++ {
		go func(i int) {
			logger.Info(fmt.Sprintf("cell-%d-polling-policy-server", i))
			pollPolicyServer(logger, cells[i], i, cfDirs)
		}(i)
	}

	fmt.Println("Press CTRL-C to exit")
	select {
	case <-time.After(expiration):
		logger.Info(fmt.Sprintf("exiting"))
		os.Exit(0)
	}
}

func addNewPolicies(logger lager.Logger, guids []string, token string) {
	logger.Info("creating-policies-for-each-application-guid")
	policies := []models.Policy{}
	for _, guid := range guids {
		for i := 0; i < policiesPerApp; i++ {
			policy := models.Policy{
				Source: models.Source{
					ID: guid,
				},
				Destination: models.Destination{
					ID:       guid, // TODO: randomness in policy creation and distributions (eg hotspot)
					Protocol: "tcp",
					Port:     9000 + i,
				},
			}
			policies = append(policies, policy)
		}
	}

	logger.Info("adding-policies")
	err := policyClient.AddPolicies(token, policies)
	if err != nil {
		logger.Fatal("adding-policies", err)
	}
	logger.Info("finished-adding-policies-to-policy-server")
}

func getPoliciesForCell(logger lager.Logger, ids []string, index, numCalls int, token string) {
	logger.Info("getting-policies-by-id", lager.Data{
		"index":    index,
		"numCalls": numCalls,
		"token":    token,
	})

	_, err := policyClient.GetPoliciesByID(token, ids...)
	if err != nil {
		logger.Fatal("getting-policies-by-id", err)
	} else {
		logger.Info(fmt.Sprintf("finished-request-from-cell-#%d-on-call-#%d", index, numCalls))
	}
}

func deleteOldPolicies(logger lager.Logger, token string) {
	logger.Info("getting-existing-policies")
	policies, err := policyClient.GetPolicies(token)
	if err != nil {
		logger.Fatal("get-policies", err)
	}
	logger.Info("number-of-existing-policies", lager.Data{"num-existing-policies": len(policies)})

	logger.Info("deleting-existing-policies")
	err = policyClient.DeletePolicies(token, policies)
	if err != nil {
		logger.Fatal("deleting-policies", err)
	}

	logger.Info("deleted-existing-policies")
}

const refreshTokenDuration = 5 * time.Minute

func jitter(baseTime time.Duration, jitterAmount time.Duration) time.Duration {
	x := rand.Int63n(int64(jitterAmount)*2) - int64(jitterAmount)
	return baseTime + time.Duration(x)
}

func pollPolicyServer(logger lager.Logger, ids []string, index int, cfDirs []string) {
	token := getCurrentToken(logger, cfDirs[index])

	numCalls := 0
	lastTokenRefresh := time.Now()
	for {
		select {
		case <-time.After(jitter(pollInterval, 1*time.Second)):
			if time.Now().Sub(lastTokenRefresh) > jitter(refreshTokenDuration, 1*time.Minute) {
				lastTokenRefresh = time.Now()

				token = getCurrentToken(logger, cfDirs[index])
			}

			go getPoliciesForCell(logger, ids, index, numCalls, token)
			numCalls = numCalls + 1
			continue
		}
	}
}

func getCurrentToken(logger lager.Logger, cfHomeDir string) string {
	cmd := exec.Command("cf", "oauth-token")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("CF_HOME=%s", cfHomeDir))

	tokenBytes, err := cmd.Output()
	if err != nil {
		logger.Fatal("running-command-cf-oauth-token`", err)
	}

	token := string(tokenBytes[0 : len(tokenBytes)-1]) // remove trailing \n
	logger.Info("parsed-cf-oauth-token", lager.Data{"token": token})

	return token
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

	api = config.Api
	apps = config.Apps
	cfUser = config.AdminUser
	cfPassword = config.AdminPassword
	createNewPolicies = config.CreateNewPolicies
	logs = config.Logs
	expiration = time.Duration(config.ExpirationMinutes) * time.Minute
	numCells = config.NumCells
	policiesPerApp = config.PoliciesPerApp
	pollInterval = time.Duration(config.PollIntervalSeconds) * time.Second
}

func createTempCfDirs(logger lager.Logger, numCells int, defaultCfDir string) []string {
	cfDirs := make([]string, numCells, numCells)

	for i := 0; i < numCells; i++ {
		cfDir, err := ioutil.TempDir("", "cfhome")
		if err != nil {
			logger.Fatal("creating-temp-cf-dir", err)
		}

		cfDirs[i] = cfDir

		cmd := exec.Command("cp", "-r", filepath.Join(defaultCfDir, ".cf"), filepath.Join(cfDir, ".cf"))
		err = cmd.Run()
		if err != nil {
			logger.Fatal("copying-cf-config", err)
		}
	}

	logger.Info("created-temp-cf-dirs", lager.Data{"cfDirs": cfDirs})
	return cfDirs
}
