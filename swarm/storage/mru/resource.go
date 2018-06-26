// Copyright 2018 The go-ethereum Authors
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

package mru

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/net/idna"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/multihash"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	signatureLength         = 65
	metadataChunkOffsetSize = 18
	DbDirName               = "resource"
	chunkSize               = 4096 // temporary until we implement FileStore in the resourcehandler
	defaultStoreTimeout     = 4000 * time.Millisecond
	hasherCount             = 8
	resourceHash            = storage.SHA3Hash
	defaultRetrieveTimeout  = 100 * time.Millisecond
)

type blockEstimator struct {
	Start   time.Time
	Average time.Duration
}

// TODO: Average must  be adjusted when blockchain connection is present and synced
func NewBlockEstimator() *blockEstimator {
	sampleDate, _ := time.Parse(time.RFC3339, "2018-05-04T20:35:22Z")   // from etherscan.io
	sampleBlock := int64(3169691)                                       // from etherscan.io
	ropstenStart, _ := time.Parse(time.RFC3339, "2016-11-20T11:48:50Z") // from etherscan.io
	ns := sampleDate.Sub(ropstenStart).Nanoseconds()
	period := int(ns / sampleBlock)
	parsestring := fmt.Sprintf("%dns", int(float64(period)*1.0005)) // increase the blockcount a little, so we don't overshoot the read block height; if we do, we will never find the updates when getting synced data
	periodNs, _ := time.ParseDuration(parsestring)
	return &blockEstimator{
		Start:   ropstenStart,
		Average: periodNs,
	}
}

func (b *blockEstimator) HeaderByNumber(context.Context, string, *big.Int) (*types.Header, error) {
	return &types.Header{
		Number: big.NewInt(time.Since(b.Start).Nanoseconds() / b.Average.Nanoseconds()),
	}, nil
}

type Error struct {
	code int
	err  string
}

func (e *Error) Error() string {
	return e.err
}

func (e *Error) Code() int {
	return e.code
}

func NewError(code int, s string) error {
	if code < 0 || code >= ErrCnt {
		panic("no such error code!")
	}
	r := &Error{
		err: s,
	}
	switch code {
	case ErrNotFound, ErrIO, ErrUnauthorized, ErrInvalidValue, ErrDataOverflow, ErrNothingToReturn, ErrInvalidSignature, ErrNotSynced, ErrPeriodDepth, ErrCorruptData:
		r.code = code
	}
	return r
}

type Signature [signatureLength]byte

type LookupParams struct {
	Limit bool
	Max   uint32
}

// Encapsulates an specific resource update. When synced it contains the most recent
// version of the resource update data.
type resource struct {
	*bytes.Reader
	Multihash  bool
	name       string
	nameHash   common.Hash
	startBlock uint64
	lastPeriod uint32
	lastKey    storage.Address
	frequency  uint64
	version    uint32
	data       []byte
	updated    time.Time
}

// TODO Expire content after a defined period (to force resync)
func (r *resource) isSynced() bool {
	return !r.updated.IsZero()
}

func (r *resource) NameHash() common.Hash {
	return r.nameHash
}

func (r *resource) Size(chan bool) (int64, error) {
	if !r.isSynced() {
		return 0, NewError(ErrNotSynced, "Not synced")
	}
	return int64(len(r.data)), nil
}

func (r *resource) Name() string {
	return r.name
}

func (r *resource) UnmarshalBinary(data []byte) error {
	r.startBlock = binary.LittleEndian.Uint64(data[:8])
	r.frequency = binary.LittleEndian.Uint64(data[8:16])
	r.name = string(data[16:])
	return nil
}

func (r *resource) MarshalBinary() ([]byte, error) {
	b := make([]byte, 16+len(r.name))
	binary.LittleEndian.PutUint64(b, r.startBlock)
	binary.LittleEndian.PutUint64(b[8:], r.frequency)
	copy(b[16:], []byte(r.name))
	return b, nil
}

type headerGetter interface {
	HeaderByNumber(context.Context, string, *big.Int) (*types.Header, error)
}

