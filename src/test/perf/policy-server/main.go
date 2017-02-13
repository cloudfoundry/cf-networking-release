package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/models"
	"lib/policy_client"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
)

func refreshToken(logger lager.Logger, cfUser, cfPassword, cfHomeDir string) error {
	cmd := exec.Command("cf", "auth", cfUser, cfPassword)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("CF_HOME=%s", cfHomeDir))

	outputBytes, err := cmd.Output()
	if err != nil {
		logger.Error("failed-to-run-cf-auth", err)
		return err
	}

	output := string(outputBytes)

	if strings.Contains(output, "FAILED") {
		logger.Error("Failed to authenticate", nil, lager.Data{
			"cfUser":     cfUser,
			"cfPassword": cfPassword,
		})
		return errors.New("failed to authenticate")
	}

	return nil
}

func getCurrentToken(logger lager.Logger, cfHomeDir string) (string, error) {
	cmd := exec.Command("cf", "oauth-token")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("CF_HOME=%s", cfHomeDir))

	tokenBytes, err := cmd.Output()
	if err != nil {
		logger.Error("failed-to-run-cf-oauth-token", err)
		return "", err
	}

	token := string(tokenBytes[0 : len(tokenBytes)-1]) // remove trailing \n
	logger.Info("got-token", lager.Data{"token": token})

	return token, nil
}

