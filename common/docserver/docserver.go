// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package docserver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type DocServer struct {
	*http.Transport
	DocRoot string
	schemes []string
}

func New(docRoot string) (self *DocServer) {
	self = &DocServer{
		Transport: &http.Transport{},
		DocRoot:   docRoot,
		schemes:   []string{"file"},
	}
	self.DocRoot = "/tmp/"
	self.RegisterProtocol("file", http.NewFileTransport(http.Dir(self.DocRoot)))
	return
}

// Clients should be reused instead of created as needed. Clients are safe for concurrent use by multiple goroutines.

// A Client is higher-level than a RoundTripper (such as Transport) and additionally handles HTTP details such as cookies and redirects.

func (self *DocServer) Client() *http.Client {
	return &http.Client{
		Transport: self,
	}
}

func (self *DocServer) RegisterScheme(scheme string, rt http.RoundTripper) {
	self.schemes = append(self.schemes, scheme)
	self.RegisterProtocol(scheme, rt)
}

func (self *DocServer) HasScheme(scheme string) bool {
	for _, s := range self.schemes {
		if s == scheme {
			return true
		}
	}
	return false
}

func (self *DocServer) GetAuthContent(uri string, hash common.Hash) (content []byte, err error) {
	// retrieve content
	content, err = self.Get(uri, "")
	if err != nil {
		return
	}

	// check hash to authenticate content
	chash := crypto.Sha3Hash(content)
	if chash != hash {
		content = nil
		err = fmt.Errorf("content hash mismatch %x != %x (exp)", hash[:], chash[:])
	}

	return

}

// Get(uri, path) downloads the document at uri, if path is non-empty it
// is interpreted as a filepath to which the contents are saved
func (self *DocServer) Get(uri, path string) (content []byte, err error) {
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

	if path != "" {
		var abspath string
		abspath, err = filepath.Abs(path)
		ioutil.WriteFile(abspath, content, 0700)
	}

	return

}
