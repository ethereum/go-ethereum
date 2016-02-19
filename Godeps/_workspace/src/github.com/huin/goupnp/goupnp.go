// goupnp is an implementation of a client for various UPnP services.
//
// For most uses, it is recommended to use the code-generated packages under
// github.com/huin/goupnp/dcps. Example use is shown at
// http://godoc.org/github.com/huin/goupnp/example
//
// A commonly used client is internetgateway1.WANPPPConnection1:
// http://godoc.org/github.com/huin/goupnp/dcps/internetgateway1#WANPPPConnection1
//
// Currently only a couple of schemas have code generated for them from the
// UPnP example XML specifications. Not all methods will work on these clients,
// because the generated stubs contain the full set of specified methods from
// the XML specifications, and the discovered services will likely support a
// subset of those methods.
package goupnp

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"time"
	"golang.org/x/net/html/charset"

	"github.com/huin/goupnp/httpu"
	"github.com/huin/goupnp/ssdp"
)

// ContextError is an error that wraps an error with some context information.
type ContextError struct {
	Context string
	Err     error
}

func (err ContextError) Error() string {
	return fmt.Sprintf("%s: %v", err.Context, err.Err)
}

// MaybeRootDevice contains either a RootDevice or an error.
type MaybeRootDevice struct {
	Root *RootDevice
	Err  error
}

// DiscoverDevices attempts to find targets of the given type. This is
// typically the entry-point for this package. searchTarget is typically a URN
// in the form "urn:schemas-upnp-org:device:..." or
// "urn:schemas-upnp-org:service:...". A single error is returned for errors
// while attempting to send the query. An error or RootDevice is returned for
// each discovered RootDevice.
func DiscoverDevices(searchTarget string) ([]MaybeRootDevice, error) {
	httpu, err := httpu.NewHTTPUClient()
	if err != nil {
		return nil, err
	}
	defer httpu.Close()
	responses, err := ssdp.SSDPRawSearch(httpu, string(searchTarget), 2, 3)
	if err != nil {
		return nil, err
	}

	results := make([]MaybeRootDevice, len(responses))
	for i, response := range responses {
		maybe := &results[i]
		loc, err := response.Location()
		if err != nil {
			maybe.Err = ContextError{"unexpected bad location from search", err}
			continue
		}
		locStr := loc.String()
		root := new(RootDevice)
		if err := requestXml(locStr, DeviceXMLNamespace, root); err != nil {
			maybe.Err = ContextError{fmt.Sprintf("error requesting root device details from %q", locStr), err}
			continue
		}
		var urlBaseStr string
		if root.URLBaseStr != "" {
			urlBaseStr = root.URLBaseStr
		} else {
			urlBaseStr = locStr
		}
		urlBase, err := url.Parse(urlBaseStr)
		if err != nil {
			maybe.Err = ContextError{fmt.Sprintf("error parsing location URL %q", locStr), err}
			continue
		}
		root.SetURLBase(urlBase)
		maybe.Root = root
	}

	return results, nil
}

func requestXml(url string, defaultSpace string, doc interface{}) error {
	timeout := time.Duration(3 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("goupnp: got response status %s from %q",
			resp.Status, url)
	}

	decoder := xml.NewDecoder(resp.Body)
	decoder.DefaultSpace = defaultSpace
	decoder.CharsetReader = charset.NewReaderLabel

	return decoder.Decode(doc)
}
