package storage

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/idna"

	"github.com/ethereum/go-ethereum/common"
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
	nameHash   common.Hash
	startBlock uint64
	lastPeriod uint32
	frequency  uint64
	version    uint32
	data       []byte
	updated    time.Time
}

// Mutable resource is an entity which allows updates to a resource
// without resorting to ENS on each update.
// The update scheme is built on swarm chunks with chunk keys following
// a predictable, versionable pattern.
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
// actual updates. Thus, a resource update for identifier "føø.bar"
// starting at block 4200 with frequency 42 will have updates on block 4242,
// 4284, 4326 and so on.
//
// The identifier is supplied as a string, but will be IDNA converted and
// passed through the ENS namehash function. Pure ascii identifiers without
// periods will thus merely be hashed.
//
// Note that the root entry is not required for the resource update scheme to
// work. A normal chunk of the blocknumber/frequency data can also be created,
// and pointed to by an external resource (ENS or manifest entry)
//
// Actual data updates are also made in the form of swarm chunks. The keys
// of the updates are the hash of a concatenation of properties as follows:
//
// sha256(namehash|period|version)
//
// The period is (currentblock - startblock) / frequency
//
// Using our previous example, this means that a period 3 will have 4326 as
// the block number.
//
// If more than one update is made to the same block number, incremental
// version numbers are used successively.
//
// A lookup agent need only know the identifier name in order to get the versions
//
// the chunk data is: sign(resourcedata)|resourcedata
// the resourcedata is: headerlength|period|version|name|data
//
// headerlength is a 16 bit value containing the byte length of period|version|name
// period and version are both 32 bit values. name can have arbitrary length
//
// NOTE: the following is yet to be implemented
// The resource update chunks will be stored in the swarm, but receive special
// treatment as their keys do not validate as hashes of their data. They are also
// stored using a separate store, and forwarding/syncing protocols carry per-chunk
// flags to tell whether the chunk can be validated or not; if not it is to be
// treated as a resource update chunk.

type ResourceValidator interface {
	isOwner(string) (bool, error)
	nameHash(string) common.Hash
}

type ResourceHandler struct {
	ChunkStore
	validator    ResourceValidator
	rpcClient    *rpc.Client
	resources    map[string]*resource
	hashLock     sync.Mutex
	resourceLock sync.RWMutex
	hasher       SwarmHash
	privKey      *ecdsa.PrivateKey
	maxChunkData int64
}

// Create or open resource update chunk store
func NewResourceHandler(privKey *ecdsa.PrivateKey, hasher SwarmHasher, chunkStore ChunkStore, rpcClient *rpc.Client, validator ResourceValidator) (*ResourceHandler, error) {
	rh := &ResourceHandler{
		ChunkStore:   chunkStore,
		rpcClient:    rpcClient,
		resources:    make(map[string]*resource),
		hasher:       hasher(),
		privKey:      privKey,
		maxChunkData: DefaultBranches * int64(hasher().Size()),
	}

	if validator != nil {
		rh.validator = validator
	} else {
		rh.validator = NewGenericValidator(func(name string) common.Hash {
			rh.hashLock.Lock()
			defer rh.hashLock.Unlock()
			rh.hasher.Reset()
			rh.hasher.Write([]byte(name))
			return common.BytesToHash(rh.hasher.Sum(nil))
		})
	}

	return rh, nil
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
func NewResource(name string, startBlock uint64, frequency uint64, nameHashFunc func(name string) common.Hash) (*resource, error) {

	validname, err := validateInput(name, frequency)
	if err != nil {
		return nil, err
	}

	return &resource{
		name:       validname,
		nameHash:   nameHashFunc(validname),
		startBlock: startBlock,
		frequency:  frequency,
	}, nil
}

// Creates a new root entry for a mutable resource identified by `name` with the specified `frequency`.
//
// The start block of the resource update will be the actual current block height of the connected network.
func (self *ResourceHandler) NewResource(name string, frequency uint64) (*resource, error) {

	ok, err := self.validator.isOwner(name)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, fmt.Errorf("Not owner of '%s'", name)
	}

	validname, err := validateInput(name, frequency)
	if err != nil {
		return nil, err
	}

	nameHash := self.validator.nameHash(validname)

	// get our blockheight at this time
	currentblock, err := self.getBlock()
	if err != nil {
		return nil, err
	}

	// chunk with key equal to namehash points to data of first blockheight + update frequency
	// from this we know from what blockheight we should look for updates, and how often
	chunk := NewChunk(Key(nameHash[:]), nil)
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
	log.Debug("new resource", "name", validname, "key", nameHash, "startBlock", currentblock, "frequency", frequency)

	rsrc := &resource{
		name:       validname,
		nameHash:   nameHash,
		startBlock: currentblock,
		frequency:  frequency,
		updated:    time.Now(),
	}
	self.setResource(name, rsrc)

	return self.resources[name], nil
}

