package storage

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/idna"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	signatureLength = 65
	indexSize       = 24
)

// Encapsulates an actual resource update. When synced it contains the most recent
// version of the resource update data.
type resource struct {
	name       string
	ensName    common.Hash
	startBlock uint64
	lastBlock  uint64
	frequency  uint64
	version    uint64
	data       []byte
	updated    time.Time
}

// Mutable resource is an entity which allows updates to a resource
// without resorting to ENS on each update.
// The update scheme is built on swarm chunks with chunk keys following
// a predictable, versionable pattern.
//
// The data of the chunk contains the content hash of the version in question.
// In order to be valid, the hash is signed by the owner of the ENS record
// of the mutable resource.
//
// Updates are defined to be periodic in nature, where periods are
// expressed in terms of number of blocks.
//
// The root entry of a mutable resource is tied to a unique identifier,
// typically - but not necessarily - an ens name. It also contains the
// block number when the resource update was first registered, and
// the block frequency with which the resource will be updated, both of
// which are stored as little-endian uint64 values in the database (for a
// total of 16 bytes).

// The root entry tells the requester from when the mutable resource was
// first added (block number) and in which block number to look for the
// actual updates. Thus, a resource update for identifier "foo.bar"
// starting at block 4200 with frequency 42 will have updates on block 4242,
// 4284, 4326 and so on.
//
// The identifier is supplied as a string, but will be IDNA converted and
// passed through the ENS namehash function. Pure ascii identifiers without
// periods will thus merely be hashed.
//
// Note that the root entry is not required for the resource update scheme to
// work. A normal chunk of the blocknumber/frequency data can also be created,
// and pointed to by an actual ENS entry (or manifest entry) instead.
//
// Actual data updates are also made in the form of swarm chunks. The keys
// of the updates are the hash of a concatenation of properties as follows:
//
// sha256(namehash|blocknumber|version)
//
// The blocknumber here is the next block period after the current block
// calculated from the start block and frequency of the resource update.
// Using our previous example, this means that an update made at block 4285,
// and even 4284, will have 4326 as the block number.
//
// If more than one update is made to the same block number, incremental
// version numbers are used successively.
//
// A lookup agent need only know the identifier name in order to get the versions
//
// the data itself is prefixed with a signed hash of the data. The sigining key
// is used to verify the authenticity of the update, for example by looking
// up the ownership of the namehash in ENS and comparing to the address derived
// from it
//
// NOTE: the following is yet to be implemented
// The resource update chunks will be stored in the swarm, but receive special
// treatment as their keys do not validate as hashes of their data. They are also
// stored using a separate store, and forwarding/syncing protocols carry per-chunk
// flags to tell whether the chunk can be validated or not; if not it is to be
// treated as a resource update chunk.
//
// TODO: signature validation
type ResourceHandler struct {
	ChunkStore
	ethapi       *rpc.Client
	resources    map[string]*resource
	hashLock     sync.Mutex
	resourceLock sync.Mutex
	hasher       SwarmHash
	privKey      *ecdsa.PrivateKey
	maxChunkData int64
}

// Create or open resource update chunk store
func NewResourceHandler(privKey *ecdsa.PrivateKey, datadir string, cloudStore CloudStore, ethapi *rpc.Client) (*ResourceHandler, error) {
	path := filepath.Join(datadir, "resource")
	dbStore, err := NewDbStore(datadir, nil, singletonSwarmDbCapacity, 0)
	if err != nil {
		return nil, err
	}
	localStore := &LocalStore{
		memStore: NewMemStore(dbStore, singletonSwarmDbCapacity),
		DbStore:  dbStore,
	}
	hasher := MakeHashFunc("SHA3")
	return &ResourceHandler{
		ChunkStore:   newResourceChunkStore(path, hasher, localStore, cloudStore),
		ethapi:       ethapi,
		resources:    make(map[string]*resource),
		hasher:       hasher(),
		privKey:      privKey,
		maxChunkData: DefaultBranches * int64(hasher().Size()),
	}, nil
}

