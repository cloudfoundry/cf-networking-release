package version

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"

	"code.cloudfoundry.org/cli/plugin"
)

type Getter struct {
	Filename string
}

var versionRegex = regexp.MustCompile(`\d+.\d+.\d+`)
var numberRegex = regexp.MustCompile(`\d+`)

func (g *Getter) Get() (plugin.VersionType, error) {

	if _, err := os.Stat(g.Filename); err != nil {
		return plugin.VersionType{}, fmt.Errorf("file does not exist: %s", err)
	}

	jsonBytes, err := ioutil.ReadFile(g.Filename)
	if err != nil {
		return plugin.VersionType{}, fmt.Errorf("reading version: %s", err) // not tested
	}

	if !versionRegex.Match(jsonBytes) {
		return plugin.VersionType{}, fmt.Errorf("invalid version: %s", jsonBytes)
	}

	numbers := numberRegex.FindAll(jsonBytes, 3)
	var major, minor, build int
	if major, err = strconv.Atoi(string(numbers[0])); err != nil {
		return plugin.VersionType{}, fmt.Errorf("invalid major number: %s", string(numbers[0]))
	}
	if minor, err = strconv.Atoi(string(numbers[1])); err != nil {
		return plugin.VersionType{}, fmt.Errorf("invalid minor number: %s", string(numbers[1]))
	}
	if build, err = strconv.Atoi(string(numbers[2])); err != nil {
		return plugin.VersionType{}, fmt.Errorf("invalid build number: %s", string(numbers[2]))
	}

	return plugin.VersionType{
		Major: major,
		Minor: minor,
		Build: build,
	}, nil
}
