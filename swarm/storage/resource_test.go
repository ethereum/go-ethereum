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
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/contracts/ens/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	blockCount = uint64(4200)
	cleanF     func()
	hashfunc   = sha3.NewKeccak256()
)

func init() {
	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))
}

type FakeRPC struct {
	blockcount *uint64
}

func (r *FakeRPC) BlockNumber() (string, error) {
	return strconv.FormatUint(*r.blockcount, 10), nil
}

func TestResourceValidContent(t *testing.T) {

	rh, privkey, _, err, teardownTest := setupTest()
	if err != nil {
		teardownTest(t, err)
	}

	// create a new resource
	validname, err := idna.ToASCII("føø.bar")
	if err != nil {
		teardownTest(t, err)
	}

	// generate a hash for block 4200 version 1
	key := rh.resourceHash(ens.EnsNode(validname), 4200, 1)
	chunk := NewChunk(key, nil)

	// generate some bogus data for the chunk and sign it
	data := make([]byte, 8)
	_, err = rand.Read(data)
	if err != nil {
		teardownTest(t, err)
	}
	rh.hasher.Reset()
	rh.hasher.Write(data)
	datahash := rh.hasher.Sum(nil)
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
	recoveredaddress, err := rh.getContentAccount(chunk.SData)
	if err != nil {
		teardownTest(t, err)
	}
	originaladdress := crypto.PubkeyToAddress(privkey.PublicKey)

	if recoveredaddress != originaladdress {
		teardownTest(t, fmt.Errorf("addresses dont match: %x != %x", originaladdress, recoveredaddress))
	}
	teardownTest(t, nil)
}

