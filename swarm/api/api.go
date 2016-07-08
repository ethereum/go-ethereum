package api

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var (
	hashMatcher      = regexp.MustCompile("^[0-9A-Fa-f]{64}")
	slashes          = regexp.MustCompile("/+")
	domainAndVersion = regexp.MustCompile("[@:;,]+")
)

type Resolver interface {
	Resolve(string) (storage.Key, error)
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
	return self.dpa.Store(data, size, wg)
}

type ErrResolve error

// DNS Resolver
func (self *Api) Resolve(hostPort string, nameresolver bool) (contentHash storage.Key, err error) {
	if hashMatcher.MatchString(hostPort) || self.dns == nil {
		glog.V(logger.Detail).Infof("[BZZ] host is a contentHash: '%v'", hostPort)
		return storage.Key(common.Hex2Bytes(hostPort)), nil
	}
	if !nameresolver {
		err = fmt.Errorf("'%s' is not a content hash value.", hostPort)
		return
	}
	contentHash, err = self.dns.Resolve(hostPort)
	if err != nil {
		err = ErrResolve(err)
		glog.V(logger.Warn).Infof("[BZZ] DNS error : %v", err)
	}
	glog.V(logger.Detail).Infof("[BZZ] host lookup: %v -> %v", err)
	return
}

func parse(uri string) (hostPort, path string) {
	parts := slashes.Split(uri, 3)
	var i int
	if len(parts) == 0 {
		return
	}
	// beginning with slash is now optional
	for len(parts[i]) == 0 {
		i++
	}
	hostPort = parts[i]
	for i < len(parts)-1 {
		i++
		if len(path) > 0 {
			path = path + "/" + parts[i]
		} else {
			path = parts[i]
		}
	}
	glog.V(logger.Debug).Infof("[BZZ] Swarm: host: '%s', path '%s' requested.", hostPort, path)
	return
}

func (self *Api) parseAndResolve(uri string, nameresolver bool) (contentHash storage.Key, hostPort, path string, err error) {
	hostPort, path = parse(uri)
	//resolving host and port
	contentHash, err = self.Resolve(hostPort, nameresolver)
	glog.V(logger.Debug).Infof("[BZZ] Resolved '%s' to contentHash: '%s', path: '%s'", uri, contentHash, path)
	return
}

// Put provides singleton manifest creation on top of dpa store
func (self *Api) Put(content, contentType string) (string, error) {
	r := strings.NewReader(content)
	wg := &sync.WaitGroup{}
	key, err := self.dpa.Store(r, int64(len(content)), wg)
	if err != nil {
		return "", err
	}
	manifest := fmt.Sprintf(`{"entries":[{"hash":"%v","contentType":"%s"}]}`, key, contentType)
	r = strings.NewReader(manifest)
	key, err = self.dpa.Store(r, int64(len(manifest)), wg)
	if err != nil {
		return "", err
	}
	wg.Wait()
	return key.String(), nil
}

// Get uses iterative manifest retrieval and prefix matching
// to resolve path to content using dpa retrieve
// it returns a section reader, mimeType, status and an error
func (self *Api) Get(uri string, nameresolver bool) (reader storage.LazySectionReader, mimeType string, status int, err error) {

	key, _, path, err := self.parseAndResolve(uri, nameresolver)
	quitC := make(chan bool)
	trie, err := loadManifest(self.dpa, key, quitC)
	if err != nil {
		glog.V(logger.Warn).Infof("[BZZ] Swarm: loadManifestTrie error: %v", err)
		return
	}

	glog.V(logger.Detail).Infof("[BZZ] Swarm: getEntry(%s)", path)
	entry, _ := trie.getEntry(path)
	if entry != nil {
		key = common.Hex2Bytes(entry.Hash)
		status = entry.Status
		mimeType = entry.ContentType
		glog.V(logger.Detail).Infof("[BZZ] Swarm: content lookup key: '%v' (%v)", key, mimeType)
		reader = self.dpa.Retrieve(key)
	} else {
		err = fmt.Errorf("manifest entry for '%s' not found", path)
		glog.V(logger.Warn).Infof("[BZZ] Swarm: %v", err)
	}
	return
}

func (self *Api) Modify(uri, contentHash, contentType string, nameresolver bool) (newRootHash string, err error) {
	root, _, path, err := self.parseAndResolve(uri, nameresolver)
	quitC := make(chan bool)
	trie, err := loadManifest(self.dpa, root, quitC)
	if err != nil {
		return
	}

	if contentHash != "" {
		entry := &manifestTrieEntry{
			Path:        path,
			Hash:        contentHash,
			ContentType: contentType,
		}
		trie.addEntry(entry, quitC)
	} else {
		trie.deleteEntry(path, quitC)
	}

	err = trie.recalcAndStore()
	if err != nil {
		return
	}
	return trie.hash.String(), nil
}
