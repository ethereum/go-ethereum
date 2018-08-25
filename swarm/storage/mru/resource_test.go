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
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/multihash"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var (
	loglevel   = flag.Int("loglevel", 3, "loglevel")
	testHasher = storage.MakeHashFunc(resourceHashAlgorithm)()
	startTime  = Timestamp{
		Time: uint64(4200),
	}
	resourceFrequency = uint64(42)
	cleanF            func()
	resourceName      = "føø.bar"
	hashfunc          = storage.MakeHashFunc(storage.DefaultHash)
)

func init() {
	flag.Parse()
	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))
}

// simulated timeProvider
type fakeTimeProvider struct {
	currentTime uint64
}

func (f *fakeTimeProvider) Tick() {
	f.currentTime++
}

func (f *fakeTimeProvider) Now() Timestamp {
	return Timestamp{
		Time: f.currentTime,
	}
}

func TestUpdateChunkSerializationErrorChecking(t *testing.T) {

	// Test that parseUpdate fails if the chunk is too small
	var r SignedResourceUpdate
	if err := r.fromChunk(storage.ZeroAddr, make([]byte, minimumUpdateDataLength-1)); err == nil {
		t.Fatalf("Expected parseUpdate to fail when chunkData contains less than %d bytes", minimumUpdateDataLength)
	}

	r = SignedResourceUpdate{}
	// Test that parseUpdate fails when the length header does not match the data array length
	fakeChunk := make([]byte, 150)
	binary.LittleEndian.PutUint16(fakeChunk, 44)
	if err := r.fromChunk(storage.ZeroAddr, fakeChunk); err == nil {
		t.Fatal("Expected parseUpdate to fail when the header length does not match the actual data array passed in")
	}

	r = SignedResourceUpdate{
		resourceUpdate: resourceUpdate{
			updateHeader: updateHeader{
				UpdateLookup: UpdateLookup{

					rootAddr: make([]byte, 79), // put the wrong length, should be storage.KeyLength
				},
				metaHash:  nil,
				multihash: false,
			},
		},
	}
	_, err := r.toChunk()
	if err == nil {
		t.Fatal("Expected newUpdateChunk to fail when rootAddr or metaHash have the wrong length")
	}
	r.rootAddr = make([]byte, storage.KeyLength)
	r.metaHash = make([]byte, storage.KeyLength)
	_, err = r.toChunk()
	if err == nil {
		t.Fatal("Expected newUpdateChunk to fail when there is no data")
	}
	r.data = make([]byte, 79) // put some arbitrary length data
	_, err = r.toChunk()
	if err == nil {
		t.Fatal("expected newUpdateChunk to fail when there is no signature", err)
	}

	alice := newAliceSigner()
	if err := r.Sign(alice); err != nil {
		t.Fatalf("error signing:%s", err)

	}
	_, err = r.toChunk()
	if err != nil {
		t.Fatalf("error creating update chunk:%s", err)
	}

	r.multihash = true
	r.data[1] = 79 // mess with the multihash, corrupting one byte of it.
	if err := r.Sign(alice); err == nil {
		t.Fatal("expected Sign() to fail when an invalid multihash is in data and multihash=true", err)
	}
}

