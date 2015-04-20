package docserver

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// http://golang.org/pkg/net/http/#RoundTripper
var (
	schemes = map[string]func(*DocServer) http.RoundTripper{
		// Simple File server from local disk file:///etc/passwd :)
		"file": fileServerOnDocRoot,
	}
)

func fileServerOnDocRoot(ds *DocServer) http.RoundTripper {
	return http.NewFileTransport(http.Dir(ds.DocRoot))
}

type DocServer struct {
	*http.Transport
	DocRoot string
}

func New(docRoot string) (self *DocServer, err error) {
	self = &DocServer{
		Transport: &http.Transport{},
		DocRoot:   docRoot,
	}
	err = self.RegisterProtocols(schemes)
	return
}

// Clients should be reused instead of created as needed. Clients are safe for concurrent use by multiple goroutines.

// A Client is higher-level than a RoundTripper (such as Transport) and additionally handles HTTP details such as cookies and redirects.

func (self *DocServer) Client() *http.Client {
	return &http.Client{
		Transport: self,
	}
}

func (self *DocServer) RegisterProtocols(schemes map[string]func(*DocServer) http.RoundTripper) (err error) {
	for scheme, rtf := range schemes {
		self.RegisterProtocol(scheme, rtf(self))
	}
	return
}

func (self *DocServer) GetAuthContent(uri string, hash common.Hash) (content []byte, err error) {
	// retrieve content
	resp, err := self.Client().Get(uri)
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()
	if err != nil {
		return
	}
	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// check hash to authenticate content
	hashbytes := crypto.Sha3(content)
	var chash common.Hash
	copy(chash[:], hashbytes)
	if chash != hash {
		content = nil
		err = fmt.Errorf("content hash mismatch")
	}

	return

}
