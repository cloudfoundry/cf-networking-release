/*
Package network provides an HTTP network abstraction that is bound to the
request/response cycle of commands to the UAA service. The requests and
responses that it consumes and produces are particular to that service,
although they may have some common overlap with JSON HTTP API requests
for other services.

Example

Here is an example request/response to show how the library works:

	package main

	import (
		"log"
		"net/http"
		"net/http/httptest"

		"github.com/pivotal-cf-experimental/warrant/internal/network"
	)

	func main() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusTeapot)
			w.Write([]byte("{}"))
		}))
		client := network.NewClient(network.Config{
			Host: server.URL,
		})

		response, err := client.MakeRequest(network.Request{
			Method:                "GET",
			Path:                  "/banana",
			Authorization:         network.NewBasicAuthorization("username", "password"),
			AcceptableStatusCodes: []int{http.StatusTeapot},
		})
		if err != nil {
			log.Fatalf("request failed: %s", err)
		}

		log.Printf("%#v\n", response)
		//	network.Response{
		//	  Code: 418,
		//	  Body: []uint8{0x7b, 0x7d},
		//	  Headers: http.Header{
		//		"Content-Type": []string{"text/plain; charset=utf-8"},
		//		"Date": []string{"Tue, 07 Jul 2015 00:46:30 GMT"},
		//		"Content-Length": []string{"2"},
		//	  },
		//	}
	}
*/
package network
