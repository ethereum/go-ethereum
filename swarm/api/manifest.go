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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	ManifestType        = "application/bzz-manifest+json"
	ResourceContentType = "application/bzz-resource"

	manifestSizeLimit = 5 * 1024 * 1024
)

// Manifest represents a swarm manifest
type Manifest struct {
	Entries []ManifestEntry `json:"entries,omitempty"`
}

// ManifestEntry represents an entry in a swarm manifest
type ManifestEntry struct {
	Hash        string    `json:"hash,omitempty"`
	Path        string    `json:"path,omitempty"`
	ContentType string    `json:"contentType,omitempty"`
	Mode        int64     `json:"mode,omitempty"`
	Size        int64     `json:"size,omitempty"`
	ModTime     time.Time `json:"mod_time,omitempty"`
	Status      int       `json:"status,omitempty"`
}

// ManifestList represents the result of listing files in a manifest
type ManifestList struct {
	CommonPrefixes []string         `json:"common_prefixes,omitempty"`
	Entries        []*ManifestEntry `json:"entries,omitempty"`
}

// NewManifest creates and stores a new, empty manifest
func (a *API) NewManifest(ctx context.Context, toEncrypt bool) (storage.Address, error) {
	var manifest Manifest
	data, err := json.Marshal(&manifest)
	if err != nil {
		return nil, err
	}
	key, wait, err := a.Store(ctx, bytes.NewReader(data), int64(len(data)), toEncrypt)
	wait(ctx)
	return key, err
}

// Manifest hack for supporting Mutable Resource Updates from the bzz: scheme
// see swarm/api/api.go:API.Get() for more information
func (a *API) NewResourceManifest(ctx context.Context, resourceAddr string) (storage.Address, error) {
	var manifest Manifest
	entry := ManifestEntry{
		Hash:        resourceAddr,
		ContentType: ResourceContentType,
	}
	manifest.Entries = append(manifest.Entries, entry)
	data, err := json.Marshal(&manifest)
	if err != nil {
		return nil, err
	}
	key, _, err := a.Store(ctx, bytes.NewReader(data), int64(len(data)), false)
	return key, err
}

// ManifestWriter is used to add and remove entries from an underlying manifest
type ManifestWriter struct {
	api   *API
	trie  *manifestTrie
	quitC chan bool
}

func (a *API) NewManifestWriter(ctx context.Context, addr storage.Address, quitC chan bool) (*ManifestWriter, error) {
	trie, err := loadManifest(ctx, a.fileStore, addr, quitC)
	if err != nil {
		return nil, fmt.Errorf("error loading manifest %s: %s", addr, err)
	}
	return &ManifestWriter{a, trie, quitC}, nil
}

// AddEntry stores the given data and adds the resulting key to the manifest
func (m *ManifestWriter) AddEntry(ctx context.Context, data io.Reader, e *ManifestEntry) (storage.Address, error) {
	key, _, err := m.api.Store(ctx, data, e.Size, m.trie.encrypted)
	if err != nil {
		return nil, err
	}
	entry := newManifestTrieEntry(e, nil)
	entry.Hash = key.Hex()
	m.trie.addEntry(entry, m.quitC)
	return key, nil
}

// RemoveEntry removes the given path from the manifest
func (m *ManifestWriter) RemoveEntry(path string) error {
	m.trie.deleteEntry(path, m.quitC)
	return nil
}

// Store stores the manifest, returning the resulting storage key
func (m *ManifestWriter) Store() (storage.Address, error) {
	return m.trie.ref, m.trie.recalcAndStore()
}

// ManifestWalker is used to recursively walk the entries in the manifest and
// all of its submanifests
type ManifestWalker struct {
	api   *API
	trie  *manifestTrie
	quitC chan bool
}

func (a *API) NewManifestWalker(ctx context.Context, addr storage.Address, quitC chan bool) (*ManifestWalker, error) {
	trie, err := loadManifest(ctx, a.fileStore, addr, quitC)
	if err != nil {
		return nil, fmt.Errorf("error loading manifest %s: %s", addr, err)
	}
	return &ManifestWalker{a, trie, quitC}, nil
}

