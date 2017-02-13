// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package mailserver

import (
	"crypto/ecdsa"
	"encoding/binary"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

const powRequirement = 0.00001
const keyName = "6d604bac5401ce9a6b995f1b45a4ab"

var shh *whisper.Whisper
var seed = time.Now().Unix()

type ServerTestParams struct {
	topic whisper.TopicType
	low   uint32
	upp   uint32
	key   *ecdsa.PrivateKey
}

func assert(statement bool, text string, t *testing.T) {
	if !statement {
		t.Fatal(text)
	}
}

func TestDBKey(t *testing.T) {
	var h common.Hash
	i := uint32(time.Now().Unix())
	k := NewDbKey(i, h)
	assert(len(k.raw) == common.HashLength+4, "wrong DB key length", t)
	assert(byte(i%0x100) == k.raw[3], "raw representation should be big endian", t)
	assert(byte(i/0x1000000) == k.raw[0], "big endian expected", t)
}

func generateEnvelope(t *testing.T) *whisper.Envelope {
	params := &whisper.MessageParams{
		KeySym:   []byte("test key"),
		Topic:    whisper.TopicType{},
		Payload:  []byte("test payload"),
		PoW:      powRequirement,
		WorkTime: 2,
	}

	msg := whisper.NewSentMessage(params)
	env, err := msg.Wrap(params)
	if err != nil {
		t.Fatalf("failed to wrap with seed %d: %s.", seed, err)
	}
	return env
}

func TestMailServer(t *testing.T) {
	const password = "password_for_this_test"
	const dbPath = "whisper-server-test"

	_, err := ioutil.TempDir("", dbPath)
	if err != nil {
		t.Fatal(err)
	}

	var server WMailServer
	shh = whisper.NewWhisper(&server)
	server.Init(shh, dbPath, password, powRequirement)
	defer server.Close()

	err = shh.AddSymKey(keyName, []byte(password))
	if err != nil {
		t.Fatalf("Failed to create symmetric key for mail request: %s", err)
	}

	rand.Seed(seed)
	env := generateEnvelope(t)
	server.Archive(env)
	deliverTest(t, &server, env)
}

func deliverTest(t *testing.T, server *WMailServer, env *whisper.Envelope) {
	testPeerID := shh.NewIdentity()
	birth := env.Expiry - env.TTL
	p := &ServerTestParams{
		topic: env.Topic,
		low:   birth - 1,
		upp:   birth + 1,
		key:   testPeerID,
	}
	singleRequest(t, server, env, p, true)

	p.low, p.upp = birth+1, 0xffffffff
	singleRequest(t, server, env, p, false)

	p.low, p.upp = 0, birth-1
	singleRequest(t, server, env, p, false)

	p.low = birth - 1
	p.upp = birth + 1
	p.topic[0]++
	singleRequest(t, server, env, p, false)
}

func singleRequest(t *testing.T, server *WMailServer, env *whisper.Envelope, p *ServerTestParams, expect bool) {
	request := createRequest(t, p)
	src := crypto.FromECDSAPub(&p.key.PublicKey)
	ok, lower, upper, topic := server.validateRequest(src, request)
	if !ok {
		t.Fatalf("request validation failed, seed: %d.", seed)
	}
	if lower != p.low {
		t.Fatalf("request validation failed (lower bound), seed: %d.", seed)
	}
	if upper != p.upp {
		t.Fatalf("request validation failed (upper bound), seed: %d.", seed)
	}
	if topic != p.topic {
		t.Fatalf("request validation failed (topic), seed: %d.", seed)
	}

	var exist bool
	mail := server.processRequest(nil, p.low, p.upp, p.topic)
	for _, msg := range mail {
		if msg.Hash() == env.Hash() {
			exist = true
			break
		}
	}

	if exist != expect {
		t.Fatalf("error: exist = %v, seed: %d.", exist, seed)
	}

	src[0]++
	ok, lower, upper, topic = server.validateRequest(src, request)
	if ok {
		t.Fatalf("request validation false positive, seed: %d.", seed)
	}
}

func createRequest(t *testing.T, p *ServerTestParams) *whisper.Envelope {
	data := make([]byte, 8+whisper.TopicLength)
	binary.BigEndian.PutUint32(data, p.low)
	binary.BigEndian.PutUint32(data[4:], p.upp)
	copy(data[8:], p.topic[:])

	params := &whisper.MessageParams{
		KeySym:   shh.GetSymKey(keyName),
		Topic:    p.topic,
		Payload:  data,
		PoW:      powRequirement * 2,
		WorkTime: 2,
		Src:      p.key,
	}

	msg := whisper.NewSentMessage(params)
	env, err := msg.Wrap(params)
	if err != nil {
		t.Fatalf("failed to wrap with seed %d: %s.", seed, err)
	}
	return env
}