func main() {
	// Our GA scalability target is: 100 cells, 100 apps and 200 instances per app with 3 policies per app.

	// config:
	// - total policies (default: 60,000. 10000 apps and 2 instances per app with 3 policies per app)

	// - number of cells (default: 100)
	// - policies per cell (default: 600 src, 600 dst) // policies assumed to be uniform/bi-directional
	// - containers per cell (default: 200) // 200 unique app ids per cell
	// - polling frequency (default 5)
	// - run forever unitl Ctrl+C? or set some duration?

	// can 1 workstation generate enough load?
	// 1 request per cell, every 5 seconds (* 100 cells)
	// 100 requests, every 5 seconds
	// 20 requests per second

	// before test run:
	// clean up policies
	// if necessary, disable cleanup (restart server with long cleanup polling interval)

	logger := lager.NewLogger("container-networking.policy-server-test")

	var (
		apps, numCells, policiesPerApp           int
		pollInterval, expiration                 time.Duration
		policyServerAPI, cfUser, cfPassword, out string
		setup                                    bool
	)
	flag.IntVar(&apps, "apps", 10000, "number of apps")
	flag.IntVar(&numCells, "numCells", 100, "number of cells")
	// TODO app instances
	flag.IntVar(&policiesPerApp, "policiesPerApp", 3, "policies per app")
	flag.DurationVar(&pollInterval, "pollInterval", 5*time.Second, "polling interval on each cell")
	flag.StringVar(&cfUser, "cfUser", "", "cf user")
	flag.StringVar(&cfPassword, "cfPassword", "", "cf password for cf user")
	flag.StringVar(&policyServerAPI, "api", "", "policy server base URL")
	flag.BoolVar(&setup, "setup", true, "if true, remove existing policies and create new policies")
	flag.StringVar(&out, "out", "out.txt", "lager stdout")
	flag.DurationVar(&expiration, "expiration", time.Hour, "length of polling")
	flag.Parse()

	// Write to file
	file, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		logger.Fatal("unable-to-write-to-file", err)
	}
	logger.RegisterSink(lager.NewWriterSink(file, lager.INFO))
	logger.Info("started")
	defer logger.Info("exited")

	if policyServerAPI == "" {
		logger.Fatal("Specify policy server", errors.New(""))
	}

	refreshTokenTime := 5 * time.Minute

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // insecure!!!
			},
		},
	}

	user, err := user.Current()
	if err != nil {
		logger.Error("get-home-dir-failed", err)
	}
	homeDir := user.HomeDir
	if _, err = os.Stat(filepath.Join(homeDir, ".cf")); os.IsNotExist(err) {
		logger.Error("cf-dir-unavailable", err)
		panic("cf-dir-unavailable")
	}

	defaultCfDir := filepath.Join(homeDir, ".cf")

	// Create temp cf home dirs for each cell, with config copied from ~/.cf
	cfDirs := make([]string, numCells, numCells)
	for i := 0; i < numCells; i++ {
		cfDir, err := ioutil.TempDir("", "cfhome")
		if err != nil {
			logger.Error("init-temp-cf-home-dir-failed", err)
		}
		cfDirs[i] = cfDir

		cmd := exec.Command("cp", "-r", filepath.Join(homeDir, ".cf"), filepath.Join(cfDir, ".cf"))
		err = cmd.Run()
		if err != nil {
			logger.Error("copy-cf-config-failed", err)
		}
	}
	logger.Info("cfDirs", lager.Data{"cfDirs": cfDirs})

	err = refreshToken(logger, cfUser, cfPassword, defaultCfDir)
	if err != nil {
		logger.Fatal("Unable to refresh token", err)
	}
	homeToken, err := getCurrentToken(logger, defaultCfDir)
	if err != nil {
		logger.Fatal("Unable to get token", err)
	}

	client := policy_client.NewExternal(logger, httpClient, policyServerAPI)

	if setup {
		logger.Info("getting-existing-policies")
		policies, err := client.GetPolicies(homeToken)
		logger.Info("existing-policies", lager.Data{"num-existing-policies": len(policies)})
		if err != nil {
			logger.Fatal("Failed to get policies", err)
		}

		logger.Info("deleting-existing-policies")
		err = client.DeletePolicies(homeToken, policies)
		if err != nil {
			logger.Fatal("Failed to delete policies", err)
		}
		logger.Info("done-deleting-existing-policies")
	} else {
		logger.Info("not-cleaning-up-existing-policies")
	}

	// creates "applications" (10,000 guids) (in local memory)
	logger.Info("creating-applications")
	var guids []string
	for i := 0; i < apps; i++ {
		guid := fmt.Sprintf("9cb281b-e272-4df7-b398-b6663ca-%04d", i) // TODO we should do better... indexes and what not
		guids = append(guids, guid)
	}
	logger.Info("done-creating-applications")

	if setup {
		// creates the policies (30,000) (using the fake app guids)
		policies := []models.Policy{}
		logger.Info("creating-policies")
		for _, guid := range guids {
			for i := 0; i < policiesPerApp; i++ {
				policy := models.Policy{
					Source: models.Source{
						ID: guid,
					},
					Destination: models.Destination{
						ID:       guid, // TODO make this random or explore other distrubutions (eg hotspot)
						Protocol: "tcp",
						Port:     9000 + i,
					},
				}
				policies = append(policies, policy)
			}
		}
		err := client.AddPolicies(homeToken, policies)
		if err != nil {
			logger.Fatal("Failed to create policies", err)
		}
		logger.Info("done-creating-policies")

		err = refreshToken(logger, cfUser, cfPassword, defaultCfDir)
		if err != nil {
			logger.Fatal("Unable to refresh token", err)
		}
		homeToken, err = getCurrentToken(logger, defaultCfDir)
		if err != nil {
			logger.Fatal("Unable to get token", err)
		}

	} else {
		logger.Info("skipping-creating-policies")
	}

	// simulate placing "app instances on cells" (100 cells, with 100 app guids per cell)
	appsPerCell := apps / numCells
	var cells [][]string
	for i := 0; i < numCells; i++ {
		cells = append(cells, guids[i*appsPerCell:(i+1)*appsPerCell])
	}

	// each "cell" is its own goroutine which spawns goroutines to make requests
	callPolicyServer := func(ids []string, index, numCalls int, token string) {
		logger.Info("callPolicyServer", lager.Data{
			"index":    index,
			"numCalls": numCalls,
			"token":    token,
		})
		_, err := client.GetPoliciesByID(token, ids...)
		if err != nil {
			logger.Error("failed-to-get-policies", err)
		} else {
			logger.Info(fmt.Sprintf("completed-request-from-cell-%d-call-%d", index, numCalls))
		}
	}

	pollPolicyServer := func(ids []string, index int) {
		err := refreshToken(logger, cfUser, cfPassword, cfDirs[index])
		if err != nil {
			logger.Fatal("Unable to refresh token", err)
		}
		token, err := getCurrentToken(logger, cfDirs[index])
		if err != nil {
			logger.Fatal("Unable to get token", err)
		}

		numCalls := 0
		lastTokenRefresh := time.Now()
		for {
			select {
			case <-time.After(pollInterval): // TODO jitter?
				if time.Now().Sub(lastTokenRefresh) > refreshTokenTime {
					// refresh token
					lastTokenRefresh = time.Now()
					err := refreshToken(logger, cfUser, cfPassword, cfDirs[index])
					if err != nil {
						logger.Fatal("Unable to refresh token", err)
					}
					token, err = getCurrentToken(logger, cfDirs[index])
					if err != nil {
						logger.Fatal("Unable to get token", err)
					}
				}
				go callPolicyServer(ids, index, numCalls, token)
				numCalls = numCalls + 1
				continue
			}
		}
	}

	// each "cell" makes requests to server for its app instances
	for i := 0; i < len(cells); i++ {
		go func(i int) {
			logger.Info(fmt.Sprintf("cell-%d-polling-server", i))
			pollPolicyServer(cells[i], i)
		}(i)
	}

	fmt.Println("Press CTRL-C to exit")
	select {
	case <-time.After(expiration):
		os.Exit(0)
	}
}
