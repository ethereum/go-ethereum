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

//go:generate mimegen --types=./../../cmd/swarm/mimegen/mime.types --package=api --out=gen_mime.go
//go:generate gofmt -s -w gen_mime.go

import (
	"archive/tar"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"path"
	"strings"

	"bytes"
	"mime"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/spancontext"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/feed"
	"github.com/ethereum/go-ethereum/swarm/storage/feed/lookup"

	opentracing "github.com/opentracing/opentracing-go"
)

var (
	ErrNotFound = errors.New("not found")
)

var (
	apiResolveCount        = metrics.NewRegisteredCounter("api.resolve.count", nil)
	apiResolveFail         = metrics.NewRegisteredCounter("api.resolve.fail", nil)
	apiPutCount            = metrics.NewRegisteredCounter("api.put.count", nil)
	apiPutFail             = metrics.NewRegisteredCounter("api.put.fail", nil)
	apiGetCount            = metrics.NewRegisteredCounter("api.get.count", nil)
	apiGetNotFound         = metrics.NewRegisteredCounter("api.get.notfound", nil)
	apiGetHTTP300          = metrics.NewRegisteredCounter("api.get.http.300", nil)
	apiManifestUpdateCount = metrics.NewRegisteredCounter("api.manifestupdate.count", nil)
	apiManifestUpdateFail  = metrics.NewRegisteredCounter("api.manifestupdate.fail", nil)
	apiManifestListCount   = metrics.NewRegisteredCounter("api.manifestlist.count", nil)
	apiManifestListFail    = metrics.NewRegisteredCounter("api.manifestlist.fail", nil)
	apiDeleteCount         = metrics.NewRegisteredCounter("api.delete.count", nil)
	apiDeleteFail          = metrics.NewRegisteredCounter("api.delete.fail", nil)
	apiGetTarCount         = metrics.NewRegisteredCounter("api.gettar.count", nil)
	apiGetTarFail          = metrics.NewRegisteredCounter("api.gettar.fail", nil)
	apiUploadTarCount      = metrics.NewRegisteredCounter("api.uploadtar.count", nil)
	apiUploadTarFail       = metrics.NewRegisteredCounter("api.uploadtar.fail", nil)
	apiModifyCount         = metrics.NewRegisteredCounter("api.modify.count", nil)
	apiModifyFail          = metrics.NewRegisteredCounter("api.modify.fail", nil)
	apiAddFileCount        = metrics.NewRegisteredCounter("api.addfile.count", nil)
	apiAddFileFail         = metrics.NewRegisteredCounter("api.addfile.fail", nil)
	apiRmFileCount         = metrics.NewRegisteredCounter("api.removefile.count", nil)
	apiRmFileFail          = metrics.NewRegisteredCounter("api.removefile.fail", nil)
	apiAppendFileCount     = metrics.NewRegisteredCounter("api.appendfile.count", nil)
	apiAppendFileFail      = metrics.NewRegisteredCounter("api.appendfile.fail", nil)
	apiGetInvalid          = metrics.NewRegisteredCounter("api.get.invalid", nil)
)

// Resolver interface resolve a domain name to a hash using ENS
type Resolver interface {
	Resolve(string) (common.Hash, error)
}

// ResolveValidator is used to validate the contained Resolver
type ResolveValidator interface {
	Resolver
	Owner(node [32]byte) (common.Address, error)
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
}

// NoResolverError is returned by MultiResolver.Resolve if no resolver
// can be found for the address.
type NoResolverError struct {
	TLD string
}

// NewNoResolverError creates a NoResolverError for the given top level domain
func NewNoResolverError(tld string) *NoResolverError {
	return &NoResolverError{TLD: tld}
}

// Error NoResolverError implements error
func (e *NoResolverError) Error() string {
	if e.TLD == "" {
		return "no ENS resolver"
	}
	return fmt.Sprintf("no ENS endpoint configured to resolve .%s TLD names", e.TLD)
}

// MultiResolver is used to resolve URL addresses based on their TLDs.
// Each TLD can have multiple resolvers, and the resolution from the
// first one in the sequence will be returned.
type MultiResolver struct {
	resolvers map[string][]ResolveValidator
	nameHash  func(string) common.Hash
}

// MultiResolverOption sets options for MultiResolver and is used as
// arguments for its constructor.
type MultiResolverOption func(*MultiResolver)

