package middleware

import (
	"context"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/lager/v3"
)

type Key string

//go:generate counterfeiter -o fakes/uuid_generator.go --fake-name UUIDGenerator . uuidGenerator
type uuidGenerator interface {
	GenerateUUID() (string, error)
}

type LogWrapper struct {
	UUIDGenerator uuidGenerator
}

const LoggerKey = Key("logger")

func (l *LogWrapper) getUUID(r *http.Request) string {
	previousUUID := r.Header.Get("X-VCAP-Request-ID")

	generatedUUID, err := l.UUIDGenerator.GenerateUUID()
	if err == nil {
		if previousUUID != "" {
			return fmt.Sprintf("%s::%s", previousUUID, generatedUUID)
		} else {
			return generatedUUID
		}
	}
	return ""
}

func (l *LogWrapper) LogWrap(logger lager.Logger, wrappedHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestLogger lager.Logger

		sessionName := "request"
		data := lager.Data{
			"method":  r.Method,
			"request": r.URL.String(),
		}

		uuid := l.getUUID(r)
		if uuid != "" {
			data["request_guid"] = uuid
			sessionName = fmt.Sprintf("%s_%s", sessionName, uuid)
		}

		requestLogger = logger.Session(sessionName, data)

		contextWithLogger := context.WithValue(r.Context(), LoggerKey, requestLogger)
		r = r.WithContext(contextWithLogger)

		requestLogger.Debug("serving")
		defer requestLogger.Debug("done")

		w.Header().Add("X-VCAP-Request-ID", uuid)

		wrappedHandler.ServeHTTP(w, r)
	})
}
