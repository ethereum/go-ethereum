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

	"bytes"
	"mime"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var hashMatcher = regexp.MustCompile("^[0-9A-Fa-f]{64}")

type Resolver interface {
	Resolve(string) (common.Hash, error)
}

// errNoResolver is returned by MultiResolver.Resolve if no resolver
// can be found for the address.
var errNoResolver = errors.New("no resolver")

// MultiResolver is used to resolve URL addresses based on their TLDs.
// Each TLD can have multiple resolvers, and the resoluton from the
// first one in the sequence will be returned.
type MultiResolver struct {
	resolvers map[string][]Resolver
}

// MultiResolverOption sets options for MultiResolver and is used as
// arguments for its constructor.
type MultiResolverOption func(*MultiResolver)

// MultiResolverOptionWithResolver adds a Resolver to a list of resolvers
// for a specific TLD. If TLD is an empty string, the resolver will be added
// to the list of default resolver, the ones that will be used for resolution
// of addresses which do not have their TLD resolver specified.
func MultiResolverOptionWithResolver(r Resolver, tld string) MultiResolverOption {
	return func(m *MultiResolver) {
		m.resolvers[tld] = append(m.resolvers[tld], r)
	}
}

// NewMultiResolver creates a new instance of MultiResolver.
func NewMultiResolver(opts ...MultiResolverOption) (m *MultiResolver) {
	m = &MultiResolver{
		resolvers: make(map[string][]Resolver),
	}
	for _, o := range opts {
		o(m)
	}
	return m
}