// MultiResolverOptionWithResolver adds a Resolver to a list of resolvers
// for a specific TLD. If TLD is an empty string, the resolver will be added
// to the list of default resolver, the ones that will be used for resolution
// of addresses which do not have their TLD resolver specified.
func MultiResolverOptionWithResolver(r ResolveValidator, tld string) MultiResolverOption {
	return func(m *MultiResolver) {
		m.resolvers[tld] = append(m.resolvers[tld], r)
	}
}

// MultiResolverOptionWithNameHash is unused at the time of this writing
func MultiResolverOptionWithNameHash(nameHash func(string) common.Hash) MultiResolverOption {
	return func(m *MultiResolver) {
		m.nameHash = nameHash
	}
}

// NewMultiResolver creates a new instance of MultiResolver.
func NewMultiResolver(opts ...MultiResolverOption) (m *MultiResolver) {
	m = &MultiResolver{
		resolvers: make(map[string][]ResolveValidator),
		nameHash:  ens.EnsNode,
	}
	for _, o := range opts {
		o(m)
	}
	return m
}

// Resolve resolves address by choosing a Resolver by TLD.
// If there are more default Resolvers, or for a specific TLD,
// the Hash from the first one which does not return error
// will be returned.
func (m *MultiResolver) Resolve(addr string) (h common.Hash, err error) {
	rs, err := m.getResolveValidator(addr)
	if err != nil {
		return h, err
	}
	for _, r := range rs {
		h, err = r.Resolve(addr)
		if err == nil {
			return
		}
	}
	return
}

// ValidateOwner checks the ENS to validate that the owner of the given domain is the given eth address
func (m *MultiResolver) ValidateOwner(name string, address common.Address) (bool, error) {
	rs, err := m.getResolveValidator(name)
	if err != nil {
		return false, err
	}
	var addr common.Address
	for _, r := range rs {
		addr, err = r.Owner(m.nameHash(name))
		// we hide the error if it is not for the last resolver we check
		if err == nil {
			return addr == address, nil
		}
	}
	return false, err
}

// HeaderByNumber uses the validator of the given domainname and retrieves the header for the given block number
func (m *MultiResolver) HeaderByNumber(ctx context.Context, name string, blockNr *big.Int) (*types.Header, error) {
	rs, err := m.getResolveValidator(name)
	if err != nil {
		return nil, err
	}
	for _, r := range rs {
		var header *types.Header
		header, err = r.HeaderByNumber(ctx, blockNr)
		// we hide the error if it is not for the last resolver we check
		if err == nil {
			return header, nil
		}
	}
	return nil, err
}

// getResolveValidator uses the hostname to retrieve the resolver associated with the top level domain
func (m *MultiResolver) getResolveValidator(name string) ([]ResolveValidator, error) {
	rs := m.resolvers[""]
	tld := path.Ext(name)
	if tld != "" {
		tld = tld[1:]
		rstld, ok := m.resolvers[tld]
		if ok {
			return rstld, nil
		}
	}
	if len(rs) == 0 {
		return rs, NewNoResolverError(tld)
	}
	return rs, nil
}

// SetNameHash sets the hasher function that hashes the domain into a name hash that ENS uses
func (m *MultiResolver) SetNameHash(nameHash func(string) common.Hash) {
	m.nameHash = nameHash
}

/*
API implements webserver/file system related content storage and retrieval
on top of the FileStore
it is the public interface of the FileStore which is included in the ethereum stack
*/
type API struct {
	feed      *feed.Handler
	fileStore *storage.FileStore
	dns       Resolver
	Decryptor func(context.Context, string) DecryptFunc
}

// NewAPI the api constructor initialises a new API instance.
func NewAPI(fileStore *storage.FileStore, dns Resolver, feedHandler *feed.Handler, pk *ecdsa.PrivateKey) (self *API) {
	self = &API{
		fileStore: fileStore,
		dns:       dns,
		feed:      feedHandler,
		Decryptor: func(ctx context.Context, credentials string) DecryptFunc {
			return self.doDecrypt(ctx, credentials, pk)
		},
	}
	return
}

// Retrieve FileStore reader API
func (a *API) Retrieve(ctx context.Context, addr storage.Address) (reader storage.LazySectionReader, isEncrypted bool) {
	return a.fileStore.Retrieve(ctx, addr)
}

// Store wraps the Store API call of the embedded FileStore
func (a *API) Store(ctx context.Context, data io.Reader, size int64, toEncrypt bool) (addr storage.Address, wait func(ctx context.Context) error, err error) {
	log.Debug("api.store", "size", size)
	return a.fileStore.Store(ctx, data, size, toEncrypt)
}

