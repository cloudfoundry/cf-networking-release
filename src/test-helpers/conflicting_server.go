package testhelpers

import (
	"net/http"
	"fmt"
	. "github.com/onsi/gomega"
)

func LaunchConflictingServer(port int) *http.Server {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	conflictingServer := &http.Server{Addr: address}
	go func() { conflictingServer.ListenAndServe() }()
	client := &http.Client{}
	Eventually(func() bool {
		resp, err := client.Get(fmt.Sprintf("http://%s", address))
		if err != nil {
			return false
		}
		return resp.StatusCode == 404
	}).Should(BeTrue())
	return conflictingServer
}
