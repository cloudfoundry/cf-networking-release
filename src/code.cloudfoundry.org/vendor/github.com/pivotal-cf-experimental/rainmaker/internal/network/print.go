package network

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

func (c Client) printRequest(request *http.Request) {
	if c.config.TraceWriter != nil {
		logger := log.New(c.config.TraceWriter, "", 0)

		bodyCopy := bytes.NewBuffer([]byte{})
		if request.Body != nil {
			body := bytes.NewBuffer([]byte{})
			_, err := io.Copy(io.MultiWriter(body, bodyCopy), request.Body)
			if err != nil {
				panic(err)
			}

			request.Body = ioutil.NopCloser(body)
		}

		logger.Printf("REQUEST: %s %s %s %v\n", request.Method, request.URL, bodyCopy.String(), request.Header)
	}
}

func (c Client) printResponse(resp Response) {
	if c.config.TraceWriter != nil {
		logger := log.New(c.config.TraceWriter, "", 0)

		logger.Printf("RESPONSE: %d %s %+v\n", resp.Code, resp.Body, resp.Headers)
	}
}
