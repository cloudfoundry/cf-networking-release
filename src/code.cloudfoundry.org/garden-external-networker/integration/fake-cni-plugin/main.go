package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/020"
)

func parseEnviron(pairs []string) (map[string]string, error) {
	hash := make(map[string]string)
	for i, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("can't parse env var %d: %s", i, p)
		}
		hash[parts[0]] = parts[1]
	}
	return hash, nil
}

type LogInfo struct {
	Args  []string
	Env   map[string]string
	Stdin string
}

func main() {
	const logDirEnvVar = "FAKE_LOG_DIR"
	logDir := os.Getenv(logDirEnvVar)
	if logDir == "" {
		log.Fatalf("missing required arg %q", logDirEnvVar)
	}

	stdin, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("error reading stdin bytes: %s", err)
	}

	env, err := parseEnviron(os.Environ())
	if err != nil {
		log.Fatalf("unable to parse environment: %s", err)
	}

	args := os.Args

	logInfo := LogInfo{
		Args:  args,
		Env:   env,
		Stdin: string(stdin),
	}

	logBytes, err := json.Marshal(logInfo)
	if err != nil {
		log.Fatalf("unable to json marshal log info")
	}

	logFilePath := filepath.Join(logDir, filepath.Base(strings.TrimSuffix(args[0], filepath.Ext(args[0])))+".log")
	err = ioutil.WriteFile(logFilePath, logBytes, 0600)
	if err != nil {
		log.Fatalf("unable to write log file: %s", err)
	}

	nameservers := []string{"1.2.3.4"}
	if os.Getenv("FAKE_CNI_DEBUG") == "no_dns_result" {
		nameservers = []string{}
	}

	result := &types020.Result{
		IP4: &types020.IPConfig{
			IP: net.IPNet{
				IP:   net.ParseIP("169.254.1.2"),
				Mask: net.IPv4Mask(255, 255, 255, 0),
			},
		},
		DNS: types.DNS{
			Nameservers: nameservers,
		},
	}

	outputBytes, err := json.Marshal(result)
	if err != nil {
		log.Fatalf("unable to json marshal result data: %s", err)
	}

	_, err = os.Stdout.Write(outputBytes)
	if err != nil {
		log.Fatalf("unable to write result to stdout: %s", err)
	}

}
