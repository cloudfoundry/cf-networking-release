package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/tedsuo/rata"
)

type CORSOptionsWrapper struct {
	RataRoutes         rata.Routes
	AllowedCORSDomains []string
}

func (c CORSOptionsWrapper) Wrap(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "OPTIONS" {
			methods := []string{}
			for _, route := range c.RataRoutes {
				match, err := c.matchRoute(route.Path, req.URL.Path)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				if match {
					methods = append(methods, route.Method)
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
			w.Header().Set("Access-Control-Allow-Headers", "authorization")
		}
		if ok, allowedOrigin := c.allowedOrigin(req.Header["Origin"]); ok {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}

		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
		handler.ServeHTTP(w, req)
	})
}

func (c CORSOptionsWrapper) matchRoute(rataPath, requestPath string) (bool, error) {
	pathReplacer := regexp.MustCompile("\\:\\w+")
	pathPattern := pathReplacer.ReplaceAll([]byte(rataPath), []byte("\\w+"))
	return regexp.Match(fmt.Sprintf("^%s$", pathPattern), []byte(requestPath))
}

func (c CORSOptionsWrapper) allowedOrigin(requestOrigins []string) (bool, string) {
	if len(requestOrigins) < 1 {
		return false, ""
	}
	requestOrigin := requestOrigins[0]
	for _, allowedOrigin := range c.AllowedCORSDomains {
		if allowedOrigin == requestOrigin || allowedOrigin == "*" {
			return true, allowedOrigin
		}
	}
	return false, ""
}
