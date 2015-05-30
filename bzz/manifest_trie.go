package bzz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

const (
	manifestType = "application/bzz-manifest+json"
)

type manifestTrie struct {
	dpa     *DPA
	entries [257]*manifestTrieEntry // indexed by first character of path, entries[256] is the empty path entry
	hash    Key                     // if hash != nil, it is stored
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

func loadManifestTrie(dpa *DPA, hash Key) (trie *manifestTrie, err error) { // non-recursive, subtrees are downloaded on-demand

	dpaLogger.Debugf("Swarm: manifest lookup key: '%064x'.", hash)
	// retrieve manifest via DPA
	manifestReader := dpa.Retrieve(hash)
	// TODO check size for oversized manifests
	manifestData := make([]byte, manifestReader.Size())
	var size int
	size, err = manifestReader.Read(manifestData)
	if int64(size) < manifestReader.Size() {
		dpaLogger.Debugf("Swarm: Manifest %064x not found.", hash)
		if err == nil {
			err = fmt.Errorf("Manifest retrieval cut short: %v &lt; %v", size, manifestReader.Size())
		}
		return
	}

	dpaLogger.Debugf("Swarm: Manifest %064x retrieved", hash)
	man := manifestJSON{}
	err = json.Unmarshal(manifestData, &man)
	if err != nil {
		err = fmt.Errorf("Manifest %064x is malformed: %v", hash, err)
		dpaLogger.Debugf("Swarm: %v", err)
		return
	}

	dpaLogger.Debugf("Swarm: Manifest %064x has %d entries.", hash, len(man.Entries))

	trie = &manifestTrie{
		dpa: dpa,
	}
	for _, entry := range man.Entries {
		trie.addEntry(entry)
	}
	return
}

func (self *manifestTrie) addEntry(entry *manifestTrieEntry) {
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
		if self.loadSubTrie(oldentry) != nil {
			return
		}
		entry.Path = entry.Path[cpl:]
		oldentry.subtrie.addEntry(entry)
		oldentry.Hash = ""
		return
	}

	commonPrefix := entry.Path[:cpl]

	subtrie := &manifestTrie{
		dpa: self.dpa,
	}
	entry.Path = entry.Path[cpl:]
	oldentry.Path = oldentry.Path[cpl:]
	subtrie.addEntry(entry)
	subtrie.addEntry(oldentry)

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

func (self *manifestTrie) deleteEntry(path string) {
	self.hash = nil // trie modified, hash needs to be re-calculated on demand

	if len(path) == 0 {
		self.entries[256] = nil
		return
	}

	b := byte(path[0])
	entry := self.entries[b]
	if (entry != nil) && (entry.Path == path) {
		self.entries[b] = nil
		return
	}

	epl := len(entry.Path)
	if (entry.ContentType == manifestType) && (len(path) >= epl) && (path[:epl] == entry.Path) {
		if self.loadSubTrie(entry) != nil {
			return
		}
		entry.subtrie.deleteEntry(path[epl:])
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
				entry.Hash = fmt.Sprintf("%064x", entry.subtrie.hash)
			}
			list.Entries = append(list.Entries, entry)
		}
	}

	manifest, err := json.Marshal(list)
	if err != nil {
		return err
	}

	sr := io.NewSectionReader(bytes.NewReader(manifest), 0, int64(len(manifest)))
	wg := &sync.WaitGroup{}
	key, err2 := self.dpa.Store(sr, wg)
	wg.Wait()
	self.hash = key
	return err2
}

func (self *manifestTrie) loadSubTrie(entry *manifestTrieEntry) (err error) {
	if entry.subtrie == nil {
		hash := common.Hex2Bytes(entry.Hash)
		entry.subtrie, err = loadManifestTrie(self.dpa, hash)
		entry.Hash = "" // might not match, should be recalculated
	}
	return
}

func (self *manifestTrie) findPrefixOf(path string) (entry *manifestTrieEntry, pos int) {

	dpaLogger.Debugf("findPrefixOf(%s)", path)

	if len(path) == 0 {
		return self.entries[256], 0
	}

	b := byte(path[0])
	entry = self.entries[b]
	if entry == nil {
		return nil, 0
	}
	epl := len(entry.Path)
	dpaLogger.Debugf("path = %v  entry.Path = %v  epl = %v", path, entry.Path, epl)
	if (len(path) >= epl) && (path[:epl] == entry.Path) {
		dpaLogger.Debugf("entry.ContentType = %v", entry.ContentType)
		if entry.ContentType == manifestType {
			if self.loadSubTrie(entry) != nil {
				return nil, 0
			}
			entry, pos = entry.subtrie.findPrefixOf(path[epl:])
			if entry != nil {
				pos += epl
			}
		} else {
			pos = epl
		}
	} else {
		entry = nil
	}
	return
}

func (self *manifestTrie) getEntryNLS(path string) (entry *manifestTrieEntry, pos int) {
	entry, pos = self.findPrefixOf(path)
	if entry != nil {
		for (pos < len(path)) && (path[pos] == '/') {
			pos++
		}
		if (pos < len(path)) && (pos > 0) && (path[pos-1] != '/') {
			return nil, 0
		}
	}
	return
}

func (self *manifestTrie) getEntry(path string) (entry *manifestTrieEntry, pos int) {
	var slash string
	for {
		entry, pos = self.getEntryNLS(slash + path)
		dpaLogger.Debugf("getEntryNLS(%s) pos=%v", slash+path, pos)
		if pos < len(slash) {
			dpaLogger.Debugf("Path '%s' on manifest not found.", path)
			return nil, 0
		}
		if entry != nil {
			pos -= len(slash)
			dpaLogger.Debugf("Swarm: '%s' matches '%s'.", path, entry.Path)
			return
		}
		slash = slash + "/"
	}
}