// ErrResolve is returned when an URI cannot be resolved from ENS.
type ErrResolve error

// Resolve a name into a content-addressed hash
// where address could be an ENS name, or a content addressed hash
func (a *API) Resolve(ctx context.Context, address string) (storage.Address, error) {
	// if DNS is not configured, return an error
	if a.dns == nil {
		if hashMatcher.MatchString(address) {
			return common.Hex2Bytes(address), nil
		}
		apiResolveFail.Inc(1)
		return nil, fmt.Errorf("no DNS to resolve name: %q", address)
	}
	// try and resolve the address
	resolved, err := a.dns.Resolve(address)
	if err != nil {
		if hashMatcher.MatchString(address) {
			return common.Hex2Bytes(address), nil
		}
		return nil, err
	}
	return resolved[:], nil
}

// Resolve resolves a URI to an Address using the MultiResolver.
func (a *API) ResolveURI(ctx context.Context, uri *URI, credentials string) (storage.Address, error) {
	apiResolveCount.Inc(1)
	log.Trace("resolving", "uri", uri.Addr)

	var sp opentracing.Span
	ctx, sp = spancontext.StartSpan(
		ctx,
		"api.resolve")
	defer sp.Finish()

	// if the URI is immutable, check if the address looks like a hash
	if uri.Immutable() {
		key := uri.Address()
		if key == nil {
			return nil, fmt.Errorf("immutable address not a content hash: %q", uri.Addr)
		}
		return key, nil
	}

	addr, err := a.Resolve(ctx, uri.Addr)
	if err != nil {
		return nil, err
	}

	if uri.Path == "" {
		return addr, nil
	}
	walker, err := a.NewManifestWalker(ctx, addr, a.Decryptor(ctx, credentials), nil)
	if err != nil {
		return nil, err
	}
	var entry *ManifestEntry
	walker.Walk(func(e *ManifestEntry) error {
		// if the entry matches the path, set entry and stop
		// the walk
		if e.Path == uri.Path {
			entry = e
			// return an error to cancel the walk
			return errors.New("found")
		}
		// ignore non-manifest files
		if e.ContentType != ManifestType {
			return nil
		}
		// if the manifest's path is a prefix of the
		// requested path, recurse into it by returning
		// nil and continuing the walk
		if strings.HasPrefix(uri.Path, e.Path) {
			return nil
		}
		return ErrSkipManifest
	})
	if entry == nil {
		return nil, errors.New("not found")
	}
	addr = storage.Address(common.Hex2Bytes(entry.Hash))
	return addr, nil
}

// Put provides singleton manifest creation on top of FileStore store
func (a *API) Put(ctx context.Context, content string, contentType string, toEncrypt bool) (k storage.Address, wait func(context.Context) error, err error) {
	apiPutCount.Inc(1)
	r := strings.NewReader(content)
	key, waitContent, err := a.fileStore.Store(ctx, r, int64(len(content)), toEncrypt)
	if err != nil {
		apiPutFail.Inc(1)
		return nil, nil, err
	}
	manifest := fmt.Sprintf(`{"entries":[{"hash":"%v","contentType":"%s"}]}`, key, contentType)
	r = strings.NewReader(manifest)
	key, waitManifest, err := a.fileStore.Store(ctx, r, int64(len(manifest)), toEncrypt)
	if err != nil {
		apiPutFail.Inc(1)
		return nil, nil, err
	}
	return key, func(ctx context.Context) error {
		err := waitContent(ctx)
		if err != nil {
			return err
		}
		return waitManifest(ctx)
	}, nil
}

