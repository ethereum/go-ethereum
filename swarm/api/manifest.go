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
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	manifestType = "application/bzz-manifest+json"
)

type manifestTrie struct {
	dpa     *storage.DPA
	entries [257]*manifestTrieEntry // indexed by first character of path, entries[256] is the empty path entry
	hash    storage.Key             // if hash != nil, it is stored
}

type manifestJSON struct {
	Entries []*manifestTrieEntry `json:"entries"`
}

type manifestTrieEntry struct {
	Path        string `json:"path"`
	Hash        string `json:"hash"` // for manifest content type, empty until subtrie is evaluated
	ContentType string `json:"contentType"`
	Status      int    `json:"status"`
	subtrie     *manifestTrie
}

func loadManifest(dpa *storage.DPA, hash storage.Key, quitC chan bool) (trie *manifestTrie, err error) { // non-recursive, subtrees are downloaded on-demand

	glog.V(logger.Detail).Infof("manifest lookup key: '%v'.", hash.Log())
	// retrieve manifest via DPA
	manifestReader := dpa.Retrieve(hash)
	return readManifest(manifestReader, hash, dpa, quitC)
}

func readManifest(manifestReader storage.LazySectionReader, hash storage.Key, dpa *storage.DPA, quitC chan bool) (trie *manifestTrie, err error) { // non-recursive, subtrees are downloaded on-demand

	// TODO check size for oversized manifests
	size, err := manifestReader.Size(quitC)
	if err != nil { // size == 0
		// can't determine size means we don't have the root chunk
		err = fmt.Errorf("Manifest not Found")
		return
	}
	manifestData := make([]byte, size)
	read, err := manifestReader.Read(manifestData)
	if int64(read) < size {
		glog.V(logger.Detail).Infof("Manifest %v not found.", hash.Log())
		if err == nil {
			err = fmt.Errorf("Manifest retrieval cut short: read %v, expect %v", read, size)
		}
		return
	}

	glog.V(logger.Detail).Infof("Manifest %v retrieved", hash.Log())
	man := manifestJSON{}
	err = json.Unmarshal(manifestData, &man)
	if err != nil {
		err = fmt.Errorf("Manifest %v is malformed: %v", hash.Log(), err)
		glog.V(logger.Detail).Infof("%v", err)
		return
	}

	glog.V(logger.Detail).Infof("Manifest %v has %d entries.", hash.Log(), len(man.Entries))

	trie = &manifestTrie{
		dpa: dpa,
	}
	for _, entry := range man.Entries {
		trie.addEntry(entry, quitC)
	}
	return
}

func (self *manifestTrie) addEntry(entry *manifestTrieEntry, quitC chan bool) {
	self.hash = nil // trie modified, hash needs to be re-calculated on demand

	if len(entry.Path) == 0 {
		self.entries[256] = entry
		return
	}

	b := byte(entry.Path[0])
	if (self.entries[b] == nil) || (self.entries[b].Path == entry.Path) {
		self.entries[b] = entry
		return
	}

	oldentry := self.entries[b]
	cpl := 0
	for (len(entry.Path) > cpl) && (len(oldentry.Path) > cpl) && (entry.Path[cpl] == oldentry.Path[cpl]) {
		cpl++
	}

	if (oldentry.ContentType == manifestType) && (cpl == len(oldentry.Path)) {
		if self.loadSubTrie(oldentry, quitC) != nil {
			return
		}
		entry.Path = entry.Path[cpl:]
		oldentry.subtrie.addEntry(entry, quitC)
		oldentry.Hash = ""
		return
	}

	commonPrefix := entry.Path[:cpl]

	subtrie := &manifestTrie{
		dpa: self.dpa,
	}
	entry.Path = entry.Path[cpl:]
	oldentry.Path = oldentry.Path[cpl:]
	subtrie.addEntry(entry, quitC)
	subtrie.addEntry(oldentry, quitC)

	self.entries[b] = &manifestTrieEntry{
		Path:        commonPrefix,
		Hash:        "",
		ContentType: manifestType,
		subtrie:     subtrie,
	}
}

