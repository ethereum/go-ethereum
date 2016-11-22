// Copyright 2016 The go-ethereum Authors
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

package whisperv5

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var seed int64

// InitSingleTest should be called in the beginning of every
// test, which uses RNG, in order to make the tests
// reproduciblity independent of their sequence.
func InitSingleTest() {
	seed = time.Now().Unix()
	rand.Seed(seed)
}

func InitDebugTest(i int64) {
	seed = i
	rand.Seed(seed)
}

type FilterTestCase struct {
	f      *Filter
	id     int
	alive  bool
	msgCnt int
}

func generateFilter(x *testing.T, symmetric bool) (*Filter, error) {
	var f Filter
	f.Messages = make(map[common.Hash]*ReceivedMessage)

	const topicNum = 8
	f.Topics = make([]TopicType, topicNum)
	for i := 0; i < topicNum; i++ {
		randomize(f.Topics[i][:])
		f.Topics[i][0] = 0x01
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		x.Errorf("generateFilter failed 1 with seed %d.", seed)
		return nil, err
	}
	f.Src = &key.PublicKey

	if symmetric {
		f.KeySym = make([]byte, 12)
		randomize(f.KeySym)
		f.SymKeyHash = crypto.Keccak256Hash(f.KeySym)
	} else {
		f.KeyAsym, err = crypto.GenerateKey()
		if err != nil {
			x.Errorf("generateFilter failed 2 with seed %d.", seed)
			return nil, err
		}
	}

	// AcceptP2P & PoW are not set
	return &f, nil
}

func generateTestCases(x *testing.T, SizeTestFilters int) []FilterTestCase {
	cases := make([]FilterTestCase, SizeTestFilters)
	for i := 0; i < SizeTestFilters; i++ {
		f, _ := generateFilter(x, true)
		cases[i].f = f
		cases[i].alive = (rand.Int()&int(1) == 0)
	}
	return cases
}

func TestInstallFilters(x *testing.T) {
	InitSingleTest()

	const SizeTestFilters = 256
	w := NewWhisper(nil)
	filters := NewFilters(w)
	tst := generateTestCases(x, SizeTestFilters)

	var j int
	for i := 0; i < SizeTestFilters; i++ {
		j = filters.Install(tst[i].f)
		tst[i].id = j
	}

	if j < SizeTestFilters-1 {
		x.Errorf("seed %d: wrong index %d", seed, j)
		return
	}

	for _, t := range tst {
		if !t.alive {
			filters.Uninstall(t.id)
		}
	}

	for i, t := range tst {
		fil := filters.Get(t.id)
		exist := (fil != nil)
		if exist != t.alive {
			x.Errorf("seed %d: failed alive: %d, %v, %v", seed, i, exist, t.alive)
			return
		}
		if exist && fil.PoW != t.f.PoW {
			x.Errorf("seed %d: failed Get: %d, %v, %v", seed, i, exist, t.alive)
			return
		}
	}
}

func TestComparePubKey(x *testing.T) {
	InitSingleTest()

	key1, err := crypto.GenerateKey()
	if err != nil {
		x.Errorf("failed GenerateKey 1 with seed %d: %s.", seed, err)
		return
	}
	key2, err := crypto.GenerateKey()
	if err != nil {
		x.Errorf("failed GenerateKey 2 with seed %d: %s.", seed, err)
		return
	}
	if isPubKeyEqual(&key1.PublicKey, &key2.PublicKey) {
		x.Errorf("failed !equal with seed %d.", seed)
		return
	}

	// generate key3 == key1
	rand.Seed(seed)
	key3, err := crypto.GenerateKey()
	if err != nil {
		x.Errorf("failed GenerateKey 3 with seed %d: %s.", seed, err)
		return
	}
	if isPubKeyEqual(&key1.PublicKey, &key3.PublicKey) {
		x.Errorf("failed equal with seed %d.", seed)
		return
	}
}

