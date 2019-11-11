package cloudflare

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// AccessServiceToken represents an Access Service Token.
type AccessServiceToken struct {
	ClientID  string     `json:"client_id"`
	CreatedAt *time.Time `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// AccessServiceTokenUpdateResponse represents the response from the API
// when a new Service Token is updated. This base struct is also used in the
// Create as they are very similar responses.
type AccessServiceTokenUpdateResponse struct {
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	ClientID  string     `json:"client_id"`
}

// AccessServiceTokenCreateResponse is the same API response as the Update
// operation with the exception that the `ClientSecret` is present in a
// Create operation.
type AccessServiceTokenCreateResponse struct {
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	ClientID     string     `json:"client_id"`
	ClientSecret string     `json:"client_secret"`
}

// AccessServiceTokensListResponse represents the response from the list
// Access Service Tokens endpoint.
type AccessServiceTokensListResponse struct {
	Result []AccessServiceToken `json:"result"`
	Response
	ResultInfo `json:"result_info"`
}

// AccessServiceTokensDetailResponse is the API response, containing a single
// Access Service Token.
type AccessServiceTokensDetailResponse struct {
	Success  bool               `json:"success"`
	Errors   []string           `json:"errors"`
	Messages []string           `json:"messages"`
	Result   AccessServiceToken `json:"result"`
}

// AccessServiceTokensCreationDetailResponse is the API response, containing a
// single Access Service Token.
type AccessServiceTokensCreationDetailResponse struct {
	Success  bool                             `json:"success"`
	Errors   []string                         `json:"errors"`
	Messages []string                         `json:"messages"`
	Result   AccessServiceTokenCreateResponse `json:"result"`
}

// AccessServiceTokensUpdateDetailResponse is the API response, containing a
// single Access Service Token.
type AccessServiceTokensUpdateDetailResponse struct {
	Success  bool                             `json:"success"`
	Errors   []string                         `json:"errors"`
	Messages []string                         `json:"messages"`
	Result   AccessServiceTokenUpdateResponse `json:"result"`
}

// AccessServiceTokens returns all Access Service Tokens for an account.
//
// API reference: https://api.cloudflare.com/#access-service-tokens-list-access-service-tokens
func (api *API) AccessServiceTokens(accountID string) ([]AccessServiceToken, ResultInfo, error) {
	uri := "/accounts/" + accountID + "/access/service_tokens"

	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return []AccessServiceToken{}, ResultInfo{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessServiceTokensListResponse AccessServiceTokensListResponse
	err = json.Unmarshal(res, &accessServiceTokensListResponse)
	if err != nil {
		return []AccessServiceToken{}, ResultInfo{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessServiceTokensListResponse.Result, accessServiceTokensListResponse.ResultInfo, nil
}

// CreateAccessServiceToken creates a new Access Service Token for an account.
//
// API reference: https://api.cloudflare.com/#access-service-tokens-create-access-service-token
func (api *API) CreateAccessServiceToken(accountID, name string) (AccessServiceTokenCreateResponse, error) {
	uri := "/accounts/" + accountID + "/access/service_tokens"
	marshalledName, _ := json.Marshal(struct {
		Name string `json:"name"`
	}{name})

	res, err := api.makeRequest("POST", uri, marshalledName)

	if err != nil {
		return AccessServiceTokenCreateResponse{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessServiceTokenCreation AccessServiceTokensCreationDetailResponse
	err = json.Unmarshal(res, &accessServiceTokenCreation)
	if err != nil {
		return AccessServiceTokenCreateResponse{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessServiceTokenCreation.Result, nil
}

// UpdateAccessServiceToken updates an existing Access Service Token for an
// account.
//
// API reference: https://api.cloudflare.com/#access-service-tokens-update-access-service-token
func (api *API) UpdateAccessServiceToken(accountID, uuid, name string) (AccessServiceTokenUpdateResponse, error) {
	uri := fmt.Sprintf("/accounts/%s/access/service_tokens/%s", accountID, uuid)

	marshalledName, _ := json.Marshal(struct {
		Name string `json:"name"`
	}{name})

	res, err := api.makeRequest("PUT", uri, marshalledName)
	if err != nil {
		return AccessServiceTokenUpdateResponse{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessServiceTokenUpdate AccessServiceTokensUpdateDetailResponse
	err = json.Unmarshal(res, &accessServiceTokenUpdate)
	if err != nil {
		return AccessServiceTokenUpdateResponse{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessServiceTokenUpdate.Result, nil
}

// DeleteAccessServiceToken removes an existing Access Service Token for an
// account.
//
// API reference: https://api.cloudflare.com/#access-service-tokens-delete-access-service-token
func (api *API) DeleteAccessServiceToken(accountID, uuid string) (AccessServiceTokenUpdateResponse, error) {
	uri := fmt.Sprintf("/accounts/%s/access/service_tokens/%s", accountID, uuid)

	res, err := api.makeRequest("DELETE", uri, nil)
	if err != nil {
		return AccessServiceTokenUpdateResponse{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessServiceTokenUpdate AccessServiceTokensUpdateDetailResponse
	err = json.Unmarshal(res, &accessServiceTokenUpdate)
	if err != nil {
		return AccessServiceTokenUpdateResponse{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessServiceTokenUpdate.Result, nil
}