// Get uses iterative manifest retrieval and prefix matching
// to resolve basePath to content using FileStore retrieve
// it returns a section reader, mimeType, status, the key of the actual content and an error
func (a *API) Get(ctx context.Context, decrypt DecryptFunc, manifestAddr storage.Address, path string) (reader storage.LazySectionReader, mimeType string, status int, contentAddr storage.Address, err error) {
	log.Debug("api.get", "key", manifestAddr, "path", path)
	apiGetCount.Inc(1)
	trie, err := loadManifest(ctx, a.fileStore, manifestAddr, nil, decrypt)
	if err != nil {
		apiGetNotFound.Inc(1)
		status = http.StatusNotFound
		return nil, "", http.StatusNotFound, nil, err
	}

	log.Debug("trie getting entry", "key", manifestAddr, "path", path)
	entry, _ := trie.getEntry(path)

	if entry != nil {
		log.Debug("trie got entry", "key", manifestAddr, "path", path, "entry.Hash", entry.Hash)

		if entry.ContentType == ManifestType {
			log.Debug("entry is manifest", "key", manifestAddr, "new key", entry.Hash)
			adr, err := hex.DecodeString(entry.Hash)
			if err != nil {
				return nil, "", 0, nil, err
			}
			return a.Get(ctx, decrypt, adr, entry.Path)
		}

		// we need to do some extra work if this is a Swarm feed manifest
		if entry.ContentType == FeedContentType {
			if entry.Feed == nil {
				return reader, mimeType, status, nil, fmt.Errorf("Cannot decode Feed in manifest")
			}
			_, err := a.feed.Lookup(ctx, feed.NewQueryLatest(entry.Feed, lookup.NoClue))
			if err != nil {
				apiGetNotFound.Inc(1)
				status = http.StatusNotFound
				log.Debug(fmt.Sprintf("get feed update content error: %v", err))
				return reader, mimeType, status, nil, err
			}
			// get the data of the update
			_, contentAddr, err := a.feed.GetContent(entry.Feed)
			if err != nil {
				apiGetNotFound.Inc(1)
				status = http.StatusNotFound
				log.Warn(fmt.Sprintf("get feed update content error: %v", err))
				return reader, mimeType, status, nil, err
			}

			// extract content hash
			if len(contentAddr) != storage.AddressLength {
				apiGetInvalid.Inc(1)
				status = http.StatusUnprocessableEntity
				errorMessage := fmt.Sprintf("invalid swarm hash in feed update. Expected %d bytes. Got %d", storage.AddressLength, len(contentAddr))
				log.Warn(errorMessage)
				return reader, mimeType, status, nil, errors.New(errorMessage)
			}
			manifestAddr = storage.Address(contentAddr)
			log.Trace("feed update contains swarm hash", "key", manifestAddr)

			// get the manifest the swarm hash points to
			trie, err := loadManifest(ctx, a.fileStore, manifestAddr, nil, NOOPDecrypt)
			if err != nil {
				apiGetNotFound.Inc(1)
				status = http.StatusNotFound
				log.Warn(fmt.Sprintf("loadManifestTrie (feed update) error: %v", err))
				return reader, mimeType, status, nil, err
			}

			// finally, get the manifest entry
			// it will always be the entry on path ""
			entry, _ = trie.getEntry(path)
			if entry == nil {
				status = http.StatusNotFound
				apiGetNotFound.Inc(1)
				err = fmt.Errorf("manifest (feed update) entry for '%s' not found", path)
				log.Trace("manifest (feed update) entry not found", "key", manifestAddr, "path", path)
				return reader, mimeType, status, nil, err
			}
		}

		// regardless of feed update manifests or normal manifests we will converge at this point
		// get the key the manifest entry points to and serve it if it's unambiguous
		contentAddr = common.Hex2Bytes(entry.Hash)
		status = entry.Status
		if status == http.StatusMultipleChoices {
			apiGetHTTP300.Inc(1)
			return nil, entry.ContentType, status, contentAddr, err
		}
		mimeType = entry.ContentType
		log.Debug("content lookup key", "key", contentAddr, "mimetype", mimeType)
		reader, _ = a.fileStore.Retrieve(ctx, contentAddr)
	} else {
		// no entry found
		status = http.StatusNotFound
		apiGetNotFound.Inc(1)
		err = fmt.Errorf("Not found: could not find resource '%s'", path)
		log.Trace("manifest entry not found", "key", contentAddr, "path", path)
	}
	return
}

func (a *API) Delete(ctx context.Context, addr string, path string) (storage.Address, error) {
	apiDeleteCount.Inc(1)
	uri, err := Parse("bzz:/" + addr)
	if err != nil {
		apiDeleteFail.Inc(1)
		return nil, err
	}
	key, err := a.ResolveURI(ctx, uri, EMPTY_CREDENTIALS)

	if err != nil {
		return nil, err
	}
	newKey, err := a.UpdateManifest(ctx, key, func(mw *ManifestWriter) error {
		log.Debug(fmt.Sprintf("removing %s from manifest %s", path, key.Log()))
		return mw.RemoveEntry(path)
	})
	if err != nil {
		apiDeleteFail.Inc(1)
		return nil, err
	}

	return newKey, nil
}

