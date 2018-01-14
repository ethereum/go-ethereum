package storage

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/idna"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/contracts/ens/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	hasher            = MakeHashFunc("SHA3")()
	zeroAddr          = common.Address{}
	startBlock        = uint64(4200)
	resourceFrequency = uint64(42)
	cleanF            func()
	domainName        = "føø.bar"
)

func init() {
	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))
}

// simulated backend does not have the blocknumber call
// so we use this wrapper to fake returning the block count
type fakeBackend struct {
	*backends.SimulatedBackend
	blocknumber uint64
}

func (f *fakeBackend) Commit() {
	if f.SimulatedBackend != nil {
		f.SimulatedBackend.Commit()
	}
	f.blocknumber++
}

// for faking the rpc service, since we don't need the whole node stack
type FakeRPC struct {
	backend *fakeBackend
}

func (r *FakeRPC) BlockNumber() (string, error) {
	return strconv.FormatUint(r.backend.blocknumber, 10), nil
}

// check that signature address matches update signer address
func TestResourceSignature(t *testing.T) {

	// privkey for signing updates
	privkey, err := crypto.GenerateKey()
	if err != nil {
		return
	}

	// set up rpc and create resourcehandler
	rh, _, err, teardownTest := setupTest(privkey, nil, zeroAddr)
	if err != nil {
		teardownTest(t, err)
	}

	// create a new resource
	validname, err := idna.ToASCII(domainName)
	if err != nil {
		teardownTest(t, err)
	}

	// generate a hash for block 4200 version 1
	key := rh.resourceHash(ens.EnsNode(validname), 1, 1)
	chunk := NewChunk(key, nil)

	// generate some bogus data for the chunk and sign it
	data := make([]byte, 8)
	_, err = rand.Read(data)
	if err != nil {
		teardownTest(t, err)
	}
	hasher.Reset()
	hasher.Write(data)
	datahash := hasher.Sum(nil)
	sig, err := crypto.Sign(datahash, privkey)
	if err != nil {
		teardownTest(t, err)
	}

	// put sig and data in the chunk
	chunk.SData = make([]byte, 8+signatureLength)
	copy(chunk.SData[:signatureLength], sig)
	copy(chunk.SData[signatureLength:], data)

	// check that we can recover the owner account from the update chunk's signature
	// TODO: change this to verifyContent on ENS integration
	rawrh := rh.(*RawResourceHandler)
	recoveredaddress, err := rawrh.getContentAccount(chunk.SData)
	if err != nil {
		teardownTest(t, err)
	}
	originaladdress := crypto.PubkeyToAddress(privkey.PublicKey)

	if recoveredaddress != originaladdress {
		teardownTest(t, fmt.Errorf("addresses dont match: %x != %x", originaladdress, recoveredaddress))
	}
	teardownTest(t, nil)
}

// determine resource update metadata from chunk data
func TestResourceReverseLookup(t *testing.T) {

	// privkey for signing updates
	privkey, err := crypto.GenerateKey()
	if err != nil {
		return
	}

	// make fake backend, set up rpc and create resourcehandler
	backend := &fakeBackend{
		blocknumber: startBlock,
	}
	rh, _, err, teardownTest := setupTest(privkey, backend, zeroAddr)
	if err != nil {
		teardownTest(t, err)
	}

	// create a new resource
	rsrc, err := rh.NewResource(domainName, resourceFrequency)
	if err != nil {
		teardownTest(t, err)
	}

	// update data
	fwdBlocks(int(resourceFrequency+1), backend)
	data := []byte("foo")
	resourcekey, err := rh.Update(domainName, data)
	if err != nil {
		teardownTest(t, err)
	}
	rawrh := rh.(*RawResourceHandler)
	chunk, err := rawrh.ChunkStore.(*resourceChunkStore).localStore.(*LocalStore).memStore.Get(Key(resourcekey))
	if err != nil {
		teardownTest(t, err)
	}

	// check if data after header length offset is as expected
	headerlength := binary.LittleEndian.Uint16(chunk.SData[signatureLength : signatureLength+2])
	if !bytes.Equal(chunk.SData[signatureLength+headerlength+2:], data) {
		teardownTest(t, fmt.Errorf("Expected chunk data with header length %d (pos %d) to match %x, but was %x", headerlength, signatureLength+headerlength+2, data, chunk.SData[signatureLength+headerlength+2:]))
	}

	// get name, period, version from chunk and check
	revperiod, revversion, revname, revdata, err := parseUpdate(chunk.SData[signatureLength:])

	if !bytes.Equal(revname, rsrc.nameHash.Bytes()) {
		teardownTest(t, fmt.Errorf("Expected retrieved name from chunk data to be '%x', was '%x'", rsrc.nameHash.Bytes(), revname))
	}
	if !bytes.Equal(revdata, data) {
		teardownTest(t, fmt.Errorf("Expected retrieved data from chunk data to be '%x', was '%x'", data, revdata))
	}

	if revperiod != 2 {
		teardownTest(t, fmt.Errorf("Expected retrieved period from chunk data to be 1, was %d", revperiod))
	}
	if revversion != 1 {
		teardownTest(t, fmt.Errorf("Expected retrieved version from chunk data to be 1, was %d", revversion))
	}
}

