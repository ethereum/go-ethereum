package bzz

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
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
it is the public interface of the dpa which is included in the ethereum stack
*/
type Api struct {
	Chunker  *TreeChunker
	Port     string
	Resolver *resolver.Resolver
	dpa      *DPA
	netStore *netStore
}

/*
the api constructor initialises
- the netstore endpoint for chunk store logic
- the chunker (bzz hash)
- the dpa - single document retrieval api
*/
func NewApi(datadir, port string) (self *Api, err error) {

	self = &Api{
		Chunker: &TreeChunker{},
		Port:    port,
	}

	self.netStore, err = newNetStore(filepath.Join(datadir, "bzz"), filepath.Join(datadir, "bzzpeers.json"))
	if err != nil {
		return
	}

	self.dpa = &DPA{
		Chunker:    self.Chunker,
		ChunkStore: self.netStore,
	}
	return
}

// Bzz returns the bzz protocol class instances of which run on every peer
func (self *Api) Bzz() (p2p.Protocol, error) {
	return BzzProtocol(self.netStore)
}

/*
Start is called when the ethereum stack is started
- calls Init() on treechunker
- launches the dpa (listening for chunk store/retrieve requests)
- launches the netStore (starts kademlia hive peer management)
- starts an http server
*/
func (self *Api) Start(node *discover.Node, connectPeer func(string) error) {
	self.Chunker.Init()
	self.dpa.Start()
	self.netStore.start(node, connectPeer)
	dpaLogger.Infof("Swarm started.")
	go startHttpServer(self, self.Port)
}

func (self *Api) Stop() {
	self.dpa.Stop()
	self.netStore.stop()
}

// Get uses iterative manifest retrieval and prefix matching
// to resolve path to content using dpa retrieve
func (self *Api) Get(bzzpath string) (content []byte, mimeType string, status int, size int, err error) {
	var reader SectionReader
	reader, mimeType, status, err = self.getPath("/" + bzzpath)
	if err != nil {
		return
	}
	content = make([]byte, reader.Size())
	size, err = reader.Read(content)
	if err == io.EOF {
		err = nil
	}
	return
}

// Put provides singleton manifest creation and optional name registration
// on top of dpa store
func (self *Api) Put(content, contentType string) (string, error) {
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

func (self *Api) Modify(rootHash, path, contentHash, contentType string) (newRootHash string, err error) {
	root := common.Hex2Bytes(rootHash)
	trie, err := loadManifestTrie(self.dpa, root)
	if err != nil {
		return
	}

	if contentHash != "" {
		entry := &manifestTrieEntry{
			Path:        path,
			Hash:        contentHash,
			ContentType: contentType,
		}
		trie.addEntry(entry)
	} else {
		trie.deleteEntry(path)
	}

	err = trie.recalcAndStore()
	if err != nil {
		return
	}
	return fmt.Sprintf("%064x", trie.hash), nil
}

// Download replicates the manifest path structure on the local filesystem
// under localpath
func (self *Api) Download(bzzpath, localpath string) (string, error) {
	return "", nil
}

const maxParallelFiles = 5

// Upload replicates a local directory as a manifest file and uploads it
// using dpa store
// TODO: localpath should point to a manifest
func (self *Api) Upload(lpath string) (string, error) {
	var list []*manifestTrieEntry
	localpath, err1 := filepath.Abs(filepath.Clean(lpath))
	if err1 != nil {
		return "", err1
	}
	start := len(localpath)
	if (start > 0) && (localpath[start-1] != os.PathSeparator) {
		start++
	}
	dpaLogger.Debugf("uploading '%s'", localpath)
	err := filepath.Walk(localpath, func(path string, info os.FileInfo, err error) error {
		if (err == nil) && !info.IsDir() {
			//fmt.Printf("lp %s  path %s\n", localpath, path)
			if len(path) <= start {
				return fmt.Errorf("Path is too short")
			}
			if path[:len(localpath)] != localpath {
				return fmt.Errorf("Path prefix of '%s' does not match localpath '%s'", path, localpath)
			}
			entry := &manifestTrieEntry{
				Path: path,
			}
			list = append(list, entry)
		}
		return err
	})
	if err != nil {
		return "", err
	}

	cnt := len(list)
	errors := make([]error, cnt)
	done := make(chan bool, maxParallelFiles)
	dcnt := 0

	for i, entry := range list {
		if i >= dcnt+maxParallelFiles {
			<-done
			dcnt++
		}
		go func(i int, entry *manifestTrieEntry, done chan bool) {
			f, err := os.Open(entry.Path)
			if err == nil {
				stat, _ := f.Stat()
				sr := io.NewSectionReader(f, 0, stat.Size())
				wg := &sync.WaitGroup{}
				var hash Key
				hash, err = self.dpa.Store(sr, wg)
				if hash != nil {
					list[i].Hash = fmt.Sprintf("%064x", hash)
				}
				wg.Wait()
			}
			if err == nil {
				cmd := exec.Command("file", "--mime-type", "-b", entry.Path)
				var out bytes.Buffer
				cmd.Stdout = &out
				err = cmd.Run()
				if err == nil {
					list[i].ContentType = strings.TrimSuffix(out.String(), "\n")
				}
			}
			errors[i] = err
			done <- true
		}(i, entry, done)
	}
	for dcnt < cnt {
		<-done
		dcnt++
	}

	trie := &manifestTrie{
		dpa: self.dpa,
	}
	for i, entry := range list {
		if errors[i] != nil {
			return "", errors[i]
		}
		entry.Path = entry.Path[start:]
		trie.addEntry(entry)
	}

	err2 := trie.recalcAndStore()
	var hs string
	if err2 == nil {
		hs = fmt.Sprintf("%064x", trie.hash)
	}
	return hs, err2
}

func (self *Api) Register(sender common.Address, hash common.Hash, domain string) (err error) {
	domainhash := common.BytesToHash(crypto.Sha3([]byte(domain)))

	if self.Resolver != nil {
		_, err = self.Resolver.RegisterContentHash(sender, domainhash, hash)
	} else {
		err = fmt.Errorf("no registry: %v", err)
	}
	return
}

type errResolve error

func (self *Api) Resolve(hostport string) (contentHash Key, errR errResolve) {
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
		dpaLogger.Debugf("Swarm: host is a contentHash: '%064x'", contentHash)
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
			dpaLogger.Debugf("Swarm: resolve host to contentHash: '%064x'", contentHash)
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
	key, err = self.Resolve(hostPort)
	if err != nil {
		return
	}

	trie, err := loadManifestTrie(self.dpa, key)
	if err != nil {
		return
	}

	entry, _ := trie.getEntry(path)
	if entry != nil {
		key = common.Hex2Bytes(entry.Hash)
		status = entry.Status
		mimeType = entry.ContentType
		dpaLogger.Debugf("Swarm: content lookup key: '%064x' (%v)", key, mimeType)
		reader = self.dpa.Retrieve(key)
	}
	return
}
