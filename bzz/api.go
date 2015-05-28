package bzz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/resolver"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

var (
	hashMatcher = regexp.MustCompile("^[0-9A-Fa-f]{64}")
	slashes     = regexp.MustCompile("/+")
)

/*
Api implements webserver/file system related content storage and retrieval
on top of the dpa
*/
type Api struct {
	dpa      *DPA
	netStore *netStore
	port     string
	Resolver *resolver.Resolver
}

func NewApi(datadir, port string) (api *Api, err error) {

	api = &Api{port: port}

	api.netStore, err = newNetStore(filepath.Join(datadir, "bzz"), filepath.Join(datadir, "bzzpeers.json"))
	if err != nil {
		return
	}

	chunker := &TreeChunker{}
	chunker.Init()
	api.dpa = &DPA{
		Chunker:    chunker,
		ChunkStore: api.netStore,
	}
	return
}

func (self *Api) Bzz() (p2p.Protocol, error) {
	return BzzProtocol(self.netStore)
}

func (self *Api) Start(node *discover.Node, connectPeer func(string) error) {
	self.dpa.Start()
	self.netStore.Start(node, connectPeer)
	dpaLogger.Infof("Swarm started.")
	go startHttpServer(self, self.port)
}

func (self *Api) Stop() {
	self.dpa.Stop()
	self.netStore.Stop()
}

// Get uses iterative manifest retrieval and prefix matching
// to resolve path to content using dpa retrieve
func (self *Api) Get(bzzpath string) (string, error) {
	return "", nil
}

// Put provides singleton manifest creation and optional name registration
// on top of dpa store
func (self *Api) Put(content, contentType, address, domain string) (string, error) {
	sr := io.NewSectionReader(strings.NewReader(content), 0, int64(len(content)))
	wg := &sync.WaitGroup{}
	key, err := self.dpa.Store(sr, wg)
	if err != nil {
		return "", err
	}
	manifest := fmt.Sprintf(`{"entries":[{"hash":"%064x","contentType":"%s"}]}`, key, contentType)
	sr = io.NewSectionReader(strings.NewReader(manifest), 0, int64(len(manifest)))
	key, err = self.dpa.Store(sr, wg)
	if err != nil {
		return "", err
	}
	wg.Wait()
	return fmt.Sprintf("%064x", key), nil
}

// Download replicates the manifest path structure on the local filesystem
// under localpath
func (self *Api) Download(bzzpath, localpath string) (string, error) {
	return "", nil
}

// Upload replicates a local directory as a manifest file and uploads it
// using dpa store
// TODO: localpath should point to a manifest
func (self *Api) Upload(localpath, address, domain string) (string, error) {
	var files []string
	localpath = common.ExpandHomePath(localpath)
	dpaLogger.Debugf("uploading '%s'", localpath)
	err := filepath.Walk(localpath, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		return "", err
	}

	cnt := len(files)
	hashes := make([]Key, cnt)
	errors := make([]error, cnt)
	wg := &sync.WaitGroup{}

	for i, path := range files {
		wg.Add(1)
		go func(i int, path string) {
			f, err := os.Open(path)
			if err == nil {
				stat, _ := f.Stat()
				sr := io.NewSectionReader(f, 0, stat.Size())
				hashes[i], err = self.dpa.Store(sr, wg)
			}
			errors[i] = err
			wg.Done()
		}(i, path)
	}
	wg.Wait()

	var buffer bytes.Buffer
	buffer.WriteString(`{"entries":[`)
	sc := ","
	var pathPrefix *regexp.Regexp
	var relativePath string
	pathPrefix, err = regexp.Compile("^" + localpath + fmt.Sprintf("%s", os.PathSeparator))
	if err != nil {
		return "", err
	}

	for i, path := range files {
		if errors[i] != nil {
			return "", errors[i]
		}
		if i == cnt-1 {
			sc = "]}"
		}
		relativePath = pathPrefix.ReplaceAllString(path, "")
		buffer.WriteString(fmt.Sprintf(`{"hash":"%064x","path":"%s","contentType":"text/plain"}%s`, hashes[i], relativePath, sc))
	}

	manifest := buffer.Bytes()
	sr := io.NewSectionReader(bytes.NewReader(manifest), 0, int64(len(manifest)))
	key, err2 := self.dpa.Store(sr, wg)
	wg.Wait()
	return fmt.Sprintf("%064x", key), err2
}