// check that signature address matches update signer address
func TestReverse(t *testing.T) {

	period := uint32(4)
	version := uint32(2)

	// make fake timeProvider
	timeProvider := &fakeTimeProvider{
		currentTime: startTime.Time,
	}

	// signer containing private key
	signer := newAliceSigner()

	// set up rpc and create resourcehandler
	_, _, teardownTest, err := setupTest(timeProvider, signer)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	metadata := ResourceMetadata{
		Name:      resourceName,
		StartTime: startTime,
		Frequency: resourceFrequency,
		Owner:     signer.Address(),
	}

	rootAddr, metaHash, _, err := metadata.serializeAndHash()
	if err != nil {
		t.Fatal(err)
	}

	// generate some bogus data for the chunk and sign it
	data := make([]byte, 8)
	_, err = rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	testHasher.Reset()
	testHasher.Write(data)

	update := &SignedResourceUpdate{
		resourceUpdate: resourceUpdate{
			updateHeader: updateHeader{
				UpdateLookup: UpdateLookup{
					period:   period,
					version:  version,
					rootAddr: rootAddr,
				},
				metaHash: metaHash,
			},
			data: data,
		},
	}
	// generate a hash for t=4200 version 1
	key := update.UpdateAddr()

	if err = update.Sign(signer); err != nil {
		t.Fatal(err)
	}

	chunk, err := update.toChunk()
	if err != nil {
		t.Fatal(err)
	}

	// check that we can recover the owner account from the update chunk's signature
	var checkUpdate SignedResourceUpdate
	if err := checkUpdate.fromChunk(chunk.Addr, chunk.SData); err != nil {
		t.Fatal(err)
	}
	checkdigest, err := checkUpdate.GetDigest()
	if err != nil {
		t.Fatal(err)
	}
	recoveredaddress, err := getOwner(checkdigest, *checkUpdate.signature)
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
	if period != checkUpdate.period {
		t.Fatalf("Expected period '%d', was '%d'", period, checkUpdate.period)
	}
	if version != checkUpdate.version {
		t.Fatalf("Expected version '%d', was '%d'", version, checkUpdate.version)
	}
	if !bytes.Equal(data, checkUpdate.data) {
		t.Fatalf("Expectedn data '%x', was '%x'", data, checkUpdate.data)
	}
}