func validateInput(name string, frequency uint64) (string, error) {
	// frequency 0 is invalid
	if frequency == 0 {
		return "", fmt.Errorf("Frequency cannot be 0")
	}

	// must have name
	if name == "" {
		return "", fmt.Errorf("Name cannot be empty")
	}

	// make sure our ens identifier is idna safe
	validname, err := idna.ToASCII(name)
	if err != nil {
		return "", err
	}

	return validname, nil
}

// Creates a standalone resource object
//
// Can be passed to SetResource if external root data lookups are used
func NewResource(name string, startBlock uint64, frequency uint64) (*resource, error) {

	validname, err := validateInput(name, frequency)
	if err != nil {
		return nil, err
	}

	return &resource{
		name:       validname,
		ensName:    ens.EnsNode(validname),
		startBlock: startBlock,
		frequency:  frequency,
	}, nil
}

// Creates a new root entry for a resource update identified by `name` with the specified `frequency`.
//
// The start block of the resource update will be the actual current block height of the connected network.
func (self *ResourceHandler) NewResource(name string, frequency uint64) (*resource, error) {

	validname, err := validateInput(name, frequency)
	if err != nil {
		return nil, err
	}

	ensName := ens.EnsNode(validname)

	// get our blockheight at this time
	currentblock, err := self.getBlock()
	if err != nil {
		return nil, err
	}

	// chunk with key equal to namehash points to data of first blockheight + update frequency
	// from this we know from what blockheight we should look for updates, and how often
	chunk := NewChunk(Key(ensName[:]), nil)
	chunk.SData = make([]byte, indexSize)

	// resource update root chunks follow same convention as "normal" chunks
	// with 8 bytes prefix specifying size
	val := make([]byte, 8)
	chunk.SData[0] = 16 // size, little-endian
	binary.LittleEndian.PutUint64(val, currentblock)
	copy(chunk.SData[8:16], val)
	binary.LittleEndian.PutUint64(val, frequency)
	copy(chunk.SData[16:], val)
	self.Put(chunk)
	log.Debug("new resource", "name", validname, "key", ensName, "startBlock", currentblock, "frequency", frequency)

	self.resourceLock.Lock()
	defer self.resourceLock.Unlock()
	self.resources[name] = &resource{
		name:       validname,
		ensName:    ensName,
		startBlock: currentblock,
		frequency:  frequency,
		updated:    time.Now(),
	}
	return self.resources[name], nil
}

// Set an externally defined resource object
//
// If the resource update root chunk is located externally (for example as a normal
// chunk looked up by ENS) the data would be manually added with this method).
//
// Method will fail if resource is already registered in this session, unless
// `allowOverwrite` is set
func (self *ResourceHandler) SetResource(rsrc *resource, allowOverwrite bool) error {

	utfname, err := idna.ToUnicode(rsrc.name)
	if err != nil {
		return fmt.Errorf("Invalid IDNA rsrc name '%s'", rsrc.name)
	}
	if !allowOverwrite {
		self.resourceLock.Lock()
		_, ok := self.resources[utfname]
		self.resourceLock.Unlock()
		if ok {
			return fmt.Errorf("Resource exists")
		}
	}

	// get our blockheight at this time
	currentblock, err := self.getBlock()
	if err != nil {
		return err
	}

	if rsrc.startBlock > currentblock {
		return fmt.Errorf("Startblock cannot be higher than current block (%d > %d)", rsrc.startBlock, currentblock)
	}

	self.resources[utfname] = rsrc
	return nil
}

