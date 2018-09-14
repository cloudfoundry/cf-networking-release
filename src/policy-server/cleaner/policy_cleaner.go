package cleaner

import (
	"fmt"
	"policy-server/store"
	"time"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/uua_client.go --fake-name UAAClient . uaaClient
type uaaClient interface {
	GetToken() (string, error)
}

//go:generate counterfeiter -o fakes/cc_client.go --fake-name CCClient . ccClient
type ccClient interface {
	GetLiveAppGUIDs(token string, appGUIDs []string) (map[string]struct{}, error)
	GetLiveSpaceGUIDs(token string, spaceGUIDs []string) (map[string]struct{}, error)
}

//go:generate counterfeiter -o fakes/policy_store.go --fake-name PolicyStore . policyStore
type policyStore interface {
	All() ([]store.Policy, error)
	Delete([]store.Policy) error
}

//go:generate counterfeiter -o fakes/egress_policy_store.go --fake-name EgressPolicyStore . egressPolicyStore
type egressPolicyStore interface {
	All() ([]store.EgressPolicy, error)
	Delete([]store.EgressPolicy) error
}

type PolicyCleaner struct {
	Logger                lager.Logger
	Store                 policyStore
	EgressStore           egressPolicyStore
	UAAClient             uaaClient
	CCClient              ccClient
	CCAppRequestChunkSize int
	RequestTimeout        time.Duration
}

func NewPolicyCleaner(logger lager.Logger, store policyStore, egressStore egressPolicyStore, uaaClient uaaClient,
	ccClient ccClient, ccAppRequestChunkSize int, requestTimeout time.Duration) *PolicyCleaner {
	return &PolicyCleaner{
		Logger:                logger,
		Store:                 store,
		EgressStore:           egressStore,
		UAAClient:             uaaClient,
		CCClient:              ccClient,
		CCAppRequestChunkSize: ccAppRequestChunkSize,
		RequestTimeout:        requestTimeout,
	}
}

func (p *PolicyCleaner) DeleteStalePolicies() ([]store.Policy, []store.EgressPolicy, error) {
	policies, err := p.Store.All()
	if err != nil {
		p.Logger.Error("store-list-policies-failed", err)
		return []store.Policy{}, []store.EgressPolicy{}, fmt.Errorf("database read failed for c2c policies: %s", err)
	}

	egressPolicies, err := p.EgressStore.All()
	if err != nil {
		p.Logger.Error("store-list-policies-failed", err)
		return []store.Policy{}, []store.EgressPolicy{}, fmt.Errorf("database read failed for egress policies: %s", err)
	}

	token, err := p.UAAClient.GetToken()
	if err != nil {
		p.Logger.Error("get-uaa-token-failed", err)
		return []store.Policy{}, []store.EgressPolicy{}, fmt.Errorf("get UAA token failed: %s", err)
	}

	policiesToDelete, err := p.getC2CPoliciesToDelete(policies, token)
	if err != nil {
		return []store.Policy{}, []store.EgressPolicy{}, err
	}

	egressPoliciesToDelete, err := p.getEgressPoliciesToDelete(egressPolicies, token)
	if err != nil {
		return []store.Policy{}, []store.EgressPolicy{}, err
	}

	p.Logger.Info("deleting stale policies:", lager.Data{
		"total_c2c_policies":    len(policiesToDelete),
		"stale_c2c_policies":    policiesToDelete,
		"total_egress_policies": len(egressPoliciesToDelete),
		"stale_egress_policies": egressPoliciesToDelete,
	})
	err = p.Store.Delete(policiesToDelete)
	if err != nil {
		p.Logger.Error("store-delete-policies-failed", err)
		return []store.Policy{}, []store.EgressPolicy{}, fmt.Errorf("database write failed: %s", err)
	}

	err = p.EgressStore.Delete(egressPoliciesToDelete)
	if err != nil {
		p.Logger.Error("egress-store-delete-policies-failed", err)
		return []store.Policy{}, []store.EgressPolicy{}, fmt.Errorf("database write failed: %s", err)
	}

	return policiesToDelete, egressPoliciesToDelete, nil
}

func (p *PolicyCleaner) DeleteStalePoliciesWrapper() error {
	_, _, err := p.DeleteStalePolicies()
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

func (p *PolicyCleaner) getEgressPoliciesToDelete(egressPolicies []store.EgressPolicy, token string) ([]store.EgressPolicy, error) {
	var spaceEgressPolicyGUIDs, appEgressPolicyGUIDs []string
	spaceEgressPolicies := make(map[string][]store.EgressPolicy)
	var egressPoliciesToDelete []store.EgressPolicy
	appEgressPolicies := make(map[string][]store.EgressPolicy)

	for _, egressPolicy := range egressPolicies {
		if egressPolicy.Source.Type == "space" {
			spaceEgressPolicyGUIDs = append(spaceEgressPolicyGUIDs, egressPolicy.Source.ID)
			spaceEgressPolicies[egressPolicy.Source.ID] = append(spaceEgressPolicies[egressPolicy.Source.ID], egressPolicy)
		}
		if egressPolicy.Source.Type == "app" {
			appEgressPolicyGUIDs = append(appEgressPolicyGUIDs, egressPolicy.Source.ID)
			appEgressPolicies[egressPolicy.Source.ID] = append(appEgressPolicies[egressPolicy.Source.ID], egressPolicy)
		}
	}

	appGUIDchunks := getChunks(appEgressPolicyGUIDs, p.CCAppRequestChunkSize)

	for _, appGUIDchunk := range appGUIDchunks {
		liveAppGUIDs, err := p.CCClient.GetLiveAppGUIDs(token, appGUIDchunk)
		if err != nil {
			p.Logger.Error("cc-get-app-guids-failed", err)
			return nil, fmt.Errorf("get app guids from Cloud-Controller failed: %s", err)
		}

		staleAppGUIDs := getStaleAppGUIDs(liveAppGUIDs, appGUIDchunk)
		egressPoliciesToDelete = append(egressPoliciesToDelete, getStaleEgressAppPolicies(appEgressPolicies, staleAppGUIDs)...)
	}

	liveSpaceGUIDs, err := p.CCClient.GetLiveSpaceGUIDs(token, spaceEgressPolicyGUIDs)
	if err != nil {
		p.Logger.Error("get-live-space-guids-failed", err)
		return nil, fmt.Errorf("get live space guids failed: %s", err)
	}
	egressPoliciesToDelete = append(egressPoliciesToDelete, getStaleEgressSpacePolicies(spaceEgressPolicies, liveSpaceGUIDs)...)
	return egressPoliciesToDelete, nil
}

func getStaleEgressSpacePolicies(spacePolicies map[string][]store.EgressPolicy, liveSpaceGUIDs map[string]struct{}) []store.EgressPolicy {
	var staleSpaceEgressPolicies []store.EgressPolicy
	for spaceGUID := range liveSpaceGUIDs {
		delete(spacePolicies, spaceGUID)
	}
	for _, policies := range spacePolicies {
		staleSpaceEgressPolicies = append(staleSpaceEgressPolicies, policies...)
	}

	return staleSpaceEgressPolicies
}

func getStaleEgressAppPolicies(appPolicies map[string][]store.EgressPolicy, staleAppGUIDs map[string]struct{}) []store.EgressPolicy {
	var staleAppEgressPolicies []store.EgressPolicy
	for appGUID := range staleAppGUIDs {
		staleAppEgressPolicies = append(staleAppEgressPolicies, appPolicies[appGUID]...)
	}

	return staleAppEgressPolicies
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
