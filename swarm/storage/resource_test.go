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
	testHasher        = MakeHashFunc(SHA3Hash)()
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
func TestResourceReverse(t *testing.T) {

	period := uint32(4)
	version := uint32(2)

	// set up rpc and create resourcehandler
	rh, _, signer, teardownTest, err := setupTest(nil, nil)
	if err != nil {
		teardownTest(t, err)
	}

	// create a new resource
	validname, err := idna.ToASCII(domainName)
	if err != nil {
		teardownTest(t, err)
	}

	// generate a hash for block 4200 version 1
	key := rh.resourceHash(period, version, rh.validator.nameHash(validname))

	// generate some bogus data for the chunk and sign it
	data := make([]byte, 8)
	_, err = rand.Read(data)
	if err != nil {
		teardownTest(t, err)
	}
	testHasher.Reset()
	testHasher.Write(data)
	digest := rh.keyDataHash(key, data)
	sig, err := rh.validator.sign(digest)
	if err != nil {
		teardownTest(t, err)
	}

	chunk := newUpdateChunk(key, sig, period, version, validname, data)

	// check that we can recover the owner account from the update chunk's signature
	checksig, checkperiod, checkversion, checkname, checkdata, err := parseUpdate(chunk.SData)
	checkdigest := rh.keyDataHash(chunk.Key, checkdata)
	recoveredaddress, err := getAddressFromDataSig(checkdigest, checksig)
	if err != nil {
		teardownTest(t, err)
	}
	originaladdress := crypto.PubkeyToAddress(signer.privKey.PublicKey)

	if recoveredaddress != originaladdress {
		teardownTest(t, fmt.Errorf("addresses dont match: %x != %x", originaladdress, recoveredaddress))
	}

	if !bytes.Equal(key[:], chunk.Key[:]) {
		teardownTest(t, fmt.Errorf("Expected chunk key '%x', was '%x'", key, chunk.Key))
	}
	if period != checkperiod {
		teardownTest(t, fmt.Errorf("Expected period '%d', was '%d'", period, checkperiod))
	}
	if version != checkversion {
		teardownTest(t, fmt.Errorf("Expected version '%d', was '%d'", version, checkversion))
	}
	if validname != checkname {
		teardownTest(t, fmt.Errorf("Expected name '%s', was '%s'", validname, checkname))
	}
	if !bytes.Equal(data, checkdata) {
		teardownTest(t, fmt.Errorf("Expectedn data '%x', was '%x'", data, checkdata))
	}
	teardownTest(t, nil)
}