// make updates and retrieve them based on periods and versions
func TestResourceHandler(t *testing.T) {

	// make fake timeProvider
	timeProvider := &fakeTimeProvider{
		currentTime: startTime.Time,
	}

	// signer containing private key
	signer := newAliceSigner()

	rh, datadir, teardownTest, err := setupTest(timeProvider, signer)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// create a new resource
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metadata := &ResourceMetadata{
		Name:      resourceName,
		Frequency: resourceFrequency,
		StartTime: Timestamp{Time: timeProvider.Now().Time},
		Owner:     signer.Address(),
	}

	request, err := NewCreateUpdateRequest(metadata)
	if err != nil {
		t.Fatal(err)
	}
	request.Sign(signer)
	if err != nil {
		t.Fatal(err)
	}
	err = rh.New(ctx, request)
	if err != nil {
		t.Fatal(err)
	}

	chunk, err := rh.chunkStore.Get(context.TODO(), storage.Address(request.rootAddr))
	if err != nil {
		t.Fatal(err)
	} else if len(chunk.SData) < 16 {
		t.Fatalf("chunk data must be minimum 16 bytes, is %d", len(chunk.SData))
	}

	var recoveredMetadata ResourceMetadata

	recoveredMetadata.binaryGet(chunk.SData)
	if err != nil {
		t.Fatal(err)
	}
	if recoveredMetadata.StartTime.Time != timeProvider.currentTime {
		t.Fatalf("stored startTime %d does not match provided startTime %d", recoveredMetadata.StartTime.Time, timeProvider.currentTime)
	}
	if recoveredMetadata.Frequency != resourceFrequency {
		t.Fatalf("stored frequency %d does not match provided frequency %d", recoveredMetadata.Frequency, resourceFrequency)
	}

	// data for updates:
	updates := []string{
		"blinky",
		"pinky",
		"inky",
		"clyde",
	}

	// update halfway to first period. period=1, version=1
	resourcekey := make(map[string]storage.Address)
	fwdClock(int(resourceFrequency/2), timeProvider)
	data := []byte(updates[0])
	request.SetData(data, false)
	if err := request.Sign(signer); err != nil {
		t.Fatal(err)
	}
	resourcekey[updates[0]], err = rh.Update(ctx, &request.SignedResourceUpdate)
	if err != nil {
		t.Fatal(err)
	}

	// update on first period with version = 1 to make it fail since there is already one update with version=1
	request, err = rh.NewUpdateRequest(ctx, request.rootAddr)
	if err != nil {
		t.Fatal(err)
	}
	if request.version != 2 || request.period != 1 {
		t.Fatal("Suggested period should be 1 and version should be 2")
	}

	request.version = 1 // force version 1 instead of 2 to make it fail
	data = []byte(updates[1])
	request.SetData(data, false)
	if err := request.Sign(signer); err != nil {
		t.Fatal(err)
	}
	resourcekey[updates[1]], err = rh.Update(ctx, &request.SignedResourceUpdate)
	if err == nil {
		t.Fatal("Expected update to fail since this version already exists")
	}

	// update on second period with version = 1, correct. period=2, version=1
	fwdClock(int(resourceFrequency/2), timeProvider)
	request, err = rh.NewUpdateRequest(ctx, request.rootAddr)
	if err != nil {
		t.Fatal(err)
	}
	request.SetData(data, false)
	if err := request.Sign(signer); err != nil {
		t.Fatal(err)
	}
	resourcekey[updates[1]], err = rh.Update(ctx, &request.SignedResourceUpdate)
	if err != nil {
		t.Fatal(err)
	}

	fwdClock(int(resourceFrequency), timeProvider)
	// Update on third period, with version = 1
	request, err = rh.NewUpdateRequest(ctx, request.rootAddr)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(updates[2])
	request.SetData(data, false)
	if err := request.Sign(signer); err != nil {
		t.Fatal(err)
	}
	resourcekey[updates[2]], err = rh.Update(ctx, &request.SignedResourceUpdate)
	if err != nil {
		t.Fatal(err)
	}

	// update just after third period
	fwdClock(1, timeProvider)
	request, err = rh.NewUpdateRequest(ctx, request.rootAddr)
	if err != nil {
		t.Fatal(err)
	}
	if request.period != 3 || request.version != 2 {
		t.Fatal("Suggested period should be 3 and version should be 2")
	}
	data = []byte(updates[3])
	request.SetData(data, false)

	if err := request.Sign(signer); err != nil {
		t.Fatal(err)
	}
	resourcekey[updates[3]], err = rh.Update(ctx, &request.SignedResourceUpdate)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)
	rh.Close()

	// check we can retrieve the updates after close
	// it will match on second iteration startTime + (resourceFrequency * 3)
	fwdClock(int(resourceFrequency*2)-1, timeProvider)

	rhparams := &HandlerParams{}

	rh2, err := NewTestHandler(datadir, rhparams)
	if err != nil {
		t.Fatal(err)
	}

	rsrc2, err := rh2.Load(context.TODO(), request.rootAddr)
	if err != nil {
		t.Fatal(err)
	}

	_, err = rh2.Lookup(ctx, LookupLatest(request.rootAddr))
	if err != nil {
		t.Fatal(err)
	}

	// last update should be "clyde", version two, time= startTime + (resourcefrequency * 3)
	if !bytes.Equal(rsrc2.data, []byte(updates[len(updates)-1])) {
		t.Fatalf("resource data was %v, expected %v", string(rsrc2.data), updates[len(updates)-1])
	}
	if rsrc2.version != 2 {
		t.Fatalf("resource version was %d, expected 2", rsrc2.version)
	}
	if rsrc2.period != 3 {
		t.Fatalf("resource period was %d, expected 3", rsrc2.period)
	}
	log.Debug("Latest lookup", "period", rsrc2.period, "version", rsrc2.version, "data", rsrc2.data)

	// specific period, latest version
	rsrc, err := rh2.Lookup(ctx, LookupLatestVersionInPeriod(request.rootAddr, 3))
	if err != nil {
		t.Fatal(err)
	}
	// check data
	if !bytes.Equal(rsrc.data, []byte(updates[len(updates)-1])) {
		t.Fatalf("resource data (historical) was %v, expected %v", string(rsrc2.data), updates[len(updates)-1])
	}
	log.Debug("Historical lookup", "period", rsrc2.period, "version", rsrc2.version, "data", rsrc2.data)

	// specific period, specific version
	lookupParams := LookupVersion(request.rootAddr, 3, 1)
	rsrc, err = rh2.Lookup(ctx, lookupParams)
	if err != nil {
		t.Fatal(err)
	}
	// check data
	if !bytes.Equal(rsrc.data, []byte(updates[2])) {
		t.Fatalf("resource data (historical) was %v, expected %v", string(rsrc2.data), updates[2])
	}
	log.Debug("Specific version lookup", "period", rsrc2.period, "version", rsrc2.version, "data", rsrc2.data)

	// we are now at third update
	// check backwards stepping to the first
	for i := 1; i >= 0; i-- {
		rsrc, err := rh2.LookupPrevious(ctx, lookupParams)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(rsrc.data, []byte(updates[i])) {
			t.Fatalf("resource data (previous) was %v, expected %v", rsrc.data, updates[i])

		}
	}

	// beyond the first should yield an error
	rsrc, err = rh2.LookupPrevious(ctx, lookupParams)
	if err == nil {
		t.Fatalf("expected previous to fail, returned period %d version %d data %v", rsrc.period, rsrc.version, rsrc.data)
	}

}