// ErrSkipManifest is used as a return value from WalkFn to indicate that the
// manifest should be skipped
var ErrSkipManifest = errors.New("skip this manifest")

// WalkFn is the type of function called for each entry visited by a recursive
// manifest walk
type WalkFn func(entry *ManifestEntry) error

// Walk recursively walks the manifest calling walkFn for each entry in the
// manifest, including submanifests
func (m *ManifestWalker) Walk(walkFn WalkFn) error {
	return m.walk(m.trie, "", walkFn)
}

func (m *ManifestWalker) walk(trie *manifestTrie, prefix string, walkFn WalkFn) error {
	for _, entry := range trie.entries {
		if entry == nil {
			continue
		}
		entry.Path = prefix + entry.Path
		err := walkFn(&entry.ManifestEntry)
		if err != nil {
			if entry.ContentType == ManifestType && err == ErrSkipManifest {
				continue
			}
			return err
		}
		if entry.ContentType != ManifestType {
			continue
		}
		if err := trie.loadSubTrie(entry, nil); err != nil {
			return err
		}
		if err := m.walk(entry.subtrie, entry.Path, walkFn); err != nil {
			return err
		}
	}
	return nil
}

type manifestTrie struct {
	fileStore *storage.FileStore
	entries   [257]*manifestTrieEntry // indexed by first character of basePath, entries[256] is the empty basePath entry
	ref       storage.Address         // if ref != nil, it is stored
	encrypted bool
}

func newManifestTrieEntry(entry *ManifestEntry, subtrie *manifestTrie) *manifestTrieEntry {
	return &manifestTrieEntry{
		ManifestEntry: *entry,
		subtrie:       subtrie,
	}
}

type manifestTrieEntry struct {
	ManifestEntry

	subtrie *manifestTrie
}

func loadManifest(ctx context.Context, fileStore *storage.FileStore, hash storage.Address, quitC chan bool) (trie *manifestTrie, err error) { // non-recursive, subtrees are downloaded on-demand
	log.Trace("manifest lookup", "key", hash)
	// retrieve manifest via FileStore
	manifestReader, isEncrypted := fileStore.Retrieve(ctx, hash)
	log.Trace("reader retrieved", "key", hash)
	return readManifest(manifestReader, hash, fileStore, isEncrypted, quitC)
}

func readManifest(manifestReader storage.LazySectionReader, hash storage.Address, fileStore *storage.FileStore, isEncrypted bool, quitC chan bool) (trie *manifestTrie, err error) { // non-recursive, subtrees are downloaded on-demand

	// TODO check size for oversized manifests
	size, err := manifestReader.Size(quitC)
	if err != nil { // size == 0
		// can't determine size means we don't have the root chunk
		log.Trace("manifest not found", "key", hash)
		err = fmt.Errorf("Manifest not Found")
		return
	}
	if size > manifestSizeLimit {
		log.Warn("manifest exceeds size limit", "key", hash, "size", size, "limit", manifestSizeLimit)
		err = fmt.Errorf("Manifest size of %v bytes exceeds the %v byte limit", size, manifestSizeLimit)
		return
	}
	manifestData := make([]byte, size)
	read, err := manifestReader.Read(manifestData)
	if int64(read) < size {
		log.Trace("manifest not found", "key", hash)
		if err == nil {
			err = fmt.Errorf("Manifest retrieval cut short: read %v, expect %v", read, size)
		}
		return
	}

	log.Debug("manifest retrieved", "key", hash)
	var man struct {
		Entries []*manifestTrieEntry `json:"entries"`
	}
	err = json.Unmarshal(manifestData, &man)
	if err != nil {
		err = fmt.Errorf("Manifest %v is malformed: %v", hash.Log(), err)
		log.Trace("malformed manifest", "key", hash)
		return
	}

	log.Trace("manifest entries", "key", hash, "len", len(man.Entries))

	trie = &manifestTrie{
		fileStore: fileStore,
		encrypted: isEncrypted,
	}
	for _, entry := range man.Entries {
		trie.addEntry(entry, quitC)
	}
	return
}

