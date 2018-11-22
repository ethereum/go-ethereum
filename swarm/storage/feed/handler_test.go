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

package feed

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/feed/lookup"
)

var (
	loglevel  = flag.Int("loglevel", 3, "loglevel")
	startTime = Timestamp{
		Time: uint64(4200),
	}
	cleanF       func()
	subtopicName = "føø.bar"
	hashfunc     = storage.MakeHashFunc(storage.DefaultHash)
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

func (f *fakeTimeProvider) Set(time uint64) {
	f.currentTime = time
}

func (f *fakeTimeProvider) FastForward(offset uint64) {
	f.currentTime += offset
}

func (f *fakeTimeProvider) Now() Timestamp {
	return Timestamp{
		Time: f.currentTime,
	}
}

// make updates and retrieve them based on periods and versions
func TestFeedsHandler(t *testing.T) {

	// make fake timeProvider
	clock := &fakeTimeProvider{
		currentTime: startTime.Time, // clock starts at t=4200
	}

	// signer containing private key
	signer := newAliceSigner()

	feedsHandler, datadir, teardownTest, err := setupTest(clock, signer)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// create a new feed
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	topic, _ := NewTopic("Mess with Swarm feeds code and see what ghost catches you", nil)
	fd := Feed{
		Topic: topic,
		User:  signer.Address(),
	}

	// data for updates:
	updates := []string{
		"blinky", // t=4200
		"pinky",  // t=4242
		"inky",   // t=4284
		"clyde",  // t=4285
	}

	request := NewFirstRequest(fd.Topic) // this timestamps the update at t = 4200 (start time)
	chunkAddress := make(map[string]storage.Address)
	data := []byte(updates[0])
	request.SetData(data)
	if err := request.Sign(signer); err != nil {
		t.Fatal(err)
	}
	chunkAddress[updates[0]], err = feedsHandler.Update(ctx, request)
	if err != nil {
		t.Fatal(err)
	}

	// move the clock ahead 21 seconds
	clock.FastForward(21) // t=4221

	request, err = feedsHandler.NewRequest(ctx, &request.Feed) // this timestamps the update at t = 4221
	if err != nil {
		t.Fatal(err)
	}
	if request.Epoch.Base() != 0 || request.Epoch.Level != lookup.HighestLevel-1 {
		t.Fatalf("Suggested epoch BaseTime should be 0 and Epoch level should be %d", lookup.HighestLevel-1)
	}

	request.Epoch.Level = lookup.HighestLevel // force level 25 instead of 24 to make it fail
	data = []byte(updates[1])
	request.SetData(data)
	if err := request.Sign(signer); err != nil {
		t.Fatal(err)
	}
	chunkAddress[updates[1]], err = feedsHandler.Update(ctx, request)
	if err == nil {
		t.Fatal("Expected update to fail since an update in this epoch already exists")
	}

	// move the clock ahead 21 seconds
	clock.FastForward(21) // t=4242
	request, err = feedsHandler.NewRequest(ctx, &request.Feed)
	if err != nil {
		t.Fatal(err)
	}
	request.SetData(data)
	if err := request.Sign(signer); err != nil {
		t.Fatal(err)
	}
	chunkAddress[updates[1]], err = feedsHandler.Update(ctx, request)
	if err != nil {
		t.Fatal(err)
	}

	// move the clock ahead 42 seconds
	clock.FastForward(42) // t=4284
	request, err = feedsHandler.NewRequest(ctx, &request.Feed)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(updates[2])
	request.SetData(data)
	if err := request.Sign(signer); err != nil {
		t.Fatal(err)
	}
	chunkAddress[updates[2]], err = feedsHandler.Update(ctx, request)
	if err != nil {
		t.Fatal(err)
	}

	// move the clock ahead 1 second
	clock.FastForward(1) // t=4285
	request, err = feedsHandler.NewRequest(ctx, &request.Feed)
	if err != nil {
		t.Fatal(err)
	}
	if request.Epoch.Base() != 0 || request.Epoch.Level != 22 {
		t.Fatalf("Expected epoch base time to be %d, got %d. Expected epoch level to be %d, got %d", 0, request.Epoch.Base(), 22, request.Epoch.Level)
	}
	data = []byte(updates[3])
	request.SetData(data)

	if err := request.Sign(signer); err != nil {
		t.Fatal(err)
	}
	chunkAddress[updates[3]], err = feedsHandler.Update(ctx, request)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)
	feedsHandler.Close()

	// check we can retrieve the updates after close
	clock.FastForward(2000) // t=6285

	feedParams := &HandlerParams{}

	feedsHandler2, err := NewTestHandler(datadir, feedParams)
	if err != nil {
		t.Fatal(err)
	}

	update2, err := feedsHandler2.Lookup(ctx, NewQueryLatest(&request.Feed, lookup.NoClue))
	if err != nil {
		t.Fatal(err)
	}

	// last update should be "clyde"
	if !bytes.Equal(update2.data, []byte(updates[len(updates)-1])) {
		t.Fatalf("feed update data was %v, expected %v", string(update2.data), updates[len(updates)-1])
	}
	if update2.Level != 22 {
		t.Fatalf("feed update epoch level was %d, expected 22", update2.Level)
	}
	if update2.Base() != 0 {
		t.Fatalf("feed update epoch base time was %d, expected 0", update2.Base())
	}
	log.Debug("Latest lookup", "epoch base time", update2.Base(), "epoch level", update2.Level, "data", update2.data)

	// specific point in time
	update, err := feedsHandler2.Lookup(ctx, NewQuery(&request.Feed, 4284, lookup.NoClue))
	if err != nil {
		t.Fatal(err)
	}
	// check data
	if !bytes.Equal(update.data, []byte(updates[2])) {
		t.Fatalf("feed update data (historical) was %v, expected %v", string(update2.data), updates[2])
	}
	log.Debug("Historical lookup", "epoch base time", update2.Base(), "epoch level", update2.Level, "data", update2.data)

	// beyond the first should yield an error
	update, err = feedsHandler2.Lookup(ctx, NewQuery(&request.Feed, startTime.Time-1, lookup.NoClue))
	if err == nil {
		t.Fatalf("expected previous to fail, returned epoch %s data %v", update.Epoch.String(), update.data)
	}

}