func TestMatchEnvelope(x *testing.T) {
	InitSingleTest()

	fsym, err := generateFilter(x, true)
	if err != nil {
		x.Errorf("failed generateFilter 1 with seed %d: %s.", seed, err)
		return
	}

	fasym, err := generateFilter(x, false)
	if err != nil {
		x.Errorf("failed generateFilter 2 with seed %d: %s.", seed, err)
		return
	}

	params, err := generateMessageParams()
	if err != nil {
		x.Errorf("failed generateMessageParams 3 with seed %d: %s.", seed, err)
		return
	}

	params.Topic[0] = 0xFF // ensure mismatch

	// mismatch with pseudo-random data
	msg := NewSentMessage(params)
	env, err := msg.Wrap(params)
	if err != nil {
		x.Errorf("failed Wrap 4 with seed %d: %s.", seed, err)
		return
	}
	match := fsym.MatchEnvelope(env)
	if match {
		x.Errorf("failed test case 5 with seed %d.", seed)
		return
	}
	match = fasym.MatchEnvelope(env)
	if match {
		x.Errorf("failed test case 6 with seed %d.", seed)
		return
	}

	// encrypt symmetrically
	i := rand.Int() % 4
	fsym.Topics[i] = params.Topic
	fasym.Topics[i] = params.Topic
	msg = NewSentMessage(params)
	env, err = msg.Wrap(params)
	if err != nil {
		x.Errorf("failed test case 7 with seed %d, test case 3: %s.", seed, err)
		return
	}

	// symmetric + matching topic: match
	match = fsym.MatchEnvelope(env)
	if !match {
		x.Errorf("failed test case 8 with seed %d.", seed)
		return
	}

	// asymmetric + matching topic: mismatch
	match = fasym.MatchEnvelope(env)
	if match {
		x.Errorf("failed test case 9 with seed %d.", seed)
		return
	}

	// symmetric + matching topic + insufficient PoW: mismatch
	fsym.PoW = env.PoW() + 1.0
	match = fsym.MatchEnvelope(env)
	if match {
		x.Errorf("failed test case 10 with seed %d.", seed)
		return
	}

	// symmetric + matching topic + sufficient PoW: match
	fsym.PoW = env.PoW() / 2
	match = fsym.MatchEnvelope(env)
	if !match {
		x.Errorf("failed test case 11 with seed %d.", seed)
		return
	}

	// symmetric + topics are nil: mismatch
	prevTopics := fsym.Topics
	fsym.Topics = nil
	match = fasym.MatchEnvelope(env)
	if match {
		x.Errorf("failed test case 12 with seed %d.", seed)
		return
	}
	fsym.Topics = prevTopics

	// encrypt asymmetrically
	key, err := crypto.GenerateKey()
	if err != nil {
		x.Errorf("failed GenerateKey 13 with seed %d: %s.", seed, err)
		return
	}
	params.KeySym = nil
	params.Dst = &key.PublicKey
	msg = NewSentMessage(params)
	env, err = msg.Wrap(params)
	if err != nil {
		x.Errorf("failed test case 14 with seed %d, test case 3: %s.", seed, err)
		return
	}

	// encryption method mismatch
	match = fsym.MatchEnvelope(env)
	if match {
		x.Errorf("failed test case 15 with seed %d.", seed)
		return
	}

	// asymmetric + mismatching topic: mismatch
	match = fasym.MatchEnvelope(env)
	if !match {
		x.Errorf("failed test case 16 with seed %d.", seed)
		return
	}

	// asymmetric + matching topic: match
	fasym.Topics[i] = fasym.Topics[i+1]
	match = fasym.MatchEnvelope(env)
	if match {
		x.Errorf("failed test case 17 with seed %d.", seed)
		return
	}

	// asymmetric + topic is nil (wildcard): match
	fasym.Topics = nil
	match = fasym.MatchEnvelope(env)
	if !match {
		x.Errorf("failed test case 18 with seed %d.", seed)
		return
	}

	// asymmetric + insufficient PoW: mismatch
	fasym.PoW = env.PoW() + 1.0
	match = fasym.MatchEnvelope(env)
	if match {
		x.Errorf("failed test case 19 with seed %d.", seed)
		return
	}

	// asymmetric + sufficient PoW: match
	fasym.PoW = env.PoW() / 2
	match = fasym.MatchEnvelope(env)
	if !match {
		x.Errorf("failed test case 20 with seed %d.", seed)
		return
	}
}

