package storage

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// FileServiceClient contains operations for Microsoft Azure File Service.
type FileServiceClient struct {
	client Client
}

// A Share is an entry in ShareListResponse.
type Share struct {
	Name       string          `xml:"Name"`
	Properties ShareProperties `xml:"Properties"`
}

// ShareProperties contains various properties of a share returned from
// various endpoints like ListShares.
type ShareProperties struct {
	LastModified string `xml:"Last-Modified"`
	Etag         string `xml:"Etag"`
	Quota        string `xml:"Quota"`
}

// ShareListResponse contains the response fields from
// ListShares call.
//
// See https://msdn.microsoft.com/en-us/library/azure/dn167009.aspx
type ShareListResponse struct {
	XMLName    xml.Name `xml:"EnumerationResults"`
	Xmlns      string   `xml:"xmlns,attr"`
	Prefix     string   `xml:"Prefix"`
	Marker     string   `xml:"Marker"`
	NextMarker string   `xml:"NextMarker"`
	MaxResults int64    `xml:"MaxResults"`
	Shares     []Share  `xml:"Shares>Share"`
}

// ListSharesParameters defines the set of customizable parameters to make a
// List Shares call.
//
// See https://msdn.microsoft.com/en-us/library/azure/dn167009.aspx
type ListSharesParameters struct {
	Prefix     string
	Marker     string
	Include    string
	MaxResults uint
	Timeout    uint
}

// ShareHeaders contains various properties of a file and is an entry
// in SetShareProperties
type ShareHeaders struct {
	Quota string `header:"x-ms-share-quota"`
}

func (p ListSharesParameters) getParameters() url.Values {
	out := url.Values{}

	if p.Prefix != "" {
		out.Set("prefix", p.Prefix)
	}
	if p.Marker != "" {
		out.Set("marker", p.Marker)
	}
	if p.Include != "" {
		out.Set("include", p.Include)
	}
	if p.MaxResults != 0 {
		out.Set("maxresults", fmt.Sprintf("%v", p.MaxResults))
	}
	if p.Timeout != 0 {
		out.Set("timeout", fmt.Sprintf("%v", p.Timeout))
	}

	return out
}

// pathForFileShare returns the URL path segment for a File Share resource
func pathForFileShare(name string) string {
	return fmt.Sprintf("/%s", name)
}

// ListShares returns the list of shares in a storage account along with
// pagination token and other response details.
//
// See https://msdn.microsoft.com/en-us/library/azure/dd179352.aspx
func (f FileServiceClient) ListShares(params ListSharesParameters) (ShareListResponse, error) {
	q := mergeParams(params.getParameters(), url.Values{"comp": {"list"}})
	uri := f.client.getEndpoint(fileServiceName, "", q)
	headers := f.client.getStandardHeaders()

	var out ShareListResponse
	resp, err := f.client.exec("GET", uri, headers, nil)
	if err != nil {
		return out, err
	}
	defer resp.body.Close()

	err = xmlUnmarshal(resp.body, &out)
	return out, err
}