// Searches and retrieves the specific version of the resource update identified by `name`
// at the specific block height
//
//
// If refresh is set to true, the resource data will be reloaded from the resource update
// root chunk.
// It is the callers responsibility to make sure that this chunk exists (if the resource
// update root data was retrieved externally, it typically doesn't)
func (self *ResourceHandler) LookupVersion(name string, nextblock uint64, version uint64, refresh bool) (*resource, error) {
	rsrc, err := self.loadResource(name, refresh)
	if err != nil {
		return nil, err
	}
	return self.lookup(rsrc, name, nextblock, version, refresh)
}

// Retrieves the latest version of the resource update identified by `name`
// at the specified block height
//
// If an update is found, version numbers are iterated until failure, and the last
// successfully retrieved version is copied to the corresponding resources map entry
// and returned.
//
// See also (*ResourceHandler).LookupVersion
func (self *ResourceHandler) LookupHistorical(name string, nextblock uint64, refresh bool) (*resource, error) {
	rsrc, err := self.loadResource(name, refresh)
	if err != nil {
		return nil, err
	}
	return self.lookup(rsrc, name, nextblock, 0, refresh)
}

// Retrieves the latest version of the resource update identified by `name`
// at the next update block height
//
// It starts at the next period after the current block height, and upon failure
// tries the corresponding keys of each previous period until one is found
// (or startBlock is reached, in which case there are no updates).
//
// Version iteration is done as in (*ResourceHandler).LookupHistorical
//
// See also (*ResourceHandler).LookupHistorical
func (self *ResourceHandler) LookupLatest(name string, refresh bool) (*resource, error) {

	// get our blockheight at this time and the next block of the update period
	rsrc, err := self.loadResource(name, refresh)
	if err != nil {
		return nil, err
	}
	currentblock, err := self.getBlock()
	if err != nil {
		return nil, err
	}
	nextblock := getNextBlock(rsrc.startBlock, currentblock, rsrc.frequency)
	return self.lookup(rsrc, name, nextblock, 0, refresh)
}

// base code for public lookup methods
func (self *ResourceHandler) lookup(rsrc *resource, name string, nextblock uint64, version uint64, refresh bool) (*resource, error) {

	if nextblock == 0 {
		return nil, fmt.Errorf("blocknumber must be >0")
	}

	// start from the last possible block period, and iterate previous ones until we find a match
	// if we hit startBlock we're out of options
	var specificversion bool
	if version > 0 {
		specificversion = true
	} else {
		version = 1
	}

	for nextblock > rsrc.startBlock {
		key := self.resourceHash(rsrc.ensName, nextblock, version)
		chunk, err := self.Get(key)
		if err == nil {
			if specificversion {
				return self.updateResourceIndex(rsrc, chunk, nextblock, version, &name)
			}
			// check if we have versions > 1. If a version fails, the previous version is used and returned.
			log.Trace("rsrc update version 1 found, checking for version updates", "nextblock", nextblock, "key", key)
			for {
				newversion := version + 1
				key := self.resourceHash(rsrc.ensName, nextblock, newversion)
				newchunk, err := self.Get(key)
				if err != nil {
					return self.updateResourceIndex(rsrc, chunk, nextblock, version, &name)
				}
				log.Trace("version update found, checking next", "version", version, "block", nextblock, "key", key)
				chunk = newchunk
				version = newversion
			}
		}
		log.Trace("rsrc update not found, checking previous period", "block", nextblock, "key", key)
		nextblock -= rsrc.frequency
	}
	return nil, fmt.Errorf("no updates found")
}