// Set an externally defined resource object
//
// If the resource update root chunk is located externally (for example as a normal
// chunk looked up by ENS) the data would be manually added with this method).
//
// Method will fail if resource is already registered in this session, unless
// `allowOverwrite` is set
func (self *ResourceHandler) SetExternalResource(rsrc *resource, allowOverwrite bool) error {

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
func (self *ResourceHandler) LookupVersion(name string, period uint32, version uint32, refresh bool) (*resource, error) {
	rsrc, err := self.loadResource(name, refresh)
	if err != nil {
		return nil, err
	}
	return self.lookup(rsrc, name, period, version, refresh)
}

// Retrieves the latest version of the resource update identified by `name`
// at the specified block height
//
// If an update is found, version numbers are iterated until failure, and the last
// successfully retrieved version is copied to the corresponding resources map entry
// and returned.
//
// See also (*ResourceHandler).LookupVersion
func (self *ResourceHandler) LookupHistorical(name string, period uint32, refresh bool) (*resource, error) {
	rsrc, err := self.loadResource(name, refresh)
	if err != nil {
		return nil, err
	}
	return self.lookup(rsrc, name, period, 0, refresh)
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
	nextperiod := getNextPeriod(rsrc.startBlock, currentblock, rsrc.frequency)
	return self.lookup(rsrc, name, nextperiod, 0, refresh)
}

// base code for public lookup methods
func (self *ResourceHandler) lookup(rsrc *resource, name string, period uint32, version uint32, refresh bool) (*resource, error) {

	if period == 0 {
		return nil, fmt.Errorf("period must be >0")
	}

	// start from the last possible block period, and iterate previous ones until we find a match
	// if we hit startBlock we're out of options
	var specificversion bool
	if version > 0 {
		specificversion = true
	} else {
		version = 1
	}

	for period > 0 {
		key := self.resourceHash(rsrc.nameHash, period, version)
		chunk, err := self.Get(key)
		if err == nil {
			if specificversion {
				return self.updateResourceIndex(rsrc, chunk, &name)
			}
			// check if we have versions > 1. If a version fails, the previous version is used and returned.
			log.Trace("rsrc update version 1 found, checking for version updates", "period", period, "key", key)
			for {
				newversion := version + 1
				key := self.resourceHash(rsrc.nameHash, period, newversion)
				newchunk, err := self.Get(key)
				if err != nil {
					return self.updateResourceIndex(rsrc, chunk, &name)
				}
				log.Trace("version update found, checking next", "version", version, "period", period, "key", key)
				chunk = newchunk
				version = newversion
			}
		}
		log.Trace("rsrc update not found, checking previous period", "period", period, "key", key)
		period--
	}
	return nil, fmt.Errorf("no updates found")
}

// load existing mutable resource into resource struct
func (self *ResourceHandler) loadResource(name string, refresh bool) (*resource, error) {
	// if the resource is not known to this session we must load it
	// if refresh is set, we force load

	rsrc := self.getResource(name)
	if rsrc == nil || refresh {
		rsrc = &resource{}
		// make sure our ens identifier is idna safe
		validname, err := idna.ToASCII(name)
		if err != nil {
			return nil, err
		}
		rsrc.name = validname
		rsrc.nameHash = self.validator.nameHash(validname)

		// get the root info chunk and update the cached value
		chunk, err := self.Get(Key(rsrc.nameHash[:]))
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
		rsrc.nameHash = self.resources[name].nameHash
		rsrc.startBlock = self.resources[name].startBlock
		rsrc.frequency = self.resources[name].frequency
	}
	return rsrc, nil
}

// update mutable resource index map with specified content
func (self *ResourceHandler) updateResourceIndex(rsrc *resource, chunk *Chunk, indexname *string) (*resource, error) {

	// rsrc update data chunks are total hacks
	// and have no size prefix :D
	err := self.verifyContent(chunk.SData)
	if err != nil {
		return nil, err
	}

	// update our rsrcs entry map
	period, version, _, data, err := parseUpdate(chunk.SData[signatureLength:])
	rsrc.lastPeriod = period
	rsrc.version = version
	rsrc.updated = time.Now()
	rsrc.data = make([]byte, len(data))
	copy(rsrc.data, data)
	log.Debug("Resource synced", "name", rsrc.name, "key", chunk.Key, "period", rsrc.lastPeriod, "version", rsrc.version)
	self.setResource(*indexname, rsrc)
	return rsrc, nil
}

func parseUpdate(blob []byte) (period uint32, version uint32, ensname []byte, data []byte, err error) {
	headerlength := binary.LittleEndian.Uint16(blob[:2])
	if int(headerlength+2) > len(blob) {
		return 0, 0, nil, nil, fmt.Errorf("Reported header length %d longer than actual data length %d", headerlength, len(blob))
	}
	cursor := 2
	period = binary.LittleEndian.Uint32(blob[cursor : cursor+4])
	cursor += 4
	version = binary.LittleEndian.Uint32(blob[cursor : cursor+4])
	cursor += 4
	namelength := int(headerlength) - cursor + 2
	ensname = make([]byte, namelength)
	copy(ensname, blob[cursor:])
	cursor += namelength
	data = make([]byte, len(blob)-cursor)
	copy(data, blob[cursor:])
	return
}

// Adds an actual data update
//
// Uses the data currently loaded in the resources map entry.
// It is the caller's responsibility to make sure that this data is not stale.
//
// A resource update cannot span chunks, and thus has max length 4096
func (self *ResourceHandler) Update(name string, data []byte) (Key, error) {

	ok, err := self.validator.isOwner(name)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, fmt.Errorf("Not owner of '%s'", name)
	}

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
	nextperiod := getNextPeriod(resource.startBlock, currentblock, resource.frequency)

	// if we already have an update for this block then increment version
	var version uint32
	if self.hasUpdate(name, nextperiod) {
		version = resource.version
	}
	version++

	// prepend version and period to allow reverse lookups
	// data header length does NOT include the header length prefix bytes themselves
	headerlength := uint16(len(resource.nameHash) + 4 + 4)
	fulldata := make([]byte, int(headerlength)+2+len(data))

	cursor := 0
	binary.LittleEndian.PutUint16(fulldata, headerlength)
	cursor += 2

	binary.LittleEndian.PutUint32(fulldata[cursor:], nextperiod)
	cursor += 4

	binary.LittleEndian.PutUint32(fulldata[cursor:], version)
	cursor += 4

	copy(fulldata[cursor:], resource.nameHash[:])
	cursor += len(resource.nameHash)

	copy(fulldata[cursor:], data)

	// create the update chunk and send it
	key := self.resourceHash(resource.nameHash, nextperiod, version)
	chunk := NewChunk(key, nil)
	chunk.SData, err = self.signContent(fulldata)
	if err != nil {
		return nil, err
	}
	chunk.Size = int64(len(fulldata))
	self.Put(chunk)
	log.Trace("resource update", "name", resource.name, "key", key, "currentblock", currentblock, "lastperiod", nextperiod, "version", version, "data", chunk.SData)

	// update our resources map entry and return the new key
	resource.lastPeriod = nextperiod
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
	err := self.rpcClient.Call(&currentblock, "eth_blockNumber")
	if err != nil {
		return 0, err
	}
	if currentblock == "0x0" {
		return 0, nil
	}
	return strconv.ParseUint(currentblock, 10, 64)
}

