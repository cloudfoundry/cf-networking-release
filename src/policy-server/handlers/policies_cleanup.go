package handlers

import (
	"lib/marshal"
	"net/http"
	"policy-server/models"
	"policy-server/store"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/uua_client.go --fake-name UAAClient . uaaClient
type uaaClient interface {
	GetToken() (string, error)
}

//go:generate counterfeiter -o ../fakes/cc_client.go --fake-name CCClient . ccClient
type ccClient interface {
	GetAllAppGUIDs(token string) (map[string]interface{}, error)
}

type PoliciesCleanup struct {
	Logger    lager.Logger
	Store     store.Store
	Marshaler marshal.Marshaler
	UAAClient uaaClient
	CCClient  ccClient
}

func (h *PoliciesCleanup) ServeHTTP(w http.ResponseWriter, req *http.Request, currentUserName string) {

	policies, err := h.Store.All()
	if err != nil {
		h.Logger.Error("store-list-policies-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "database read failed"}`))
		return
	}

	token, err := h.UAAClient.GetToken()
	if err != nil {
		h.Logger.Error("get-uaa-token-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "get UAA token failed"}`))
		return
	}

	ccAppGuids, err := h.CCClient.GetAllAppGUIDs(token)
	if err != nil {
		h.Logger.Error("cc-get-app-guids-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "get app guids from Cloud-Controller failed"}`))
		return
	}

	stalePolicies := getStalePolicies(policies, ccAppGuids)

	//h.Logger.Info("I am cleaning up policies")
	//ret, err := h.Store.DeleteByGroup(staleAppGuids)

	for i, _ := range stalePolicies {
		stalePolicies[i].Source.Tag = ""
		stalePolicies[i].Destination.Tag = ""
	}

	policyCleanup := struct {
		TotalPolicies int             `json:"total_policies"`
		Policies      []models.Policy `json:"policies"`
	}{len(stalePolicies), stalePolicies}

	bytes, err := h.Marshaler.Marshal(policyCleanup)
	if err != nil {
		h.Logger.Error("marshal-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "marshal response failed"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

func getStalePolicies(policyList []models.Policy, ccList map[string]interface{}) (ret []models.Policy) {
	for _, p := range policyList {
		_, foundSrc := ccList[p.Source.ID]
		_, foundDst := ccList[p.Destination.ID]
		if !foundSrc || !foundDst {
			ret = append(ret, p)
		}
	}
	return ret
}
