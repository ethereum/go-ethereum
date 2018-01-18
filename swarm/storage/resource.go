package storage

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
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
	indexSize       = 16
	dbDirName       = "resource"
	chunkSize       = 4096 // temporary until we implement DPA in the resourcehandler
)

type Signature [signatureLength]byte

type SignFunc func(common.Hash) (Signature, error)

type nameHashFunc func(string) common.Hash

// Encapsulates an specific resource update. When synced it contains the most recent
// version of the resource update data.
type resource struct {
	name       *string
	nameHash   common.Hash
	startBlock uint64
	lastPeriod uint32
	frequency  uint64
	version    uint32
	data       []byte
	updated    time.Time
}

// TODO Expire content after a defined period (to force resync)
func (r *resource) isSynced() bool {
	return !r.updated.IsZero()
}

// Implement to activate validation of resource updates
// Specifically signing data and verification of signatures
type ResourceValidator interface {
	checkAccess(string, common.Address) (bool, error)
	nameHash(string) common.Hash         // nameHashFunc
	sign(common.Hash) (Signature, error) // SignFunc
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
// typically - but not necessarily - an ens name.  The identifier must be
// an valid IDNA string. It also contains the block number
// when the resource update was first registered, and
// the block frequency with which the resource will be updated, both of
// which are stored as little-endian uint64 values in the database (for a
// total of 16 bytes).

// The root entry tells the requester from when the mutable resource was
// first added (block number) and in which block number to look for the
// actual updates. Thus, a resource update for identifier "føø.bar"
// starting at block 4200 with frequency 42 will have updates on block 4242,
// 4284, 4326 and so on.
//
// Note that the root entry is not required for the resource update scheme to
// work. A normal chunk of the blocknumber/frequency data can also be created,
// and pointed to by an external resource (ENS or manifest entry)
//
// Actual data updates are also made in the form of swarm chunks. The keys
// of the updates are the hash of a concatenation of properties as follows:
//
// sha256(period|version|namehash)
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
// the resourcedata is:
// headerlength|period|version|identifier|data
//
// if a validator is active, the chunk data is:
// sign(resourcedata)|resourcedata
// otherwise, the chunk data is the same as the resourcedata
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
type ResourceHandler struct {
	ChunkStore
	validator    ResourceValidator
	rpcClient    *rpc.Client
	resources    map[string]*resource
	hashLock     sync.Mutex
	resourceLock sync.RWMutex
	hasher       SwarmHash
	nameHash     nameHashFunc
}

// Create or open resource update chunk store
//
// If validator is nil, signature and access validation will be deactivated
func NewResourceHandler(datadir string, cloudStore CloudStore, rpcClient *rpc.Client, validator ResourceValidator) (*ResourceHandler, error) {

	hashfunc := MakeHashFunc(SHA3Hash)

	path := filepath.Join(datadir, dbDirName)
	dbStore, err := NewDbStore(datadir, hashfunc, singletonSwarmDbCapacity, 0)
	if err != nil {
		return nil, err
	}
	localStore := &LocalStore{
		memStore: NewMemStore(dbStore, singletonSwarmDbCapacity),
		DbStore:  dbStore,
	}

	rh := &ResourceHandler{
		ChunkStore: newResourceChunkStore(path, hashfunc, localStore, cloudStore),
		rpcClient:  rpcClient,
		resources:  make(map[string]*resource),
		hasher:     hashfunc(),
		validator:  validator,
	}

	if rh.validator != nil {
		rh.nameHash = rh.validator.nameHash
	} else {
		rh.nameHash = func(name string) common.Hash {
			rh.hashLock.Lock()
			defer rh.hashLock.Unlock()
			rh.hasher.Reset()
			rh.hasher.Write([]byte(name))
			return common.BytesToHash(rh.hasher.Sum(nil))
		}
	}

	return rh, nil
}

// \TODO should be hashsize * branches from the chosen chunker, implement with dpa
func (self *ResourceHandler) chunkSize() int64 {
	return chunkSize
}

// Creates a new root entry for a mutable resource identified by `name` with the specified `frequency`.
//
// The signature data should match the hash of the idna-converted name by the validator's namehash function, NOT the raw name bytes.
//
// The start block of the resource update will be the actual current block height of the connected network.
func (self *ResourceHandler) NewResource(name string, frequency uint64) (*resource, error) {

	// frequency 0 is invalid
	if frequency == 0 {
		return nil, fmt.Errorf("Frequency cannot be 0")
	}

	if !isSafeName(name) {
		return nil, fmt.Errorf("Invalid name: '%s'", name)
	}

	nameHash := self.nameHash(name)

	if self.validator != nil {
		signature, err := self.validator.sign(nameHash)
		if err != nil {
			return nil, fmt.Errorf("Sign fail: %v", err)
		}
		addr, err := getAddressFromDataSig(nameHash, signature)
		if err != nil {
			return nil, fmt.Errorf("Retrieve address from signature fail: %v", err)
		}
		ok, err := self.validator.checkAccess(name, addr)
		if err != nil {
			return nil, err
		} else if !ok {
			return nil, fmt.Errorf("Not owner of '%s'", name)
		}
	}

	// get our blockheight at this time
	currentblock, err := self.getBlock()
	if err != nil {
		return nil, err
	}

	// chunk with key equal to namehash points to data of first blockheight + update frequency
	// from this we know from what blockheight we should look for updates, and how often
	chunk := NewChunk(Key(nameHash.Bytes()), nil)
	chunk.SData = make([]byte, indexSize)

	val := make([]byte, 8)
	binary.LittleEndian.PutUint64(val, currentblock)
	copy(chunk.SData[:8], val)
	binary.LittleEndian.PutUint64(val, frequency)
	copy(chunk.SData[8:], val)
	self.Put(chunk)
	log.Debug("new resource", "name", name, "key", nameHash, "startBlock", currentblock, "frequency", frequency)

	rsrc := &resource{
		name:       &name,
		nameHash:   nameHash,
		startBlock: currentblock,
		frequency:  frequency,
		updated:    time.Now(),
	}
	self.setResource(name, rsrc)

	return rsrc, nil
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
	return self.lookup(rsrc, period, version, refresh)
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
	return self.lookup(rsrc, period, 0, refresh)
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
	return self.lookup(rsrc, nextperiod, 0, refresh)
}

// base code for public lookup methods
func (self *ResourceHandler) lookup(rsrc *resource, period uint32, version uint32, refresh bool) (*resource, error) {

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
		key := self.resourceHash(period, version, rsrc.nameHash)
		chunk, err := self.Get(key)
		if err == nil {
			if specificversion {
				return self.updateResourceIndex(rsrc, chunk)
			}
			// check if we have versions > 1. If a version fails, the previous version is used and returned.
			log.Trace("rsrc update version 1 found, checking for version updates", "period", period, "key", key)
			for {
				newversion := version + 1
				key := self.resourceHash(period, newversion, rsrc.nameHash)
				newchunk, err := self.Get(key)
				if err != nil {
					return self.updateResourceIndex(rsrc, chunk)
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
		// make sure our name is safe to use
		if !isSafeName(name) {
			return nil, fmt.Errorf("Invalid name '%s'")
		}
		rsrc.name = &name
		rsrc.nameHash = self.nameHash(name)

		// get the root info chunk and update the cached value
		chunk, err := self.Get(Key(rsrc.nameHash[:]))
		if err != nil {
			return nil, err
		}

		// minimum sanity check for chunk data
		if len(chunk.SData) != indexSize {
			return nil, fmt.Errorf("Invalid chunk length %d, should be %d", len(chunk.SData), indexSize)
		}
		rsrc.startBlock = binary.LittleEndian.Uint64(chunk.SData[:8])
		rsrc.frequency = binary.LittleEndian.Uint64(chunk.SData[8:])
	} else {
		rsrc.name = self.resources[name].name
		rsrc.nameHash = self.resources[name].nameHash
		rsrc.startBlock = self.resources[name].startBlock
		rsrc.frequency = self.resources[name].frequency
	}
	return rsrc, nil
}

// update mutable resource index map with specified content
func (self *ResourceHandler) updateResourceIndex(rsrc *resource, chunk *Chunk) (*resource, error) {

	// retrieve metadata from chunk data and check that it matches this mutable resource
	signature, period, version, name, data, err := self.parseUpdate(chunk.SData)
	if *rsrc.name != name {
		return nil, fmt.Errorf("Update belongs to '%s', but have '%s'", name, rsrc.name)
	}
	// only check signature if validator is present
	if self.validator != nil {
		digest := self.keyDataHash(chunk.Key, data)
		_, err = getAddressFromDataSig(digest, *signature)
		if err != nil {
			return nil, fmt.Errorf("Invalid signature: %v", err)
		}
	}

	// update our rsrcs entry map
	rsrc.lastPeriod = period
	rsrc.version = version
	rsrc.updated = time.Now()
	rsrc.data = make([]byte, len(data))
	copy(rsrc.data, data)
	log.Debug("Resource synced", "name", *rsrc.name, "key", chunk.Key, "period", rsrc.lastPeriod, "version", rsrc.version)
	self.setResource(*rsrc.name, rsrc)
	return rsrc, nil
}

// retrieve update metadata from chunk data
// mirrors newUpdateChunk()
func (self *ResourceHandler) parseUpdate(chunkdata []byte) (signature *Signature, period uint32, version uint32, name string, data []byte, err error) {
	cursor := 0

	// omit signatures if we have no validator
	var sigoffset int
	if self.validator != nil {
		signature = &Signature{}
		copy(signature[:], chunkdata[:signatureLength])
		sigoffset = signatureLength
		cursor = sigoffset
	}

	headerlength := binary.LittleEndian.Uint16(chunkdata[cursor : cursor+2])
	if int(headerlength+2) > len(chunkdata) {
		err = fmt.Errorf("Reported header length %d longer than actual data length %d", headerlength, len(chunkdata))
		return
	}
	cursor += 2
	period = binary.LittleEndian.Uint32(chunkdata[cursor : cursor+4])
	cursor += 4
	version = binary.LittleEndian.Uint32(chunkdata[cursor : cursor+4])
	cursor += 4
	namelength := int(headerlength) - cursor + sigoffset + 2
	name = string(chunkdata[cursor : cursor+namelength])
	cursor += namelength
	data = make([]byte, len(chunkdata)-cursor)
	copy(data, chunkdata[cursor:])
	return
}

// Adds an actual data update
//
// Uses the data currently loaded in the resources map entry.
// It is the caller's responsibility to make sure that this data is not stale.
//
// A resource update cannot span chunks, and thus has max length 4096
func (self *ResourceHandler) Update(name string, data []byte) (Key, error) {

	var sigoffset int
	if self.validator != nil {
		sigoffset = signatureLength
	}

	// get the cached information
	rsrc := self.getResource(name)
	if rsrc == nil {
		return nil, fmt.Errorf("Resource object not in index")
	}
	if !rsrc.isSynced() {
		return nil, fmt.Errorf("Resource object not in sync")
	}

	// an update can be only one chunk long
	datalimit := self.chunkSize() - int64(sigoffset-len(name)-8)
	if int64(len(data)) > datalimit {
		return nil, fmt.Errorf("Data overflow: %d / %d bytes", len(data), datalimit)
	}

	// get our blockheight at this time and the next block of the update period
	currentblock, err := self.getBlock()
	if err != nil {
		return nil, err
	}
	nextperiod := getNextPeriod(rsrc.startBlock, currentblock, rsrc.frequency)

	// if we already have an update for this block then increment version
	// (resource object MUST be in sync for version to be correct)
	var version uint32
	if self.hasUpdate(name, nextperiod) {
		version = rsrc.version
	}
	version++

	// calculate the chunk key
	key := self.resourceHash(nextperiod, version, rsrc.nameHash)

	var signature *Signature
	if self.validator != nil {
		// sign the data hash with the key
		digest := self.keyDataHash(key, data)
		sig, err := self.validator.sign(digest)
		if err != nil {
			return nil, err
		}
		signature = &sig

		// get the address of the signer (which also checks that it's a valid signature)
		addr, err := getAddressFromDataSig(digest, *signature)
		if err != nil {
			return nil, fmt.Errorf("Invalid data/signature: %v", err)
		}

		// check if the signer has access to update
		ok, err := self.validator.checkAccess(name, addr)
		if err != nil {
			return nil, err
		} else if !ok {
			return nil, fmt.Errorf("Address %x does not have access to update %s", addr, name)
		}
	}

	chunk := newUpdateChunk(key, signature, nextperiod, version, name, data)

	// send the chunk
	self.Put(chunk)
	log.Trace("resource update", "name", name, "key", key, "currentblock", currentblock, "lastperiod", nextperiod, "version", version, "data", chunk.SData)

	// update our resources map entry and return the new key
	rsrc.lastPeriod = nextperiod
	rsrc.version = version
	rsrc.data = make([]byte, len(data))
	copy(rsrc.data, data)
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

// Calculate the period index (aka major version number) from a given block number
func (self *ResourceHandler) BlockToPeriod(name string, blocknumber uint64) uint32 {
	return getNextPeriod(self.resources[name].startBlock, blocknumber, self.resources[name].frequency)
}

// Calculate the block number from a given period index (aka major version number)
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

// used for chunk keys
func (self *ResourceHandler) resourceHash(period uint32, version uint32, namehash common.Hash) Key {
	// format is: hash(period|version|namehash)
	self.hashLock.Lock()
	defer self.hashLock.Unlock()
	self.hasher.Reset()
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, period)
	self.hasher.Write(b)
	binary.LittleEndian.PutUint32(b, version)
	self.hasher.Write(b)
	self.hasher.Write(namehash[:])
	return self.hasher.Sum(nil)
}

func (self *ResourceHandler) hasUpdate(name string, period uint32) bool {
	if self.resources[name].lastPeriod == period {
		return true
	}
	return false
}

func getAddressFromDataSig(datahash common.Hash, signature Signature) (common.Address, error) {
	pub, err := crypto.SigToPub(datahash.Bytes(), signature[:])
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pub), nil
}

// create an update chunk
func newUpdateChunk(key Key, signature *Signature, period uint32, version uint32, name string, data []byte) *Chunk {

	// no signatures if no validator
	var sigoffset int
	if signature != nil {
		sigoffset = signatureLength
	}

	// prepend version and period to allow reverse lookups
	headerlength := uint16(len(name) + 4 + 4)

	chunk := NewChunk(key, nil)
	chunk.SData = make([]byte, sigoffset+int(headerlength)+2+len(data))

	cursor := 0
	if signature != nil {
		copy(chunk.SData, (*signature)[:])
		cursor += signatureLength
	}

	// data header length does NOT include the header length prefix bytes themselves
	binary.LittleEndian.PutUint16(chunk.SData[cursor:], headerlength)
	cursor += 2

	binary.LittleEndian.PutUint32(chunk.SData[cursor:], period)
	cursor += 4

	binary.LittleEndian.PutUint32(chunk.SData[cursor:], version)
	cursor += 4

	namebytes := []byte(name)
	copy(chunk.SData[cursor:], namebytes)
	cursor += len(namebytes)

	copy(chunk.SData[cursor:], data)

	chunk.Size = int64(len(chunk.SData))
	return chunk
}

// \TODO chunkSize is a workaround until the ChunkStore interface exports a method to get the chunk size directly
type resourceChunkStore struct {
	localStore ChunkStore
	netStore   ChunkStore
	chunkSize  int64
}

func newResourceChunkStore(path string, hasher SwarmHasher, localStore *LocalStore, cloudStore CloudStore) *resourceChunkStore {
	return &resourceChunkStore{
		localStore: localStore,
		netStore:   NewNetStore(hasher, localStore, cloudStore, NewDefaultStoreParams()),
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

func getNextPeriod(start uint64, current uint64, frequency uint64) uint32 {
	blockdiff := current - start
	period := blockdiff / frequency
	return uint32(period + 1)
}

func ToSafeName(name string) (string, error) {
	validname, err := idna.ToASCII(name)
	if err != nil {
		return "", err
	}
	return validname, nil
}

// check that name identifiers contain valid bytes
func isSafeName(name string) bool {
	if name == "" {
		return false
	}
	validname, err := idna.ToASCII(name)
	if err != nil {
		return false
	}
	if validname != name {
		return false
	}
	return true
}

// convenience for creating signature hashes of update data
func (self *ResourceHandler) keyDataHash(key Key, data []byte) common.Hash {
	self.hashLock.Lock()
	defer self.hashLock.Unlock()
	self.hasher.Reset()
	self.hasher.Write(key[:])
	self.hasher.Write(data)
	return common.BytesToHash(self.hasher.Sum(nil))
}