func (mt *manifestTrie) addEntry(entry *manifestTrieEntry, quitC chan bool) {
	mt.ref = nil // trie modified, hash needs to be re-calculated on demand

	if len(entry.Path) == 0 {
		mt.entries[256] = entry
		return
	}

	b := entry.Path[0]
	oldentry := mt.entries[b]
	if (oldentry == nil) || (oldentry.Path == entry.Path && oldentry.ContentType != ManifestType) {
		mt.entries[b] = entry
		return
	}

	cpl := 0
	for (len(entry.Path) > cpl) && (len(oldentry.Path) > cpl) && (entry.Path[cpl] == oldentry.Path[cpl]) {
		cpl++
	}

	if (oldentry.ContentType == ManifestType) && (cpl == len(oldentry.Path)) {
		if mt.loadSubTrie(oldentry, quitC) != nil {
			return
		}
		entry.Path = entry.Path[cpl:]
		oldentry.subtrie.addEntry(entry, quitC)
		oldentry.Hash = ""
		return
	}

	commonPrefix := entry.Path[:cpl]

	subtrie := &manifestTrie{
		fileStore: mt.fileStore,
		encrypted: mt.encrypted,
	}
	entry.Path = entry.Path[cpl:]
	oldentry.Path = oldentry.Path[cpl:]
	subtrie.addEntry(entry, quitC)
	subtrie.addEntry(oldentry, quitC)

	mt.entries[b] = newManifestTrieEntry(&ManifestEntry{
		Path:        commonPrefix,
		ContentType: ManifestType,
	}, subtrie)
}

func (mt *manifestTrie) getCountLast() (cnt int, entry *manifestTrieEntry) {
	for _, e := range mt.entries {
		if e != nil {
			cnt++
			entry = e
		}
	}
	return
}

func (mt *manifestTrie) deleteEntry(path string, quitC chan bool) {
	mt.ref = nil // trie modified, hash needs to be re-calculated on demand

	if len(path) == 0 {
		mt.entries[256] = nil
		return
	}

	b := path[0]
	entry := mt.entries[b]
	if entry == nil {
		return
	}
	if entry.Path == path {
		mt.entries[b] = nil
		return
	}

	epl := len(entry.Path)
	if (entry.ContentType == ManifestType) && (len(path) >= epl) && (path[:epl] == entry.Path) {
		if mt.loadSubTrie(entry, quitC) != nil {
			return
		}
		entry.subtrie.deleteEntry(path[epl:], quitC)
		entry.Hash = ""
		// remove subtree if it has less than 2 elements
		cnt, lastentry := entry.subtrie.getCountLast()
		if cnt < 2 {
			if lastentry != nil {
				lastentry.Path = entry.Path + lastentry.Path
			}
			mt.entries[b] = lastentry
		}
	}
}

func (mt *manifestTrie) recalcAndStore() error {
	if mt.ref != nil {
		return nil
	}

	var buffer bytes.Buffer
	buffer.WriteString(`{"entries":[`)

	list := &Manifest{}
	for _, entry := range mt.entries {
		if entry != nil {
			if entry.Hash == "" { // TODO: paralellize
				err := entry.subtrie.recalcAndStore()
				if err != nil {
					return err
				}
				entry.Hash = entry.subtrie.ref.Hex()
			}
			list.Entries = append(list.Entries, entry.ManifestEntry)
		}

	}

	manifest, err := json.Marshal(list)
	if err != nil {
		return err
	}

	sr := bytes.NewReader(manifest)
	ctx := context.TODO()
	key, wait, err2 := mt.fileStore.Store(ctx, sr, int64(len(manifest)), mt.encrypted)
	if err2 != nil {
		return err2
	}
	err2 = wait(ctx)
	mt.ref = key
	return err2
}

func (mt *manifestTrie) loadSubTrie(entry *manifestTrieEntry, quitC chan bool) (err error) {
	if entry.subtrie == nil {
		hash := common.Hex2Bytes(entry.Hash)
		entry.subtrie, err = loadManifest(context.TODO(), mt.fileStore, hash, quitC)
		entry.Hash = "" // might not match, should be recalculated
	}
	return
}

