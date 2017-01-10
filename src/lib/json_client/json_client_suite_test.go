package json_client_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestJsonClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "JsonClient Suite")
}
