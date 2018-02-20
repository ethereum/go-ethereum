package storage

import (
	"bytes"
	"context"
	"crypto/ecdsa"
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

	"github.com/multiformats/go-multihash"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/contracts/ens/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

var (
	testHasher        = MakeHashFunc(SHA3Hash)()
	zeroAddr          = common.Address{}
	startBlock        = uint64(4200)
	resourceFrequency = uint64(42)
	cleanF            func()
	domainName        = "føø.bar"
	safeName          string
)

func init() {
	var err error
	verbose := flag.Bool("v", false, "verbose")
	flag.Parse()
	if *verbose {
		log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))
	}
	safeName, err = ToSafeName(domainName)
	if err != nil {
		panic(err)
	}
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

func (f *fakeBackend) HeaderByNumber(context context.Context, bigblock *big.Int) (*types.Header, error) {
	f.blocknumber++
	biggie := big.NewInt(f.blocknumber)
	return &types.Header{
		Number: biggie,
	}, nil
}

// check that signature address matches update signer address
func TestResourceReverse(t *testing.T) {

	period := uint32(4)
	version := uint32(2)

	// signer containing private key
	signer, err := newTestSigner()
	if err != nil {
		t.Fatal(err)
	}

	// set up rpc and create resourcehandler
	rh, _, _, teardownTest, err := setupTest(nil, newTestValidator(signer.signContent))
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// generate a hash for block 4200 version 1
	key := rh.resourceHash(period, version, rh.nameHash(safeName))

	// generate some bogus data for the chunk and sign it
	data := make([]byte, 8)
	_, err = rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	testHasher.Reset()
	testHasher.Write(data)
	digest := rh.keyDataHash(key, data)
	sig, err := rh.validator.sign(digest)
	if err != nil {
		t.Fatal(err)
	}

	chunk := newUpdateChunk(key, &sig, period, version, safeName, data, len(data))

	// check that we can recover the owner account from the update chunk's signature
	checksig, checkperiod, checkversion, checkname, checkdata, err := rh.parseUpdate(chunk.SData)
	if err != nil {
		t.Fatal(err)
	}
	checkdigest := rh.keyDataHash(chunk.Key, checkdata)
	recoveredaddress, err := getAddressFromDataSig(checkdigest, *checksig)
	if err != nil {
		t.Fatalf("Retrieve address from signature fail: %v", err)
	}
	originaladdress := crypto.PubkeyToAddress(signer.privKey.PublicKey)

	// check that the metadata retrieved from the chunk matches what we gave it
	if recoveredaddress != originaladdress {
		t.Fatalf("addresses dont match: %x != %x", originaladdress, recoveredaddress)
	}

	if !bytes.Equal(key[:], chunk.Key[:]) {
		t.Fatalf("Expected chunk key '%x', was '%x'", key, chunk.Key)
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
func TestResourceHandler(t *testing.T) {

	// make fake backend, set up rpc and create resourcehandler
	backend := &fakeBackend{
		blocknumber: int64(startBlock),
	}
	rh, datadir, _, teardownTest, err := setupTest(backend, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// create a new resource
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = rh.NewResource(ctx, safeName, resourceFrequency)
	if err != nil {
		t.Fatal(err)
	}

	// check that the new resource is stored correctly
	namehash := rh.nameHash(safeName)
	chunk, err := rh.ChunkStore.(*resourceChunkStore).localStore.(*LocalStore).memStore.Get(Key(namehash[:]))
	if err != nil {
		t.Fatal(err)
	} else if len(chunk.SData) < 16 {
		t.Fatalf("chunk data must be minimum 16 bytes, is %d", len(chunk.SData))
	}
	startblocknumber := binary.LittleEndian.Uint64(chunk.SData[:8])
	chunkfrequency := binary.LittleEndian.Uint64(chunk.SData[8:])
	if startblocknumber != uint64(backend.blocknumber) {
		t.Fatalf("stored block number %d does not match provided block number %d", startblocknumber, backend.blocknumber)
	}
	if chunkfrequency != resourceFrequency {
		t.Fatalf("stored frequency %d does not match provided frequency %d", chunkfrequency, resourceFrequency)
	}

	// update halfway to first period
	resourcekey := make(map[string]Key)
	fwdBlocks(int(resourceFrequency/2), backend)
	data := []byte("blinky")
	resourcekey["blinky"], err = rh.Update(ctx, safeName, data)
	if err != nil {
		t.Fatal(err)
	}

	// update on first period
	fwdBlocks(int(resourceFrequency/2), backend)
	data = []byte("pinky")
	resourcekey["pinky"], err = rh.Update(ctx, safeName, data)
	if err != nil {
		t.Fatal(err)
	}

	// update on second period
	fwdBlocks(int(resourceFrequency), backend)
	data = []byte("inky")
	resourcekey["inky"], err = rh.Update(ctx, safeName, data)
	if err != nil {
		t.Fatal(err)
	}

	// update just after second period
	fwdBlocks(1, backend)
	data = []byte("clyde")
	resourcekey["clyde"], err = rh.Update(ctx, safeName, data)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	rh.Close()

	// check we can retrieve the updates after close
	// it will match on second iteration startblocknumber + (resourceFrequency * 3)
	fwdBlocks(int(resourceFrequency*2)-1, backend)

	rh2, err := NewTestResourceHandler(datadir, rh.ethClient, nil)
	_, err = rh2.LookupLatestByName(ctx, safeName, true)
	if err != nil {
		t.Fatal(err)
	}

	// last update should be "clyde", version two, blockheight startblocknumber + (resourcefrequency * 3)
	if !bytes.Equal(rh2.resources[safeName].data, []byte("clyde")) {
		t.Fatalf("resource data was %v, expected %v", rh2.resources[safeName].data, []byte("clyde"))
	}
	if rh2.resources[safeName].version != 2 {
		t.Fatalf("resource version was %d, expected 2", rh2.resources[safeName].version)
	}
	if rh2.resources[safeName].lastPeriod != 3 {
		t.Fatalf("resource period was %d, expected 3", rh2.resources[safeName].lastPeriod)
	}
	log.Debug("Latest lookup", "period", rh2.resources[safeName].lastPeriod, "version", rh2.resources[safeName].version, "data", rh2.resources[safeName].data)

	// specific block, latest version
	rsrc, err := rh2.LookupHistoricalByName(ctx, safeName, 3, true)
	if err != nil {
		t.Fatal(err)
	}
	// check data
	if !bytes.Equal(rsrc.data, []byte("clyde")) {
		t.Fatalf("resource data (historical) was %v, expected %v", rh2.resources[domainName].data, []byte("clyde"))
	}
	log.Debug("Historical lookup", "period", rh2.resources[safeName].lastPeriod, "version", rh2.resources[safeName].version, "data", rh2.resources[safeName].data)

	// specific block, specific version
	rsrc, err = rh2.LookupVersionByName(ctx, safeName, 3, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	// check data
	if !bytes.Equal(rsrc.data, []byte("inky")) {
		t.Fatalf("resource data (historical) was %v, expected %v", rh2.resources[domainName].data, []byte("inky"))
	}
	log.Debug("Specific version lookup", "period", rh2.resources[safeName].lastPeriod, "version", rh2.resources[safeName].version, "data", rh2.resources[safeName].data)

}

func TestResourceMultihash(t *testing.T) {

	// signer containing private key
	signer, err := newTestSigner()
	if err != nil {
		t.Fatal(err)
	}
	validator := newTestValidator(signer.signContent)

	// make fake backend, set up rpc and create resourcehandler
	backend := &fakeBackend{
		blocknumber: int64(startBlock),
	}

	// set up rpc and create resourcehandler
	rh, datadir, _, teardownTest, err := setupTest(backend, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// create a new resource
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = rh.NewResource(ctx, safeName, resourceFrequency)
	if err != nil {
		t.Fatal(err)
	}

	// we're naïvely assuming keccak256 for swarm hashes
	// if it ever changes this test should also change
	swarmhashbytes := rh.nameHash("foo")
	swarmhashmulti, err := multihash.Encode(swarmhashbytes.Bytes(), multihash.KECCAK_256)
	if err != nil {
		t.Fatal(err)
	}
	swarmhashkey, err := rh.UpdateMultihash(ctx, safeName, swarmhashmulti)
	if err != nil {
		t.Fatal(err)
	}

	sha1bytes := make([]byte, multihash.DefaultLengths[multihash.SHA1])
	sha1multi, err := multihash.Encode(sha1bytes, multihash.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	sha1key, err := rh.UpdateMultihash(ctx, safeName, sha1multi)
	if err != nil {
		t.Fatal(err)
	}

	// invalid multihashes
	_, err = rh.UpdateMultihash(ctx, safeName, swarmhashmulti[1:])
	if err == nil {
		t.Fatalf("Expected update to fail with first byte skipped")
	}
	_, err = rh.UpdateMultihash(ctx, safeName, swarmhashmulti[:len(swarmhashmulti)-2])
	if err == nil {
		t.Fatalf("Expected update to fail with last byte skipped")
	}

	data, err := getUpdateDirect(rh, swarmhashkey)
	if err != nil {
		t.Fatal(err)
	}
	swarmhashdecode, err := multihash.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(swarmhashdecode.Digest, swarmhashbytes.Bytes()) {
		t.Fatalf("Decoded SHA1 hash '%x' does not match original hash '%x'", swarmhashdecode.Digest, swarmhashbytes.Bytes())
	}
	data, err = getUpdateDirect(rh, sha1key)
	if err != nil {
		t.Fatal(err)
	}
	sha1decode, err := multihash.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sha1decode.Digest, sha1bytes) {
		t.Fatalf("Decoded SHA1 hash '%x' does not match original hash '%x'", sha1decode.Digest, sha1bytes)
	}
	rh.Close()

	// test with signed data
	rh2, err := NewTestResourceHandler(datadir, rh.ethClient, validator)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rh2.NewResource(ctx, safeName, resourceFrequency)
	if err != nil {
		t.Fatal(err)
	}
	swarmhashsignedkey, err := rh2.UpdateMultihash(ctx, safeName, swarmhashmulti)
	if err != nil {
		t.Fatal(err)
	}
	sha1signedkey, err := rh2.UpdateMultihash(ctx, safeName, sha1multi)
	if err != nil {
		t.Fatal(err)
	}

	data, err = getUpdateDirect(rh2, swarmhashsignedkey)
	if err != nil {
		t.Fatal(err)
	}
	swarmhashdecode, err = multihash.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(swarmhashdecode.Digest, swarmhashbytes.Bytes()) {
		t.Fatalf("Decoded SHA1 hash '%x' does not match original hash '%x'", swarmhashdecode.Digest, swarmhashbytes.Bytes())
	}
	data, err = getUpdateDirect(rh2, sha1signedkey)
	if err != nil {
		t.Fatal(err)
	}
	sha1decode, err = multihash.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sha1decode.Digest, sha1bytes) {
		t.Fatalf("Decoded SHA1 hash '%x' does not match original hash '%x'", sha1decode.Digest, sha1bytes)
	}
}

// create ENS enabled resource update, with and without valid owner
func TestResourceENSOwner(t *testing.T) {

	// signer containing private key
	signer, err := newTestSigner()
	if err != nil {
		t.Fatal(err)
	}

	// ens address and transact options
	addr := crypto.PubkeyToAddress(signer.privKey.PublicKey)
	transactOpts := bind.NewKeyedTransactor(signer.privKey)

	// set up ENS sim
	domainparts := strings.Split(safeName, ".")
	contractAddr, contractbackend, err := setupENS(addr, transactOpts, domainparts[0], domainparts[1])
	if err != nil {
		t.Fatal(err)
	}

	validator, err := NewENSValidator(contractAddr, contractbackend, transactOpts, signer.signContent)
	if err != nil {
		t.Fatal(err)
	}

	// set up rpc and create resourcehandler with ENS sim backend
	rh, _, _, teardownTest, err := setupTest(contractbackend, validator)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// create new resource when we are owner = ok
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = rh.NewResource(ctx, safeName, resourceFrequency)
	if err != nil {
		t.Fatalf("Create resource fail: %v", err)
	}

	data := []byte("foo")
	// update resource when we are owner = ok
	_, err = rh.Update(ctx, safeName, data)
	if err != nil {
		t.Fatalf("Update resource fail: %v", err)
	}

	// update resource when we are owner = ok
	signertwo, err := newTestSigner()
	if err != nil {
		t.Fatal(err)
	}
	rh.validator.(*ENSValidator).signFunc = signertwo.signContent
	_, err = rh.Update(ctx, safeName, data)
	if err == nil {
		t.Fatalf("Expected resource update fail due to owner mismatch")
	}
}

// fast-forward blockheight
func fwdBlocks(count int, backend *fakeBackend) {
	for i := 0; i < count; i++ {
		backend.Commit()
	}
}

// create rpc and resourcehandler
func setupTest(backend ethApi, validator ResourceValidator) (rh *ResourceHandler, datadir string, signer *testSigner, teardown func(), err error) {

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
		return nil, "", nil, nil, err
	}
	fsClean = func() {
		os.RemoveAll(datadir)
	}

	rh, err = NewTestResourceHandler(datadir, backend, validator)
	return rh, datadir, signer, cleanF, nil
}

// Set up simulated ENS backend for use with ENSResourceHandler tests
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

// implementation of an external signer to pass to validator
type testSigner struct {
	privKey     *ecdsa.PrivateKey
	hasher      SwarmHash
	signContent SignFunc
}

func newTestSigner() (*testSigner, error) {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	return &testSigner{
		privKey:     privKey,
		hasher:      testHasher,
		signContent: NewGenericResourceSigner(privKey),
	}, nil
}

// Default fallthrough validation of mutable resource ownership
type testValidator struct {
	*baseValidator
	hashFunc func(string) common.Hash
}

func newTestValidator(signFunc SignFunc) *testValidator {
	return &testValidator{
		baseValidator: &baseValidator{
			signFunc: signFunc,
		},
		hashFunc: func(name string) common.Hash {
			testHasher.Reset()
			testHasher.Write([]byte(name))
			return common.BytesToHash(testHasher.Sum(nil))
		},
	}

}

func (self *testValidator) checkAccess(name string, address common.Address) (bool, error) {
	return true, nil
}

func (self *testValidator) nameHash(name string) common.Hash {
	return self.hashFunc(name)
}

func getUpdateDirect(rh *ResourceHandler, key Key) ([]byte, error) {
	chunk, err := rh.ChunkStore.(*resourceChunkStore).localStore.(*LocalStore).memStore.Get(key)
	if err != nil {
		return nil, err
	}
	_, _, _, _, data, err := rh.parseUpdate(chunk.SData)
	if err != nil {
		return nil, err
	}
	return data, nil
}