// make updates and retrieve them based on periods and versions
func TestResourceHandler(t *testing.T) {

	// privkey for signing updates
	privkey, err := crypto.GenerateKey()
	if err != nil {
		return
	}

	// make fake backend, set up rpc and create resourcehandler
	backend := &fakeBackend{
		blocknumber: startBlock,
	}
	rh, datadir, err, teardownTest := setupTest(privkey, backend, zeroAddr)
	if err != nil {
		teardownTest(t, err)
	}

	// create a new resource
	resourcevalidname, err := idna.ToASCII(domainName)
	if err != nil {
		teardownTest(t, err)
	}
	_, err = rh.NewResource(domainName, resourceFrequency)
	if err != nil {
		teardownTest(t, err)
	}

	// check that the new resource is stored correctly
	rawrh := rh.(*RawResourceHandler)
	namehash := rawrh.nameHashFunc(resourcevalidname)
	chunk, err := rawrh.ChunkStore.(*resourceChunkStore).localStore.(*LocalStore).memStore.Get(Key(namehash[:]))
	if err != nil {
		teardownTest(t, err)
	} else if len(chunk.SData) < 16 {
		teardownTest(t, fmt.Errorf("chunk data must be minimum 16 bytes, is %d", len(chunk.SData)))
	}
	startblocknumber := binary.LittleEndian.Uint64(chunk.SData[8:16])
	chunkfrequency := binary.LittleEndian.Uint64(chunk.SData[16:])
	if startblocknumber != backend.blocknumber {
		teardownTest(t, fmt.Errorf("stored block number %d does not match provided block number %d", startblocknumber, backend.blocknumber))
	}
	if chunkfrequency != resourceFrequency {
		teardownTest(t, fmt.Errorf("stored frequency %d does not match provided frequency %d", chunkfrequency, resourceFrequency))
	}

	// update halfway to first period
	resourcekey := make(map[string]Key)
	fwdBlocks(int(resourceFrequency/2), backend)
	resourcekey["blinky"], err = rh.Update(domainName, []byte("blinky"))
	if err != nil {
		teardownTest(t, err)
	}

	// update on first period
	fwdBlocks(int(resourceFrequency/2), backend)
	resourcekey["pinky"], err = rh.Update(domainName, []byte("pinky"))
	if err != nil {
		teardownTest(t, err)
	}

	// update on second period
	fwdBlocks(int(resourceFrequency), backend)
	resourcekey["inky"], err = rh.Update(domainName, []byte("inky"))
	if err != nil {
		teardownTest(t, err)
	}

	// update just after second period
	fwdBlocks(1, backend)
	resourcekey["clyde"], err = rh.Update(domainName, []byte("clyde"))
	if err != nil {
		teardownTest(t, err)
	}
	time.Sleep(time.Second)
	rh.Close()

	// check we can retrieve the updates after close
	// it will match on second iteration startblocknumber + (resourceFrequency * 3)
	fwdBlocks(int(resourceFrequency*2)-1, backend)

	rh2, err := NewRawResourceHandler(privkey, datadir, &testCloudStore{}, rawrh.rpcClient, nil)
	_, err = rh2.LookupLatest(domainName, true)
	if err != nil {
		teardownTest(t, err)
	}

	// last update should be "clyde", version two, blockheight startblocknumber + (resourcefrequency * 3)
	if !bytes.Equal(rh2.resources[domainName].data, []byte("clyde")) {
		teardownTest(t, fmt.Errorf("resource data was %v, expected %v", rh2.resources[domainName].data, []byte("clyde")))
	}
	if rh2.resources[domainName].version != 2 {
		teardownTest(t, fmt.Errorf("resource version was %d, expected 2", rh2.resources[domainName].version))
	}
	if rh2.resources[domainName].lastPeriod != 3 {
		teardownTest(t, fmt.Errorf("resource period was %d, expected 3", rh2.resources[domainName].lastPeriod))
	}

	rsrc, err := NewResource(domainName, startblocknumber, resourceFrequency, rh2.nameHashFunc)
	if err != nil {
		teardownTest(t, err)
	}
	err = rh2.SetResource(rsrc, true)
	if err != nil {
		teardownTest(t, err)
	}

	// latest block, latest version
	resource, err := rh2.LookupLatest(domainName, false) // if key is specified, refresh is implicit
	if err != nil {
		teardownTest(t, err)
	}

	// check data
	if !bytes.Equal(resource.data, []byte("clyde")) {
		teardownTest(t, fmt.Errorf("resource data (latest) was %v, expected %v", rh2.resources[domainName].data, []byte("clyde")))
	}

	// specific block, latest version
	resource, err = rh2.LookupHistorical(domainName, 3, true)
	if err != nil {
		teardownTest(t, err)
	}

	// check data
	if !bytes.Equal(resource.data, []byte("clyde")) {
		teardownTest(t, fmt.Errorf("resource data (historical) was %v, expected %v", rh2.resources[domainName].data, []byte("clyde")))
	}

	// specific block, specific version
	resource, err = rh2.LookupVersion(domainName, 3, 1, true)
	if err != nil {
		teardownTest(t, err)
	}

	// check data
	if !bytes.Equal(resource.data, []byte("inky")) {
		teardownTest(t, fmt.Errorf("resource data (historical) was %v, expected %v", rh2.resources[domainName].data, []byte("inky")))
	}
	teardownTest(t, nil)

}