// CreateShare operation creates a new share under the specified account. If the
// share with the same name already exists, the operation fails.
//
// See https://msdn.microsoft.com/en-us/library/azure/dn167008.aspx
func (f FileServiceClient) CreateShare(name string) error {
	resp, err := f.createShare(name)
	if err != nil {
		return err
	}
	defer resp.body.Close()
	return checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

// ShareExists returns true if a share with given name exists
// on the storage account, otherwise returns false.
func (f FileServiceClient) ShareExists(name string) (bool, error) {
	uri := f.client.getEndpoint(fileServiceName, pathForFileShare(name), url.Values{"restype": {"share"}})
	headers := f.client.getStandardHeaders()

	resp, err := f.client.exec("HEAD", uri, headers, nil)
	if resp != nil {
		defer resp.body.Close()
		if resp.statusCode == http.StatusOK || resp.statusCode == http.StatusNotFound {
			return resp.statusCode == http.StatusOK, nil
		}
	}
	return false, err
}

// GetShareURL gets the canonical URL to the share with the specified name in the
// specified container. This method does not create a publicly accessible URL if
// the file is private and this method does not check if the file
// exists.
func (f FileServiceClient) GetShareURL(name string) string {
	return f.client.getEndpoint(fileServiceName, pathForFileShare(name), url.Values{})
}

// CreateShareIfNotExists creates a new share under the specified account if
// it does not exist. Returns true if container is newly created or false if
// container already exists.
//
// See https://msdn.microsoft.com/en-us/library/azure/dn167008.aspx
func (f FileServiceClient) CreateShareIfNotExists(name string) (bool, error) {
	resp, err := f.createShare(name)
	if resp != nil {
		defer resp.body.Close()
		if resp.statusCode == http.StatusCreated || resp.statusCode == http.StatusConflict {
			return resp.statusCode == http.StatusCreated, nil
		}
	}
	return false, err
}

// CreateShare creates a Azure File Share and returns its response
func (f FileServiceClient) createShare(name string) (*storageResponse, error) {
	if err := f.checkForStorageEmulator(); err != nil {
		return nil, err
	}
	uri := f.client.getEndpoint(fileServiceName, pathForFileShare(name), url.Values{"restype": {"share"}})
	headers := f.client.getStandardHeaders()
	return f.client.exec("PUT", uri, headers, nil)
}

// GetShareProperties provides various information about the specified
// file. See https://msdn.microsoft.com/en-us/library/azure/dn689099.aspx
func (f FileServiceClient) GetShareProperties(name string) (*ShareProperties, error) {
	uri := f.client.getEndpoint(fileServiceName, pathForFileShare(name), url.Values{"restype": {"share"}})

	headers := f.client.getStandardHeaders()
	resp, err := f.client.exec("HEAD", uri, headers, nil)
	if err != nil {
		return nil, err
	}
	defer resp.body.Close()

	if err := checkRespCode(resp.statusCode, []int{http.StatusOK}); err != nil {
		return nil, err
	}

	return &ShareProperties{
		LastModified: resp.headers.Get("Last-Modified"),
		Etag:         resp.headers.Get("Etag"),
		Quota:        resp.headers.Get("x-ms-share-quota"),
	}, nil
}

// SetShareProperties replaces the ShareHeaders for the specified file.
//
// Some keys may be converted to Camel-Case before sending. All keys
// are returned in lower case by SetShareProperties. HTTP header names
// are case-insensitive so case munging should not matter to other
// applications either.
//
// See https://msdn.microsoft.com/en-us/library/azure/mt427368.aspx
func (f FileServiceClient) SetShareProperties(name string, shareHeaders ShareHeaders) error {
	params := url.Values{}
	params.Set("restype", "share")
	params.Set("comp", "properties")

	uri := f.client.getEndpoint(fileServiceName, pathForFileShare(name), params)
	headers := f.client.getStandardHeaders()

	extraHeaders := headersFromStruct(shareHeaders)

	for k, v := range extraHeaders {
		headers[k] = v
	}

	resp, err := f.client.exec("PUT", uri, headers, nil)
	if err != nil {
		return err
	}
	defer resp.body.Close()

	return checkRespCode(resp.statusCode, []int{http.StatusOK})
}

// DeleteShare operation marks the specified share for deletion. The share
// and any files contained within it are later deleted during garbage
// collection.
//
// See https://msdn.microsoft.com/en-us/library/azure/dn689090.aspx
func (f FileServiceClient) DeleteShare(name string) error {
	resp, err := f.deleteShare(name)
	if err != nil {
		return err
	}
	defer resp.body.Close()
	return checkRespCode(resp.statusCode, []int{http.StatusAccepted})
}

// DeleteShareIfExists operation marks the specified share for deletion if it
// exists. The share and any files contained within it are later deleted during
// garbage collection. Returns true if share existed and deleted with this call,
// false otherwise.
//
// See https://msdn.microsoft.com/en-us/library/azure/dn689090.aspx
func (f FileServiceClient) DeleteShareIfExists(name string) (bool, error) {
	resp, err := f.deleteShare(name)
	if resp != nil {
		defer resp.body.Close()
		if resp.statusCode == http.StatusAccepted || resp.statusCode == http.StatusNotFound {
			return resp.statusCode == http.StatusAccepted, nil
		}
	}
	return false, err
}

// deleteShare makes the call to Delete Share operation endpoint and returns
// the response
func (f FileServiceClient) deleteShare(name string) (*storageResponse, error) {
	if err := f.checkForStorageEmulator(); err != nil {
		return nil, err
	}
	uri := f.client.getEndpoint(fileServiceName, pathForFileShare(name), url.Values{"restype": {"share"}})
	return f.client.exec("DELETE", uri, f.client.getStandardHeaders(), nil)
}

// SetShareMetadata replaces the metadata for the specified Share.
//
// Some keys may be converted to Camel-Case before sending. All keys
// are returned in lower case by GetShareMetadata. HTTP header names
// are case-insensitive so case munging should not matter to other
// applications either.
//
// See https://msdn.microsoft.com/en-us/library/azure/dd179414.aspx
func (f FileServiceClient) SetShareMetadata(name string, metadata map[string]string, extraHeaders map[string]string) error {
	params := url.Values{}
	params.Set("restype", "share")
	params.Set("comp", "metadata")

	uri := f.client.getEndpoint(fileServiceName, pathForFileShare(name), params)
	headers := f.client.getStandardHeaders()
	for k, v := range metadata {
		headers[userDefinedMetadataHeaderPrefix+k] = v
	}

	for k, v := range extraHeaders {
		headers[k] = v
	}

	resp, err := f.client.exec("PUT", uri, headers, nil)
	if err != nil {
		return err
	}
	defer resp.body.Close()

	return checkRespCode(resp.statusCode, []int{http.StatusOK})
}

// GetShareMetadata returns all user-defined metadata for the specified share.
//
// All metadata keys will be returned in lower case. (HTTP header
// names are case-insensitive.)
//
// See https://msdn.microsoft.com/en-us/library/azure/dd179414.aspx
func (f FileServiceClient) GetShareMetadata(name string) (map[string]string, error) {
	params := url.Values{}
	params.Set("restype", "share")
	params.Set("comp", "metadata")

	uri := f.client.getEndpoint(fileServiceName, pathForFileShare(name), params)
	headers := f.client.getStandardHeaders()

	resp, err := f.client.exec("GET", uri, headers, nil)
	if err != nil {
		return nil, err
	}
	defer resp.body.Close()

	if err := checkRespCode(resp.statusCode, []int{http.StatusOK}); err != nil {
		return nil, err
	}

	metadata := make(map[string]string)
	for k, v := range resp.headers {
		// Can't trust CanonicalHeaderKey() to munge case
		// reliably. "_" is allowed in identifiers:
		// https://msdn.microsoft.com/en-us/library/azure/dd179414.aspx
		// https://msdn.microsoft.com/library/aa664670(VS.71).aspx
		// http://tools.ietf.org/html/rfc7230#section-3.2
		// ...but "_" is considered invalid by
		// CanonicalMIMEHeaderKey in
		// https://golang.org/src/net/textproto/reader.go?s=14615:14659#L542
		// so k can be "X-Ms-Meta-Foo" or "x-ms-meta-foo_bar".
		k = strings.ToLower(k)
		if len(v) == 0 || !strings.HasPrefix(k, strings.ToLower(userDefinedMetadataHeaderPrefix)) {
			continue
		}
		// metadata["foo"] = content of the last X-Ms-Meta-Foo header
		k = k[len(userDefinedMetadataHeaderPrefix):]
		metadata[k] = v[len(v)-1]
	}
	return metadata, nil
}

//checkForStorageEmulator determines if the client is setup for use with
//Azure Storage Emulator, and returns a relevant error
func (f FileServiceClient) checkForStorageEmulator() error {
	if f.client.accountName == StorageEmulatorAccountName {
		return fmt.Errorf("Error: File service is not currently supported by Azure Storage Emulator")
	}
	return nil
}