func TestMatchMessageSym(x *testing.T) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		x.Errorf("failed generateMessageParams with seed %d: %s.", seed, err)
		return
	}

	f, err := generateFilter(x, true)
	if err != nil {
		x.Errorf("failed generateFilter 1 with seed %d: %s.", seed, err)
		return
	}

	const index = 1
	params.KeySym = f.KeySym
	params.Topic = f.Topics[index]

	sentMessage := NewSentMessage(params)
	env, err := sentMessage.Wrap(params)
	if err != nil {
		x.Errorf("failed Wrap 2 with seed %d: %s.", seed, err)
		return
	}

	msg := env.Open(f)
	if msg == nil {
		x.Errorf("failed to open 3 with seed %d.", seed)
		return
	}

	// Src mismatch
	if f.MatchMessage(msg) {
		x.Errorf("failed test case 4 with seed %d.", seed)
		return
	}

	// Src: match
	*f.Src.X = *params.Src.PublicKey.X
	*f.Src.Y = *params.Src.PublicKey.Y
	if !f.MatchMessage(msg) {
		x.Errorf("failed test case 5 with seed %d.", seed)
		return
	}

	// insufficient PoW: mismatch
	f.PoW = msg.PoW + 1.0
	if f.MatchMessage(msg) {
		x.Errorf("failed test case 6 with seed %d.", seed)
		return
	}

	// sufficient PoW: match
	f.PoW = msg.PoW / 2
	if !f.MatchMessage(msg) {
		x.Errorf("failed test case 7 with seed %d.", seed)
		return
	}

	// topic mismatch
	f.Topics[index][0]++
	if f.MatchMessage(msg) {
		x.Errorf("failed test case 8 with seed %d.", seed)
		return
	}
	f.Topics[index][0]--

	// key mismatch
	f.SymKeyHash[0]++
	if f.MatchMessage(msg) {
		x.Errorf("failed test case 9 with seed %d.", seed)
		return
	}
	f.SymKeyHash[0]--

	// Src absent: match
	f.Src = nil
	if !f.MatchMessage(msg) {
		x.Errorf("failed test case 10 with seed %d.", seed)
		return
	}

	// key hash mismatch mismatch
	h := f.SymKeyHash
	f.SymKeyHash = common.Hash{}
	if f.MatchMessage(msg) {
		x.Errorf("failed test case 11 with seed %d.", seed)
		return
	}
	f.SymKeyHash = h
	if !f.MatchMessage(msg) {
		x.Errorf("failed test case 12 with seed %d.", seed)
		return
	}

	// encryption method mismatch
	f.KeySym = nil
	f.KeyAsym, err = crypto.GenerateKey()
	if err != nil {
		x.Errorf("failed GenerateKey 13 with seed %d: %s.", seed, err)
		return
	}
	if f.MatchMessage(msg) {
		x.Errorf("failed test case 14 with seed %d.", seed)
		return
	}
}

