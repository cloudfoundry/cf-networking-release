package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SpamPath          = "/spam/"
	ProxyBaseURLField = "PROXY_URL"
	httpTimeoutInSec  = 7
)

type ErrorResp struct {
	Err string `json:"error"`
}

type SpamResp struct {
	SuccessCount int `json:"success_count"`
}

func SpamHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("begin-spam")

	callCount, err := requestedCallCount(req.URL.Path)

	if err != nil {
		writeError(w, err)
		log.Println("spam-failed-parsing-requested-call-count")
		return
	}

	proxyBaseURL := os.Getenv(ProxyBaseURLField)
	successfullCallCount := successfulProxyCallsCount(proxyBaseURL, callCount)

	writeSpamResponse(w, successfullCallCount)

	log.Println("finish-spam")
}

func requestedCallCount(path string) (int, error) {
	chunks := strings.Split(path, SpamPath)
	return strconv.Atoi(chunks[len(chunks)-1])
}

func writeSpamResponse(w http.ResponseWriter, successfulCallCount int) {
	resp := &SpamResp{SuccessCount: successfulCallCount}
	writeResp(w, resp)
}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)

	resp := &ErrorResp{Err: fmt.Sprintf("Error: %s", err)}
	writeResp(w, resp)
}

func successfulProxyCallsCount(proxyBaseURL string, requestedCallCount int) int {
	count := 0
	client := http.Client{
		// Timeout is required since the client retries to open connections
		// even though they were once refused.
		Timeout: httpTimeoutInSec * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	caller := proxyCaller(client)

	var wg sync.WaitGroup

	for i := 0; i < requestedCallCount; i++ {
		wg.Add(1)
		go caller(proxyBaseURL, &count, &wg)
	}

	wg.Wait()

	return count
}

func proxyCaller(client http.Client) func(proxyBaseURL string, count *int, wg *sync.WaitGroup) {
	return func(proxyBaseURL string, count *int, wg *sync.WaitGroup) {
		defer wg.Done()

		resp, err := client.Get(proxyBaseURL)

		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Expected exit code %d but received %d", http.StatusOK, resp.StatusCode)
			return
		}

		*count++
	}
}

func writeResp(w http.ResponseWriter, resp interface{}) {
	rawResp, err := json.Marshal(resp)

	if err != nil {
		log.Fatalf("Error while marshalling: %s", err)
	}

	fmt.Println(string(rawResp))
	// #nosec G104 - ignore error writing http response to avoid spamming logs on a DoS
	w.Write(rawResp)
}