// make updates and retrieve them based on periods and versions
func TestResourceHandler(t *testing.T) {

	// make fake backend, set up rpc and create resourcehandler
	backend := &fakeBackend{
		blocknumber: startBlock,
	}
	rh, datadir, _, teardownTest, err := setupTest(backend, nil)
	if err != nil {
		teardownTest(t, err)
	}

	// create a new resource
	resourcevalidname, err := idna.ToASCII(domainName)
	if err != nil {
		teardownTest(t, err)
	}
	_, err = rh.NewResource(domainName, resourceFrequency, false)
	if err != nil {
		teardownTest(t, err)
	}

	// check that the new resource is stored correctly
	namehash := rh.validator.nameHash(resourcevalidname)
	chunk, err := rh.ChunkStore.(*resourceChunkStore).localStore.(*LocalStore).memStore.Get(Key(namehash[:]))
	if err != nil {
		teardownTest(t, err)
	} else if len(chunk.SData) < 16 {
		teardownTest(t, fmt.Errorf("chunk data must be minimum 16 bytes, is %d", len(chunk.SData)))
	}
	startblocknumber := binary.LittleEndian.Uint64(chunk.SData[:8])
	chunkfrequency := binary.LittleEndian.Uint64(chunk.SData[8:])
	if startblocknumber != backend.blocknumber {
		teardownTest(t, fmt.Errorf("stored block number %d does not match provided block number %d", startblocknumber, backend.blocknumber))
	}
	if chunkfrequency != resourceFrequency {
		teardownTest(t, fmt.Errorf("stored frequency %d does not match provided frequency %d", chunkfrequency, resourceFrequency))
	}

	// update halfway to first period
	resourcekey := make(map[string]Key)
	fwdBlocks(int(resourceFrequency/2), backend)
	data := []byte("blinky")
	resourcekey["blinky"], err = rh.Update(domainName, data)
	if err != nil {
		teardownTest(t, err)
	}

	// update on first period
	fwdBlocks(int(resourceFrequency/2), backend)
	data = []byte("pinky")
	resourcekey["pinky"], err = rh.Update(domainName, data)
	if err != nil {
		teardownTest(t, err)
	}

	// update on second period
	fwdBlocks(int(resourceFrequency), backend)
	data = []byte("inky")
	resourcekey["inky"], err = rh.Update(domainName, data)
	if err != nil {
		teardownTest(t, err)
	}

	// update just after second period
	fwdBlocks(1, backend)
	data = []byte("clyde")
	resourcekey["clyde"], err = rh.Update(domainName, data)
	if err != nil {
		teardownTest(t, err)
	}
	time.Sleep(time.Second)
	rh.Close()

	// check we can retrieve the updates after close
	// it will match on second iteration startblocknumber + (resourceFrequency * 3)
	fwdBlocks(int(resourceFrequency*2)-1, backend)

	rh2, err := NewResourceHandler(datadir, &testCloudStore{}, rh.rpcClient, nil)
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

	// specific block, latest version
	rsrc, err := rh2.LookupHistorical(domainName, 3, true)
	if err != nil {
		teardownTest(t, err)
	}

	// check data
	if !bytes.Equal(rsrc.data, []byte("clyde")) {
		teardownTest(t, fmt.Errorf("resource data (historical) was %v, expected %v", rh2.resources[domainName].data, []byte("clyde")))
	}

	// specific block, specific version
	rsrc, err = rh2.LookupVersion(domainName, 3, 1, true)
	if err != nil {
		teardownTest(t, err)
	}

	// check data
	if !bytes.Equal(rsrc.data, []byte("inky")) {
		teardownTest(t, fmt.Errorf("resource data (historical) was %v, expected %v", rh2.resources[domainName].data, []byte("inky")))
	}
	teardownTest(t, nil)

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
	domainparts := strings.Split(domainName, ".")
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
		teardownTest(t, err)
	}

	// create new resource when we are owner = ok
	_, err = rh.NewResource(domainName, resourceFrequency, true)
	if err != nil {
		teardownTest(t, fmt.Errorf("Create resource fail: %v", err))
	}

	data := []byte("foo")
	// update resource when we are owner = ok
	_, err = rh.Update(domainName, data)
	if err != nil {
		teardownTest(t, fmt.Errorf("Update resource fail: %v", err))
	}

	// update resource when we are owner = ok
	signertwo, err := newTestSigner()
	if err != nil {
		teardownTest(t, err)
	}
	rh.validator.(*ENSValidator).signFunc = signertwo.signContent
	_, err = rh.Update(domainName, data)
	if err == nil {
		teardownTest(t, fmt.Errorf("Expected resource update fail due to owner mismatch"))
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
func setupTest(contractbackend bind.ContractBackend, validator ResourceValidator) (rh *ResourceHandler, datadir string, signer *testSigner, teardown func(*testing.T, error), err error) {

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

	if validator == nil {
		// create a new signer, which creates the private key
		signer, err = newTestSigner()
		if err != nil {
			return
		}
		validator = NewGenericValidator(testHashFunc, signer.signContent)
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

	rh, err = NewResourceHandler(datadir, &testCloudStore{}, rpcclient, validator)
	teardown = func(t *testing.T, err error) {
		cleanF()
		if err != nil {
			t.Fatal(err)
		}
	}

	return
}

// Set up simulated ENS backend for use with ENSResourceHandler tests
func setupENS(addr common.Address, transactOpts *bind.TransactOpts, sub string, top string) (common.Address, bind.ContractBackend, error) {

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

func testHashFunc(name string) common.Hash {
	testHasher.Reset()
	testHasher.Write([]byte(name))
	return common.BytesToHash(testHasher.Sum(nil))
}

type testSigner struct {
	privKey *ecdsa.PrivateKey
	hasher  SwarmHash
}

func newTestSigner() (*testSigner, error) {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	return &testSigner{
		privKey: privKey,
		hasher:  testHasher,
	}, nil
}

func (self *testSigner) signContent(data common.Hash) (signature Signature, err error) {
	signaturebytes, err := crypto.Sign(data.Bytes(), self.privKey)
	if err != nil {
		return
	}
	copy(signature[:], signaturebytes)
	return
}

type testCloudStore struct {
}

func (c *testCloudStore) Store(*Chunk) {
}

func (c *testCloudStore) Deliver(*Chunk) {
}

func (c *testCloudStore) Retrieve(*Chunk) {
}
