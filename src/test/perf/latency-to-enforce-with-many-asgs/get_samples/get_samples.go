package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lib/policy_client"
	"math/rand"
	"net/http"
	"os"
	"policy-server/api/api_v0"
	"strconv"
	"time"

	"code.cloudfoundry.org/lager"
)

func main() {
	if err := mainWithError(); err != nil {
		os.Stderr.Write([]byte(err.Error() + "\n"))
		os.Exit(1)
	}
}

func mainWithError() error {
	args := os.Args
	if len(args) < 1 {
		return fmt.Errorf("usage: lots of args")
	}

	oauthToken := args[1]

	numExistingASGs, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("parsing num existing asgs: %s", err)
	}

	baseURL := args[3]
	proxyAppGUID := args[4]
	proxyAppRoute := args[5]
	statsFile := args[6]
	nSamples, err := strconv.Atoi(args[7])
	if err != nil {
		return fmt.Errorf("parsing num samples: %s", err)
	}
	backendAppGUID := args[8]
	backendAppRoute := args[9]

	logger := lager.NewLogger("test")
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.INFO))

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // insecure!!!
			},
		},
	}
	policyClient := policy_client.NewExternal(logger, httpClient, baseURL)

	destIP, err := getIPAddr(backendAppRoute)
	if err != nil {
		return fmt.Errorf("get ip addr for app: %s", err)
	}

	t := &TestThingy{
		logger:       logger,
		policyClient: policyClient,
		token:        oauthToken,

		sourceAppGUID: proxyAppGUID,
		destAppGUID:   backendAppGUID,
		proxyAppRoute: proxyAppRoute,
	}

	for i := 0; i < nSamples; i++ {
		duration, err := t.measureLatencyFor1More(destIP)
		if err != nil {
			return fmt.Errorf("measuring latency: %s", err)
		}

		logger.Info("measured-latency", lager.Data{"duration in seconds": duration.Seconds()})

		err = appendStat(statsFile, numExistingASGs, duration)
		if err != nil {
			return fmt.Errorf("writing stats: %s", err)
		}

		jitter := time.Duration(100+rand.Intn(3000)) * time.Millisecond
		time.Sleep(jitter)
	}

	return nil
}

type TestThingy struct {
	logger       lager.Logger
	policyClient policy_client.ExternalPolicyClient
	token        string

	sourceAppGUID, destAppGUID string
	proxyAppRoute              string
}

func appendStat(statsFile string, numExisting int, duration time.Duration) error {
	f, err := os.OpenFile(statsFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("open file: %s", err)
	}
	defer f.Close()

	newLine := fmt.Sprintf("%d\t%f\n", numExisting, duration.Seconds())
	if _, err = f.WriteString(newLine); err != nil {
		return fmt.Errorf("write string: %s", err)
	}
	return nil
}

func getIPAddr(appRoute string) (string, error) {
	resp, err := http.Get(appRoute)
	if err != nil {
		return "", fmt.Errorf("get: %s", err)
	}
	defer resp.Body.Close()

	var proxyInfo struct {
		ListenAddresses []string `json:"ListenAddresses"`
	}
	err = json.NewDecoder(resp.Body).Decode(&proxyInfo)
	if err != nil {
		return "", fmt.Errorf("decode json: %s", err)
	}

	if len(proxyInfo.ListenAddresses) < 2 {
		return "", fmt.Errorf("unexpectedly short list of addresses: %+v", proxyInfo.ListenAddresses)
	}

	return proxyInfo.ListenAddresses[1], nil
}

func (t *TestThingy) measureLatencyFor1More(destIP string) (time.Duration, error) {
	_, err := t.tryCheckReachable(destIP, 100, false)
	if err != nil {
		return 0, fmt.Errorf("waiting for pre-state to settle: %s", err)
	}

	oneMorePolicy := []api_v0.Policy{
		api_v0.Policy{
			Source: api_v0.Source{
				ID: t.sourceAppGUID,
			},
			Destination: api_v0.Destination{
				ID:       t.destAppGUID,
				Protocol: "tcp",
				Port:     8080,
			},
		},
	}
	err = t.policyClient.AddPoliciesV0(t.token, oneMorePolicy)
	if err != nil {
		return 0, fmt.Errorf("add policies: %s", err)
	}
	defer t.policyClient.DeletePoliciesV0(t.token, oneMorePolicy)

	return t.tryCheckReachable(destIP, 100, true)
}

func (t *TestThingy) tryCheckReachable(destIP string, numAttempts int, desiredReachability bool) (time.Duration, error) {
	t.logger.Info("waiting-for-reachability", lager.Data{"desired-state": desiredReachability})
	startTime := time.Now()
	for attempt := 0; attempt < numAttempts; attempt++ {
		reachable := t.checkReachable(destIP, desiredReachability)
		if reachable {
			t.logger.Info("achieved-reachability", lager.Data{"desired-state": desiredReachability})
			return time.Since(startTime), nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return 0, fmt.Errorf("app never acheived the desired reachability state of %v", desiredReachability)
}

func (t *TestThingy) checkReachable(destIP string, desiredReachable bool) bool {
	resp, err := http.Get(fmt.Sprintf("%s/proxy/%s:8080", t.proxyAppRoute, destIP))
	defer resp.Body.Close()
	if err != nil {
		return false
	}
	body, _ := ioutil.ReadAll(resp.Body)
	if desiredReachable {
		t.logger.Info("desired-reachable-and-got", lager.Data{"status-code": resp.StatusCode, "body": string(body)})
		return (resp.StatusCode == 200)
	} else {
		t.logger.Info("desired-reachable-and-got", lager.Data{"status-code": resp.StatusCode, "body": string(body)})
		return resp.StatusCode == 500
	}
}