type ownerValidator interface {
	ValidateOwner(name string, address common.Address) (bool, error)
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
// total of 16 bytes). It also contains the unique identifier.
// It is stored in a separate content-addressed chunk (call it the metadata chunk),
// with the following layout:
//
// (0x0000|startblock|frequency|identifier)
//
// (The two first zero-value bytes are used for disambiguation by the chunk validator,
// and update chunk will always have a value > 0 there.)
//
// The root entry tells the requester from when the mutable resource was
// first added (block number) and in which block number to look for the
// actual updates. Thus, a resource update for identifier "føø.bar"
// starting at block 4200 with frequency 42 will have updates on block 4242,
// 4284, 4326 and so on.
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
// resourcedata|sign(resourcedata)
// otherwise, the chunk data is the same as the resourcedata
//
// headerlength is a 16 bit value containing the byte length of period|version|name
//
// TODO: Include modtime in chunk data + signature
type Handler struct {
	chunkStore      *storage.NetStore
	HashSize        int
	signer          Signer
	headerGetter    headerGetter
	ownerValidator  ownerValidator
	resources       map[string]*resource
	hashPool        sync.Pool
	resourceLock    sync.RWMutex
	storeTimeout    time.Duration
	queryMaxPeriods *LookupParams
}

type HandlerParams struct {
	QueryMaxPeriods *LookupParams
	Signer          Signer
	HeaderGetter    headerGetter
	OwnerValidator  ownerValidator
}

// Create or open resource update chunk store
func NewHandler(params *HandlerParams) (*Handler, error) {
	if params.QueryMaxPeriods == nil {
		params.QueryMaxPeriods = &LookupParams{
			Limit: false,
		}
	}
	rh := &Handler{
		headerGetter:   params.HeaderGetter,
		ownerValidator: params.OwnerValidator,
		resources:      make(map[string]*resource),
		storeTimeout:   defaultStoreTimeout,
		signer:         params.Signer,
		hashPool: sync.Pool{
			New: func() interface{} {
				return storage.MakeHashFunc(resourceHash)()
			},
		},
		queryMaxPeriods: params.QueryMaxPeriods,
	}

	for i := 0; i < hasherCount; i++ {
		hashfunc := storage.MakeHashFunc(resourceHash)()
		if rh.HashSize == 0 {
			rh.HashSize = hashfunc.Size()
		}
		rh.hashPool.Put(hashfunc)
	}

	return rh, nil
}

// SetStore sets the store backend for resource updates
func (h *Handler) SetStore(store *storage.NetStore) {
	h.chunkStore = store
}

// Validate is a chunk validation method (matches ChunkValidatorFunc signature)
//
// If resource update, owner is checked against ENS record of resource name inferred from chunk data
// If parsed signature is nil, validates automatically
// If not resource update, it validates are root chunk if length is metadataChunkOffsetSize and first two bytes are 0
func (h *Handler) Validate(addr storage.Address, data []byte) bool {
	signature, period, version, name, parseddata, _, err := h.parseUpdate(data)
	if err != nil {
		log.Warn(err.Error())
		if len(data) > metadataChunkOffsetSize { // identifier comes after this byte range, and must be at least one byte
			if bytes.Equal(data[:2], []byte{0, 0}) {
				return true
			}
		}
		log.Error("Invalid resource chunk")
		return false
	} else if signature == nil {
		return bytes.Equal(h.resourceHash(period, version, ens.EnsNode(name)), addr)
	}

	digest := h.keyDataHash(addr, parseddata)
	addrSig, err := getAddressFromDataSig(digest, *signature)
	if err != nil {
		log.Error("Invalid signature on resource chunk")
		return false
	}
	ok, _ := h.checkAccess(name, addrSig)
	return ok
}

// If no ens client is supplied, resource updates are not validated
func (h *Handler) IsValidated() bool {
	return h.ownerValidator != nil
}

// Create the resource update digest used in signatures
func (h *Handler) keyDataHash(addr storage.Address, data []byte) common.Hash {
	hasher := h.hashPool.Get().(storage.SwarmHash)
	defer h.hashPool.Put(hasher)
	hasher.Reset()
	hasher.Write(addr[:])
	hasher.Write(data)
	return common.BytesToHash(hasher.Sum(nil))
}

// Checks if current address matches owner address of ENS
func (h *Handler) checkAccess(name string, address common.Address) (bool, error) {
	if h.ownerValidator == nil {
		return true, nil
	}
	return h.ownerValidator.ValidateOwner(name, address)
}

// get data from current resource
func (h *Handler) GetContent(name string) (storage.Address, []byte, error) {
	rsrc := h.get(name)
	if rsrc == nil || !rsrc.isSynced() {
		return nil, nil, NewError(ErrNotFound, " does not exist or is not synced")
	}
	return rsrc.lastKey, rsrc.data, nil
}

// Gets the period of the current data loaded in the resource
func (h *Handler) GetLastPeriod(nameHash string) (uint32, error) {
	rsrc := h.get(nameHash)
	if rsrc == nil {
		return 0, NewError(ErrNotFound, " does not exist")
	} else if !rsrc.isSynced() {
		return 0, NewError(ErrNotSynced, " is not synced")
	}
	return rsrc.lastPeriod, nil
}

// Gets the version of the current data loaded in the resource
func (h *Handler) GetVersion(nameHash string) (uint32, error) {
	rsrc := h.get(nameHash)
	if rsrc == nil {
		return 0, NewError(ErrNotFound, " does not exist")
	} else if !rsrc.isSynced() {
		return 0, NewError(ErrNotSynced, " is not synced")
	}
	return rsrc.version, nil
}

// \TODO should be hashsize * branches from the chosen chunker, implement with FileStore
func (h *Handler) chunkSize() int64 {
	return chunkSize
}

// Creates a new root entry for a mutable resource identified by `name` with the specified `frequency`.
//
// The signature data should match the hash of the idna-converted name by the validator's namehash function, NOT the raw name bytes.
//
// The start block of the resource update will be the actual current block height of the connected network.
func (h *Handler) New(ctx context.Context, name string, frequency uint64) (storage.Address, *resource, error) {

	// frequency 0 is invalid
	if frequency == 0 {
		return nil, nil, NewError(ErrInvalidValue, "Frequency cannot be 0")
	}

	// make sure name only contains ascii values
	if !isSafeName(name) {
		return nil, nil, NewError(ErrInvalidValue, fmt.Sprintf("Invalid name: '%s'", name))
	}

	nameHash := ens.EnsNode(name)

	// if the signer function is set, validate that the key of the signer has access to modify this ENS name
	if h.signer != nil {
		signature, err := h.signer.Sign(nameHash)
		if err != nil {
			return nil, nil, NewError(ErrInvalidSignature, fmt.Sprintf("Sign fail: %v", err))
		}
		addr, err := getAddressFromDataSig(nameHash, signature)
		if err != nil {
			return nil, nil, NewError(ErrInvalidSignature, fmt.Sprintf("Retrieve address from signature fail: %v", err))
		}
		ok, err := h.checkAccess(name, addr)
		if err != nil {
			return nil, nil, err
		} else if !ok {
			return nil, nil, NewError(ErrUnauthorized, fmt.Sprintf("Not owner of '%s'", name))
		}
	}

	// get our blockheight at this time
	currentblock, err := h.getBlock(ctx, name)
	if err != nil {
		return nil, nil, err
	}

	chunk := h.newMetaChunk(name, currentblock, frequency)

	h.chunkStore.Put(chunk)
	log.Debug("new resource", "name", name, "key", nameHash, "startBlock", currentblock, "frequency", frequency)

	// create the internal index for the resource and populate it with the data of the first version
	rsrc := &resource{
		startBlock: currentblock,
		frequency:  frequency,
		name:       name,
		nameHash:   nameHash,
		updated:    time.Now(),
	}
	h.set(nameHash.Hex(), rsrc)

	return chunk.Addr, rsrc, nil
}

func (h *Handler) newMetaChunk(name string, startBlock uint64, frequency uint64) *storage.Chunk {
	// the metadata chunk points to data of first blockheight + update frequency
	// from this we know from what blockheight we should look for updates, and how often
	// it also contains the name of the resource, so we know what resource we are working with
	data := make([]byte, metadataChunkOffsetSize+len(name))

	// root block has first two bytes both set to 0, which distinguishes from update bytes
	val := make([]byte, 8)
	binary.LittleEndian.PutUint64(val, startBlock)
	copy(data[2:10], val)
	binary.LittleEndian.PutUint64(val, frequency)
	copy(data[10:18], val)
	copy(data[18:], []byte(name))

	// the key of the metadata chunk is content-addressed
	// if it wasn't we couldn't replace it later
	// resolving this relationship is left up to external agents (for example ENS)
	hasher := h.hashPool.Get().(storage.SwarmHash)
	hasher.Reset()
	hasher.Write(data)
	key := hasher.Sum(nil)
	h.hashPool.Put(hasher)

	// make the chunk and send it to swarm
	chunk := storage.NewChunk(key, nil)
	chunk.SData = make([]byte, metadataChunkOffsetSize+len(name))
	copy(chunk.SData, data)
	return chunk
}

// Searches and retrieves the specific version of the resource update identified by `name`
// at the specific block height
//
// If refresh is set to true, the resource data will be reloaded from the resource update
// metadata chunk.
// It is the callers responsibility to make sure that this chunk exists (if the resource
// update root data was retrieved externally, it typically doesn't)
func (h *Handler) LookupVersionByName(ctx context.Context, name string, period uint32, version uint32, refresh bool, maxLookup *LookupParams) (*resource, error) {
	return h.LookupVersion(ctx, ens.EnsNode(name), period, version, refresh, maxLookup)
}

func (h *Handler) LookupVersion(ctx context.Context, nameHash common.Hash, period uint32, version uint32, refresh bool, maxLookup *LookupParams) (*resource, error) {
	rsrc := h.get(nameHash.Hex())
	if rsrc == nil {
		return nil, NewError(ErrNothingToReturn, "resource not loaded")
	}
	return h.lookup(rsrc, period, version, refresh, maxLookup)
}

// Retrieves the latest version of the resource update identified by `name`
// at the specified block height
//
// If an update is found, version numbers are iterated until failure, and the last
// successfully retrieved version is copied to the corresponding resources map entry
// and returned.
//
// See also (*Handler).LookupVersion
func (h *Handler) LookupHistoricalByName(ctx context.Context, name string, period uint32, refresh bool, maxLookup *LookupParams) (*resource, error) {
	return h.LookupHistorical(ctx, ens.EnsNode(name), period, refresh, maxLookup)
}

func (h *Handler) LookupHistorical(ctx context.Context, nameHash common.Hash, period uint32, refresh bool, maxLookup *LookupParams) (*resource, error) {
	rsrc := h.get(nameHash.Hex())
	if rsrc == nil {
		return nil, NewError(ErrNothingToReturn, "resource not loaded")
	}
	return h.lookup(rsrc, period, 0, refresh, maxLookup)
}

// Retrieves the latest version of the resource update identified by `name`
// at the next update block height
//
// It starts at the next period after the current block height, and upon failure
// tries the corresponding keys of each previous period until one is found
// (or startBlock is reached, in which case there are no updates).
//
// Version iteration is done as in (*Handler).LookupHistorical
//
// See also (*Handler).LookupHistorical
func (h *Handler) LookupLatestByName(ctx context.Context, name string, refresh bool, maxLookup *LookupParams) (*resource, error) {
	return h.LookupLatest(ctx, ens.EnsNode(name), refresh, maxLookup)
}

func (h *Handler) LookupLatest(ctx context.Context, nameHash common.Hash, refresh bool, maxLookup *LookupParams) (*resource, error) {

	// get our blockheight at this time and the next block of the update period
	rsrc := h.get(nameHash.Hex())
	if rsrc == nil {
		return nil, NewError(ErrNothingToReturn, "resource not loaded")
	}
	currentblock, err := h.getBlock(ctx, rsrc.name)
	if err != nil {
		return nil, err
	}
	nextperiod, err := getNextPeriod(rsrc.startBlock, currentblock, rsrc.frequency)
	if err != nil {
		return nil, err
	}
	return h.lookup(rsrc, nextperiod, 0, refresh, maxLookup)
}

// Returns the resource before the one currently loaded in the resource index
//
// This is useful where resource updates are used incrementally in contrast to
// merely replacing content.
//
// Requires a synced resource object
func (h *Handler) LookupPreviousByName(ctx context.Context, name string, maxLookup *LookupParams) (*resource, error) {
	return h.LookupPrevious(ctx, ens.EnsNode(name), maxLookup)
}

func (h *Handler) LookupPrevious(ctx context.Context, nameHash common.Hash, maxLookup *LookupParams) (*resource, error) {
	rsrc := h.get(nameHash.Hex())
	if rsrc == nil {
		return nil, NewError(ErrNothingToReturn, "resource not loaded")
	}
	if !rsrc.isSynced() {
		return nil, NewError(ErrNotSynced, "LookupPrevious requires synced resource.")
	} else if rsrc.lastPeriod == 0 {
		return nil, NewError(ErrNothingToReturn, " not found")
	}
	if rsrc.version > 1 {
		rsrc.version--
	} else if rsrc.lastPeriod == 1 {
		return nil, NewError(ErrNothingToReturn, "Current update is the oldest")
	} else {
		rsrc.version = 0
		rsrc.lastPeriod--
	}
	return h.lookup(rsrc, rsrc.lastPeriod, rsrc.version, false, maxLookup)
}

// base code for public lookup methods
func (h *Handler) lookup(rsrc *resource, period uint32, version uint32, refresh bool, maxLookup *LookupParams) (*resource, error) {

	// we can't look for anything without a store
	if h.chunkStore == nil {
		return nil, NewError(ErrInit, "Call Handler.SetStore() before performing lookups")
	}

	// period 0 does not exist
	if period == 0 {
		return nil, NewError(ErrInvalidValue, "period must be >0")
	}

	// start from the last possible block period, and iterate previous ones until we find a match
	// if we hit startBlock we're out of options
	var specificversion bool
	if version > 0 {
		specificversion = true
	} else {
		version = 1
	}

	var hops uint32
	if maxLookup == nil {
		maxLookup = h.queryMaxPeriods
	}
	log.Trace("resource lookup", "period", period, "version", version, "limit", maxLookup.Limit, "max", maxLookup.Max)
	for period > 0 {
		if maxLookup.Limit && hops > maxLookup.Max {
			return nil, NewError(ErrPeriodDepth, fmt.Sprintf("Lookup exceeded max period hops (%d)", maxLookup.Max))
		}
		key := h.resourceHash(period, version, rsrc.nameHash)
		chunk, err := h.chunkStore.GetWithTimeout(key, defaultRetrieveTimeout)
		if err == nil {
			if specificversion {
				return h.updateIndex(rsrc, chunk)
			}
			// check if we have versions > 1. If a version fails, the previous version is used and returned.
			log.Trace("rsrc update version 1 found, checking for version updates", "period", period, "key", key)
			for {
				newversion := version + 1
				key := h.resourceHash(period, newversion, rsrc.nameHash)
				newchunk, err := h.chunkStore.GetWithTimeout(key, defaultRetrieveTimeout)
				if err != nil {
					return h.updateIndex(rsrc, chunk)
				}
				chunk = newchunk
				version = newversion
				log.Trace("version update found, checking next", "version", version, "period", period, "key", key)
			}
		}
		log.Trace("rsrc update not found, checking previous period", "period", period, "key", key)
		period--
		hops++
	}
	return nil, NewError(ErrNotFound, "no updates found")
}

// Retrieves a resource metadata chunk and creates/updates the index entry for it
// with the resulting metadata
func (h *Handler) Load(addr storage.Address) (*resource, error) {
	chunk, err := h.chunkStore.GetWithTimeout(addr, defaultRetrieveTimeout)
	if err != nil {
		return nil, NewError(ErrNotFound, err.Error())
	}

	// minimum sanity check for chunk data (an update chunk first two bytes is headerlength uint16, and cannot be 0)
	// \TODO this is not enough to make sure the data isn't bogus. A normal content addressed chunk could still satisfy these criteria
	if !bytes.Equal(chunk.SData[:2], []byte{0x0, 0x0}) {
		return nil, NewError(ErrCorruptData, fmt.Sprintf("Chunk is not a resource metadata chunk"))
	} else if len(chunk.SData) <= metadataChunkOffsetSize {
		return nil, NewError(ErrNothingToReturn, fmt.Sprintf("Invalid chunk length %d, should be minimum %d", len(chunk.SData), metadataChunkOffsetSize+1))
	}

	// create the index entry
	rsrc := &resource{}
	rsrc.UnmarshalBinary(chunk.SData[2:])
	rsrc.nameHash = ens.EnsNode(rsrc.name)
	h.set(rsrc.nameHash.Hex(), rsrc)
	log.Trace("resource index load", "rootkey", addr, "name", rsrc.name, "namehash", rsrc.nameHash, "startblock", rsrc.startBlock, "frequency", rsrc.frequency)
	return rsrc, nil
}

// update mutable resource index map with specified content
func (h *Handler) updateIndex(rsrc *resource, chunk *storage.Chunk) (*resource, error) {

	// retrieve metadata from chunk data and check that it matches this mutable resource
	signature, period, version, name, data, multihash, err := h.parseUpdate(chunk.SData)
	if rsrc.name != name {
		return nil, NewError(ErrNothingToReturn, fmt.Sprintf("Update belongs to '%s', but have '%s'", name, rsrc.name))
	}
	log.Trace("resource index update", "name", rsrc.name, "namehash", rsrc.nameHash, "updatekey", chunk.Addr, "period", period, "version", version)

	// check signature (if signer algorithm is present)
	// \TODO maybe this check is redundant if also checked upon retrieval of chunk
	if signature != nil {
		digest := h.keyDataHash(chunk.Addr, data)
		_, err = getAddressFromDataSig(digest, *signature)
		if err != nil {
			return nil, NewError(ErrUnauthorized, fmt.Sprintf("Invalid signature: %v", err))
		}
	}

	// update our rsrcs entry map
	rsrc.lastKey = chunk.Addr
	rsrc.lastPeriod = period
	rsrc.version = version
	rsrc.updated = time.Now()
	rsrc.data = make([]byte, len(data))
	rsrc.Multihash = multihash
	rsrc.Reader = bytes.NewReader(rsrc.data)
	copy(rsrc.data, data)
	log.Debug(" synced", "name", rsrc.name, "key", chunk.Addr, "period", rsrc.lastPeriod, "version", rsrc.version)
	h.set(rsrc.nameHash.Hex(), rsrc)
	return rsrc, nil
}

// retrieve update metadata from chunk data
// mirrors newUpdateChunk()
func (h *Handler) parseUpdate(chunkdata []byte) (*Signature, uint32, uint32, string, []byte, bool, error) {
	// absolute minimum an update chunk can contain:
	// 14 = header + one byte of name + one byte of data
	if len(chunkdata) < 14 {
		return nil, 0, 0, "", nil, false, NewError(ErrNothingToReturn, "chunk less than 13 bytes cannot be a resource update chunk")
	}
	cursor := 0
	headerlength := binary.LittleEndian.Uint16(chunkdata[cursor : cursor+2])
	cursor += 2
	datalength := binary.LittleEndian.Uint16(chunkdata[cursor : cursor+2])
	cursor += 2
	var exclsignlength int
	// we need extra magic if it's a multihash, since we used datalength 0 in header as an indicator of multihash content
	// retrieve the second varint and set this as the data length
	// TODO: merge with isMultihash code
	if datalength == 0 {
		uvarintbuf := bytes.NewBuffer(chunkdata[headerlength+4:])
		r, err := binary.ReadUvarint(uvarintbuf)
		if err != nil {
			errstr := fmt.Sprintf("corrupt multihash, hash id varint could not be read: %v", err)
			log.Warn(errstr)
			return nil, 0, 0, "", nil, false, NewError(ErrCorruptData, errstr)

		}
		r, err = binary.ReadUvarint(uvarintbuf)
		if err != nil {
			errstr := fmt.Sprintf("corrupt multihash, hash length field could not be read: %v", err)
			log.Warn(errstr)
			return nil, 0, 0, "", nil, false, NewError(ErrCorruptData, errstr)

		}
		exclsignlength = int(headerlength + uint16(r))
	} else {
		exclsignlength = int(headerlength + datalength + 4)
	}

	// the total length excluding signature is headerlength and datalength fields plus the length of the header and the data given in these fields
	exclsignlength = int(headerlength + datalength + 4)
	if exclsignlength > len(chunkdata) || exclsignlength < 14 {
		return nil, 0, 0, "", nil, false, NewError(ErrNothingToReturn, fmt.Sprintf("Reported headerlength %d + datalength %d longer than actual chunk data length %d", headerlength, exclsignlength, len(chunkdata)))
	} else if exclsignlength < 14 {
		return nil, 0, 0, "", nil, false, NewError(ErrNothingToReturn, fmt.Sprintf("Reported headerlength %d + datalength %d is smaller than minimum valid resource chunk length %d", headerlength, datalength, 14))
	}

	// at this point we can be satisfied that the data integrity is ok
	var period uint32
	var version uint32
	var name string
	var data []byte
	period = binary.LittleEndian.Uint32(chunkdata[cursor : cursor+4])
	cursor += 4
	version = binary.LittleEndian.Uint32(chunkdata[cursor : cursor+4])
	cursor += 4
	namelength := int(headerlength) - cursor + 4
	if l := len(chunkdata); l < cursor+namelength {
		return nil, 0, 0, "", nil, false, NewError(ErrNothingToReturn, fmt.Sprintf("chunk less than %v bytes is too short to read the name", l))
	}
	name = string(chunkdata[cursor : cursor+namelength])
	cursor += namelength

	// if multihash content is indicated we check the validity of the multihash
	// \TODO the check above for multihash probably is sufficient also for this case (or can be with a small adjustment) and if so this code should be removed
	var intdatalength int
	var ismultihash bool
	if datalength == 0 {
		var intheaderlength int
		var err error
		intdatalength, intheaderlength, err = multihash.GetMultihashLength(chunkdata[cursor:])
		if err != nil {
			log.Error("multihash parse error", "err", err)
			return nil, 0, 0, "", nil, false, err
		}
		intdatalength += intheaderlength
		multihashboundary := cursor + intdatalength
		if len(chunkdata) != multihashboundary && len(chunkdata) < multihashboundary+signatureLength {
			log.Debug("multihash error", "chunkdatalen", len(chunkdata), "multihashboundary", multihashboundary)
			return nil, 0, 0, "", nil, false, errors.New("Corrupt multihash data")
		}
		ismultihash = true
	} else {
		intdatalength = int(datalength)
	}
	data = make([]byte, intdatalength)
	copy(data, chunkdata[cursor:cursor+intdatalength])

	// omit signatures if we have no validator
	var signature *Signature
	cursor += intdatalength
	if h.signer != nil {
		sigdata := chunkdata[cursor : cursor+signatureLength]
		if len(sigdata) > 0 {
			signature = &Signature{}
			copy(signature[:], sigdata)
		}
	}

	return signature, period, version, name, data, ismultihash, nil
}

// Adds an actual data update
//
// Uses the data currently loaded in the resources map entry.
// It is the caller's responsibility to make sure that this data is not stale.
//
// A resource update cannot span chunks, and thus has max length 4096
func (h *Handler) UpdateMultihash(ctx context.Context, name string, data []byte) (storage.Address, error) {
	// \TODO perhaps this check should be in newUpdateChunk()
	if _, _, err := multihash.GetMultihashLength(data); err != nil {
		return nil, NewError(ErrNothingToReturn, err.Error())
	}
	return h.update(ctx, name, data, true)
}

func (h *Handler) Update(ctx context.Context, name string, data []byte) (storage.Address, error) {
	return h.update(ctx, name, data, false)
}

// create and commit an update
func (h *Handler) update(ctx context.Context, name string, data []byte, multihash bool) (storage.Address, error) {

	// zero-length updates are bogus
	if len(data) == 0 {
		return nil, NewError(ErrInvalidValue, "I refuse to waste swarm space for updates with empty values, amigo (data length is 0)")
	}

	// we can't update anything without a store
	if h.chunkStore == nil {
		return nil, NewError(ErrInit, "Call Handler.SetStore() before updating")
	}

	// signature length is 0 if we are not using them
	var signaturelength int
	if h.signer != nil {
		signaturelength = signatureLength
	}

	// get the cached information
	nameHash := ens.EnsNode(name)
	nameHashHex := nameHash.Hex()
	rsrc := h.get(nameHashHex)
	if rsrc == nil {
		return nil, NewError(ErrNotFound, fmt.Sprintf(" object '%s' not in index", name))
	} else if !rsrc.isSynced() {
		return nil, NewError(ErrNotSynced, " object not in sync")
	}

	// an update can be only one chunk long; data length less header and signature data
	// 12 = length of header and data length fields (2xuint16) plus period and frequency value fields (2xuint32)
	datalimit := h.chunkSize() - int64(signaturelength-len(name)-12)
	if int64(len(data)) > datalimit {
		return nil, NewError(ErrDataOverflow, fmt.Sprintf("Data overflow: %d / %d bytes", len(data), datalimit))
	}

	// get our blockheight at this time and the next block of the update period
	currentblock, err := h.getBlock(ctx, name)
	if err != nil {
		return nil, NewError(ErrIO, fmt.Sprintf("Could not get block height: %v", err))
	}
	nextperiod, err := getNextPeriod(rsrc.startBlock, currentblock, rsrc.frequency)
	if err != nil {
		return nil, err
	}

	// if we already have an update for this block then increment version
	// resource object MUST be in sync for version to be correct, but we checked this earlier in the method already
	var version uint32
	if h.hasUpdate(nameHashHex, nextperiod) {
		version = rsrc.version
	}
	version++

	// calculate the chunk key
	key := h.resourceHash(nextperiod, version, rsrc.nameHash)

	// if we have a signing function, sign the update
	// \TODO this code should probably be consolidated with corresponding code in New()
	var signature *Signature
	if h.signer != nil {
		// sign the data hash with the key
		digest := h.keyDataHash(key, data)
		sig, err := h.signer.Sign(digest)
		if err != nil {
			return nil, NewError(ErrInvalidSignature, fmt.Sprintf("Sign fail: %v", err))
		}
		signature = &sig

		// get the address of the signer (which also checks that it's a valid signature)
		addr, err := getAddressFromDataSig(digest, *signature)
		if err != nil {
			return nil, NewError(ErrInvalidSignature, fmt.Sprintf("Invalid data/signature: %v", err))
		}
		if h.signer != nil {
			// check if the signer has access to update
			ok, err := h.checkAccess(name, addr)
			if err != nil {
				return nil, NewError(ErrIO, fmt.Sprintf("Access check fail: %v", err))
			} else if !ok {
				return nil, NewError(ErrUnauthorized, fmt.Sprintf("Address %x does not have access to update %s", addr, name))
			}
		}
	}

	// a datalength field set to 0 means the content is a multihash
	var datalength int
	if !multihash {
		datalength = len(data)
	}
	chunk := newUpdateChunk(key, signature, nextperiod, version, name, data, datalength)

	// send the chunk
	h.chunkStore.Put(chunk)
	log.Trace("resource update", "name", name, "key", key, "currentblock", currentblock, "lastperiod", nextperiod, "version", version, "data", chunk.SData, "multihash", multihash)

	// update our resources map entry and return the new key
	rsrc.lastPeriod = nextperiod
	rsrc.version = version
	rsrc.data = make([]byte, len(data))
	copy(rsrc.data, data)
	return key, nil
}

// Closes the datastore.
// Always call this at shutdown to avoid data corruption.
func (h *Handler) Close() {
	h.chunkStore.Close()
}

// gets the current block height
func (h *Handler) getBlock(ctx context.Context, name string) (uint64, error) {
	blockheader, err := h.headerGetter.HeaderByNumber(ctx, name, nil)
	if err != nil {
		return 0, err
	}
	return blockheader.Number.Uint64(), nil
}

// Calculate the period index (aka major version number) from a given block number
func (h *Handler) BlockToPeriod(name string, blocknumber uint64) (uint32, error) {
	return getNextPeriod(h.resources[name].startBlock, blocknumber, h.resources[name].frequency)
}

// Calculate the block number from a given period index (aka major version number)
func (h *Handler) PeriodToBlock(name string, period uint32) uint64 {
	return h.resources[name].startBlock + (uint64(period) * h.resources[name].frequency)
}

// Retrieves the resource index value for the given nameHash
func (h *Handler) get(nameHash string) *resource {
	h.resourceLock.RLock()
	defer h.resourceLock.RUnlock()
	rsrc := h.resources[nameHash]
	return rsrc
}

// Sets the resource index value for the given nameHash
func (h *Handler) set(nameHash string, rsrc *resource) {
	h.resourceLock.Lock()
	defer h.resourceLock.Unlock()
	h.resources[nameHash] = rsrc
}

// used for chunk keys
func (h *Handler) resourceHash(period uint32, version uint32, namehash common.Hash) storage.Address {
	// format is: hash(period|version|namehash)
	hasher := h.hashPool.Get().(storage.SwarmHash)
	defer h.hashPool.Put(hasher)
	hasher.Reset()
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, period)
	hasher.Write(b)
	binary.LittleEndian.PutUint32(b, version)
	hasher.Write(b)
	hasher.Write(namehash[:])
	return hasher.Sum(nil)
}

