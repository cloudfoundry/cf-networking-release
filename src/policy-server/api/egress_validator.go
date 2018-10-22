package api

import (
	"errors"
	"fmt"
	"policy-server/store"
	"sort"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/httperror"
)

//go:generate counterfeiter -o fakes/egress_destination_store.go --fake-name EgressDestinationStore . EgressDestinationStore
type EgressDestinationStore interface {
	GetByGUID(guid ...string) ([]store.EgressDestination, error)
	GetByName(name ...string) ([]store.EgressDestination, error)
}

//go:generate counterfeiter -o fakes/cc_client.go --fake-name CCClient . ccClient
type ccClient interface {
	GetLiveAppGUIDs(token string, appGUIDs []string) (map[string]struct{}, error)
	GetLiveSpaceGUIDs(token string, spaceGUIDs []string) (map[string]struct{}, error)
}

//go:generate counterfeiter -o fakes/uua_client.go --fake-name UAAClient . uaaClient
type uaaClient interface {
	GetToken() (string, error)
}

type EgressValidator struct {
	CCClient         ccClient
	UAAClient        uaaClient
	DestinationStore EgressDestinationStore
}

func DestinationKeyFunc(policy EgressPolicy) string { return policy.Destination.GUID }
func SourceKeyFunc(policy EgressPolicy) string      { return policy.Source.ID }

func (v *EgressValidator) ValidateEgressPolicies(policies []EgressPolicy) error {
	for _, policy := range policies {
		if policy.Source == nil {
			return policyMetadataError("missing egress source", policy)
		}
		if policy.Source.ID == "" {
			return policyMetadataError("missing egress source ID", policy)
		}
		if policy.Source.Type != "" && policy.Source.Type != "app" && policy.Source.Type != "space" {
			return policyMetadataError("source type must be app or space", policy)
		}
		if policy.Destination == nil {
			return policyMetadataError("missing egress destination", policy)
		}
		if policy.Destination.GUID == "" {
			return policyMetadataError("missing egress destination id", policy)
		}
	}

	token, err := v.UAAClient.GetToken()
	if err != nil {
		return fmt.Errorf("failed to get uaa token: %s", err)
	}

	appGUIDSet := sourceAppGUIDs(policies)

	if len(appGUIDSet) > 0 {
		liveAppGUIDs, err := v.CCClient.GetLiveAppGUIDs(token, keys(appGUIDSet))
		if err != nil {
			return fmt.Errorf("failed to get live app guids: %s", err)
		}

		missingAppGUIDs := relativeComplement(appGUIDSet, liveAppGUIDs)

		if len(missingAppGUIDs) > 0 {
			return composeMetadataError("app", missingAppGUIDs, SourceKeyFunc, policies)
		}
	}

	spaceGUIDSet := sourceSpaceGUIDs(policies)

	if len(spaceGUIDSet) > 0 {
		liveSpaceGUIDs, err := v.CCClient.GetLiveSpaceGUIDs(token, keys(spaceGUIDSet))
		if err != nil {
			return fmt.Errorf("failed to get live space guids: %s", err)
		}

		missingSpaceGUIDs := relativeComplement(spaceGUIDSet, liveSpaceGUIDs)

		if len(missingSpaceGUIDs) > 0 {
			return composeMetadataError("space", missingSpaceGUIDs, SourceKeyFunc, policies)
		}
	}

	destinationGUIDSet := destinationGUIDs(policies)
	destinations, err := v.DestinationStore.GetByGUID(keys(destinationGUIDSet)...)
	if err != nil {
		return fmt.Errorf("failed to get egress destinations: can't get destinations")
	}

	foundGUIDSet := make(map[string]struct{})
	for _, destination := range destinations {
		foundGUIDSet[destination.GUID] = struct{}{}
	}

	missingDestinations := relativeComplement(destinationGUIDSet, foundGUIDSet)
	if len(missingDestinations) > 0 {
		return composeMetadataError("destination", missingDestinations, DestinationKeyFunc, policies)

	}

	return nil
}

func composeMetadataError(dataType string, missingGuids []string, keyFunc func(policy EgressPolicy) string, policies []EgressPolicy) error {
	errorMsg := fmt.Sprintf("%s guids not found: [%s]", dataType, strings.Join(missingGuids, ", "))
	deficientPolicies := findPoliciesWithKeyGUID(policies, missingGuids, keyFunc)
	policyAsMap := map[string]interface{}{fmt.Sprintf("policies with missing %ss", dataType): deficientPolicies}
	return httperror.NewMetadataError(errors.New(errorMsg), policyAsMap)
}

func findPoliciesWithKeyGUID(policies []EgressPolicy, guids []string, keyFunc func(EgressPolicy) string) []EgressPolicy {
	guidSet := set(guids)
	var policiesToReturn []EgressPolicy

	for _, policy := range policies {
		_, ok := guidSet[keyFunc(policy)]
		if ok {
			policiesToReturn = append(policiesToReturn, policy)
		}
	}

	return policiesToReturn
}

func policyMetadataError(message string, policy EgressPolicy) error {
	policyAsMap := map[string]interface{}{"bad_egress_policy": policy}
	return httperror.NewMetadataError(errors.New(message), policyAsMap)
}

func sourceAppGUIDs(policies []EgressPolicy) map[string]struct{} {
	appGUIDSet := make(map[string]struct{})
	for _, policy := range policies {
		if policy.Source.Type == "" || policy.Source.Type == "app" {
			appGUIDSet[policy.Source.ID] = struct{}{}
		}
	}
	return appGUIDSet
}

func destinationGUIDs(policies []EgressPolicy) map[string]struct{} {
	destSet := make(map[string]struct{})
	for _, policy := range policies {
		destSet[policy.Destination.GUID] = struct{}{}
	}
	return destSet
}

func sourceSpaceGUIDs(policies []EgressPolicy) map[string]struct{} {
	guidSet := make(map[string]struct{})
	for _, policy := range policies {
		if policy.Source.Type == "space" {
			guidSet[policy.Source.ID] = struct{}{}
		}
	}
	return guidSet
}

func keys(set map[string]struct{}) []string {
	var keys []string
	for key, _ := range set {
		keys = append(keys, key)
	}
	return keys
}

func set(items []string) map[string]struct{} {
	itemSet := make(map[string]struct{})
	for _, item := range items {
		itemSet[item] = struct{}{}
	}
	return itemSet
}

func relativeComplement(a map[string]struct{}, b map[string]struct{}) []string {
	result := []string{}
	for key, _ := range a {
		_, ok := b[key]
		if !ok {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}