const Day = 60 * 60 * 24
const Year = Day * 365
const Month = Day * 30

func generateData(x uint64) []byte {
	return []byte(fmt.Sprintf("%d", x))
}

func TestSparseUpdates(t *testing.T) {

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
	defer os.RemoveAll(datadir)

	// create a new feed
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	topic, _ := NewTopic("Very slow updates", nil)
	fd := Feed{
		Topic: topic,
		User:  signer.Address(),
	}

	// publish one update every 5 years since Unix 0 until today
	today := uint64(1533799046)
	var epoch lookup.Epoch
	var lastUpdateTime uint64
	for T := uint64(0); T < today; T += 5 * Year {
		request := NewFirstRequest(fd.Topic)
		request.Epoch = lookup.GetNextEpoch(epoch, T)
		request.data = generateData(T) // this generates some data that depends on T, so we can check later
		request.Sign(signer)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := rh.Update(ctx, request); err != nil {
			t.Fatal(err)
		}
		epoch = request.Epoch
		lastUpdateTime = T
	}

	query := NewQuery(&fd, today, lookup.NoClue)

	_, err = rh.Lookup(ctx, query)
	if err != nil {
		t.Fatal(err)
	}

	_, content, err := rh.GetContent(&fd)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(generateData(lastUpdateTime), content) {
		t.Fatalf("Expected to recover last written value %d, got %s", lastUpdateTime, string(content))
	}

	// lookup the closest update to 35*Year + 6* Month (~ June 2005):
	// it should find the update we put on 35*Year, since we were updating every 5 years.

	query.TimeLimit = 35*Year + 6*Month

	_, err = rh.Lookup(ctx, query)
	if err != nil {
		t.Fatal(err)
	}

	_, content, err = rh.GetContent(&fd)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(generateData(35*Year), content) {
		t.Fatalf("Expected to recover %d, got %s", 35*Year, string(content))
	}
}

