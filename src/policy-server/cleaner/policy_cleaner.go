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

//go:generate counterfeiter -o fakes/list_delete_store.go --fake-name ListDeleteStore . listDeleteStore
type listDeleteStore interface {
	All() (store.PolicyCollection, error)
	Delete(store.PolicyCollection) error
}

type PolicyCleaner struct {
	Logger                lager.Logger
	Store                 listDeleteStore
	UAAClient             uaaClient
	CCClient              ccClient
	CCAppRequestChunkSize int
	RequestTimeout        time.Duration
}

func NewPolicyCleaner(logger lager.Logger, store listDeleteStore, uaaClient uaaClient,
	ccClient ccClient, ccAppRequestChunkSize int, requestTimeout time.Duration) *PolicyCleaner {
	return &PolicyCleaner{
		Logger:                logger,
		Store:                 store,
		UAAClient:             uaaClient,
		CCClient:              ccClient,
		CCAppRequestChunkSize: ccAppRequestChunkSize,
		RequestTimeout:        requestTimeout,
	}
}

func (p *PolicyCleaner) DeleteStalePolicies() (store.PolicyCollection, error) {
	allPolicies, err := p.Store.All()
	if err != nil {
		p.Logger.Error("store-list-policies-failed", err)
		return store.PolicyCollection{}, fmt.Errorf("database read failed: %s", err)
	}
	token, err := p.UAAClient.GetToken()
	if err != nil {
		p.Logger.Error("get-uaa-token-failed", err)
		return store.PolicyCollection{}, fmt.Errorf("get UAA token failed: %s", err)
	}

	policies := allPolicies.Policies

	appGUIDs := policyAppGUIDs(policies)
	appGUIDchunks := getChunks(appGUIDs, p.CCAppRequestChunkSize)

	allPoliciesToDelete := store.PolicyCollection{}

	for _, appGUIDchunk := range appGUIDchunks {
		liveAppGUIDs, err := p.CCClient.GetLiveAppGUIDs(token, appGUIDchunk)
		if err != nil {
			p.Logger.Error("cc-get-app-guids-failed", err)
			return store.PolicyCollection{}, fmt.Errorf("get app guids from Cloud-Controller failed: %s", err)
		}

		staleAppGUIDs := getStaleAppGUIDs(liveAppGUIDs, appGUIDchunk)
		toDelete := getStalePolicies(policies, staleAppGUIDs)

		allPoliciesToDelete.Policies = append(allPoliciesToDelete.Policies, toDelete...)
	}

	egressPolicies := allPolicies.EgressPolicies

	var spaceEgressPolicyGUIDs []string
	spaceEgressPolicy := make(map[string][]store.EgressPolicy)
	for _, egressPolicy := range egressPolicies {
		if egressPolicy.Source.Type == "space" {
			spaceEgressPolicyGUIDs = append(spaceEgressPolicyGUIDs, egressPolicy.Source.ID)
			spaceEgressPolicy[egressPolicy.Source.ID] = append(spaceEgressPolicy[egressPolicy.Source.ID], egressPolicy)
		}
	}

	liveSpaceGUIDs, err := p.CCClient.GetLiveSpaceGUIDs(token, spaceEgressPolicyGUIDs)
	if err != nil {
		p.Logger.Error("get-live-space-guids-failed", err)
		return store.PolicyCollection{}, fmt.Errorf("get live space guids failed: %s", err)
	}

	allPoliciesToDelete.EgressPolicies = getStaleEgressSpacePolicies(spaceEgressPolicy, liveSpaceGUIDs)

	p.Logger.Info("deleting stale policies:", lager.Data{
		"total_c2c_policies": len(allPoliciesToDelete.Policies),
		"stale_c2c_policies": allPoliciesToDelete.Policies,
		"total_egress_policies": len(allPoliciesToDelete.EgressPolicies),
		"stale_egress_policies": allPoliciesToDelete.EgressPolicies,
	})
	err = p.Store.Delete(allPoliciesToDelete)
	if err != nil {
		p.Logger.Error("store-delete-policies-failed", err)
		return store.PolicyCollection{}, fmt.Errorf("database write failed: %s", err)
	}

	return allPoliciesToDelete, nil
}

func (p *PolicyCleaner) DeleteStalePoliciesWrapper() error {
	_, err := p.DeleteStalePolicies()
	return err
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
