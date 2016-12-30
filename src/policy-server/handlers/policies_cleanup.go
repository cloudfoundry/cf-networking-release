package handlers

import (
	"lib/marshal"
	"net/http"
	"policy-server/models"
	"policy-server/store"
	"policy-server/uaa_client"

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

func (h *PoliciesCleanup) ServeHTTP(w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
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

	policyCleanup := struct {
		TotalPolicies int             `json:"total_policies"`
		Policies      []models.Policy `json:"policies"`
	}{len(stalePolicies), stalePolicies}

	h.Logger.Info("deleting stale policies:", lager.Data{"stale_policies": policyCleanup})
	err = h.Store.Delete(stalePolicies)
	if err != nil {
		h.Logger.Error("store-delete-policies-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "database write failed"}`))
		return
	}

	for i, _ := range policyCleanup.Policies {
		policyCleanup.Policies[i].Source.Tag = ""
		policyCleanup.Policies[i].Destination.Tag = ""
	}

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

func getStalePolicies(policyList []models.Policy, ccList map[string]interface{}) (stalePolicies []models.Policy) {
	for _, p := range policyList {
		_, foundSrc := ccList[p.Source.ID]
		_, foundDst := ccList[p.Destination.ID]
		if !foundSrc || !foundDst {
			stalePolicies = append(stalePolicies, p)
		}
	}
	return stalePolicies
}