// create ENS enabled resource update, with and without valid owner
func TestResourceENSNew(t *testing.T) {

	// privkey for signing updates
	privkey, err := crypto.GenerateKey()
	if err != nil {
		return
	}

	// privkey for checking wrong owner
	privkeytwo, err := crypto.GenerateKey()
	if err != nil {
		return
	}

	// set up ENS sim
	domainparts := strings.Split(domainName, ".")
	addr, contractbackend, err := setupENS(privkey, domainparts[0], domainparts[1])
	if err != nil {
		t.Fatal(err)
	}

	// set up rpc and create resourcehandler with ENS sim backend
	rh, _, err, teardownTest := setupTest(privkey, contractbackend, addr)
	if err != nil {
		teardownTest(t, err)
	}

	// create new resource when we are owner = ok
	_, err = rh.NewResource(domainName, 42)
	if err != nil {
		teardownTest(t, err)
	}

	// create new resource when we are NOT owner = !ok
	rawrh := rh.(*ENSResourceHandler)
	rawrh.privKey = privkeytwo
	rawrh.addr = crypto.PubkeyToAddress(privkeytwo.PublicKey)
	_, err = rawrh.NewResource(domainName, 42)
	if err == nil {
		teardownTest(t, fmt.Errorf("Expected resource create fail due to owner mismatch"))
	}

	teardownTest(t, nil)
}

// fast-forward blockheight
func fwdBlocks(count int, backend *fakeBackend) {
	for i := 0; i < count; i++ {
		backend.Commit()
	}
}

// create rpc and resourcehandler
func setupTest(privkey *ecdsa.PrivateKey, contractbackend bind.ContractBackend, ensaddr common.Address) (rh ResourceHandler, datadir string, err error, teardown func(*testing.T, error)) {

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
		return
	}
	fsClean = func() {
		os.RemoveAll(datadir)
	}

	// starting the whole stack just to get blocknumbers is too cumbersome
	// so we fake the rpc server to get blocknumbers for testing
	ipcpath := filepath.Join(datadir, "test.ipc")
	ipcl, err := rpc.CreateIPCListener(ipcpath)
	if err != nil {
		return
	}
	rpcserver := rpc.NewServer()
	var fake *fakeBackend
	if contractbackend != nil {
		fake = contractbackend.(*fakeBackend)
	}
	rpcserver.RegisterName("eth", &FakeRPC{
		backend: fake,
	})
	go func() {
		rpcserver.ServeListener(ipcl)
	}()
	rpcClean = func() {
		rpcserver.Stop()
	}

	// connect to fake rpc
	rpcclient, err := rpc.Dial(ipcpath)
	if err != nil {
		return
	}

	// choose if with ens or not
	if ensaddr != zeroAddr {
		rh, err = NewENSResourceHandler(privkey, datadir, &testCloudStore{}, rpcclient, contractbackend, ensaddr)
	} else {
		rh, err = NewRawResourceHandler(privkey, datadir, &testCloudStore{}, rpcclient, nil)
	}
	teardown = func(t *testing.T, err error) {
		cleanF()
		if err != nil {
			t.Fatal(err)
		}
	}

	return
}

// Set up simulated ENS backend for use with ENSResourceHandler tests
func setupENS(privkey *ecdsa.PrivateKey, sub string, top string) (common.Address, bind.ContractBackend, error) {

	// create the domain hash values to pass to the ENS contract methods
	var tophash [32]byte
	var subhash [32]byte

	hasher.Reset()
	hasher.Write([]byte(top))
	copy(tophash[:], hasher.Sum(nil))
	hasher.Reset()
	hasher.Write([]byte(sub))
	copy(subhash[:], hasher.Sum(nil))

	// private key -> address is owner of domain
	addr := crypto.PubkeyToAddress(privkey.PublicKey)

	// initialize contract backend and deploy
	transactOpts := bind.NewKeyedTransactor(privkey)
	contractBackend := &fakeBackend{
		SimulatedBackend: backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}}),
	}

	ensAddr, _, ensinstance, err := contract.DeployENS(transactOpts, contractBackend)
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

	return ensAddr, contractBackend, nil
}

type testCloudStore struct {
}

func (c *testCloudStore) Store(*Chunk) {
}

func (c *testCloudStore) Deliver(*Chunk) {
}

func (c *testCloudStore) Retrieve(*Chunk) {
}
