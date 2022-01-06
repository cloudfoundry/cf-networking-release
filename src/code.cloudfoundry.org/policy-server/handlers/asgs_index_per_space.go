package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/policy-server/store"
)

type AsgsPerSpaceIndex struct {
	Store         store.SecurityGroupsStore
	Marshaler     marshal.Marshaler
	ErrorResponse errorResponse
}

func NewAsgsPerSpaceIndex(store store.SecurityGroupsStore, marshaler marshal.Marshaler, errorResponse errorResponse) *AsgsPerSpaceIndex {
	return &AsgsPerSpaceIndex{
		Store:         store,
		Marshaler:     marshaler,
		ErrorResponse: errorResponse,
	}
}

func (h *AsgsPerSpaceIndex) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("index-security-group-rules-per-space")
	// queryValues := req.URL.Query()
	// spaceGuids := parseSpaceGuids(queryValues)

	// var asgsBySpace store.SecurityGroupRulesBySpace
	// var err error
	// if len(spaceGuids) > 0 {
	// asgsBySpace, err = h.Store.BySpaceGuids(spaceGuids)
	// } else {
	// asgsBySpace, err := h.Store.All()
	// }

	// if err != nil {
	// 	h.ErrorResponse.InternalServerError(logger, w, err, "database read failed")
	// 	return
	// }

	// bytes, err := h.Marshaler.Marshal(asgsBySpace)
	// if err != nil {
	// 	h.ErrorResponse.InternalServerError(logger, w, err, "map asgs as bytes failed")
	// 	return
	// }

	// bytes, err := h.Mapper.AsBytes(policies)
	// if err != nil {
	// 	h.ErrorResponse.InternalServerError(logger, w, err, "map policy as bytes failed")
	// 	return
	// }

	w.WriteHeader(http.StatusOK)
	// w.Write(bytes)
}

func parseSpaceGuids(queryValues url.Values) []string {
	var guids []string
	guidList, ok := queryValues["space_guids"]
	if ok {
		guids = strings.Split(guidList[0], ",")
	}
	return guids
}