// load existing mutable resource into resource struct
func (self *ResourceHandler) loadResource(name string, refresh bool) (*resource, error) {
	// if the resource is not known to this session we must load it
	// if refresh is set, we force load

	rsrc := &resource{}

	self.resourceLock.Lock()
	_, ok := self.resources[name]
	self.resourceLock.Unlock()
	if !ok || refresh {
		// make sure our ens identifier is idna safe
		validname, err := idna.ToASCII(name)
		if err != nil {
			return nil, err
		}
		rsrc.name = validname
		rsrc.ensName = ens.EnsNode(validname)

		// get the root info chunk and update the cached value
		chunk, err := self.Get(Key(rsrc.ensName[:]))
		if err != nil {
			return nil, err
		}

		// sanity check for chunk data
		// data is prefixed by 8 bytes of size
		if len(chunk.SData) < indexSize {
			return nil, fmt.Errorf("Invalid chunk length %d", len(chunk.SData))
		} else {
			chunklength := binary.LittleEndian.Uint64(chunk.SData[:8])
			if chunklength != uint64(16) {
				return nil, fmt.Errorf("Invalid chunk length header %d", chunklength)
			}
		}
		rsrc.startBlock = binary.LittleEndian.Uint64(chunk.SData[8:16])
		rsrc.frequency = binary.LittleEndian.Uint64(chunk.SData[16:])
	} else {
		rsrc.name = self.resources[name].name
		rsrc.ensName = self.resources[name].ensName
		rsrc.startBlock = self.resources[name].startBlock
		rsrc.frequency = self.resources[name].frequency
	}
	return rsrc, nil
}

// update mutable resource index map with specified content
func (self *ResourceHandler) updateResourceIndex(rsrc *resource, chunk *Chunk, nextblock uint64, version uint64, indexname *string) (*resource, error) {

	// rsrc update data chunks are total hacks
	// and have no size prefix :D
	err := self.verifyContent(chunk.SData)
	if err != nil {
		return nil, err
	}

	// update our rsrcs entry map
	rsrc.lastBlock = nextblock
	rsrc.version = version
	rsrc.data = make([]byte, len(chunk.SData)-signatureLength)
	rsrc.updated = time.Now()
	copy(rsrc.data, chunk.SData[signatureLength:])
	log.Debug("Resource synced", "name", rsrc.name, "key", chunk.Key, "block", nextblock, "version", version)
	self.resourceLock.Lock()
	self.resources[*indexname] = rsrc
	self.resourceLock.Unlock()
	return rsrc, nil
}

// Adds an actual data update
//
// Uses the data currently loaded in the resources map entry.
// It is the caller's responsibility to make sure that this data is not stale.
//
// A resource update cannot span chunks, and thus has max length 4096
func (self *ResourceHandler) Update(name string, data []byte) (Key, error) {

	// can be only one chunk long minus 65 byte signature
	if int64(len(data)) > self.maxChunkData {
		return nil, fmt.Errorf("Data overflow: %d / %d bytes", len(data), 4096-signatureLength)
	}

	// get the cached information
	self.resourceLock.Lock()
	defer self.resourceLock.Unlock()
	resource, ok := self.resources[name]
	if !ok {
		return nil, fmt.Errorf("No such resource")
	} else if resource.updated.IsZero() {
		return nil, fmt.Errorf("Invalid resource")
	}

	// get our blockheight at this time and the next block of the update period
	currentblock, err := self.getBlock()
	if err != nil {
		return nil, err
	}
	nextblock := getNextBlock(resource.startBlock, currentblock, resource.frequency)

	// if we already have an update for this block then increment version
	var version uint64
	if nextblock == resource.lastBlock {
		version = resource.version
	}
	version++

	// create the update chunk and send it
	key := self.resourceHash(resource.ensName, nextblock, version)
	chunk := NewChunk(key, nil)
	chunk.SData, err = self.signContent(data)
	if err != nil {
		return nil, err
	}
	chunk.Size = int64(len(data))
	self.Put(chunk)
	log.Trace("resource update", "name", resource.name, "key", key, "currentblock", currentblock, "lastBlock", nextblock, "version", version)

	// update our resources map entry and return the new key
	resource.lastBlock = nextblock
	resource.version = version
	resource.data = make([]byte, len(data))
	copy(resource.data, data)
	return key, nil
}

// Closes the datastore.
// Always call this at shutdown to avoid data corruption.
func (self *ResourceHandler) Close() {
	self.ChunkStore.Close()
}