func (self *ResourceHandler) BlockToPeriod(name string, blocknumber uint64) uint32 {
	return getNextPeriod(self.resources[name].startBlock, blocknumber, self.resources[name].frequency)
}

func (self *ResourceHandler) PeriodToBlock(name string, period uint32) uint64 {
	return self.resources[name].startBlock + (uint64(period) * self.resources[name].frequency)
}

func (self *ResourceHandler) getResource(name string) *resource {
	self.resourceLock.RLock()
	defer self.resourceLock.RUnlock()
	rsrc := self.resources[name]
	return rsrc
}

func (self *ResourceHandler) setResource(name string, rsrc *resource) {
	self.resourceLock.Lock()
	defer self.resourceLock.Unlock()
	self.resources[name] = rsrc
}

func (self *ResourceHandler) resourceHash(namehash common.Hash, period uint32, version uint32) Key {
	// format is: hash(namehash|period|version)
	self.hashLock.Lock()
	defer self.hashLock.Unlock()
	self.hasher.Reset()
	self.hasher.Write(namehash[:])
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, period)
	self.hasher.Write(b)
	binary.LittleEndian.PutUint32(b, version)
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

func (self *ResourceHandler) hasUpdate(name string, period uint32) bool {
	return self.resources[name].lastPeriod == period
}

type resourceChunkStore struct {
	localStore ChunkStore
	netStore   ChunkStore
}

func newResourceChunkStore(path string, hasher SwarmHasher, localStore *LocalStore, request func(*Chunk) error) *resourceChunkStore {
	return &resourceChunkStore{
		localStore: localStore,
		netStore:   NewNetStore(localStore, request),
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
	if chunk.ReqC == nil {
		return chunk, nil
	}
	t := time.NewTimer(time.Second * 1)
	select {
	case <-t.C:
		return nil, fmt.Errorf("timeout")
	case <-chunk.C:
		log.Trace("Received resource update chunk")
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

func getNextPeriod(start uint64, current uint64, frequency uint64) uint32 {
	blockdiff := current - start
	period := blockdiff / frequency
	return uint32(period + 1)
}