// GetDirectoryTar fetches a requested directory as a tarstream
// it returns an io.Reader and an error. Do not forget to Close() the returned ReadCloser
func (a *API) GetDirectoryTar(ctx context.Context, decrypt DecryptFunc, uri *URI) (io.ReadCloser, error) {
	apiGetTarCount.Inc(1)
	addr, err := a.Resolve(ctx, uri.Addr)
	if err != nil {
		return nil, err
	}
	walker, err := a.NewManifestWalker(ctx, addr, decrypt, nil)
	if err != nil {
		apiGetTarFail.Inc(1)
		return nil, err
	}

	piper, pipew := io.Pipe()

	tw := tar.NewWriter(pipew)

	go func() {
		err := walker.Walk(func(entry *ManifestEntry) error {
			// ignore manifests (walk will recurse into them)
			if entry.ContentType == ManifestType {
				return nil
			}

			// retrieve the entry's key and size
			reader, _ := a.Retrieve(ctx, storage.Address(common.Hex2Bytes(entry.Hash)))
			size, err := reader.Size(ctx, nil)
			if err != nil {
				return err
			}

			// write a tar header for the entry
			hdr := &tar.Header{
				Name:    entry.Path,
				Mode:    entry.Mode,
				Size:    size,
				ModTime: entry.ModTime,
				Xattrs: map[string]string{
					"user.swarm.content-type": entry.ContentType,
				},
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			// copy the file into the tar stream
			n, err := io.Copy(tw, io.LimitReader(reader, hdr.Size))
			if err != nil {
				return err
			} else if n != size {
				return fmt.Errorf("error writing %s: expected %d bytes but sent %d", entry.Path, size, n)
			}

			return nil
		})
		// close tar writer before closing pipew
		// to flush remaining data to pipew
		// regardless of error value
		tw.Close()
		if err != nil {
			apiGetTarFail.Inc(1)
			pipew.CloseWithError(err)
		} else {
			pipew.Close()
		}
	}()

	return piper, nil
}

// GetManifestList lists the manifest entries for the specified address and prefix
// and returns it as a ManifestList
func (a *API) GetManifestList(ctx context.Context, decryptor DecryptFunc, addr storage.Address, prefix string) (list ManifestList, err error) {
	apiManifestListCount.Inc(1)
	walker, err := a.NewManifestWalker(ctx, addr, decryptor, nil)
	if err != nil {
		apiManifestListFail.Inc(1)
		return ManifestList{}, err
	}

	err = walker.Walk(func(entry *ManifestEntry) error {
		// handle non-manifest files
		if entry.ContentType != ManifestType {
			// ignore the file if it doesn't have the specified prefix
			if !strings.HasPrefix(entry.Path, prefix) {
				return nil
			}

			// if the path after the prefix contains a slash, add a
			// common prefix to the list, otherwise add the entry
			suffix := strings.TrimPrefix(entry.Path, prefix)
			if index := strings.Index(suffix, "/"); index > -1 {
				list.CommonPrefixes = append(list.CommonPrefixes, prefix+suffix[:index+1])
				return nil
			}
			if entry.Path == "" {
				entry.Path = "/"
			}
			list.Entries = append(list.Entries, entry)
			return nil
		}

		// if the manifest's path is a prefix of the specified prefix
		// then just recurse into the manifest by returning nil and
		// continuing the walk
		if strings.HasPrefix(prefix, entry.Path) {
			return nil
		}

		// if the manifest's path has the specified prefix, then if the
		// path after the prefix contains a slash, add a common prefix
		// to the list and skip the manifest, otherwise recurse into
		// the manifest by returning nil and continuing the walk
		if strings.HasPrefix(entry.Path, prefix) {
			suffix := strings.TrimPrefix(entry.Path, prefix)
			if index := strings.Index(suffix, "/"); index > -1 {
				list.CommonPrefixes = append(list.CommonPrefixes, prefix+suffix[:index+1])
				return ErrSkipManifest
			}
			return nil
		}

		// the manifest neither has the prefix or needs recursing in to
		// so just skip it
		return ErrSkipManifest
	})

	if err != nil {
		apiManifestListFail.Inc(1)
		return ManifestList{}, err
	}

	return list, nil
}

func (a *API) UpdateManifest(ctx context.Context, addr storage.Address, update func(mw *ManifestWriter) error) (storage.Address, error) {
	apiManifestUpdateCount.Inc(1)
	mw, err := a.NewManifestWriter(ctx, addr, nil)
	if err != nil {
		apiManifestUpdateFail.Inc(1)
		return nil, err
	}

	if err := update(mw); err != nil {
		apiManifestUpdateFail.Inc(1)
		return nil, err
	}

	addr, err = mw.Store()
	if err != nil {
		apiManifestUpdateFail.Inc(1)
		return nil, err
	}
	log.Debug(fmt.Sprintf("generated manifest %s", addr))
	return addr, nil
}

// Modify loads manifest and checks the content hash before recalculating and storing the manifest.
func (a *API) Modify(ctx context.Context, addr storage.Address, path, contentHash, contentType string) (storage.Address, error) {
	apiModifyCount.Inc(1)
	quitC := make(chan bool)
	trie, err := loadManifest(ctx, a.fileStore, addr, quitC, NOOPDecrypt)
	if err != nil {
		apiModifyFail.Inc(1)
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
		apiModifyFail.Inc(1)
		return nil, err
	}
	return trie.ref, nil
}

// AddFile creates a new manifest entry, adds it to swarm, then adds a file to swarm.
func (a *API) AddFile(ctx context.Context, mhash, path, fname string, content []byte, nameresolver bool) (storage.Address, string, error) {
	apiAddFileCount.Inc(1)

	uri, err := Parse("bzz:/" + mhash)
	if err != nil {
		apiAddFileFail.Inc(1)
		return nil, "", err
	}
	mkey, err := a.ResolveURI(ctx, uri, EMPTY_CREDENTIALS)
	if err != nil {
		apiAddFileFail.Inc(1)
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

	mw, err := a.NewManifestWriter(ctx, mkey, nil)
	if err != nil {
		apiAddFileFail.Inc(1)
		return nil, "", err
	}

	fkey, err := mw.AddEntry(ctx, bytes.NewReader(content), entry)
	if err != nil {
		apiAddFileFail.Inc(1)
		return nil, "", err
	}

	newMkey, err := mw.Store()
	if err != nil {
		apiAddFileFail.Inc(1)
		return nil, "", err

	}

	return fkey, newMkey.String(), nil
}

func (a *API) UploadTar(ctx context.Context, bodyReader io.ReadCloser, manifestPath, defaultPath string, mw *ManifestWriter) (storage.Address, error) {
	apiUploadTarCount.Inc(1)
	var contentKey storage.Address
	tr := tar.NewReader(bodyReader)
	defer bodyReader.Close()
	var defaultPathFound bool
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			apiUploadTarFail.Inc(1)
			return nil, fmt.Errorf("error reading tar stream: %s", err)
		}

		// only store regular files
		if !hdr.FileInfo().Mode().IsRegular() {
			continue
		}

		// add the entry under the path from the request
		manifestPath := path.Join(manifestPath, hdr.Name)
		contentType := hdr.Xattrs["user.swarm.content-type"]
		if contentType == "" {
			contentType = mime.TypeByExtension(filepath.Ext(hdr.Name))
		}
		//DetectContentType("")
		entry := &ManifestEntry{
			Path:        manifestPath,
			ContentType: contentType,
			Mode:        hdr.Mode,
			Size:        hdr.Size,
			ModTime:     hdr.ModTime,
		}
		contentKey, err = mw.AddEntry(ctx, tr, entry)
		if err != nil {
			apiUploadTarFail.Inc(1)
			return nil, fmt.Errorf("error adding manifest entry from tar stream: %s", err)
		}
		if hdr.Name == defaultPath {
			contentType := hdr.Xattrs["user.swarm.content-type"]
			if contentType == "" {
				contentType = mime.TypeByExtension(filepath.Ext(hdr.Name))
			}

			entry := &ManifestEntry{
				Hash:        contentKey.Hex(),
				Path:        "", // default entry
				ContentType: contentType,
				Mode:        hdr.Mode,
				Size:        hdr.Size,
				ModTime:     hdr.ModTime,
			}
			contentKey, err = mw.AddEntry(ctx, nil, entry)
			if err != nil {
				apiUploadTarFail.Inc(1)
				return nil, fmt.Errorf("error adding default manifest entry from tar stream: %s", err)
			}
			defaultPathFound = true
		}
	}
	if defaultPath != "" && !defaultPathFound {
		return contentKey, fmt.Errorf("default path %q not found", defaultPath)
	}
	return contentKey, nil
}