func (self *ResourceHandler) getBlock() (uint64, error) {
	// get the block height and convert to uint64
	var currentblock string
	err := self.ethapi.Call(&currentblock, "eth_blockNumber")
	if err != nil {
		return 0, err
	}
	if currentblock == "0x0" {
		return 0, nil
	}
	return strconv.ParseUint(currentblock, 10, 64)
}

func (self *ResourceHandler) resourceHash(namehash common.Hash, blockheight uint64, version uint64) Key {
	// format is: hash(namehash|blockheight|version)
	self.hashLock.Lock()
	defer self.hashLock.Unlock()
	self.hasher.Reset()
	self.hasher.Write(namehash[:])
	b := make([]byte, 8)
	c := binary.PutUvarint(b, blockheight)
	self.hasher.Write(b)
	// PutUvarint only overwrites first c bytes
	for i := 0; i < c; i++ {
		b[i] = 0
	}
	c = binary.PutUvarint(b, version)
	self.hasher.Write(b)
	return self.hasher.Sum(nil)
}

func (self *ResourceHandler) signContent(data []byte) ([]byte, error) {
	self.hashLock.Lock()
	self.hasher.Reset()
	self.hasher.Write(data)
	datahash := self.hasher.Sum(nil)
	self.hashLock.Unlock()

	signature, err := crypto.Sign(datahash, self.privKey)
	if err != nil {
		return nil, err
	}
	datawithsign := make([]byte, len(data)+signatureLength)
	copy(datawithsign[:signatureLength], signature)
	copy(datawithsign[signatureLength:], data)
	return datawithsign, nil
}

func (self *ResourceHandler) getContentAccount(chunkdata []byte) (common.Address, error) {
	if len(chunkdata) <= signatureLength {
		return common.Address{}, fmt.Errorf("zero-length data")
	}
	self.hashLock.Lock()
	self.hasher.Reset()
	self.hasher.Write(chunkdata[signatureLength:])
	datahash := self.hasher.Sum(nil)
	self.hashLock.Unlock()
	pub, err := crypto.SigToPub(datahash, chunkdata[:signatureLength])
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pub), nil
}

func (self *ResourceHandler) verifyContent(chunkdata []byte) error {
	address, err := self.getContentAccount(chunkdata)
	if err != nil {
		return err
	}
	log.Warn("ens owner lookup not implemented, verify will return true in all cases", "address", address)
	return nil
}

type resourceChunkStore struct {
	localStore ChunkStore
	netStore   ChunkStore
}

func newResourceChunkStore(path string, hasher SwarmHasher, localStore *LocalStore, cloudStore CloudStore) *resourceChunkStore {
	return &resourceChunkStore{
		localStore: localStore,
		netStore:   NewNetStore(hasher, localStore, cloudStore, NewStoreParams(path)),
	}
}

func (r *resourceChunkStore) Get(key Key) (*Chunk, error) {
	chunk, err := r.netStore.Get(key)
	if err != nil {
		return nil, err
	}
	// if the chunk has to be remotely retrieved, we define a timeout of how long to wait for it before failing.
	// sadly due to the nature of swarm, the error will never be conclusive as to whether it was a network issue
	// that caused the failure or that the chunk doesn't exist.
	if chunk.Req == nil {
		return chunk, nil
	}
	t := time.NewTimer(time.Second * 1)
	select {
	case <-t.C:
		return nil, fmt.Errorf("timeout")
	case <-chunk.C:
		log.Trace("Received resource update chunk", "peer", chunk.Req.Source)
	}
	return chunk, nil
}

func (r *resourceChunkStore) Put(chunk *Chunk) {
	r.netStore.Put(chunk)
}

func (r *resourceChunkStore) Close() {
	r.netStore.Close()
	r.localStore.Close()
}

func getNextBlock(start uint64, current uint64, frequency uint64) uint64 {
	blockdiff := current - start
	periods := (blockdiff / frequency) + 1
	return start + (frequency * periods)
}
