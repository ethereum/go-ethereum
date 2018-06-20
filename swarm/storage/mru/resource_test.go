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
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/contracts/ens/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/multihash"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var (
	loglevel          = flag.Int("loglevel", 3, "loglevel")
	testHasher        = storage.MakeHashFunc(storage.SHA3Hash)()
	zeroAddr          = common.Address{}
	startBlock        = uint64(4200)
	resourceFrequency = uint64(42)
	cleanF            func()
	domainName        = "føø.bar"
	safeName          string
	nameHash          common.Hash
	hashfunc          = storage.MakeHashFunc(storage.DefaultHash)
)

func init() {
	var err error
	flag.Parse()
	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))
	safeName, err = ToSafeName(domainName)
	if err != nil {
		panic(err)
	}
	nameHash = ens.EnsNode(safeName)
}

// simulated backend does not have the blocknumber call
// so we use this wrapper to fake returning the block count
type fakeBackend struct {
	*backends.SimulatedBackend
	blocknumber int64
}

func (f *fakeBackend) Commit() {
	if f.SimulatedBackend != nil {
		f.SimulatedBackend.Commit()
	}
	f.blocknumber++
}

func (f *fakeBackend) HeaderByNumber(context context.Context, name string, bigblock *big.Int) (*types.Header, error) {
	f.blocknumber++
	biggie := big.NewInt(f.blocknumber)
	return &types.Header{
		Number: biggie,
	}, nil
}

// check that signature address matches update signer address
func TestReverse(t *testing.T) {

	period := uint32(4)
	version := uint32(2)

	// signer containing private key
	signer, err := newTestSigner()
	if err != nil {
		t.Fatal(err)
	}

	// set up rpc and create resourcehandler
	rh, _, teardownTest, err := setupTest(nil, nil, signer)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// generate a hash for block 4200 version 1
	key := rh.resourceHash(period, version, ens.EnsNode(safeName))

	// generate some bogus data for the chunk and sign it
	data := make([]byte, 8)
	_, err = rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	testHasher.Reset()
	testHasher.Write(data)
	digest := rh.keyDataHash(key, data)
	sig, err := rh.signer.Sign(digest)
	if err != nil {
		t.Fatal(err)
	}

	chunk := newUpdateChunk(key, &sig, period, version, safeName, data, len(data))

	// check that we can recover the owner account from the update chunk's signature
	checksig, checkperiod, checkversion, checkname, checkdata, _, err := rh.parseUpdate(chunk.SData)
	if err != nil {
		t.Fatal(err)
	}
	checkdigest := rh.keyDataHash(chunk.Addr, checkdata)
	recoveredaddress, err := getAddressFromDataSig(checkdigest, *checksig)
	if err != nil {
		t.Fatalf("Retrieve address from signature fail: %v", err)
	}
	originaladdress := crypto.PubkeyToAddress(signer.PrivKey.PublicKey)

	// check that the metadata retrieved from the chunk matches what we gave it
	if recoveredaddress != originaladdress {
		t.Fatalf("addresses dont match: %x != %x", originaladdress, recoveredaddress)
	}

	if !bytes.Equal(key[:], chunk.Addr[:]) {
		t.Fatalf("Expected chunk key '%x', was '%x'", key, chunk.Addr)
	}
	if period != checkperiod {
		t.Fatalf("Expected period '%d', was '%d'", period, checkperiod)
	}
	if version != checkversion {
		t.Fatalf("Expected version '%d', was '%d'", version, checkversion)
	}
	if safeName != checkname {
		t.Fatalf("Expected name '%s', was '%s'", safeName, checkname)
	}
	if !bytes.Equal(data, checkdata) {
		t.Fatalf("Expectedn data '%x', was '%x'", data, checkdata)
	}
}