func TestMultihash(t *testing.T) {

	// make fake timeProvider
	timeProvider := &fakeTimeProvider{
		currentTime: startTime.Time,
	}

	// signer containing private key
	signer := newAliceSigner()

	// set up rpc and create resourcehandler
	rh, datadir, teardownTest, err := setupTest(timeProvider, signer)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// create a new resource
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metadata := &ResourceMetadata{
		Name:      resourceName,
		Frequency: resourceFrequency,
		StartTime: Timestamp{Time: timeProvider.Now().Time},
		Owner:     signer.Address(),
	}

	mr, err := NewCreateRequest(metadata)
	if err != nil {
		t.Fatal(err)
	}
	err = rh.New(ctx, mr)
	if err != nil {
		t.Fatal(err)
	}

	// we're naïvely assuming keccak256 for swarm hashes
	// if it ever changes this test should also change
	multihashbytes := ens.EnsNode("foo")
	multihashmulti := multihash.ToMultihash(multihashbytes.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	mr.SetData(multihashmulti, true)
	mr.Sign(signer)
	if err != nil {
		t.Fatal(err)
	}
	multihashkey, err := rh.Update(ctx, &mr.SignedResourceUpdate)
	if err != nil {
		t.Fatal(err)
	}

	sha1bytes := make([]byte, multihash.MultihashLength)
	sha1multi := multihash.ToMultihash(sha1bytes)
	if err != nil {
		t.Fatal(err)
	}
	mr, err = rh.NewUpdateRequest(ctx, mr.rootAddr)
	if err != nil {
		t.Fatal(err)
	}
	mr.SetData(sha1multi, true)
	mr.Sign(signer)
	if err != nil {
		t.Fatal(err)
	}
	sha1key, err := rh.Update(ctx, &mr.SignedResourceUpdate)
	if err != nil {
		t.Fatal(err)
	}

	// invalid multihashes
	mr, err = rh.NewUpdateRequest(ctx, mr.rootAddr)
	if err != nil {
		t.Fatal(err)
	}
	mr.SetData(multihashmulti[1:], true)
	mr.Sign(signer)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rh.Update(ctx, &mr.SignedResourceUpdate)
	if err == nil {
		t.Fatalf("Expected update to fail with first byte skipped")
	}
	mr, err = rh.NewUpdateRequest(ctx, mr.rootAddr)
	if err != nil {
		t.Fatal(err)
	}
	mr.SetData(multihashmulti[:len(multihashmulti)-2], true)
	mr.Sign(signer)
	if err != nil {
		t.Fatal(err)
	}

	_, err = rh.Update(ctx, &mr.SignedResourceUpdate)
	if err == nil {
		t.Fatalf("Expected update to fail with last byte skipped")
	}

	data, err := getUpdateDirect(rh.Handler, multihashkey)
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
	data, err = getUpdateDirect(rh.Handler, sha1key)
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

	rhparams := &HandlerParams{}
	// test with signed data
	rh2, err := NewTestHandler(datadir, rhparams)
	if err != nil {
		t.Fatal(err)
	}
	mr, err = NewCreateRequest(metadata)
	if err != nil {
		t.Fatal(err)
	}
	err = rh2.New(ctx, mr)
	if err != nil {
		t.Fatal(err)
	}

	mr.SetData(multihashmulti, true)
	mr.Sign(signer)

	if err != nil {
		t.Fatal(err)
	}
	multihashsignedkey, err := rh2.Update(ctx, &mr.SignedResourceUpdate)
	if err != nil {
		t.Fatal(err)
	}

	mr, err = rh2.NewUpdateRequest(ctx, mr.rootAddr)
	if err != nil {
		t.Fatal(err)
	}
	mr.SetData(sha1multi, true)
	mr.Sign(signer)
	if err != nil {
		t.Fatal(err)
	}

	sha1signedkey, err := rh2.Update(ctx, &mr.SignedResourceUpdate)
	if err != nil {
		t.Fatal(err)
	}

	data, err = getUpdateDirect(rh2.Handler, multihashsignedkey)
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
	data, err = getUpdateDirect(rh2.Handler, sha1signedkey)
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

// \TODO verify testing of signature validation and enforcement
func TestValidator(t *testing.T) {

	// make fake timeProvider
	timeProvider := &fakeTimeProvider{
		currentTime: startTime.Time,
	}

	// signer containing private key. Alice will be the good girl
	signer := newAliceSigner()

	// fake signer for false results. Bob will play the bad guy today.
	falseSigner := newBobSigner()

	// set up  sim timeProvider
	rh, _, teardownTest, err := setupTest(timeProvider, signer)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// create new resource
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	metadata := &ResourceMetadata{
		Name:      resourceName,
		Frequency: resourceFrequency,
		StartTime: Timestamp{Time: timeProvider.Now().Time},
		Owner:     signer.Address(),
	}
	mr, err := NewCreateRequest(metadata)
	if err != nil {
		t.Fatal(err)
	}
	mr.Sign(signer)

	err = rh.New(ctx, mr)
	if err != nil {
		t.Fatalf("Create resource fail: %v", err)
	}

	// chunk with address
	data := []byte("foo")
	mr.SetData(data, false)
	if err := mr.Sign(signer); err != nil {
		t.Fatalf("sign fail: %v", err)
	}
	chunk, err := mr.SignedResourceUpdate.toChunk()
	if err != nil {
		t.Fatal(err)
	}
	if !rh.Validate(chunk.Addr, chunk.SData) {
		t.Fatal("Chunk validator fail on update chunk")
	}

	// chunk with address made from different publickey
	if err := mr.Sign(falseSigner); err == nil {
		t.Fatalf("Expected Sign to fail since we are using a different OwnerAddr: %v", err)
	}

	// chunk with address made from different publickey
	mr.metadata.Owner = zeroAddr // set to zero to bypass .Sign() check
	if err := mr.Sign(falseSigner); err != nil {
		t.Fatalf("sign fail: %v", err)
	}

	chunk, err = mr.SignedResourceUpdate.toChunk()
	if err != nil {
		t.Fatal(err)
	}

	if rh.Validate(chunk.Addr, chunk.SData) {
		t.Fatal("Chunk validator did not fail on update chunk with false address")
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	metadata = &ResourceMetadata{
		Name:      resourceName,
		StartTime: TimestampProvider.Now(),
		Frequency: resourceFrequency,
		Owner:     signer.Address(),
	}
	chunk, _, err = metadata.newChunk()
	if err != nil {
		t.Fatal(err)
	}

	if !rh.Validate(chunk.Addr, chunk.SData) {
		t.Fatal("Chunk validator fail on metadata chunk")
	}
}

// tests that the content address validator correctly checks the data
// tests that resource update chunks are passed through content address validator
// there is some redundancy in this test as it also tests content addressed chunks,
// which should be evaluated as invalid chunks by this validator
func TestValidatorInStore(t *testing.T) {

	// make fake timeProvider
	TimestampProvider = &fakeTimeProvider{
		currentTime: startTime.Time,
	}

	// signer containing private key
	signer := newAliceSigner()

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

	// set up resource handler and add is as a validator to the localstore
	rhParams := &HandlerParams{}
	rh := NewHandler(rhParams)
	store.Validators = append(store.Validators, rh)

	// create content addressed chunks, one good, one faulty
	chunks := storage.GenerateRandomChunks(chunk.DefaultSize, 2)
	goodChunk := chunks[0]
	badChunk := chunks[1]
	badChunk.SData = goodChunk.SData

	metadata := &ResourceMetadata{
		StartTime: startTime,
		Name:      "xyzzy",
		Frequency: resourceFrequency,
		Owner:     signer.Address(),
	}

	rootChunk, metaHash, err := metadata.newChunk()
	if err != nil {
		t.Fatal(err)
	}
	// create a resource update chunk with correct publickey
	updateLookup := UpdateLookup{
		period:   42,
		version:  1,
		rootAddr: rootChunk.Addr,
	}

	updateAddr := updateLookup.UpdateAddr()
	data := []byte("bar")

	r := SignedResourceUpdate{
		updateAddr: updateAddr,
		resourceUpdate: resourceUpdate{
			updateHeader: updateHeader{
				UpdateLookup: updateLookup,
				metaHash:     metaHash,
			},
			data: data,
		},
	}

	r.Sign(signer)

	uglyChunk, err := r.toChunk()
	if err != nil {
		t.Fatal(err)
	}

	// put the chunks in the store and check their error status
	storage.PutChunks(store, goodChunk)
	if goodChunk.GetErrored() == nil {
		t.Fatal("expected error on good content address chunk with resource validator only, but got nil")
	}
	storage.PutChunks(store, badChunk)
	if badChunk.GetErrored() == nil {
		t.Fatal("expected error on bad content address chunk with resource validator only, but got nil")
	}
	storage.PutChunks(store, uglyChunk)
	if err := uglyChunk.GetErrored(); err != nil {
		t.Fatalf("expected no error on resource update chunk with resource validator only, but got: %s", err)
	}
}

// fast-forward clock
func fwdClock(count int, timeProvider *fakeTimeProvider) {
	for i := 0; i < count; i++ {
		timeProvider.Tick()
	}
}

// create rpc and resourcehandler
func setupTest(timeProvider timestampProvider, signer Signer) (rh *TestHandler, datadir string, teardown func(), err error) {

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

	TimestampProvider = timeProvider
	rhparams := &HandlerParams{}
	rh, err = NewTestHandler(datadir, rhparams)
	return rh, datadir, cleanF, err
}

func newAliceSigner() *GenericSigner {
	privKey, _ := crypto.HexToECDSA("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	return NewGenericSigner(privKey)
}

func newBobSigner() *GenericSigner {
	privKey, _ := crypto.HexToECDSA("accedeaccedeaccedeaccedeaccedeaccedeaccedeaccedeaccedeaccedecaca")
	return NewGenericSigner(privKey)
}

func newCharlieSigner() *GenericSigner {
	privKey, _ := crypto.HexToECDSA("facadefacadefacadefacadefacadefacadefacadefacadefacadefacadefaca")
	return NewGenericSigner(privKey)
}

func getUpdateDirect(rh *Handler, addr storage.Address) ([]byte, error) {
	chunk, err := rh.chunkStore.Get(context.TODO(), addr)
	if err != nil {
		return nil, err
	}
	var r SignedResourceUpdate
	if err := r.fromChunk(addr, chunk.SData); err != nil {
		return nil, err
	}
	return r.data, nil
}