func TestMatchMessageAsym(x *testing.T) {
	InitSingleTest()

	f, err := generateFilter(x, false)
	if err != nil {
		x.Errorf("failed generateFilter with seed %d: %s.", seed, err)
		return
	}

	params, err := generateMessageParams()
	if err != nil {
		x.Errorf("failed generateMessageParams with seed %d: %s.", seed, err)
		return
	}

	const index = 1
	params.Topic = f.Topics[index]
	params.Dst = &f.KeyAsym.PublicKey
	keySymOrig := params.KeySym
	params.KeySym = nil

	sentMessage := NewSentMessage(params)
	env, err := sentMessage.Wrap(params)
	if err != nil {
		x.Errorf("failed Wrap with seed %d: %s.", seed, err)
		return
	}

	msg := env.Open(f)
	if msg == nil {
		x.Errorf("failed to open with seed %d.", seed)
		return
	}

	// Src mismatch
	if f.MatchMessage(msg) {
		x.Errorf("failed test case 4 with seed %d.", seed)
		return
	}

	// Src: match
	*f.Src.X = *params.Src.PublicKey.X
	*f.Src.Y = *params.Src.PublicKey.Y
	if !f.MatchMessage(msg) {
		x.Errorf("failed test case 5 with seed %d.", seed)
		return
	}

	// insufficient PoW: mismatch
	f.PoW = msg.PoW + 1.0
	if f.MatchMessage(msg) {
		x.Errorf("failed test case 6 with seed %d.", seed)
		return
	}

	// sufficient PoW: match
	f.PoW = msg.PoW / 2
	if !f.MatchMessage(msg) {
		x.Errorf("failed test case 7 with seed %d.", seed)
		return
	}

	// topic mismatch, but still match, because for asymmetric encryption
	// only private key matters (in case the message is already decrypted)
	f.Topics[index][0]++
	if !f.MatchMessage(msg) {
		x.Errorf("failed test case 8 with seed %d.", seed)
		return
	}
	f.Topics[index][0]--

	// key mismatch
	prev := *f.KeyAsym.PublicKey.X
	zero := *big.NewInt(0)
	*f.KeyAsym.PublicKey.X = zero
	if f.MatchMessage(msg) {
		x.Errorf("failed test case 9 with seed %d.", seed)
		return
	}
	*f.KeyAsym.PublicKey.X = prev

	// Src absent: match
	f.Src = nil
	if !f.MatchMessage(msg) {
		x.Errorf("failed test case 10 with seed %d.", seed)
		return
	}

	// encryption method mismatch
	f.KeySym = keySymOrig
	f.KeyAsym = nil
	if f.MatchMessage(msg) {
		x.Errorf("failed test case 11 with seed %d.", seed)
		return
	}
}

func cloneFilter(orig *Filter) *Filter {
	var clone Filter
	clone.Messages = make(map[common.Hash]*ReceivedMessage)
	clone.Src = orig.Src
	clone.KeyAsym = orig.KeyAsym
	clone.KeySym = orig.KeySym
	clone.Topics = orig.Topics
	clone.PoW = orig.PoW
	clone.AcceptP2P = orig.AcceptP2P
	clone.SymKeyHash = orig.SymKeyHash
	return &clone
}

func generateCompatibeEnvelope(x *testing.T, f *Filter) *Envelope {
	params, err := generateMessageParams()
	if err != nil {
		x.Errorf("failed generateMessageParams 77 with seed %d: %s.", seed, err)
		return nil
	}

	params.KeySym = f.KeySym
	params.Topic = f.Topics[2]
	sentMessage := NewSentMessage(params)
	env, err := sentMessage.Wrap(params)
	if err != nil {
		x.Errorf("failed Wrap 78 with seed %d: %s.", seed, err)
		return nil
	}
	return env
}

