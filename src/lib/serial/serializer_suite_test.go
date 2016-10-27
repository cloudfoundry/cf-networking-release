package serial_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSerializer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Serializer Suite")
}