// Checks if we already have an update on this resource, according to the value in the current state of the resource index
func (h *Handler) hasUpdate(nameHash string, period uint32) bool {
	return h.resources[nameHash].lastPeriod == period
}

func getAddressFromDataSig(datahash common.Hash, signature Signature) (common.Address, error) {
	pub, err := crypto.SigToPub(datahash.Bytes(), signature[:])
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pub), nil
}

// create an update chunk
func newUpdateChunk(addr storage.Address, signature *Signature, period uint32, version uint32, name string, data []byte, datalength int) *storage.Chunk {

	// no signatures if no validator
	var signaturelength int
	if signature != nil {
		signaturelength = signatureLength
	}

	// prepend version and period to allow reverse lookups
	headerlength := len(name) + 4 + 4

	actualdatalength := len(data)
	chunk := storage.NewChunk(addr, nil)
	chunk.SData = make([]byte, 4+signaturelength+headerlength+actualdatalength) // initial 4 are uint16 length descriptors for headerlength and datalength

	// data header length does NOT include the header length prefix bytes themselves
	cursor := 0
	binary.LittleEndian.PutUint16(chunk.SData[cursor:], uint16(headerlength))
	cursor += 2

	// data length
	binary.LittleEndian.PutUint16(chunk.SData[cursor:], uint16(datalength))
	cursor += 2

	// header = period + version + name
	binary.LittleEndian.PutUint32(chunk.SData[cursor:], period)
	cursor += 4

	binary.LittleEndian.PutUint32(chunk.SData[cursor:], version)
	cursor += 4

	namebytes := []byte(name)
	copy(chunk.SData[cursor:], namebytes)
	cursor += len(namebytes)

	// add the data
	copy(chunk.SData[cursor:], data)

	// if signature is present it's the last item in the chunk data
	if signature != nil {
		cursor += actualdatalength
		copy(chunk.SData[cursor:], signature[:])
	}

	chunk.Size = int64(len(chunk.SData))
	return chunk
}

