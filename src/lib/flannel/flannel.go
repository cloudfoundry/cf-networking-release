package flannel

import (
	"fmt"
	"io/ioutil"
	"regexp"
)

const (
	flannelRegex = `FLANNEL_SUBNET=((?:[0-9]{1,3}\.){3}[0-9]{1,3}/24)`
)

type LocalSubnet struct {
	FlannelSubnetFilePath string
}

func (l *LocalSubnet) DiscoverLocalSubnet() (string, error) {
	fileContents, err := ioutil.ReadFile(l.FlannelSubnetFilePath)
	if err != nil {
		return "", err
	}

	matches := regexp.MustCompile(flannelRegex).FindStringSubmatch(string(fileContents))
	if len(matches) < 2 {
		return "", fmt.Errorf("unable to parse flannel subnet file")
	}

	return matches[1], nil
}
