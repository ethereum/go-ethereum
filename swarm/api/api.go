// Copyright 2016 The go-ethereum Authors
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

package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var (
	hashMatcher      = regexp.MustCompile("^[0-9A-Fa-f]{64}")
	slashes          = regexp.MustCompile("/+")
	domainAndVersion = regexp.MustCompile("[@:;,]+")
)

type Resolver interface {
	Resolve(string) (common.Hash, error)
}

/*
Api implements webserver/file system related content storage and retrieval
on top of the dpa
it is the public interface of the dpa which is included in the ethereum stack
*/
type Api struct {
	dpa *storage.DPA
	dns Resolver
}

//the api constructor initialises
func NewApi(dpa *storage.DPA, dns Resolver) (self *Api) {
	self = &Api{
		dpa: dpa,
		dns: dns,
	}
	return
}

// DPA reader API
func (self *Api) Retrieve(key storage.Key) storage.LazySectionReader {
	return self.dpa.Retrieve(key)
}

func (self *Api) Store(data io.Reader, size int64, wg *sync.WaitGroup) (key storage.Key, err error) {
	return self.dpa.Store(data, size, wg, nil)
}

type ErrResolve error

// DNS Resolver
func (self *Api) Resolve(uri *URI) (storage.Key, error) {
	log.Trace(fmt.Sprintf("Resolving : %v", uri.Addr))
	if hashMatcher.MatchString(uri.Addr) {
		log.Trace(fmt.Sprintf("addr is a hash: %q", uri.Addr))
		return storage.Key(common.Hex2Bytes(uri.Addr)), nil
	}
	if uri.Immutable() {
		return nil, errors.New("refusing to resolve immutable address")
	}
	if self.dns == nil {
		return nil, fmt.Errorf("unable to resolve addr %q, resolver not configured", uri.Addr)
	}
	hash, err := self.dns.Resolve(uri.Addr)
	if err != nil {
		log.Warn(fmt.Sprintf("DNS error resolving addr %q: %s", uri.Addr, err))
		return nil, ErrResolve(err)
	}
	log.Trace(fmt.Sprintf("addr lookup: %v -> %v", uri.Addr, hash))
	return hash[:], nil
}

// Put provides singleton manifest creation on top of dpa store
func (self *Api) Put(content, contentType string) (storage.Key, error) {
	r := strings.NewReader(content)
	wg := &sync.WaitGroup{}
	key, err := self.dpa.Store(r, int64(len(content)), wg, nil)
	if err != nil {
		return nil, err
	}
	manifest := fmt.Sprintf(`{"entries":[{"hash":"%v","contentType":"%s"}]}`, key, contentType)
	r = strings.NewReader(manifest)
	key, err = self.dpa.Store(r, int64(len(manifest)), wg, nil)
	if err != nil {
		return nil, err
	}
	wg.Wait()
	return key, nil
}

// Get uses iterative manifest retrieval and prefix matching
// to resolve path to content using dpa retrieve
// it returns a section reader, mimeType, status and an error
func (self *Api) Get(key storage.Key, path string) (reader storage.LazySectionReader, mimeType string, status int, err error) {
	trie, err := loadManifest(self.dpa, key, nil)
	if err != nil {
		log.Warn(fmt.Sprintf("loadManifestTrie error: %v", err))
		return
	}

	log.Trace(fmt.Sprintf("getEntry(%s)", path))

	entry, _ := trie.getEntry(path)

	if entry != nil {
		key = common.Hex2Bytes(entry.Hash)
		status = entry.Status
		mimeType = entry.ContentType
		log.Trace(fmt.Sprintf("content lookup key: '%v' (%v)", key, mimeType))
		reader = self.dpa.Retrieve(key)
	} else {
		status = http.StatusNotFound
		err = fmt.Errorf("manifest entry for '%s' not found", path)
		log.Warn(fmt.Sprintf("%v", err))
	}
	return
}

func (self *Api) Modify(key storage.Key, path, contentHash, contentType string) (storage.Key, error) {
	quitC := make(chan bool)
	trie, err := loadManifest(self.dpa, key, quitC)
	if err != nil {
		return nil, err
	}
	if contentHash != "" {
		entry := newManifestTrieEntry(&ManifestEntry{
			Path:        path,
			ContentType: contentType,
		}, nil)
		entry.Hash = contentHash
		trie.addEntry(entry, quitC)
	} else {
		trie.deleteEntry(path, quitC)
	}

	if err := trie.recalcAndStore(); err != nil {
		return nil, err
	}
	return trie.hash, nil
}