// RemoveFile removes a file entry in a manifest.
func (a *API) RemoveFile(ctx context.Context, mhash string, path string, fname string, nameresolver bool) (string, error) {
	apiRmFileCount.Inc(1)

	uri, err := Parse("bzz:/" + mhash)
	if err != nil {
		apiRmFileFail.Inc(1)
		return "", err
	}
	mkey, err := a.ResolveURI(ctx, uri, EMPTY_CREDENTIALS)
	if err != nil {
		apiRmFileFail.Inc(1)
		return "", err
	}

	// trim the root dir we added
	if path[:1] == "/" {
		path = path[1:]
	}

	mw, err := a.NewManifestWriter(ctx, mkey, nil)
	if err != nil {
		apiRmFileFail.Inc(1)
		return "", err
	}

	err = mw.RemoveEntry(filepath.Join(path, fname))
	if err != nil {
		apiRmFileFail.Inc(1)
		return "", err
	}

	newMkey, err := mw.Store()
	if err != nil {
		apiRmFileFail.Inc(1)
		return "", err

	}

	return newMkey.String(), nil
}

// AppendFile removes old manifest, appends file entry to new manifest and adds it to Swarm.
func (a *API) AppendFile(ctx context.Context, mhash, path, fname string, existingSize int64, content []byte, oldAddr storage.Address, offset int64, addSize int64, nameresolver bool) (storage.Address, string, error) {
	apiAppendFileCount.Inc(1)

	buffSize := offset + addSize
	if buffSize < existingSize {
		buffSize = existingSize
	}

	buf := make([]byte, buffSize)

	oldReader, _ := a.Retrieve(ctx, oldAddr)
	io.ReadAtLeast(oldReader, buf, int(offset))

	newReader := bytes.NewReader(content)
	io.ReadAtLeast(newReader, buf[offset:], int(addSize))

	if buffSize < existingSize {
		io.ReadAtLeast(oldReader, buf[addSize:], int(buffSize))
	}

	combinedReader := bytes.NewReader(buf)
	totalSize := int64(len(buf))

	// TODO(jmozah): to append using pyramid chunker when it is ready
	//oldReader := a.Retrieve(oldKey)
	//newReader := bytes.NewReader(content)
	//combinedReader := io.MultiReader(oldReader, newReader)

	uri, err := Parse("bzz:/" + mhash)
	if err != nil {
		apiAppendFileFail.Inc(1)
		return nil, "", err
	}
	mkey, err := a.ResolveURI(ctx, uri, EMPTY_CREDENTIALS)
	if err != nil {
		apiAppendFileFail.Inc(1)
		return nil, "", err
	}

	// trim the root dir we added
	if path[:1] == "/" {
		path = path[1:]
	}

	mw, err := a.NewManifestWriter(ctx, mkey, nil)
	if err != nil {
		apiAppendFileFail.Inc(1)
		return nil, "", err
	}

	err = mw.RemoveEntry(filepath.Join(path, fname))
	if err != nil {
		apiAppendFileFail.Inc(1)
		return nil, "", err
	}

	entry := &ManifestEntry{
		Path:        filepath.Join(path, fname),
		ContentType: mime.TypeByExtension(filepath.Ext(fname)),
		Mode:        0700,
		Size:        totalSize,
		ModTime:     time.Now(),
	}

	fkey, err := mw.AddEntry(ctx, io.Reader(combinedReader), entry)
	if err != nil {
		apiAppendFileFail.Inc(1)
		return nil, "", err
	}

	newMkey, err := mw.Store()
	if err != nil {
		apiAppendFileFail.Inc(1)
		return nil, "", err

	}

	return fkey, newMkey.String(), nil
}