func (self *manifestTrie) getCountLast() (cnt int, entry *manifestTrieEntry) {
	for _, e := range self.entries {
		if e != nil {
			cnt++
			entry = e
		}
	}
	return
}

func (self *manifestTrie) deleteEntry(path string, quitC chan bool) {
	self.hash = nil // trie modified, hash needs to be re-calculated on demand

	if len(path) == 0 {
		self.entries[256] = nil
		return
	}

	b := byte(path[0])
	entry := self.entries[b]
	if entry == nil {
		return
	}
	if entry.Path == path {
		self.entries[b] = nil
		return
	}

	epl := len(entry.Path)
	if (entry.ContentType == manifestType) && (len(path) >= epl) && (path[:epl] == entry.Path) {
		if self.loadSubTrie(entry, quitC) != nil {
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
			self.entries[b] = lastentry
		}
	}
}

func (self *manifestTrie) recalcAndStore() error {
	if self.hash != nil {
		return nil
	}

	var buffer bytes.Buffer
	buffer.WriteString(`{"entries":[`)

	list := &manifestJSON{}
	for _, entry := range self.entries {
		if entry != nil {
			if entry.Hash == "" { // TODO: paralellize
				err := entry.subtrie.recalcAndStore()
				if err != nil {
					return err
				}
				entry.Hash = entry.subtrie.hash.String()
			}
			list.Entries = append(list.Entries, entry)
		}
	}

	manifest, err := json.Marshal(list)
	if err != nil {
		return err
	}

	sr := bytes.NewReader(manifest)
	wg := &sync.WaitGroup{}
	key, err2 := self.dpa.Store(sr, int64(len(manifest)), wg, nil)
	wg.Wait()
	self.hash = key
	return err2
}

func (self *manifestTrie) loadSubTrie(entry *manifestTrieEntry, quitC chan bool) (err error) {
	if entry.subtrie == nil {
		hash := common.Hex2Bytes(entry.Hash)
		entry.subtrie, err = loadManifest(self.dpa, hash, quitC)
		entry.Hash = "" // might not match, should be recalculated
	}
	return
}

func (self *manifestTrie) listWithPrefixInt(prefix, rp string, quitC chan bool, cb func(entry *manifestTrieEntry, suffix string)) error {
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
		entry := self.entries[i]
		if entry != nil {
			epl := len(entry.Path)
			if entry.ContentType == manifestType {
				l := plen
				if epl < l {
					l = epl
				}
				if prefix[:l] == entry.Path[:l] {
					err := self.loadSubTrie(entry, quitC)
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

func (self *manifestTrie) listWithPrefix(prefix string, quitC chan bool, cb func(entry *manifestTrieEntry, suffix string)) (err error) {
	return self.listWithPrefixInt(prefix, "", quitC, cb)
}

func (self *manifestTrie) findPrefixOf(path string, quitC chan bool) (entry *manifestTrieEntry, pos int) {

	glog.V(logger.Detail).Infof("findPrefixOf(%s)", path)

	if len(path) == 0 {
		return self.entries[256], 0
	}

	b := byte(path[0])
	entry = self.entries[b]
	if entry == nil {
		return self.entries[256], 0
	}
	epl := len(entry.Path)
	glog.V(logger.Detail).Infof("path = %v  entry.Path = %v  epl = %v", path, entry.Path, epl)
	if (len(path) >= epl) && (path[:epl] == entry.Path) {
		glog.V(logger.Detail).Infof("entry.ContentType = %v", entry.ContentType)
		if entry.ContentType == manifestType {
			err := self.loadSubTrie(entry, quitC)
			if err != nil {
				return nil, 0
			}
			entry, pos = entry.subtrie.findPrefixOf(path[epl:], quitC)
			if entry != nil {
				pos += epl
			}
		} else {
			pos = epl
		}
	}
	return
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

func (self *manifestTrie) getEntry(spath string) (entry *manifestTrieEntry, fullpath string) {
	path := RegularSlashes(spath)
	var pos int
	quitC := make(chan bool)
	entry, pos = self.findPrefixOf(path, quitC)
	return entry, path[:pos]
}
