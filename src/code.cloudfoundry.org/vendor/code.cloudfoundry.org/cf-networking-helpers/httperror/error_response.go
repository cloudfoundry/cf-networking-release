package httperror

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager/v3"
)

const HTTP_ERROR_METRIC_NAME = "http_error"

//go:generate counterfeiter -o ../fakes/metrics_sender.go --fake-name MetricsSender . metricsSender
type metricsSender interface {
	SendDuration(string, time.Duration)
	IncrementCounter(string)
}

type ErrorResponse struct {
	MetricsSender metricsSender
}

type MetadataError struct {
	metadata      map[string]interface{}
	internalError error
}

func NewMetadataError(internalError error, metadata map[string]interface{}) MetadataError {
	return MetadataError{
		internalError: internalError,
		metadata:      metadata,
	}
}

func (m MetadataError) Error() string {
	return m.internalError.Error()
}

func (m MetadataError) Metadata() map[string]interface{} {
	return m.metadata
}

func (e *ErrorResponse) InternalServerError(logger lager.Logger, w http.ResponseWriter, err error, description string) {
	e.respondWithCode(http.StatusInternalServerError, logger, w, err, description)
}

func (e *ErrorResponse) BadRequest(logger lager.Logger, w http.ResponseWriter, err error, description string) {
	e.respondWithCode(http.StatusBadRequest, logger, w, err, description)
}

func (e *ErrorResponse) Forbidden(logger lager.Logger, w http.ResponseWriter, err error, description string) {
	e.respondWithCode(http.StatusForbidden, logger, w, err, description)
}

func (e *ErrorResponse) Unauthorized(logger lager.Logger, w http.ResponseWriter, err error, description string) {
	w.Header().Add("WWW-Authenticate", "Bearer")
	e.respondWithCode(http.StatusUnauthorized, logger, w, err, description)
}

func (e *ErrorResponse) NotFound(logger lager.Logger, w http.ResponseWriter, err error, description string) {
	e.respondWithCode(http.StatusNotFound, logger, w, err, description)
}

func (e *ErrorResponse) Conflict(logger lager.Logger, w http.ResponseWriter, err error, description string) {
	e.respondWithCode(http.StatusConflict, logger, w, err, description)
}

func (e *ErrorResponse) NotAcceptable(logger lager.Logger, w http.ResponseWriter, err error, description string) {
	e.respondWithCode(http.StatusNotAcceptable, logger, w, err, description)
}

func (e *ErrorResponse) respondWithCode(statusCode int, logger lager.Logger, w http.ResponseWriter, err error, description string) {
	logger.Error(fmt.Sprintf("%s", description), err)
	w.WriteHeader(statusCode)
	if metadataError, ok := err.(MetadataError); ok {
		j, _ := json.Marshal(metadataError.Metadata())
		w.Write([]byte(fmt.Sprintf(`{"error": "%s", "metadata": %s}`, description, j)))
	} else {
		w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, description)))
	}
	e.MetricsSender.IncrementCounter(HTTP_ERROR_METRIC_NAME)
}
