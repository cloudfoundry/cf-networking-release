package warrant

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/pivotal-cf-experimental/warrant/internal/documents"
	"github.com/pivotal-cf-experimental/warrant/internal/network"
)

// TODO: Pagination for List

// GroupsService provides access to common group actions. Using this service,
// you can create, delete, fetch and list group resources.
type GroupsService struct {
	config Config
}

// NewGroupsService returns a GroupsService initialized with the given Config.
func NewGroupsService(config Config) GroupsService {
	return GroupsService{
		config: config,
	}
}

// Create will make a request to UAA to create a new group resource with the given
// DisplayName. A token with the "scim.write" scope is required.
func (gs GroupsService) Create(displayName, token string) (Group, error) {
	resp, err := newNetworkClient(gs.config).MakeRequest(network.Request{
		Method:        "POST",
		Path:          "/Groups",
		Authorization: network.NewTokenAuthorization(token),
		Body: network.NewJSONRequestBody(documents.CreateGroupRequest{
			DisplayName: displayName,
		}),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return Group{}, translateError(err)
	}

	var response documents.GroupResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return Group{}, MalformedResponseError{err}
	}

	return newGroupFromResponse(gs.config, response), nil
}

// Update will make a request to UAA to update the matching group resource.
// A token with the "scim.write" or "groups.update" scope is required.
func (gs GroupsService) Update(group Group, token string) (Group, error) {
	resp, err := newNetworkClient(gs.config).MakeRequest(network.Request{
		Method:        "PUT",
		Path:          fmt.Sprintf("/Groups/%s", group.ID),
		Authorization: network.NewTokenAuthorization(token),
		IfMatch:       strconv.Itoa(group.Version),
		Body:          network.NewJSONRequestBody(newUpdateGroupDocumentFromGroup(group)),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return Group{}, translateError(err)
	}

	var response documents.GroupResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return Group{}, MalformedResponseError{err}
	}

	return newGroupFromResponse(gs.config, response), nil
}

// AddMember will make a request to UAA to add a member to the group resource with the matching id.
// A token with the "scim.write" scope is required.
func (gs GroupsService) AddMember(groupID, memberID, token string) (Member, error) {
	resp, err := newNetworkClient(gs.config).MakeRequest(network.Request{
		Method:        "POST",
		Path:          fmt.Sprintf("/Groups/%s/members", groupID),
		Authorization: network.NewTokenAuthorization(token),
		Body: network.NewJSONRequestBody(documents.CreateMemberRequest{
			Origin: "uaa",
			Type:   "USER",
			Value:  memberID,
		}),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return Member{}, translateError(err)
	}

	var response documents.MemberResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return Member{}, MalformedResponseError{err}
	}

	return newMemberFromResponse(gs.config, response), nil
}

// CheckMembership will make a request to UAA to fetch a member resource from a group resource.
// A token with the "scim.read" scope is required.
func (gs GroupsService) CheckMembership(groupID, memberID, token string) (Member, bool, error) {
	resp, err := newNetworkClient(gs.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  fmt.Sprintf("/Groups/%s/members/%s", groupID, memberID),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK, http.StatusNotFound},
	})
	if err != nil {
		return Member{}, false, translateError(err)
	}

	if resp.Code == http.StatusNotFound {
		return Member{}, false, nil
	}

	var response documents.MemberResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return Member{}, false, MalformedResponseError{err}
	}

	return newMemberFromResponse(gs.config, response), true, nil
}

// ListMembers will make a request to UAA to fetch the members of a group resource with the matching id.
// A token with the "scim.read" scope is required.
func (gs GroupsService) ListMembers(groupID, token string) ([]Member, error) {
	resp, err := newNetworkClient(gs.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  fmt.Sprintf("/Groups/%s/members", groupID),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return []Member{}, translateError(err)
	}

	var response []documents.MemberResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return []Member{}, MalformedResponseError{err}
	}

	var memberList []Member
	for _, memberResponse := range response {
		memberList = append(memberList, newMemberFromResponse(gs.config, memberResponse))
	}

	return memberList, nil
}

// RemoveMember will make a request to UAA to remove a member from a group resource.
// A token with the "scim.write" scope is required.
func (gs GroupsService) RemoveMember(groupID, memberID, token string) error {
	_, err := newNetworkClient(gs.config).MakeRequest(network.Request{
		Method:                "DELETE",
		Path:                  fmt.Sprintf("/Groups/%s/members/%s", groupID, memberID),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return translateError(err)
	}

	return nil
}

// Get will make a request to UAA to fetch the group resource with the matching id.
// A token with the "scim.read" scope is required.
func (gs GroupsService) Get(id, token string) (Group, error) {
	resp, err := newNetworkClient(gs.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  fmt.Sprintf("/Groups/%s", id),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return Group{}, translateError(err)
	}

	var response documents.GroupResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return Group{}, MalformedResponseError{err}
	}

	return newGroupFromResponse(gs.config, response), nil
}

// List wil make a request to UAA to list the groups that match the given Query.
// A token with the "scim.read" scope is required.
func (gs GroupsService) List(query Query, token string) ([]Group, error) {
	requestPath := url.URL{
		Path: "/Groups",
		RawQuery: url.Values{
			"filter": []string{query.Filter},
			"sortBy": []string{query.SortBy},
		}.Encode(),
	}

	resp, err := newNetworkClient(gs.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  requestPath.String(),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return []Group{}, translateError(err)
	}

	var response documents.GroupListResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return []Group{}, MalformedResponseError{err}
	}

	var groupList []Group
	for _, groupResponse := range response.Resources {
		groupList = append(groupList, newGroupFromResponse(gs.config, groupResponse))
	}

	return groupList, err
}

// Delete will make a request to UAA to delete the group resource with the matching id.
// A token with the "scim.write" scope is required.
func (gs GroupsService) Delete(id, token string) error {
	_, err := newNetworkClient(gs.config).MakeRequest(network.Request{
		Method:                "DELETE",
		Path:                  fmt.Sprintf("/Groups/%s", id),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return translateError(err)
	}

	return nil
}

func newUpdateGroupDocumentFromGroup(group Group) documents.CreateUpdateGroupRequest {
	var members []documents.CreateMemberRequest
	for _, member := range group.Members {
		members = append(members, documents.CreateMemberRequest{
			Origin: member.Origin,
			Type:   member.Type,
			Value:  member.Value,
		})
	}

	return documents.CreateUpdateGroupRequest{
		Schemas:     schemas,
		ID:          group.ID,
		DisplayName: group.DisplayName,
		Description: group.Description,
		Members:     members,
		Meta: documents.Meta{
			Version:      group.Version,
			Created:      group.CreatedAt,
			LastModified: group.UpdatedAt,
		},
	}
}
