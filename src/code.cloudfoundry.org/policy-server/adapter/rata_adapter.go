package adapter

import (
	"net/http"

	"github.com/tedsuo/rata"
)

type RataAdapter struct{}

func (RataAdapter) Param(req *http.Request, name string) string {
	return rata.Param(req, name)
}
