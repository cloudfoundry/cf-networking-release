package flannel

import (
	"fmt"
	"io/ioutil"
	"regexp"
)

const (
	flannelSubnetRegex  = `FLANNEL_SUBNET=((?:[0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2})`
	flannelNetworkRegex = `FLANNEL_NETWORK=((?:[0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2})`
)

type NetworkInfo struct {
	FlannelSubnetFilePath string
}

func (l *NetworkInfo) DiscoverNetworkInfo() (string, string, error) {
	fileContents, err := ioutil.ReadFile(l.FlannelSubnetFilePath)
	if err != nil {
		return "", "", err
	}

	subnetMatches := regexp.MustCompile(flannelSubnetRegex).FindStringSubmatch(string(fileContents))
	if len(subnetMatches) < 2 {
		return "", "", fmt.Errorf("unable to parse flannel subnet file")
	}

	networkMatches := regexp.MustCompile(flannelNetworkRegex).FindStringSubmatch(string(fileContents))
	if len(networkMatches) < 2 {
		return "", "", fmt.Errorf("unable to parse flannel network from subnet file")
	}

	return subnetMatches[1], networkMatches[1], nil
}
