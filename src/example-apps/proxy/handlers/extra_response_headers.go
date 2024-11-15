package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type ExtraResponseHeadersHandler struct {
}

func (h *ExtraResponseHeadersHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	extraHeaders := strings.TrimPrefix(req.URL.Path, "/extraresponseheaders/")
	extraHeaders = strings.TrimSuffix(extraHeaders, "/")
	numExtraHeaders, err := strconv.Atoi(extraHeaders)

	if err != nil {
		logger.Println("You must pass an int in the URL. For example: 'URL/extraheaders/10'.")
		resp.WriteHeader(http.StatusBadRequest)
	}

	var header string
	for i := 0; i < numExtraHeaders; i++ {
		header = fmt.Sprintf("meow-%d", i)
		resp.Header().Set(header, header)
	}
}