// Resolve resolves address by choosing a Resolver by TLD.
// If there are more default Resolvers, or for a specific TLD,
// the Hash from the the first one which does not return error
// will be returned.
func (m MultiResolver) Resolve(addr string) (h common.Hash, err error) {
	rs := m.resolvers[""]
	if i := strings.LastIndex(addr, "."); i >= 0 {
		rstld, ok := m.resolvers[addr[i+1:]]
		if ok {
			rs = rstld
		}
	}
	if rs == nil {
		return h, errNoResolver
	}
	for _, r := range rs {
		h, err = r.Resolve(addr)
		if err == nil {
			return
		}
	}
	return
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

// to be used only in TEST
func (self *Api) Upload(uploadDir, index string) (hash string, err error) {
	fs := NewFileSystem(self)
	hash, err = fs.Upload(uploadDir, index)
	return hash, err
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

	// if the URI is immutable, check if the address is a hash
	isHash := hashMatcher.MatchString(uri.Addr)
	if uri.Immutable() {
		if !isHash {
			return nil, fmt.Errorf("immutable address not a content hash: %q", uri.Addr)
		}
		return common.Hex2Bytes(uri.Addr), nil
	}

	// if DNS is not configured, check if the address is a hash
	if self.dns == nil {
		if !isHash {
			return nil, fmt.Errorf("no DNS to resolve name: %q", uri.Addr)
		}
		return common.Hex2Bytes(uri.Addr), nil
	}

	// try and resolve the address
	resolved, err := self.dns.Resolve(uri.Addr)
	if err == nil {
		return resolved[:], nil
	} else if !isHash {
		return nil, err
	}
	return common.Hex2Bytes(uri.Addr), nil
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
// to resolve basePath to content using dpa retrieve
// it returns a section reader, mimeType, status and an error
func (self *Api) Get(key storage.Key, path string) (reader storage.LazySectionReader, mimeType string, status int, err error) {
	trie, err := loadManifest(self.dpa, key, nil)
	if err != nil {
		status = http.StatusNotFound
		log.Warn(fmt.Sprintf("loadManifestTrie error: %v", err))
		return
	}

	log.Trace(fmt.Sprintf("getEntry(%s)", path))

	entry, _ := trie.getEntry(path)

	if entry != nil {
		key = common.Hex2Bytes(entry.Hash)
		status = entry.Status
		if status == http.StatusMultipleChoices {
			return
		} else {
			mimeType = entry.ContentType
			log.Trace(fmt.Sprintf("content lookup key: '%v' (%v)", key, mimeType))
			reader = self.dpa.Retrieve(key)
		}
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

func (self *Api) AddFile(mhash, path, fname string, content []byte, nameresolver bool) (storage.Key, string, error) {

	uri, err := Parse("bzz:/" + mhash)
	if err != nil {
		return nil, "", err
	}
	mkey, err := self.Resolve(uri)
	if err != nil {
		return nil, "", err
	}

	// trim the root dir we added
	if path[:1] == "/" {
		path = path[1:]
	}

	entry := &ManifestEntry{
		Path:        filepath.Join(path, fname),
		ContentType: mime.TypeByExtension(filepath.Ext(fname)),
		Mode:        0700,
		Size:        int64(len(content)),
		ModTime:     time.Now(),
	}

	mw, err := self.NewManifestWriter(mkey, nil)
	if err != nil {
		return nil, "", err
	}

	fkey, err := mw.AddEntry(bytes.NewReader(content), entry)
	if err != nil {
		return nil, "", err
	}

	newMkey, err := mw.Store()
	if err != nil {
		return nil, "", err

	}

	return fkey, newMkey.String(), nil

}

func (self *Api) RemoveFile(mhash, path, fname string, nameresolver bool) (string, error) {

	uri, err := Parse("bzz:/" + mhash)
	if err != nil {
		return "", err
	}
	mkey, err := self.Resolve(uri)
	if err != nil {
		return "", err
	}

	// trim the root dir we added
	if path[:1] == "/" {
		path = path[1:]
	}

	mw, err := self.NewManifestWriter(mkey, nil)
	if err != nil {
		return "", err
	}

	err = mw.RemoveEntry(filepath.Join(path, fname))
	if err != nil {
		return "", err
	}

	newMkey, err := mw.Store()
	if err != nil {
		return "", err

	}

	return newMkey.String(), nil
}

func (self *Api) AppendFile(mhash, path, fname string, existingSize int64, content []byte, oldKey storage.Key, offset int64, addSize int64, nameresolver bool) (storage.Key, string, error) {

	buffSize := offset + addSize
	if buffSize < existingSize {
		buffSize = existingSize
	}

	buf := make([]byte, buffSize)

	oldReader := self.Retrieve(oldKey)
	io.ReadAtLeast(oldReader, buf, int(offset))

	newReader := bytes.NewReader(content)
	io.ReadAtLeast(newReader, buf[offset:], int(addSize))

	if buffSize < existingSize {
		io.ReadAtLeast(oldReader, buf[addSize:], int(buffSize))
	}

	combinedReader := bytes.NewReader(buf)
	totalSize := int64(len(buf))

	// TODO(jmozah): to append using pyramid chunker when it is ready
	//oldReader := self.Retrieve(oldKey)
	//newReader := bytes.NewReader(content)
	//combinedReader := io.MultiReader(oldReader, newReader)

	uri, err := Parse("bzz:/" + mhash)
	if err != nil {
		return nil, "", err
	}
	mkey, err := self.Resolve(uri)
	if err != nil {
		return nil, "", err
	}

	// trim the root dir we added
	if path[:1] == "/" {
		path = path[1:]
	}

	mw, err := self.NewManifestWriter(mkey, nil)
	if err != nil {
		return nil, "", err
	}

	err = mw.RemoveEntry(filepath.Join(path, fname))
	if err != nil {
		return nil, "", err
	}

	entry := &ManifestEntry{
		Path:        filepath.Join(path, fname),
		ContentType: mime.TypeByExtension(filepath.Ext(fname)),
		Mode:        0700,
		Size:        totalSize,
		ModTime:     time.Now(),
	}

	fkey, err := mw.AddEntry(io.Reader(combinedReader), entry)
	if err != nil {
		return nil, "", err
	}

	newMkey, err := mw.Store()
	if err != nil {
		return nil, "", err

	}

	return fkey, newMkey.String(), nil

}

func (self *Api) BuildDirectoryTree(mhash string, nameresolver bool) (key storage.Key, manifestEntryMap map[string]*manifestTrieEntry, err error) {
	uri, err := Parse("bzz:/" + mhash)
	if err != nil {
		return nil, nil, err
	}
	key, err = self.Resolve(uri)
	if err != nil {
		return nil, nil, err
	}

	quitC := make(chan bool)
	rootTrie, err := loadManifest(self.dpa, key, quitC)
	if err != nil {
		return nil, nil, fmt.Errorf("can't load manifest %v: %v", key.String(), err)
	}

	manifestEntryMap = map[string]*manifestTrieEntry{}
	err = rootTrie.listWithPrefix(uri.Path, quitC, func(entry *manifestTrieEntry, suffix string) {
		manifestEntryMap[suffix] = entry
	})

	if err != nil {
		return nil, nil, fmt.Errorf("list with prefix failed %v: %v", key.String(), err)
	}
	return key, manifestEntryMap, nil
}