type errResolve error

func (self *Api) resolveHost(hostport string) (contentHash Key, errR errResolve) {
	var host, port string
	var err error
	host, port, err = net.SplitHostPort(hostport)
	if err != nil {
		if err.Error() == "missing port in address "+hostport {
			host = hostport
		} else {
			errR = errResolve(fmt.Errorf("invalid host '%s': %v", hostport, err))
			return
		}
	}
	if hashMatcher.MatchString(host) {
		contentHash = Key(common.Hex2Bytes(host))
		dpaLogger.Debugf("Swarm host is a contentHash: '%064x'", contentHash)
	} else {
		if self.Resolver != nil {
			hostHash := common.BytesToHash(crypto.Sha3(common.Hex2Bytes(host)))
			// TODO: should take port as block number versioning
			_ = port
			var hash common.Hash
			hash, err = self.Resolver.KeyToContentHash(hostHash)
			if err != nil {
				err = errResolve(fmt.Errorf("unable to resolve '%s': %v", hostport, err))
			}
			contentHash = Key(hash.Bytes())
			dpaLogger.Debugf("Swarm resolve host to contentHash: '%064x'", contentHash)
		} else {
			err = errResolve(fmt.Errorf("no resolver '%s': %v", hostport, err))
		}
	}
	return
}

func (self *Api) getPath(uri string) (reader SectionReader, mimeType string, status int, err error) {
	parts := slashes.Split(uri, 3)
	hostPort := parts[1]
	var path string
	if len(parts) > 2 {
		path = parts[2]
	}
	dpaLogger.Debugf("Swarm: host: '%s', path '%s' requested.", hostPort, path)

	//resolving host and port
	var key Key
	key, err = self.resolveHost(hostPort)
	if err != nil {
		return
	}

	// retrieve content following path along manifests
	var pos int
	for {
		dpaLogger.Debugf("Swarm: manifest lookup key: '%064x'.", key)
		// retrieve manifest via DPA
		manifestReader := self.dpa.Retrieve(key)
		// TODO check size for oversized manifests
		manifestData := make([]byte, manifestReader.Size())
		var size int
		size, err = manifestReader.Read(manifestData)
		if int64(size) < manifestReader.Size() {
			dpaLogger.Debugf("Swarm: Manifest for '%s' not found.", uri)
			if err == nil {
				err = fmt.Errorf("Manifest retrieval cut short: %v &lt; %v", size, manifestReader.Size())
			}
			return
		}

		dpaLogger.Debugf("Swarm: Manifest for '%s' retrieved", uri)
		man := manifest{}
		err = json.Unmarshal(manifestData, &man)
		if err != nil {
			err = fmt.Errorf("Manifest for '%s' is malformed: %v", uri, err)
			dpaLogger.Debugf("Swarm: %v", err)
			return
		}

		dpaLogger.Debugf("Swarm: Manifest for '%s' has %d entries. Retrieving entry for '%s'", uri, len(man.Entries), path)

		// retrieve entry that matches path from manifest entries
		var entry *manifestEntry
		entry, pos = man.getEntry(path)
		if entry == nil {
			err = fmt.Errorf("Content for '%s' not found.", uri)
			return
		}

		// check hash of entry
		if !hashMatcher.MatchString(entry.Hash) {
			err = fmt.Errorf("Incorrect hash '%064x' for '%s'", entry.Hash, uri)
			return
		}
		key = common.Hex2Bytes(entry.Hash)
		status = entry.Status

		// get mime type of entry
		mimeType = entry.ContentType
		if mimeType == "" {
			mimeType = manifestType
		}

		// if path matched on non-manifest content type, then retrieve reader
		// and return
		if mimeType != manifestType {
			dpaLogger.Debugf("Swarm: content lookup key: '%064x' (%v)", key, mimeType)
			reader = self.dpa.Retrieve(key)
			return
		}

		// otherwise continue along the path with manifest resolution
		path = path[pos:]
	}
	return
}
