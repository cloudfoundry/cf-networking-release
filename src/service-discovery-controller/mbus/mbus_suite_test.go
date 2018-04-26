package mbus_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/nats-io/gnatsd/server"
	gnatsd "github.com/nats-io/gnatsd/test"
	"testing"
)

func TestMbus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mbus Suite")
}

func RunServerOnPort(port int) *server.Server {
	opts := gnatsd.DefaultTestOptions
	opts.Port = port
	opts.Username = "username"
	opts.Password = "password"
	return gnatsd.RunServer(&opts)
}
