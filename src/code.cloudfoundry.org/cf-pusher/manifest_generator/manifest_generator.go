package manifest_generator

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type ManifestGenerator struct{}

func (m *ManifestGenerator) Generate(o interface{}) (string, error) {
	// TODO: improve test coverage/refactor this
	tmpfile, err := ioutil.TempFile("", "manifest")
	if err != nil {
		return "", err // Not tested
	}

	content, err := yaml.Marshal(o)
	if err != nil {
		return "", err // Not tested
	}

	if _, err := tmpfile.Write(content); err != nil {
		return "", err // Not tested
	}

	if err := tmpfile.Close(); err != nil {
		return "", err // Not tested
	}

	return tmpfile.Name(), nil

}