// make updates and retrieve them based on periods and versions
func TestHandler(t *testing.T) {

	// make fake backend, set up rpc and create resourcehandler
	backend := &fakeBackend{
		blocknumber: int64(startBlock),
	}
	rh, datadir, teardownTest, err := setupTest(backend, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// create a new resource
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rootChunkKey, _, err := rh.New(ctx, safeName, resourceFrequency)
	if err != nil {
		t.Fatal(err)
	}

	chunk, err := rh.chunkStore.Get(storage.Address(rootChunkKey))
	if err != nil {
		t.Fatal(err)
	} else if len(chunk.SData) < 16 {
		t.Fatalf("chunk data must be minimum 16 bytes, is %d", len(chunk.SData))
	}
	startblocknumber := binary.LittleEndian.Uint64(chunk.SData[2:10])
	chunkfrequency := binary.LittleEndian.Uint64(chunk.SData[10:])
	if startblocknumber != uint64(backend.blocknumber) {
		t.Fatalf("stored block number %d does not match provided block number %d", startblocknumber, backend.blocknumber)
	}
	if chunkfrequency != resourceFrequency {
		t.Fatalf("stored frequency %d does not match provided frequency %d", chunkfrequency, resourceFrequency)
	}

	// data for updates:
	updates := []string{
		"blinky",
		"pinky",
		"inky",
		"clyde",
	}

	// update halfway to first period
	resourcekey := make(map[string]storage.Address)
	fwdBlocks(int(resourceFrequency/2), backend)
	data := []byte(updates[0])
	resourcekey[updates[0]], err = rh.Update(ctx, safeName, data)
	if err != nil {
		t.Fatal(err)
	}

	// update on first period
	fwdBlocks(int(resourceFrequency/2), backend)
	data = []byte(updates[1])
	resourcekey[updates[1]], err = rh.Update(ctx, safeName, data)
	if err != nil {
		t.Fatal(err)
	}

	// update on second period
	fwdBlocks(int(resourceFrequency), backend)
	data = []byte(updates[2])
	resourcekey[updates[2]], err = rh.Update(ctx, safeName, data)
	if err != nil {
		t.Fatal(err)
	}

	// update just after second period
	fwdBlocks(1, backend)
	data = []byte(updates[3])
	resourcekey[updates[3]], err = rh.Update(ctx, safeName, data)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	rh.Close()

	// check we can retrieve the updates after close
	// it will match on second iteration startblocknumber + (resourceFrequency * 3)
	fwdBlocks(int(resourceFrequency*2)-1, backend)

	rhparams := &HandlerParams{
		QueryMaxPeriods: &LookupParams{
			Limit: false,
		},
		Signer:       nil,
		HeaderGetter: rh.headerGetter,
	}

	rh2, err := NewTestHandler(datadir, rhparams)
	if err != nil {
		t.Fatal(err)
	}
	rsrc2, err := rh2.Load(rootChunkKey)
	_, err = rh2.LookupLatest(ctx, nameHash, true, nil)
	if err != nil {
		t.Fatal(err)
	}

	// last update should be "clyde", version two, blockheight startblocknumber + (resourcefrequency * 3)
	if !bytes.Equal(rsrc2.data, []byte(updates[len(updates)-1])) {
		t.Fatalf("resource data was %v, expected %v", rsrc2.data, updates[len(updates)-1])
	}
	if rsrc2.version != 2 {
		t.Fatalf("resource version was %d, expected 2", rsrc2.version)
	}
	if rsrc2.lastPeriod != 3 {
		t.Fatalf("resource period was %d, expected 3", rsrc2.lastPeriod)
	}
	log.Debug("Latest lookup", "period", rsrc2.lastPeriod, "version", rsrc2.version, "data", rsrc2.data)

	// specific block, latest version
	rsrc, err := rh2.LookupHistorical(ctx, nameHash, 3, true, rh2.queryMaxPeriods)
	if err != nil {
		t.Fatal(err)
	}
	// check data
	if !bytes.Equal(rsrc.data, []byte(updates[len(updates)-1])) {
		t.Fatalf("resource data (historical) was %v, expected %v", rsrc2.data, updates[len(updates)-1])
	}
	log.Debug("Historical lookup", "period", rsrc2.lastPeriod, "version", rsrc2.version, "data", rsrc2.data)

	// specific block, specific version
	rsrc, err = rh2.LookupVersion(ctx, nameHash, 3, 1, true, rh2.queryMaxPeriods)
	if err != nil {
		t.Fatal(err)
	}
	// check data
	if !bytes.Equal(rsrc.data, []byte(updates[2])) {
		t.Fatalf("resource data (historical) was %v, expected %v", rsrc2.data, updates[2])
	}
	log.Debug("Specific version lookup", "period", rsrc2.lastPeriod, "version", rsrc2.version, "data", rsrc2.data)

	// we are now at third update
	// check backwards stepping to the first
	for i := 1; i >= 0; i-- {
		rsrc, err := rh2.LookupPreviousByName(ctx, safeName, rh2.queryMaxPeriods)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(rsrc.data, []byte(updates[i])) {
			t.Fatalf("resource data (previous) was %v, expected %v", rsrc2.data, updates[i])

		}
	}

	// beyond the first should yield an error
	rsrc, err = rh2.LookupPreviousByName(ctx, safeName, rh2.queryMaxPeriods)
	if err == nil {
		t.Fatalf("expeected previous to fail, returned period %d version %d data %v", rsrc2.lastPeriod, rsrc2.version, rsrc2.data)
	}

}

// create ENS enabled resource update, with and without valid owner
func TestENSOwner(t *testing.T) {

	// signer containing private key
	signer, err := newTestSigner()
	if err != nil {
		t.Fatal(err)
	}

	// ens address and transact options
	addr := crypto.PubkeyToAddress(signer.PrivKey.PublicKey)
	transactOpts := bind.NewKeyedTransactor(signer.PrivKey)

	// set up ENS sim
	domainparts := strings.Split(safeName, ".")
	contractAddr, contractbackend, err := setupENS(addr, transactOpts, domainparts[0], domainparts[1])
	if err != nil {
		t.Fatal(err)
	}

	ensClient, err := ens.NewENS(transactOpts, contractAddr, contractbackend)
	if err != nil {
		t.Fatal(err)
	}

	// set up rpc and create resourcehandler with ENS sim backend
	rh, _, teardownTest, err := setupTest(contractbackend, ensClient, signer)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// create new resource when we are owner = ok
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, _, err = rh.New(ctx, safeName, resourceFrequency)
	if err != nil {
		t.Fatalf("Create resource fail: %v", err)
	}

	data := []byte("foo")
	// update resource when we are owner = ok
	_, err = rh.Update(ctx, safeName, data)
	if err != nil {
		t.Fatalf("Update resource fail: %v", err)
	}

	// update resource when we are not owner = !ok
	signertwo, err := newTestSigner()
	if err != nil {
		t.Fatal(err)
	}
	rh.signer = signertwo
	_, err = rh.Update(ctx, safeName, data)
	if err == nil {
		t.Fatalf("Expected resource update fail due to owner mismatch")
	}
}

func TestMultihash(t *testing.T) {

	// signer containing private key
	signer, err := newTestSigner()
	if err != nil {
		t.Fatal(err)
	}

	// make fake backend, set up rpc and create resourcehandler
	backend := &fakeBackend{
		blocknumber: int64(startBlock),
	}

	// set up rpc and create resourcehandler
	rh, datadir, teardownTest, err := setupTest(backend, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// create a new resource
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, _, err = rh.New(ctx, safeName, resourceFrequency)
	if err != nil {
		t.Fatal(err)
	}

	// we're naïvely assuming keccak256 for swarm hashes
	// if it ever changes this test should also change
	multihashbytes := ens.EnsNode("foo")
	multihashmulti := multihash.ToMultihash(multihashbytes.Bytes())
	multihashkey, err := rh.UpdateMultihash(ctx, safeName, multihashmulti)
	if err != nil {
		t.Fatal(err)
	}

	sha1bytes := make([]byte, multihash.MultihashLength)
	sha1multi := multihash.ToMultihash(sha1bytes)
	sha1key, err := rh.UpdateMultihash(ctx, safeName, sha1multi)
	if err != nil {
		t.Fatal(err)
	}

	// invalid multihashes
	_, err = rh.UpdateMultihash(ctx, safeName, multihashmulti[1:])
	if err == nil {
		t.Fatalf("Expected update to fail with first byte skipped")
	}
	_, err = rh.UpdateMultihash(ctx, safeName, multihashmulti[:len(multihashmulti)-2])
	if err == nil {
		t.Fatalf("Expected update to fail with last byte skipped")
	}

	data, err := getUpdateDirect(rh, multihashkey)
	if err != nil {
		t.Fatal(err)
	}
	multihashdecode, err := multihash.FromMultihash(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(multihashdecode, multihashbytes.Bytes()) {
		t.Fatalf("Decoded hash '%x' does not match original hash '%x'", multihashdecode, multihashbytes.Bytes())
	}
	data, err = getUpdateDirect(rh, sha1key)
	if err != nil {
		t.Fatal(err)
	}
	shadecode, err := multihash.FromMultihash(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(shadecode, sha1bytes) {
		t.Fatalf("Decoded hash '%x' does not match original hash '%x'", shadecode, sha1bytes)
	}
	rh.Close()

	rhparams := &HandlerParams{
		QueryMaxPeriods: &LookupParams{
			Limit: false,
		},
		Signer:         signer,
		HeaderGetter:   rh.headerGetter,
		OwnerValidator: rh.ownerValidator,
	}
	// test with signed data
	rh2, err := NewTestHandler(datadir, rhparams)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = rh2.New(ctx, safeName, resourceFrequency)
	if err != nil {
		t.Fatal(err)
	}
	multihashsignedkey, err := rh2.UpdateMultihash(ctx, safeName, multihashmulti)
	if err != nil {
		t.Fatal(err)
	}
	sha1signedkey, err := rh2.UpdateMultihash(ctx, safeName, sha1multi)
	if err != nil {
		t.Fatal(err)
	}

	data, err = getUpdateDirect(rh2, multihashsignedkey)
	if err != nil {
		t.Fatal(err)
	}
	multihashdecode, err = multihash.FromMultihash(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(multihashdecode, multihashbytes.Bytes()) {
		t.Fatalf("Decoded hash '%x' does not match original hash '%x'", multihashdecode, multihashbytes.Bytes())
	}
	data, err = getUpdateDirect(rh2, sha1signedkey)
	if err != nil {
		t.Fatal(err)
	}
	shadecode, err = multihash.FromMultihash(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(shadecode, sha1bytes) {
		t.Fatalf("Decoded hash '%x' does not match original hash '%x'", shadecode, sha1bytes)
	}
}

func TestChunkValidator(t *testing.T) {
	// signer containing private key
	signer, err := newTestSigner()
	if err != nil {
		t.Fatal(err)
	}

	// ens address and transact options
	addr := crypto.PubkeyToAddress(signer.PrivKey.PublicKey)
	transactOpts := bind.NewKeyedTransactor(signer.PrivKey)

	// set up ENS sim
	domainparts := strings.Split(safeName, ".")
	contractAddr, contractbackend, err := setupENS(addr, transactOpts, domainparts[0], domainparts[1])
	if err != nil {
		t.Fatal(err)
	}

	ensClient, err := ens.NewENS(transactOpts, contractAddr, contractbackend)
	if err != nil {
		t.Fatal(err)
	}

	// set up rpc and create resourcehandler with ENS sim backend
	rh, _, teardownTest, err := setupTest(contractbackend, ensClient, signer)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// create new resource when we are owner = ok
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	key, rsrc, err := rh.New(ctx, safeName, resourceFrequency)
	if err != nil {
		t.Fatalf("Create resource fail: %v", err)
	}

	data := []byte("foo")
	key = rh.resourceHash(1, 1, rsrc.nameHash)
	digest := rh.keyDataHash(key, data)
	sig, err := rh.signer.Sign(digest)
	if err != nil {
		t.Fatalf("sign fail: %v", err)
	}
	chunk := newUpdateChunk(key, &sig, 1, 1, safeName, data, len(data))
	if !rh.Validate(chunk.Addr, chunk.SData) {
		t.Fatal("Chunk validator fail on update chunk")
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	startBlock, err := rh.getBlock(ctx, safeName)
	if err != nil {
		t.Fatal(err)
	}
	chunk = rh.newMetaChunk(safeName, startBlock, resourceFrequency)
	if !rh.Validate(chunk.Addr, chunk.SData) {
		t.Fatal("Chunk validator fail on metadata chunk")
	}
}

// tests that the content address validator correctly checks the data
// tests that resource update chunks are passed through content address validator
// the test checking the resouce update validator internal correctness is found in resource_test.go
func TestValidator(t *testing.T) {

	// set up localstore
	datadir, err := ioutil.TempDir("", "storage-testresourcevalidator")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(datadir)

	params := storage.NewDefaultLocalStoreParams()
	params.Init(datadir)
	store, err := storage.NewLocalStore(params, nil)
	if err != nil {
		t.Fatal(err)
	}

	// add content address validator and resource validator to validators and check puts
	// bad should fail, good should pass
	store.Validators = append(store.Validators, storage.NewContentAddressValidator(hashfunc))
	rhParams := &HandlerParams{}
	rh, err := NewHandler(rhParams)
	if err != nil {
		t.Fatal(err)
	}
	store.Validators = append(store.Validators, rh)

	chunks := storage.GenerateRandomChunks(storage.DefaultChunkSize, 2)
	goodChunk := chunks[0]
	badChunk := chunks[1]
	badChunk.SData = goodChunk.SData
	key := rh.resourceHash(42, 1, ens.EnsNode("xyzzy.eth"))
	data := []byte("bar")
	uglyChunk := newUpdateChunk(key, nil, 42, 1, "xyzzy.eth", data, len(data))

	storage.PutChunks(store, goodChunk, badChunk, uglyChunk)
	if err := goodChunk.GetErrored(); err != nil {
		t.Fatalf("expected no error on good content address chunk with both validators, but got: %s", err)
	}
	if err := badChunk.GetErrored(); err == nil {
		t.Fatal("expected error on bad chunk address with both validators, but got nil")
	}
	if err := uglyChunk.GetErrored(); err != nil {
		t.Fatalf("expected no error on resource update chunk with both validators, but got: %s", err)
	}

	// (redundant check)
	// use only resource validator, and check puts
	// bad should fail, good should fail, resource should pass
	store.Validators[0] = store.Validators[1]
	store.Validators = store.Validators[:1]

	chunks = storage.GenerateRandomChunks(storage.DefaultChunkSize, 2)
	goodChunk = chunks[0]
	badChunk = chunks[1]
	badChunk.SData = goodChunk.SData

	key = rh.resourceHash(42, 2, ens.EnsNode("xyzzy.eth"))
	data = []byte("baz")
	uglyChunk = newUpdateChunk(key, nil, 42, 2, "xyzzy.eth", data, len(data))

	storage.PutChunks(store, goodChunk, badChunk, uglyChunk)
	if goodChunk.GetErrored() == nil {
		t.Fatal("expected error on good content address chunk with resource validator only, but got nil")
	}
	if badChunk.GetErrored() == nil {
		t.Fatal("expected error on bad content address chunk with resource validator only, but got nil")
	}
	if err := uglyChunk.GetErrored(); err != nil {
		t.Fatalf("expected no error on resource update chunk with resource validator only, but got: %s", err)
	}
}

// fast-forward blockheight
func fwdBlocks(count int, backend *fakeBackend) {
	for i := 0; i < count; i++ {
		backend.Commit()
	}
}

type ensOwnerValidator struct {
	*ens.ENS
}

func (e ensOwnerValidator) ValidateOwner(name string, address common.Address) (bool, error) {
	addr, err := e.Owner(ens.EnsNode(name))
	if err != nil {
		return false, err
	}
	return address == addr, nil
}

// create rpc and resourcehandler
func setupTest(backend headerGetter, ensBackend *ens.ENS, signer Signer) (rh *Handler, datadir string, teardown func(), err error) {

	var fsClean func()
	var rpcClean func()
	cleanF = func() {
		if fsClean != nil {
			fsClean()
		}
		if rpcClean != nil {
			rpcClean()
		}
	}

	// temp datadir
	datadir, err = ioutil.TempDir("", "rh")
	if err != nil {
		return nil, "", nil, err
	}
	fsClean = func() {
		os.RemoveAll(datadir)
	}

	var ov ownerValidator
	if ensBackend != nil {
		ov = ensOwnerValidator{ensBackend}
	}

	rhparams := &HandlerParams{
		QueryMaxPeriods: &LookupParams{
			Limit: false,
		},
		Signer:         signer,
		HeaderGetter:   backend,
		OwnerValidator: ov,
	}
	rh, err = NewTestHandler(datadir, rhparams)
	return rh, datadir, cleanF, err
}

// Set up simulated ENS backend for use with ENSHandler tests
func setupENS(addr common.Address, transactOpts *bind.TransactOpts, sub string, top string) (common.Address, *fakeBackend, error) {

	// create the domain hash values to pass to the ENS contract methods
	var tophash [32]byte
	var subhash [32]byte

	testHasher.Reset()
	testHasher.Write([]byte(top))
	copy(tophash[:], testHasher.Sum(nil))
	testHasher.Reset()
	testHasher.Write([]byte(sub))
	copy(subhash[:], testHasher.Sum(nil))

	// initialize contract backend and deploy
	contractBackend := &fakeBackend{
		SimulatedBackend: backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}}),
	}

	contractAddress, _, ensinstance, err := contract.DeployENS(transactOpts, contractBackend)
	if err != nil {
		return zeroAddr, nil, fmt.Errorf("can't deploy: %v", err)
	}

	// update the registry for the correct owner address
	if _, err = ensinstance.SetOwner(transactOpts, [32]byte{}, addr); err != nil {
		return zeroAddr, nil, fmt.Errorf("can't setowner: %v", err)
	}
	contractBackend.Commit()

	if _, err = ensinstance.SetSubnodeOwner(transactOpts, [32]byte{}, tophash, addr); err != nil {
		return zeroAddr, nil, fmt.Errorf("can't register top: %v", err)
	}
	contractBackend.Commit()

	if _, err = ensinstance.SetSubnodeOwner(transactOpts, ens.EnsNode(top), subhash, addr); err != nil {
		return zeroAddr, nil, fmt.Errorf("can't register top: %v", err)
	}
	contractBackend.Commit()

	return contractAddress, contractBackend, nil
}

func newTestSigner() (*GenericSigner, error) {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	return &GenericSigner{
		PrivKey: privKey,
	}, nil
}

func getUpdateDirect(rh *Handler, addr storage.Address) ([]byte, error) {
	chunk, err := rh.chunkStore.Get(addr)
	if err != nil {
		return nil, err
	}
	_, _, _, _, data, _, err := rh.parseUpdate(chunk.SData)
	if err != nil {
		return nil, err
	}
	return data, nil
}