func TestWatchers(x *testing.T) {
	InitSingleTest()

	const NumFilters = 16
	const NumMessages = 256
	var i, j int
	var e *Envelope

	w := NewWhisper(nil)
	filters := NewFilters(w)
	tst := generateTestCases(x, NumFilters)
	for i = 0; i < NumFilters; i++ {
		tst[i].f.Src = nil
		j = filters.Install(tst[i].f)
		tst[i].id = j
	}

	last := j

	var envelopes [NumMessages]*Envelope
	for i = 0; i < NumMessages; i++ {
		j = rand.Int() % NumFilters
		e = generateCompatibeEnvelope(x, tst[j].f)
		envelopes[i] = e
		tst[j].msgCnt++
	}

	for i = 0; i < NumMessages; i++ {
		filters.NotifyWatchers(envelopes[i], messagesCode)
	}

	var total int
	var mail []*ReceivedMessage
	var count [NumFilters]int

	for i = 0; i < NumFilters; i++ {
		mail = tst[i].f.Retrieve()
		count[i] = len(mail)
		total += len(mail)
	}

	if total != NumMessages {
		x.Errorf("failed test case 1 with seed %d: total = %d, want: %d.", seed, total, NumMessages)
		return
	}

	for i = 0; i < NumFilters; i++ {
		mail = tst[i].f.Retrieve()
		if len(mail) != 0 {
			x.Errorf("failed test case 2 with seed %d: i = %d.", seed, i)
			return
		}

		if tst[i].msgCnt != count[i] {
			x.Errorf("failed test case 3 with seed %d: i = %d, get %d, want %d.", seed, i, tst[i].msgCnt, count[i])
			return
		}
	}

	// another round with a cloned filter

	clone := cloneFilter(tst[0].f)
	filters.Uninstall(last)
	total = 0
	last = NumFilters - 1
	tst[last].f = clone
	filters.Install(clone)
	for i = 0; i < NumFilters; i++ {
		tst[i].msgCnt = 0
		count[i] = 0
	}

	// make sure that the first watcher receives at least one message
	e = generateCompatibeEnvelope(x, tst[0].f)
	envelopes[0] = e
	tst[0].msgCnt++
	for i = 1; i < NumMessages; i++ {
		j = rand.Int() % NumFilters
		e = generateCompatibeEnvelope(x, tst[j].f)
		envelopes[i] = e
		tst[j].msgCnt++
	}

	for i = 0; i < NumMessages; i++ {
		filters.NotifyWatchers(envelopes[i], messagesCode)
	}

	for i = 0; i < NumFilters; i++ {
		mail = tst[i].f.Retrieve()
		count[i] = len(mail)
		total += len(mail)
	}

	combined := tst[0].msgCnt + tst[last].msgCnt
	if total != NumMessages+count[0] {
		x.Errorf("failed test case 4 with seed %d: total = %d, count[0] = %d.", seed, total, count[0])
		return
	}

	if combined != count[0] {
		x.Errorf("failed test case 5 with seed %d: combined = %d, count[0] = %d.", seed, combined, count[0])
		return
	}

	if combined != count[last] {
		x.Errorf("failed test case 6 with seed %d: combined = %d, count[last] = %d.", seed, combined, count[last])
		return
	}

	for i = 1; i < NumFilters-1; i++ {
		mail = tst[i].f.Retrieve()
		if len(mail) != 0 {
			x.Errorf("failed test case 7 with seed %d: i = %d.", seed, i)
			return
		}

		if tst[i].msgCnt != count[i] {
			x.Errorf("failed test case 8 with seed %d: i = %d, get %d, want %d.", seed, i, tst[i].msgCnt, count[i])
			return
		}
	}

	// test AcceptP2P

	total = 0
	filters.NotifyWatchers(envelopes[0], p2pCode)

	for i = 0; i < NumFilters; i++ {
		mail = tst[i].f.Retrieve()
		total += len(mail)
	}

	if total != 0 {
		x.Errorf("failed test case 9 with seed %d.", seed)
		return
	}

	f := filters.Get(0)
	f.AcceptP2P = true
	total = 0
	filters.NotifyWatchers(envelopes[0], p2pCode)

	for i = 0; i < NumFilters; i++ {
		mail = tst[i].f.Retrieve()
		total += len(mail)
	}

	if total != 1 {
		x.Errorf("failed test case 10 with seed %d: total = %d.", seed, total)
		return
	}
}