// BuildDirectoryTree used by swarmfs_unix
func (a *API) BuildDirectoryTree(ctx context.Context, mhash string, nameresolver bool) (addr storage.Address, manifestEntryMap map[string]*manifestTrieEntry, err error) {

	uri, err := Parse("bzz:/" + mhash)
	if err != nil {
		return nil, nil, err
	}
	addr, err = a.Resolve(ctx, uri.Addr)
	if err != nil {
		return nil, nil, err
	}

	quitC := make(chan bool)
	rootTrie, err := loadManifest(ctx, a.fileStore, addr, quitC, NOOPDecrypt)
	if err != nil {
		return nil, nil, fmt.Errorf("can't load manifest %v: %v", addr.String(), err)
	}

	manifestEntryMap = map[string]*manifestTrieEntry{}
	err = rootTrie.listWithPrefix(uri.Path, quitC, func(entry *manifestTrieEntry, suffix string) {
		manifestEntryMap[suffix] = entry
	})

	if err != nil {
		return nil, nil, fmt.Errorf("list with prefix failed %v: %v", addr.String(), err)
	}
	return addr, manifestEntryMap, nil
}

// FeedsLookup finds Swarm feeds updates at specific points in time, or the latest update
func (a *API) FeedsLookup(ctx context.Context, query *feed.Query) ([]byte, error) {
	_, err := a.feed.Lookup(ctx, query)
	if err != nil {
		return nil, err
	}
	var data []byte
	_, data, err = a.feed.GetContent(&query.Feed)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// FeedsNewRequest creates a Request object to update a specific feed
func (a *API) FeedsNewRequest(ctx context.Context, feed *feed.Feed) (*feed.Request, error) {
	return a.feed.NewRequest(ctx, feed)
}

// FeedsUpdate publishes a new update on the given feed
func (a *API) FeedsUpdate(ctx context.Context, request *feed.Request) (storage.Address, error) {
	return a.feed.Update(ctx, request)
}

// FeedsHashSize returned the size of the digest produced by Swarm feeds' hashing function
func (a *API) FeedsHashSize() int {
	return a.feed.HashSize
}

// ErrCannotLoadFeedManifest is returned when looking up a feeds manifest fails
var ErrCannotLoadFeedManifest = errors.New("Cannot load feed manifest")

// ErrNotAFeedManifest is returned when the address provided returned something other than a valid manifest
var ErrNotAFeedManifest = errors.New("Not a feed manifest")

// ResolveFeedManifest retrieves the Swarm feed manifest for the given address, and returns the referenced Feed.
func (a *API) ResolveFeedManifest(ctx context.Context, addr storage.Address) (*feed.Feed, error) {
	trie, err := loadManifest(ctx, a.fileStore, addr, nil, NOOPDecrypt)
	if err != nil {
		return nil, ErrCannotLoadFeedManifest
	}

	entry, _ := trie.getEntry("")
	if entry.ContentType != FeedContentType {
		return nil, ErrNotAFeedManifest
	}

	return entry.Feed, nil
}

// ErrCannotResolveFeedURI is returned when the ENS resolver is not able to translate a name to a Swarm feed
var ErrCannotResolveFeedURI = errors.New("Cannot resolve Feed URI")

// ErrCannotResolveFeed is returned when values provided are not enough or invalid to recreate a
// feed out of them.
var ErrCannotResolveFeed = errors.New("Cannot resolve Feed")

// ResolveFeed attempts to extract feed information out of the manifest, if provided
// If not, it attempts to extract the feed out of a set of key-value pairs
func (a *API) ResolveFeed(ctx context.Context, uri *URI, values feed.Values) (*feed.Feed, error) {
	var fd *feed.Feed
	var err error
	if uri.Addr != "" {
		// resolve the content key.
		manifestAddr := uri.Address()
		if manifestAddr == nil {
			manifestAddr, err = a.Resolve(ctx, uri.Addr)
			if err != nil {
				return nil, ErrCannotResolveFeedURI
			}
		}

		// get the Swarm feed from the manifest
		fd, err = a.ResolveFeedManifest(ctx, manifestAddr)
		if err != nil {
			return nil, err
		}
		log.Debug("handle.get.feed: resolved", "manifestkey", manifestAddr, "feed", fd.Hex())
	} else {
		var f feed.Feed
		if err := f.FromValues(values); err != nil {
			return nil, ErrCannotResolveFeed

		}
		fd = &f
	}
	return fd, nil
}

// MimeOctetStream default value of http Content-Type header
const MimeOctetStream = "application/octet-stream"

// DetectContentType by file file extension, or fallback to content sniff
func DetectContentType(fileName string, f io.ReadSeeker) (string, error) {
	ctype := mime.TypeByExtension(filepath.Ext(fileName))
	if ctype != "" {
		return ctype, nil
	}

	// save/rollback to get content probe from begin of file
	currentPosition, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return MimeOctetStream, fmt.Errorf("seeker can't seek, %s", err)
	}

	// read a chunk to decide between utf-8 text and binary
	var buf [512]byte
	n, _ := f.Read(buf[:])
	ctype = http.DetectContentType(buf[:n])

	_, err = f.Seek(currentPosition, io.SeekStart) // rewind to output whole file
	if err != nil {
		return MimeOctetStream, fmt.Errorf("seeker can't seek, %s", err)
	}

	return ctype, nil
}
