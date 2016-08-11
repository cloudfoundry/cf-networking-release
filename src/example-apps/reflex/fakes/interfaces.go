package fakes

import "net/http"

//go:generate counterfeiter -o ../fakes/response_writer.go --fake-name ResponseWriter . responseWriter
type responseWriter interface {
	http.ResponseWriter
}
