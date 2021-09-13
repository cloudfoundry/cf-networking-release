package helpers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/gomega"
)

type ConfigurableMockCCServer struct {
	server *httptest.Server

	apps   map[string]struct{}
	spaces map[string]struct{}
}

type resource struct {
	GUID string `json:"guid"`
}

func NewConfigurableMockCCServer() *ConfigurableMockCCServer {
	c := &ConfigurableMockCCServer{
		apps:   make(map[string]struct{}),
		spaces: make(map[string]struct{}),
	}
	c.server = httptest.NewUnstartedServer(c)

	return c
}

func (c *ConfigurableMockCCServer) Start() {
	c.server.Start()
}

func (c *ConfigurableMockCCServer) Close() {
	c.server.Close()
}

func (c *ConfigurableMockCCServer) URL() string {
	return c.server.URL
}

func (c *ConfigurableMockCCServer) AddApp(guid string) {
	c.apps[guid] = struct{}{}
}

func (c *ConfigurableMockCCServer) AddSpace(guid string) {
	c.spaces[guid] = struct{}{}
}

func (c *ConfigurableMockCCServer) DeleteApp(guid string) {
	delete(c.apps, guid)
}

func (c *ConfigurableMockCCServer) DeleteSpace(guid string) {
	delete(c.spaces, guid)
}

func (c *ConfigurableMockCCServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header["Authorization"][0] != "bearer valid-token" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if r.URL.Path == "/v3/apps" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(buildCCResponse(c.apps)))
		return
	}

	if r.URL.Path == "/v3/spaces" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(buildCCResponse(c.spaces)))
		return
	}

	w.WriteHeader(http.StatusTeapot)
	return
}

func buildCCResponse(guids map[string]struct{}) string {
	var resources []resource

	for guid, _ := range guids {
		resources = append(resources, resource{GUID: guid})
	}

	resourceJSON, err := json.Marshal(resources)
	Expect(err).NotTo(HaveOccurred())

	return fmt.Sprintf(`{
		"pagination": {
			"total_results": %d,
			"total_pages": 1
		},
		"resources": %s
	}`, len(guids), string(resourceJSON))
}
