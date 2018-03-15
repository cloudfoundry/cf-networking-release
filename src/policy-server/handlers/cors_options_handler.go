package handlers

import (
	"net/http"
	"strings"

	"github.com/tedsuo/rata"
)

type CORSOptionsWrapper struct {
	RataRoutes       rata.Routes
	AllowCORSDomains []string
}

func (c CORSOptionsWrapper) Wrap(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "OPTIONS" {
			methods := []string{}
			for _, route := range c.RataRoutes {
				if route.Path == req.URL.Path {
					methods = append(methods, route.Method)
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
		}

		w.Header().Set("Access-Control-Allow-Origin", strings.Join(c.AllowCORSDomains, ","))
		handler.ServeHTTP(w, req)
	})
}
