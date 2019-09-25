package cloudflare

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// AccessOrganization represents an Access organization.
type AccessOrganization struct {
	CreatedAt   *time.Time                    `json:"created_at"`
	UpdatedAt   *time.Time                    `json:"updated_at"`
	Name        string                        `json:"name"`
	AuthDomain  string                        `json:"auth_domain"`
	LoginDesign AccessOrganizationLoginDesign `json:"login_design"`
}

// AccessOrganizationLoginDesign represents the login design options.
type AccessOrganizationLoginDesign struct {
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
	LogoPath        string `json:"logo_path"`
}

// AccessOrganizationListResponse represents the response from the list
// access organization endpoint.
type AccessOrganizationListResponse struct {
	Result AccessOrganization `json:"result"`
	Response
	ResultInfo `json:"result_info"`
}

// AccessOrganizationDetailResponse is the API response, containing a
// single access organization.
type AccessOrganizationDetailResponse struct {
	Success  bool               `json:"success"`
	Errors   []string           `json:"errors"`
	Messages []string           `json:"messages"`
	Result   AccessOrganization `json:"result"`
}

// AccessOrganization returns the Access organisation details.
//
// API reference: https://api.cloudflare.com/#access-organizations-access-organization-details
func (api *API) AccessOrganization(accountID string) (AccessOrganization, ResultInfo, error) {
	uri := "/accounts/" + accountID + "/access/organizations"

	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return AccessOrganization{}, ResultInfo{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessOrganizationListResponse AccessOrganizationListResponse
	err = json.Unmarshal(res, &accessOrganizationListResponse)
	if err != nil {
		return AccessOrganization{}, ResultInfo{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessOrganizationListResponse.Result, accessOrganizationListResponse.ResultInfo, nil
}

// CreateAccessOrganization creates the Access organisation details.
//
// API reference: https://api.cloudflare.com/#access-organizations-create-access-organization
func (api *API) CreateAccessOrganization(accountID string, accessOrganization AccessOrganization) (AccessOrganization, error) {
	uri := "/accounts/" + accountID + "/access/organizations"

	res, err := api.makeRequest("POST", uri, accessOrganization)
	if err != nil {
		return AccessOrganization{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessOrganizationDetailResponse AccessOrganizationDetailResponse
	err = json.Unmarshal(res, &accessOrganizationDetailResponse)
	if err != nil {
		return AccessOrganization{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessOrganizationDetailResponse.Result, nil
}

// UpdateAccessOrganization creates the Access organisation details.
//
// API reference: https://api.cloudflare.com/#access-organizations-update-access-organization
func (api *API) UpdateAccessOrganization(accountID string, accessOrganization AccessOrganization) (AccessOrganization, error) {
	uri := "/accounts/" + accountID + "/access/organizations"

	res, err := api.makeRequest("PUT", uri, accessOrganization)
	if err != nil {
		return AccessOrganization{}, errors.Wrap(err, errMakeRequestError)
	}

	var accessOrganizationDetailResponse AccessOrganizationDetailResponse
	err = json.Unmarshal(res, &accessOrganizationDetailResponse)
	if err != nil {
		return AccessOrganization{}, errors.Wrap(err, errUnmarshalError)
	}

	return accessOrganizationDetailResponse.Result, nil
}
