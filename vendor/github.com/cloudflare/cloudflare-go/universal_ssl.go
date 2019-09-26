package cloudflare

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// UniversalSSLSetting represents a universal ssl setting's properties.
type UniversalSSLSetting struct {
	Enabled bool `json:"enabled"`
}

type universalSSLSettingResponse struct {
	Response
	Result UniversalSSLSetting `json:"result"`
}

// UniversalSSLVerificationDetails represents a universal ssl verifcation's properties.
type UniversalSSLVerificationDetails struct {
	CertificateStatus  string                       `json:"certificate_status"`
	VerificationType   string                       `json:"verification_type"`
	ValidationMethod   string                       `json:"validation_method"`
	CertPackUUID       string                       `json:"cert_pack_uuid"`
	VerificationStatus bool                         `json:"verification_status"`
	BrandCheck         bool                         `json:"brand_check"`
	VerificationInfo   UniversalSSLVerificationInfo `json:"verification_info"`
}

// UniversalSSLVerificationInfo represents DCV record.
type UniversalSSLVerificationInfo struct {
	RecordName   string `json:"record_name"`
	RecordTarget string `json:"record_target"`
}

type universalSSLVerificationResponse struct {
	Response
	Result []UniversalSSLVerificationDetails `json:"result"`
}

// UniversalSSLSettingDetails returns the details for a universal ssl setting
//
// API reference: https://api.cloudflare.com/#universal-ssl-settings-for-a-zone-universal-ssl-settings-details
func (api *API) UniversalSSLSettingDetails(zoneID string) (UniversalSSLSetting, error) {
	uri := "/zones/" + zoneID + "/ssl/universal/settings"
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return UniversalSSLSetting{}, errors.Wrap(err, errMakeRequestError)
	}
	var r universalSSLSettingResponse
	if err := json.Unmarshal(res, &r); err != nil {
		return UniversalSSLSetting{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}

// EditUniversalSSLSetting edits the uniersal ssl setting for a zone
//
// API reference: https://api.cloudflare.com/#universal-ssl-settings-for-a-zone-edit-universal-ssl-settings
func (api *API) EditUniversalSSLSetting(zoneID string, setting UniversalSSLSetting) (UniversalSSLSetting, error) {
	uri := "/zones/" + zoneID + "/ssl/universal/settings"
	res, err := api.makeRequest("PATCH", uri, setting)
	if err != nil {
		return UniversalSSLSetting{}, errors.Wrap(err, errMakeRequestError)
	}
	var r universalSSLSettingResponse
	if err := json.Unmarshal(res, &r); err != nil {
		return UniversalSSLSetting{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil

}

// UniversalSSLVerificationDetails returns the details for a universal ssl verifcation
//
// API reference: https://api.cloudflare.com/#ssl-verification-ssl-verification-details
func (api *API) UniversalSSLVerificationDetails(zoneID string) ([]UniversalSSLVerificationDetails, error) {
	uri := "/zones/" + zoneID + "/ssl/verification"
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return []UniversalSSLVerificationDetails{}, errors.Wrap(err, errMakeRequestError)
	}
	var r universalSSLVerificationResponse
	if err := json.Unmarshal(res, &r); err != nil {
		return []UniversalSSLVerificationDetails{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}
