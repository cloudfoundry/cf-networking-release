package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/policy-server/handlers"
	"code.cloudfoundry.org/policy-server/uaa_client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
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

func MakeRequestWithAuth(handler func(http.ResponseWriter, *http.Request), resp http.ResponseWriter, request *http.Request, token uaa_client.CheckTokenResponse) {
	contextWithTokenData := context.WithValue(request.Context(), handlers.TokenDataKey, token)
	request = request.WithContext(contextWithTokenData)
	handler(resp, request)
}

func MakeRequestWithLogger(handler func(http.ResponseWriter, *http.Request), resp http.ResponseWriter, request *http.Request, logger *lagertest.TestLogger) {
	contextWithLogger := context.WithValue(request.Context(), middleware.Key("logger"), logger)
	request = request.WithContext(contextWithLogger)
	handler(resp, request)
}

func MakeRequestWithLoggerAndAuth(handler func(http.ResponseWriter, *http.Request), resp http.ResponseWriter, request *http.Request, logger *lagertest.TestLogger, token uaa_client.CheckTokenResponse) {
	contextWithLogger := context.WithValue(request.Context(), middleware.Key("logger"), logger)
	request = request.WithContext(contextWithLogger)

	contextWithTokenData := context.WithValue(request.Context(), handlers.TokenDataKey, token)
	request = request.WithContext(contextWithTokenData)

	handler(resp, request)
}
