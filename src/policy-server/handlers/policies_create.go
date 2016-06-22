package handlers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"lib/marshal"
	"net/http"
	"policy-server/models"
	"policy-server/store"

	"github.com/pivotal-golang/lager"
)

type PoliciesCreate struct {
	Logger      lager.Logger
	Store       store.Store
	Unmarshaler marshal.Unmarshaler
	Marshaler   marshal.Marshaler
}

func (h *PoliciesCreate) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		h.Logger.Error("body-read-failed", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request body format passed to API should be JSON"}`))
		return
	}

	var payload struct {
		Policies []models.Policy `json:"policies"`
	}
	err = h.Unmarshaler.Unmarshal(bodyBytes, &payload)
	if err != nil {
		h.Logger.Error("unmarshal-failed", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid values passed to API"}`))
		return
	}

	if len(payload.Policies) == 0 {
		h.Logger.Error("bad-request", errors.New("missing policies"))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "missing policies"}`))
		return
	}

	if err = validateFields(payload.Policies); err != nil {
		h.Logger.Error("bad-request", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err)))
		return
	}

	err = h.Store.Create(payload.Policies)
	if err != nil {
		h.Logger.Error("store-create-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "database create failed"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
	return
}

func validateFields(policies []models.Policy) error {
	for _, policy := range policies {
		if policy.Source.ID == "" {
			return errors.New("missing source id")
		}
		if policy.Destination.ID == "" {
			return errors.New("missing destination id")
		}
		if policy.Destination.Protocol == "" {
			return errors.New("missing destination protocol")
		}
		if policy.Destination.Port == 0 {
			return errors.New("missing destination port")
		}
	}
	return nil
}