func (mt *manifestTrie) listWithPrefixInt(prefix, rp string, quitC chan bool, cb func(entry *manifestTrieEntry, suffix string)) error {
	plen := len(prefix)
	var start, stop int
	if plen == 0 {
		start = 0
		stop = 256
	} else {
		start = int(prefix[0])
		stop = start
	}

	for i := start; i <= stop; i++ {
		select {
		case <-quitC:
			return fmt.Errorf("aborted")
		default:
		}
		entry := mt.entries[i]
		if entry != nil {
			epl := len(entry.Path)
			if entry.ContentType == ManifestType {
				l := plen
				if epl < l {
					l = epl
				}
				if prefix[:l] == entry.Path[:l] {
					err := mt.loadSubTrie(entry, quitC)
					if err != nil {
						return err
					}
					err = entry.subtrie.listWithPrefixInt(prefix[l:], rp+entry.Path[l:], quitC, cb)
					if err != nil {
						return err
					}
				}
			} else {
				if (epl >= plen) && (prefix == entry.Path[:plen]) {
					cb(entry, rp+entry.Path[plen:])
				}
			}
		}
	}
	return nil
}

func (mt *manifestTrie) listWithPrefix(prefix string, quitC chan bool, cb func(entry *manifestTrieEntry, suffix string)) (err error) {
	return mt.listWithPrefixInt(prefix, "", quitC, cb)
}

func (mt *manifestTrie) findPrefixOf(path string, quitC chan bool) (entry *manifestTrieEntry, pos int) {
	log.Trace(fmt.Sprintf("findPrefixOf(%s)", path))

	if len(path) == 0 {
		return mt.entries[256], 0
	}

	//see if first char is in manifest entries
	b := path[0]
	entry = mt.entries[b]
	if entry == nil {
		return mt.entries[256], 0
	}

	epl := len(entry.Path)
	log.Trace(fmt.Sprintf("path = %v  entry.Path = %v  epl = %v", path, entry.Path, epl))
	if len(path) <= epl {
		if entry.Path[:len(path)] == path {
			if entry.ContentType == ManifestType {
				err := mt.loadSubTrie(entry, quitC)
				if err == nil && entry.subtrie != nil {
					subentries := entry.subtrie.entries
					for i := 0; i < len(subentries); i++ {
						sub := subentries[i]
						if sub != nil && sub.Path == "" {
							return sub, len(path)
						}
					}
				}
				entry.Status = http.StatusMultipleChoices
			}
			pos = len(path)
			return
		}
		return nil, 0
	}
	if path[:epl] == entry.Path {
		log.Trace(fmt.Sprintf("entry.ContentType = %v", entry.ContentType))
		//the subentry is a manifest, load subtrie
		if entry.ContentType == ManifestType && (strings.Contains(entry.Path, path) || strings.Contains(path, entry.Path)) {
			err := mt.loadSubTrie(entry, quitC)
			if err != nil {
				return nil, 0
			}
			sub, pos := entry.subtrie.findPrefixOf(path[epl:], quitC)
			if sub != nil {
				entry = sub
				pos += epl
				return sub, pos
			} else if path == entry.Path {
				entry.Status = http.StatusMultipleChoices
			}

		} else {
			//entry is not a manifest, return it
			if path != entry.Path {
				return nil, 0
			}
			pos = epl
		}
	}
	return nil, 0
}

// file system manifest always contains regularized paths
// no leading or trailing slashes, only single slashes inside
func RegularSlashes(path string) (res string) {
	for i := 0; i < len(path); i++ {
		if (path[i] != '/') || ((i > 0) && (path[i-1] != '/')) {
			res = res + path[i:i+1]
		}
	}
	if (len(res) > 0) && (res[len(res)-1] == '/') {
		res = res[:len(res)-1]
	}
	return
}

func (mt *manifestTrie) getEntry(spath string) (entry *manifestTrieEntry, fullpath string) {
	path := RegularSlashes(spath)
	var pos int
	quitC := make(chan bool)
	entry, pos = mt.findPrefixOf(path, quitC)
	return entry, path[:pos]
}
