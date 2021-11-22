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

// Query is a representation of a search query used to list resources.
type Query struct {
	// Filter is a string representation of a filtering expression as specified in the SCIM spec.
	Filter string
	// SortBy is a string representation of what field to sort the users by.
	SortBy string
}

// TODO: Verify a user
// TODO: Query for user info
// TODO: Convert user ids to names
// TODO: Pagination for List
// TODO: Patch

// UsersService provides access to common user actions. Using this service, you can create, fetch,
// update, delete, and list users. You can also change and set their passwords, and fetch their tokens.
type UsersService struct {
	config Config
}

// NewUsersService returns a UsersService initialized with the given Config.
func NewUsersService(config Config) UsersService {
	return UsersService{
		config: config,
	}
}

// Create will make a request to UAA to create a new user resource with the given username and email.
// A token with the "scim.write" scope is required.
func (us UsersService) Create(username, email, token string) (User, error) {
	resp, err := newNetworkClient(us.config).MakeRequest(network.Request{
		Method:        "POST",
		Path:          "/Users",
		Authorization: network.NewTokenAuthorization(token),
		Body: network.NewJSONRequestBody(documents.CreateUserRequest{
			UserName: username,
			Emails: []documents.Email{
				{Value: email},
			},
		}),
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return User{}, translateError(err)
	}

	var response documents.UserResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return User{}, MalformedResponseError{err}
	}

	return newUserFromResponse(us.config, response), nil
}

// Get will make a request to UAA to fetch the user with the matching id.
// A token with the "scim.read" scope is required.
func (us UsersService) Get(id, token string) (User, error) {
	resp, err := newNetworkClient(us.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  fmt.Sprintf("/Users/%s", id),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return User{}, translateError(err)
	}

	var response documents.UserResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return User{}, MalformedResponseError{err}
	}

	return newUserFromResponse(us.config, response), nil
}

// Delete will make a request to UAA to delete the user resource with the matching id.
// A token with the "scim.write" scope is required.
func (us UsersService) Delete(id, token string) error {
	_, err := newNetworkClient(us.config).MakeRequest(network.Request{
		Method:                "DELETE",
		Path:                  fmt.Sprintf("/Users/%s", id),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return translateError(err)
	}

	return nil
}

// Update will make a request to UAA to update the matching user resource.
// A token with the "scim.write" or "uaa.admin" scope is required.
func (us UsersService) Update(user User, token string) (User, error) {
	resp, err := newNetworkClient(us.config).MakeRequest(network.Request{
		Method:                "PUT",
		Path:                  fmt.Sprintf("/Users/%s", user.ID),
		Authorization:         network.NewTokenAuthorization(token),
		IfMatch:               strconv.Itoa(user.Version),
		Body:                  network.NewJSONRequestBody(newUpdateUserDocumentFromUser(user)),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return User{}, translateError(err)
	}

	var response documents.UserResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return User{}, MalformedResponseError{err}
	}

	return newUserFromResponse(us.config, response), nil
}

// SetPassword will make a request to UAA to set the password for the user with the matching id to the
// given password value. A token with the "password.write" scope is required.
func (us UsersService) SetPassword(id, password, token string) error {
	_, err := newNetworkClient(us.config).MakeRequest(network.Request{
		Method:        "PUT",
		Path:          fmt.Sprintf("/Users/%s/password", id),
		Authorization: network.NewTokenAuthorization(token),
		Body: network.NewJSONRequestBody(documents.SetPasswordRequest{
			Password: password,
		}),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return translateError(err)
	}

	return nil
}

// ChangePassword will make a request to UAA to change the password for the user with the matching id
// to the given password value. The existing password for the user resource as well as a token for the
// user is required.
func (us UsersService) ChangePassword(id, oldPassword, password, token string) error {
	_, err := newNetworkClient(us.config).MakeRequest(network.Request{
		Method:        "PUT",
		Path:          fmt.Sprintf("/Users/%s/password", id),
		Authorization: network.NewTokenAuthorization(token),
		Body: network.NewJSONRequestBody(documents.ChangePasswordRequest{
			OldPassword: oldPassword,
			Password:    password,
		}),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return translateError(err)
	}

	return nil
}

// GetToken will make a request to UAA to retrieve the token for the user matching the given username.
// The user's password is required.
func (us UsersService) GetToken(username, password string, client Client) (string, error) {
	req := network.Request{
		Method:        "POST",
		Path:          "/oauth/token",
		Authorization: network.NewBasicAuthorization(client.ID, ""),
		Body: network.NewFormRequestBody(url.Values{
			"client_id":     []string{client.ID},
			"client_secret": []string{},
			"username":      []string{username},
			"password":      []string{password},
			"grant_type":    []string{"password"},
			"response_type": []string{"token"},
		}),
		AcceptableStatusCodes: []int{http.StatusOK},
	}

	resp, err := newNetworkClient(us.config).MakeRequest(req)
	if err != nil {
		return "", translateError(err)
	}

	var responseBody struct {
		AccessToken string `json:"access_token"`
	}
	err = json.Unmarshal(resp.Body, &responseBody)
	if err != nil {
		return "", MalformedResponseError{err}
	}

	return responseBody.AccessToken, nil
}

// List will make a request to UAA to retrieve all user resources matching the given query.
// A token with the "scim.read" or "uaa.admin" scope is required.
func (us UsersService) List(query Query, token string) ([]User, error) {
	requestPath := url.URL{
		Path: "/Users",
		RawQuery: url.Values{
			"filter": []string{query.Filter},
			"sortBy": []string{query.SortBy},
		}.Encode(),
	}

	resp, err := newNetworkClient(us.config).MakeRequest(network.Request{
		Method:                "GET",
		Path:                  requestPath.String(),
		Authorization:         network.NewTokenAuthorization(token),
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return []User{}, translateError(err)
	}

	var response documents.UserListResponse
	err = json.Unmarshal(resp.Body, &response)
	if err != nil {
		return []User{}, MalformedResponseError{err}
	}

	var userList []User
	for _, userResponse := range response.Resources {
		userList = append(userList, newUserFromResponse(us.config, userResponse))
	}

	return userList, err
}

func newUpdateUserDocumentFromUser(user User) documents.UpdateUserRequest {
	var emails []documents.Email
	for _, email := range user.Emails {
		emails = append(emails, documents.Email{
			Value: email,
		})
	}

	return documents.UpdateUserRequest{
		Schemas:    schemas,
		ID:         user.ID,
		UserName:   user.UserName,
		ExternalID: user.ExternalID,
		Name: documents.UserName{
			Formatted:  user.FormattedName,
			FamilyName: user.FamilyName,
			GivenName:  user.GivenName,
			MiddleName: user.MiddleName,
		},
		Emails: emails,
		Meta: documents.Meta{
			Version:      user.Version,
			Created:      user.CreatedAt,
			LastModified: user.UpdatedAt,
		},
	}
}
