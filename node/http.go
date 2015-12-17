package node

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func newHTTP(docRoot string) *HTTPClient {
	t := &http.Transport{}
	if len(docRoot) > 0 {
		t.RegisterProtocol("file", http.NewFileTransport(http.Dir(docRoot)))
	}
	return &HTTPClient{
		Client:    &http.Client{Transport: t},
		transport: t,
		schemes:   []string{"file"},
	}
}

func (self *HTTPClient) registerSchemes(schemes []URLScheme) {
	for _, scheme := range schemes {
		self.schemes = append(self.schemes, scheme.Name)
		self.transport.RegisterProtocol(scheme.Name, scheme.RoundTripper)
	}
}

type HTTPClient struct {
	*http.Client
	transport *http.Transport
	schemes   []string
}

// Clients should be reused instead of created as needed. Clients are safe for concurrent use by multiple goroutines.

// A Client is higher-level than a RoundTripper (such as Transport) and additionally handles HTTP details such as cookies and redirects.

type URLScheme struct {
	Name         string
	RoundTripper http.RoundTripper
}

func (self *HTTPClient) HasScheme(scheme string) bool {
	for _, s := range self.schemes {
		if s == scheme {
			return true
		}
	}
	return false
}

// GetBody(uri) downloads the document at uri, if path is non-empty it
// GetBody(uri) fetches the uri and returns the body
func (self *HTTPClient) GetBody(uri string) ([]byte, error) {
	// retrieve body
	resp, err := self.Get(uri)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		return body, fmt.Errorf("HTTP error: %s", resp.Status)
	}
	return body, nil
}

// Download(uri, path) fetches the uri and writes the body into the file
func (self *HTTPClient) Download(uri, path string) error {

	resp, err := self.Get(uri)
	if err != nil {
		return err
	}
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	var abspath string
	abspath, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	w, err := os.Create(abspath)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return err
	}

	return nil

}

func (self *HTTPClient) GetAuthBody(uri string, hash common.Hash) ([]byte, error) {
	// retrieve body
	body, err := self.GetBody(uri)
	if err != nil {
		return nil, err
	}

	// check hash to authenticate body
	chash := crypto.Sha3Hash(body)
	if chash != hash {
		return nil, fmt.Errorf("body hash mismatch %x != %x (exp)", hash[:], chash[:])
	}

	return body, nil

}

// type HTTPClientAPI interface {
// 	Get(uri string) (*http.Response, err)
// 	Download(uri, path string) error
// 	GetBody(uri) ([]byte, error)
// 	HasScheme(scheme string) bool
// 	GetAuthBody(uri string, hash common.Hash) ([]byte, error)
// }