// Helper function to calculate the next update period number from the current block, start block and frequency
func getNextPeriod(start uint64, current uint64, frequency uint64) (uint32, error) {
	if current < start {
		return 0, NewError(ErrInvalidValue, fmt.Sprintf("given current block value %d < start block %d", current, start))
	}
	blockdiff := current - start
	period := blockdiff / frequency
	return uint32(period + 1), nil
}

// ToSafeName is a helper function to create an valid idna of a given resource update name
func ToSafeName(name string) (string, error) {
	return idna.ToASCII(name)
}

// check that name identifiers contain valid bytes
// Strings created using ToSafeName() should satisfy this check
func isSafeName(name string) bool {
	if name == "" {
		return false
	}
	validname, err := idna.ToASCII(name)
	if err != nil {
		return false
	}
	return validname == name
}

func NewTestHandler(datadir string, params *HandlerParams) (*Handler, error) {
	path := filepath.Join(datadir, DbDirName)
	rh, err := NewHandler(params)
	if err != nil {
		return nil, fmt.Errorf("resource handler create fail: %v", err)
	}
	localstoreparams := storage.NewDefaultLocalStoreParams()
	localstoreparams.Init(path)
	localStore, err := storage.NewLocalStore(localstoreparams, nil)
	if err != nil {
		return nil, fmt.Errorf("localstore create fail, path %s: %v", path, err)
	}
	localStore.Validators = append(localStore.Validators, storage.NewContentAddressValidator(storage.MakeHashFunc(resourceHash)))
	localStore.Validators = append(localStore.Validators, rh)
	netStore := storage.NewNetStore(localStore, nil)
	rh.SetStore(netStore)
	return rh, nil
}