func TestResourceReverseLookup(t *testing.T) {
	//rh, privkey, datadir, err, teardownTest := setupTest()
	rh, _, _, err, teardownTest := setupTest()
	if err != nil {
		teardownTest(t, err)
	}

	// create a new resource
	resourcename := "føø.bar"
	resourcefrequency := uint64(42)
	rsrc, err := rh.NewResource(resourcename, resourcefrequency)
	if err != nil {
		teardownTest(t, err)
	}

	// update data
	blockCount += resourcefrequency + 1
	data := []byte("foo")
	resourcekey, err := rh.Update(resourcename, data)
	if err != nil {
		teardownTest(t, err)
	}
	chunk, err := rh.ChunkStore.(*resourceChunkStore).localStore.(*LocalStore).memStore.Get(Key(resourcekey))
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

	if !bytes.Equal(revname, rsrc.ensName.Bytes()) {
		teardownTest(t, fmt.Errorf("Expected retrieved name from chunk data to be '%x', was '%x'", rsrc.ensName.Bytes(), revname))
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

func TestResourceHandler(t *testing.T) {

	rh, privkey, datadir, err, teardownTest := setupTest()
	if err != nil {
		teardownTest(t, err)
	}

	// create a new resource
	resourcename := "føø.bar"
	resourcevalidname, err := idna.ToASCII(resourcename)
	if err != nil {
		teardownTest(t, err)
	}
	resourcefrequency := uint64(42)
	_, err = rh.NewResource(resourcename, resourcefrequency)
	if err != nil {
		teardownTest(t, err)
	}

	// check that the new resource is stored correctly
	namehash := ens.EnsNode(resourcevalidname)
	chunk, err := rh.ChunkStore.(*resourceChunkStore).localStore.(*LocalStore).memStore.Get(Key(namehash[:]))
	if err != nil {
		teardownTest(t, err)
	} else if len(chunk.SData) < 16 {
		teardownTest(t, fmt.Errorf("chunk data must be minimum 16 bytes, is %d", len(chunk.SData)))
	}
	startblocknumber := binary.LittleEndian.Uint64(chunk.SData[8:16])
	chunkfrequency := binary.LittleEndian.Uint64(chunk.SData[16:])
	if startblocknumber != blockCount {
		teardownTest(t, fmt.Errorf("stored block number %d does not match provided block number %d", startblocknumber, blockCount))
	}
	if chunkfrequency != resourcefrequency {
		teardownTest(t, fmt.Errorf("stored frequency %d does not match provided frequency %d", chunkfrequency, resourcefrequency))
	}

	// update halfway to first period
	resourcekey := make(map[string]Key)
	blockCount = startblocknumber + (resourcefrequency / 2)
	resourcekey["blinky"], err = rh.Update(resourcename, []byte("blinky"))
	if err != nil {
		teardownTest(t, err)
	}

	// update on first period
	blockCount = startblocknumber + resourcefrequency
	resourcekey["pinky"], err = rh.Update(resourcename, []byte("pinky"))
	if err != nil {
		teardownTest(t, err)
	}

	// update on second period
	blockCount = startblocknumber + (resourcefrequency * 2)
	resourcekey["inky"], err = rh.Update(resourcename, []byte("inky"))
	if err != nil {
		teardownTest(t, err)
	}

	// update just after second period
	blockCount = startblocknumber + (resourcefrequency * 2) + 1
	resourcekey["clyde"], err = rh.Update(resourcename, []byte("clyde"))
	if err != nil {
		teardownTest(t, err)
	}
	time.Sleep(time.Second)
	rh.Close()

	// check we can retrieve the updates after close
	// it will match on second iteration startblocknumber + (resourcefrequency * 3)
	blockCount = startblocknumber + (resourcefrequency * 4)

	rh2, err := NewResourceHandler(privkey, datadir, &testCloudStore{}, rh.ethapi)
	_, err = rh2.LookupLatest(resourcename, true)
	if err != nil {
		teardownTest(t, err)
	}

	// last update should be "clyde", version two, blockheight startblocknumber + (resourcefrequency * 3)
	if !bytes.Equal(rh2.resources[resourcename].data, []byte("clyde")) {
		teardownTest(t, fmt.Errorf("resource data was %v, expected %v", rh2.resources[resourcename].data, []byte("clyde")))
	}
	if rh2.resources[resourcename].version != 2 {
		teardownTest(t, fmt.Errorf("resource version was %d, expected 2", rh2.resources[resourcename].version))
	}
	if rh2.resources[resourcename].lastPeriod != 3 {
		teardownTest(t, fmt.Errorf("resource period was %d, expected 3", rh2.resources[resourcename].lastPeriod))
	}

	rsrc, err := NewResource(resourcename, startblocknumber, resourcefrequency)
	if err != nil {
		teardownTest(t, err)
	}
	err = rh2.SetResource(rsrc, true)
	if err != nil {
		teardownTest(t, err)
	}

	// latest block, latest version
	resource, err := rh2.LookupLatest(resourcename, false) // if key is specified, refresh is implicit
	if err != nil {
		teardownTest(t, err)
	}

	// check data
	if !bytes.Equal(resource.data, []byte("clyde")) {
		teardownTest(t, fmt.Errorf("resource data (latest) was %v, expected %v", rh2.resources[resourcename].data, []byte("clyde")))
	}

	// specific block, latest version
	resource, err = rh2.LookupHistorical(resourcename, 3, true)
	if err != nil {
		teardownTest(t, err)
	}

	// check data
	if !bytes.Equal(resource.data, []byte("clyde")) {
		teardownTest(t, fmt.Errorf("resource data (historical) was %v, expected %v", rh2.resources[resourcename].data, []byte("clyde")))
	}

	// specific block, specific version
	resource, err = rh2.LookupVersion(resourcename, 3, 1, true)
	if err != nil {
		teardownTest(t, err)
	}

	// check data
	if !bytes.Equal(resource.data, []byte("inky")) {
		teardownTest(t, fmt.Errorf("resource data (historical) was %v, expected %v", rh2.resources[resourcename].data, []byte("inky")))
	}
	teardownTest(t, nil)

}

func setupTest() (rh *ResourceHandler, privkey *ecdsa.PrivateKey, datadir string, err error, teardown func(*testing.T, error)) {

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

	// privkey for signing updates
	privkey, err = crypto.GenerateKey()
	if err != nil {
		return
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
	rpcserver.RegisterName("eth", &FakeRPC{
		blockcount: &blockCount,
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

	rh, err = NewResourceHandler(privkey, datadir, &testCloudStore{}, rpcclient)
	teardown = func(t *testing.T, err error) {
		cleanF()
		if err != nil {
			t.Fatal(err)
		}
	}

	return
}

func TestResourceENS(t *testing.T) {
	_, privkey, _, err, teardownTest := setupTest()
	if err != nil {
		teardownTest(t, err)
	}
	err = setupENS(privkey, "foo", "bar")
	if err != nil {
		teardownTest(t, err)
	}
	teardownTest(t, nil)
}

func setupENS(privkey *ecdsa.PrivateKey, sub string, top string) error {
	// ens backend
	//key := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	var tophash [32]byte
	var subhash [32]byte
	hashfunc.Reset()
	hashfunc.Write([]byte(top))
	copy(tophash[:], hashfunc.Sum(nil))
	hashfunc.Reset()
	hashfunc.Write([]byte(sub))
	copy(subhash[:], hashfunc.Sum(nil))
	addr := crypto.PubkeyToAddress(privkey.PublicKey)
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}})
	transactOpts := bind.NewKeyedTransactor(privkey)

	_, _, ensinstance, err := contract.DeployENS(transactOpts, contractBackend)
	if err != nil {
		return fmt.Errorf("can't deploy: %v", err)
	}

	// Deploy the registrar.
	//	regAddr, _, reginstance, err := contract.DeployFIFSRegistrar(transactOpts, contractBackend, ensAddr, [32]byte{})
	//	if err != nil {
	//		return fmt.Errorf("can't deploy Registrar: %v", err)
	//	}
	//	contractBackend.Commit()
	//
	// Set the registrar as owner of the ENS root.
	if _, err = ensinstance.SetOwner(transactOpts, [32]byte{}, addr); err != nil {
		return fmt.Errorf("can't setowner: %v", err)
	}
	contractBackend.Commit()

	if _, err = ensinstance.SetSubnodeOwner(transactOpts, [32]byte{}, tophash, addr); err != nil {
		return fmt.Errorf("can't register top: %v", err)
	}
	contractBackend.Commit()

	if _, err = ensinstance.SetSubnodeOwner(transactOpts, ens.EnsNode(top), subhash, addr); err != nil {
		return fmt.Errorf("can't register top: %v", err)
	}
	contractBackend.Commit()

	nodeowner, err := ensinstance.Owner(&bind.CallOpts{}, ens.EnsNode(strings.Join([]string{sub, top}, ".")))
	if err != nil {
		return fmt.Errorf("can't retrieve owner: %v", err)
	} else if !bytes.Equal(nodeowner.Bytes(), addr.Bytes()) {
		return fmt.Errorf("retrieved owner doesn't match; expected '%x', got '%x'", addr, nodeowner)
	}

	return nil
}

type testCloudStore struct {
}

func (c *testCloudStore) Store(*Chunk) {
}

func (c *testCloudStore) Deliver(*Chunk) {
}

func (c *testCloudStore) Retrieve(*Chunk) {
}