func TestValidator(t *testing.T) {

	// make fake timeProvider
	timeProvider := &fakeTimeProvider{
		currentTime: startTime.Time,
	}

	// signer containing private key. Alice will be the good girl
	signer := newAliceSigner()

	// set up  sim timeProvider
	rh, _, teardownTest, err := setupTest(timeProvider, signer)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTest()

	// create new feed
	topic, _ := NewTopic(subtopicName, nil)
	fd := Feed{
		Topic: topic,
		User:  signer.Address(),
	}
	mr := NewFirstRequest(fd.Topic)

	// chunk with address
	data := []byte("foo")
	mr.SetData(data)
	if err := mr.Sign(signer); err != nil {
		t.Fatalf("sign fail: %v", err)
	}

	chunk, err := mr.toChunk()
	if err != nil {
		t.Fatal(err)
	}
	if !rh.Validate(chunk.Address(), chunk.Data()) {
		t.Fatal("Chunk validator fail on update chunk")
	}

	address := chunk.Address()
	// mess with the address
	address[0] = 11
	address[15] = 99

	if rh.Validate(address, chunk.Data()) {
		t.Fatal("Expected Validate to fail with false chunk address")
	}
}

// tests that the content address validator correctly checks the data
// tests that feed update chunks are passed through content address validator
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
	datadir, err := ioutil.TempDir("", "storage-testfeedsvalidator")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(datadir)

	handlerParams := storage.NewDefaultLocalStoreParams()
	handlerParams.Init(datadir)
	store, err := storage.NewLocalStore(handlerParams, nil)
	if err != nil {
		t.Fatal(err)
	}

	// set up Swarm feeds handler and add is as a validator to the localstore
	fhParams := &HandlerParams{}
	fh := NewHandler(fhParams)
	store.Validators = append(store.Validators, fh)

	// create content addressed chunks, one good, one faulty
	chunks := storage.GenerateRandomChunks(chunk.DefaultSize, 2)
	goodChunk := chunks[0]
	badChunk := storage.NewChunk(chunks[1].Address(), goodChunk.Data())

	topic, _ := NewTopic("xyzzy", nil)
	fd := Feed{
		Topic: topic,
		User:  signer.Address(),
	}

	// create a feed update chunk with correct publickey
	id := ID{
		Epoch: lookup.Epoch{Time: 42,
			Level: 1,
		},
		Feed: fd,
	}

	updateAddr := id.Addr()
	data := []byte("bar")

	r := new(Request)
	r.idAddr = updateAddr
	r.Update.ID = id
	r.data = data

	r.Sign(signer)

	uglyChunk, err := r.toChunk()
	if err != nil {
		t.Fatal(err)
	}

	// put the chunks in the store and check their error status
	err = store.Put(context.Background(), goodChunk)
	if err == nil {
		t.Fatal("expected error on good content address chunk with feed update validator only, but got nil")
	}
	err = store.Put(context.Background(), badChunk)
	if err == nil {
		t.Fatal("expected error on bad content address chunk with feed update validator only, but got nil")
	}
	err = store.Put(context.Background(), uglyChunk)
	if err != nil {
		t.Fatalf("expected no error on feed update chunk with feed update validator only, but got: %s", err)
	}
}

// create rpc and feeds Handler
func setupTest(timeProvider timestampProvider, signer Signer) (fh *TestHandler, datadir string, teardown func(), err error) {

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
	datadir, err = ioutil.TempDir("", "fh")
	if err != nil {
		return nil, "", nil, err
	}
	fsClean = func() {
		os.RemoveAll(datadir)
	}

	TimestampProvider = timeProvider
	fhParams := &HandlerParams{}
	fh, err = NewTestHandler(datadir, fhParams)
	return fh, datadir, cleanF, err
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
