package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code.cloudfoundry.org/policy-server/api"
	"code.cloudfoundry.org/policy-server/store"
)

type AsgsIndex struct {
	Store         store.SecurityGroupsStore
	Mapper        api.AsgMapper
	ErrorResponse errorResponse
}

func NewAsgsIndex(store store.SecurityGroupsStore, mapper api.AsgMapper, errorResponse errorResponse) *AsgsIndex {
	return &AsgsIndex{
		Store:         store,
		Mapper:        mapper,
		ErrorResponse: errorResponse,
	}
}

func (h *AsgsIndex) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("index-security-group-rules")
	queryValues := req.URL.Query()
	spaceGuids := parseSpaceGuids(queryValues)
	from, err := parseIntQueryValue(queryValues, "from")
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "invalid value for 'from' parameter")
		return
	}
	limit, err := parseIntQueryValue(queryValues, "limit")
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "invalid value for 'limit' parameter")
		return
	}

	asgs, pagination, err := h.Store.BySpaceGuids(spaceGuids, store.Page{From: from, Limit: limit})
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "database read failed")
		return
	}
	bytes, err := h.Mapper.AsBytes(asgs, pagination)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "map asgs as bytes failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	// #nosec G104 - ignore errors writing http responses to avoid spamming logs during a DoS
	w.Write(bytes)
}

func parseSpaceGuids(queryValues url.Values) []string {
	var guids []string
	guidList, ok := queryValues["space_guids"]
	if ok {
		guids = strings.Split(guidList[0], ",")
	}
	return guids
}

func parseIntQueryValue(queryValues url.Values, name string) (int, error) {
	valStr, ok := queryValues[name]
	if ok {
		return strconv.Atoi(valStr[0])
	}
	return 0, nil
}
