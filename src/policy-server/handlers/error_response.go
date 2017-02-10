package handlers

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/lager"
)

type ErrorResponse struct {
	Logger        lager.Logger
	MetricsSender metricsSender
}

func (e *ErrorResponse) InternalServerError(w http.ResponseWriter, err error, message, description string) {
	e.Logger.Error(fmt.Sprintf("%s: %s", message, description), err)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(fmt.Sprintf(`{"error": "%s: %s"}`, message, description)))
	e.MetricsSender.IncrementCounter(message)
}
