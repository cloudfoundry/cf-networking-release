package cleaner

//go:generate counterfeiter -generate

import (
	"fmt"

	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/policy-server/cc_client"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/uaa_client"
)

//counterfeiter:generate -o fakes/policy_store.go --fake-name PolicyStore . policyStore
type policyStore interface {
	All() ([]store.Policy, error)
	Delete([]store.Policy) error
}

type PolicyCleaner struct {
	Logger                lager.Logger
	Store                 policyStore
	UAAClient             uaa_client.UAAClient
	CCClient              cc_client.CCClient
	CCAppRequestChunkSize int
}

func NewPolicyCleaner(logger lager.Logger, store policyStore, uaaClient uaa_client.UAAClient,
	ccClient cc_client.CCClient, ccAppRequestChunkSize int) *PolicyCleaner {
	return &PolicyCleaner{
		Logger:                logger,
		Store:                 store,
		UAAClient:             uaaClient,
		CCClient:              ccClient,
		CCAppRequestChunkSize: ccAppRequestChunkSize,
	}
}

func (p *PolicyCleaner) DeleteStalePolicies() ([]store.Policy, error) {
	policies, err := p.Store.All()
	if err != nil {
		p.Logger.Error("store-list-policies-failed", err)
		return []store.Policy{}, fmt.Errorf("database read failed for c2c policies: %s", err)
	}

	token, err := p.UAAClient.GetToken()
	if err != nil {
		p.Logger.Error("get-uaa-token-failed", err)
		return []store.Policy{}, fmt.Errorf("get UAA token failed: %s", err)
	}

	policiesToDelete, err := p.getC2CPoliciesToDelete(policies, token)
	if err != nil {
		return []store.Policy{}, err
	}

	p.Logger.Info("deleting stale policies:", lager.Data{
		"total_c2c_policies": len(policiesToDelete),
		"stale_c2c_policies": policiesToDelete,
	})
	err = p.Store.Delete(policiesToDelete)
	if err != nil {
		p.Logger.Error("store-delete-policies-failed", err)
		return []store.Policy{}, fmt.Errorf("database write failed: %s", err)
	}

	return policiesToDelete, nil
}

func (p *PolicyCleaner) DeleteStalePoliciesWrapper() error {
	_, err := p.DeleteStalePolicies()
	return err
}

func (p *PolicyCleaner) getC2CPoliciesToDelete(policies []store.Policy, token string) ([]store.Policy, error) {
	var c2cPoliciesToDelete []store.Policy

	appGUIDs := policyAppGUIDs(policies)
	appGUIDchunks := getChunks(appGUIDs, p.CCAppRequestChunkSize)

	for _, appGUIDchunk := range appGUIDchunks {
		liveAppGUIDs, err := p.CCClient.GetLiveAppGUIDs(token, appGUIDchunk)
		if err != nil {
			p.Logger.Error("cc-get-app-guids-failed", err)
			return nil, fmt.Errorf("get app guids from Cloud-Controller failed: %s", err)
		}

		staleAppGUIDs := getStaleAppGUIDs(liveAppGUIDs, appGUIDchunk)
		toDelete := getStalePolicies(policies, staleAppGUIDs)

		c2cPoliciesToDelete = append(c2cPoliciesToDelete, toDelete...)
	}

	return c2cPoliciesToDelete, nil
}

func getStaleAppGUIDs(liveAppGUIDs map[string]struct{}, appGUIDs []string) map[string]struct{} {
	staleAppGUIDs := make(map[string]struct{})
	for _, guid := range appGUIDs {
		if _, ok := liveAppGUIDs[guid]; !ok {
			staleAppGUIDs[guid] = struct{}{}
		}
	}
	return staleAppGUIDs
}

func getStalePolicies(policyList []store.Policy, staleAppGUIDs map[string]struct{}) []store.Policy {
	var stalePolicies []store.Policy
	for _, p := range policyList {
		_, foundSrc := staleAppGUIDs[p.Source.ID]
		_, foundDst := staleAppGUIDs[p.Destination.ID]
		if foundSrc || foundDst {
			stalePolicies = append(stalePolicies, p)
		}
	}
	return stalePolicies
}

func policyAppGUIDs(policyList []store.Policy) []string {
	appGUIDset := make(map[string]struct{})
	for _, p := range policyList {
		appGUIDset[p.Source.ID] = struct{}{}
		appGUIDset[p.Destination.ID] = struct{}{}
	}
	var appGUIDs []string
	for guid, _ := range appGUIDset {
		appGUIDs = append(appGUIDs, guid)
	}
	return appGUIDs
}

func getChunks(appGuids []string, chunkSize int) [][]string {
	if chunkSize < 1 {
		chunkSize = 100
	}
	var chunks [][]string

	for i := 0; i < len(appGuids); i += chunkSize {
		last := i + chunkSize
		if last > len(appGuids) {
			last = len(appGuids)
		}
		chunks = append(chunks, appGuids[i:last])
	}
	return chunks
}
