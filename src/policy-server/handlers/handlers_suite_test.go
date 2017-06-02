package handlers_test

import (
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	"testing"
)

func TestHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handlers Suite")
}

func LogsWith(level lager.LogLevel, msg string) types.GomegaMatcher {
	return And(
		WithTransform(func(log lager.LogFormat) string {
			return log.Message
		}, Equal(msg)),
		WithTransform(func(log lager.LogFormat) lager.LogLevel {
			return log.LogLevel
		}, Equal(level)),
	)
}

func HaveLogData(nextMatcher types.GomegaMatcher) types.GomegaMatcher {
	return WithTransform(func(log lager.LogFormat) lager.Data {
		return log.Data
	}, nextMatcher)
}
