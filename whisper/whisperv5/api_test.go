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
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestBasic(t *testing.T) {
	var id string = "test"
	w := New()
	api := NewPublicWhisperAPI(w)
	if api == nil {
		t.Fatalf("failed to create API.")
	}

	ver, err := api.Version()
	if err != nil {
		t.Fatalf("failed generateFilter: %s.", err)
	}

	if uint64(ver) != ProtocolVersion {
		t.Fatalf("wrong version: %d.", ver)
	}

	mail := api.GetNewSubscriptionMessages("non-existent-id")
	if len(mail) != 0 {
		t.Fatalf("failed GetFilterChanges: premature result")
	}

	exist, err := api.HasKeyPair(id)
	if err != nil {
		t.Fatalf("failed initial HasIdentity: %s.", err)
	}
	if exist {
		t.Fatalf("failed initial HasIdentity: false positive.")
	}

	success, err := api.DeleteKeyPair(id)
	if err != nil {
		t.Fatalf("failed DeleteIdentity: %s.", err)
	}
	if success {
		t.Fatalf("deleted non-existing identity: false positive.")
	}

	pub, err := api.NewKeyPair()
	if err != nil {
		t.Fatalf("failed NewIdentity: %s.", err)
	}
	if len(pub) == 0 {
		t.Fatalf("failed NewIdentity: empty")
	}

	exist, err = api.HasKeyPair(pub)
	if err != nil {
		t.Fatalf("failed HasIdentity: %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasIdentity: false negative.")
	}

	success, err = api.DeleteKeyPair(pub)
	if err != nil {
		t.Fatalf("failed to delete second identity: %s.", err)
	}
	if !success {
		t.Fatalf("failed to delete second identity.")
	}

	exist, err = api.HasKeyPair(pub)
	if err != nil {
		t.Fatalf("failed HasIdentity(): %s.", err)
	}
	if exist {
		t.Fatalf("failed HasIdentity(): false positive.")
	}

	id = "arbitrary text"
	id2 := "another arbitrary string"

	exist, err = api.HasSymmetricKey(id)
	if err != nil {
		t.Fatalf("failed HasSymKey: %s.", err)
	}
	if exist {
		t.Fatalf("failed HasSymKey: false positive.")
	}

	id, err = api.GenerateSymmetricKey()
	if err != nil {
		t.Fatalf("failed GenerateSymKey: %s.", err)
	}

	exist, err = api.HasSymmetricKey(id)
	if err != nil {
		t.Fatalf("failed HasSymKey(): %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasSymKey(): false negative.")
	}

	const password = "some stuff here"
	id, err = api.AddSymmetricKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed AddSymKey: %s.", err)
	}

	id2, err = api.AddSymmetricKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed AddSymKey: %s.", err)
	}

	exist, err = api.HasSymmetricKey(id2)
	if err != nil {
		t.Fatalf("failed HasSymKey(id2): %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasSymKey(id2): false negative.")
	}

	k1, err := api.GetSymmetricKey(id)
	if err != nil {
		t.Fatalf("failed GetSymKey(id): %s.", err)
	}
	k2, err := api.GetSymmetricKey(id2)
	if err != nil {
		t.Fatalf("failed GetSymKey(id2): %s.", err)
	}

	if !bytes.Equal(k1, k2) {
		t.Fatalf("installed keys are not equal")
	}

	exist, err = api.DeleteSymmetricKey(id)
	if err != nil {
		t.Fatalf("failed DeleteSymKey(id): %s.", err)
	}
	if !exist {
		t.Fatalf("failed DeleteSymKey(id): false negative.")
	}

	exist, err = api.HasSymmetricKey(id)
	if err != nil {
		t.Fatalf("failed HasSymKey(id): %s.", err)
	}
	if exist {
		t.Fatalf("failed HasSymKey(id): false positive.")
	}
}

func TestUnmarshalFilterArgs(t *testing.T) {
	s := []byte(`{
	"type":"sym",
	"key":"0x70c87d191324e6712a591f304b4eedef6ad9bb9d",
	"sig":"0x9b2055d370f73ec7d8a03e965129118dc8f5bf83",
	"minPoW":2.34,
	"topics":["0x00000000", "0x007f80ff", "0xff807f00", "0xf26e7779"],
	"allowP2P":true
	}`)

	var f WhisperFilterArgs
	err := f.UnmarshalJSON(s)
	if err != nil {
		t.Fatalf("failed UnmarshalJSON: %s.", err)
	}

	if !f.Symmetric {
		t.Fatalf("wrong type.")
	}
	if f.Key != "0x70c87d191324e6712a591f304b4eedef6ad9bb9d" {
		t.Fatalf("wrong key: %s.", f.Key)
	}
	if f.Sig != "0x9b2055d370f73ec7d8a03e965129118dc8f5bf83" {
		t.Fatalf("wrong sig: %s.", f.Sig)
	}
	if f.MinPoW != 2.34 {
		t.Fatalf("wrong MinPoW: %f.", f.MinPoW)
	}
	if !f.AllowP2P {
		t.Fatalf("wrong AllowP2P.")
	}
	if len(f.Topics) != 4 {
		t.Fatalf("wrong topics number: %d.", len(f.Topics))
	}

	i := 0
	if !bytes.Equal(f.Topics[i], []byte{0x00, 0x00, 0x00, 0x00}) {
		t.Fatalf("wrong topic[%d]: %x.", i, f.Topics[i])
	}

	i++
	if !bytes.Equal(f.Topics[i], []byte{0x00, 0x7f, 0x80, 0xff}) {
		t.Fatalf("wrong topic[%d]: %x.", i, f.Topics[i])
	}

	i++
	if !bytes.Equal(f.Topics[i], []byte{0xff, 0x80, 0x7f, 0x00}) {
		t.Fatalf("wrong topic[%d]: %x.", i, f.Topics[i])
	}

	i++
	if !bytes.Equal(f.Topics[i], []byte{0xf2, 0x6e, 0x77, 0x79}) {
		t.Fatalf("wrong topic[%d]: %x.", i, f.Topics[i])
	}
}

func TestUnmarshalPostArgs(t *testing.T) {
	s := []byte(`{
	"type":"sym",
	"ttl":12345,
	"sig":"0x70c87d191324e6712a591f304b4eedef6ad9bb9d",
	"key":"0x9b2055d370f73ec7d8a03e965129118dc8f5bf83",
	"topic":"0xf26e7779",
	"padding":"0x74686973206973206D79207465737420737472696E67",
	"payload":"0x7061796C6F61642073686F756C642062652070736575646F72616E646F6D",
	"powTime":777,
	"powTarget":3.1416,
	"targetPeer":"enode://915533f667b1369793ebb9bda022416b1295235a1420799cd87a969467372546d808ebf59c5c9ce23f103d59b61b97df8af91f0908552485975397181b993461@127.0.0.1:12345"
	}`)

	var a PostArgs
	err := json.Unmarshal(s, &a)
	if err != nil {
		t.Fatalf("failed UnmarshalJSON: %s.", err)
	}

	if a.Type != "sym" {
		t.Fatalf("wrong Type: %s.", a.Type)
	}
	if a.TTL != 12345 {
		t.Fatalf("wrong ttl: %d.", a.TTL)
	}
	if a.Sig != "0x70c87d191324e6712a591f304b4eedef6ad9bb9d" {
		t.Fatalf("wrong From: %s.", a.Sig)
	}
	if a.Key != "0x9b2055d370f73ec7d8a03e965129118dc8f5bf83" {
		t.Fatalf("wrong Key: %s.", a.Key)
	}

	if BytesToTopic(a.Topic) != (TopicType{0xf2, 0x6e, 0x77, 0x79}) {
		t.Fatalf("wrong topic: %x.", a.Topic)
	}
	if string(a.Padding) != "this is my test string" {
		t.Fatalf("wrong Padding: %s.", string(a.Padding))
	}
	if string(a.Payload) != "payload should be pseudorandom" {
		t.Fatalf("wrong Payload: %s.", string(a.Payload))
	}
	if a.PowTime != 777 {
		t.Fatalf("wrong PowTime: %d.", a.PowTime)
	}
	if a.PowTarget != 3.1416 {
		t.Fatalf("wrong PowTarget: %f.", a.PowTarget)
	}
	if a.TargetPeer != "enode://915533f667b1369793ebb9bda022416b1295235a1420799cd87a969467372546d808ebf59c5c9ce23f103d59b61b97df8af91f0908552485975397181b993461@127.0.0.1:12345" {
		t.Fatalf("wrong PeerID: %s.", a.TargetPeer)
	}
}

func waitForMessages(api *PublicWhisperAPI, id string, target int) []*WhisperMessage {
	// timeout: 2 seconds
	result := make([]*WhisperMessage, 0, target)
	for i := 0; i < 100; i++ {
		mail := api.GetNewSubscriptionMessages(id)
		if len(mail) > 0 {
			for _, m := range mail {
				result = append(result, m)
			}
			if len(result) >= target {
				break
			}
		}
		time.Sleep(time.Millisecond * 20)
	}

	return result
}

func TestIntegrationAsym(t *testing.T) {
	w := New()
	api := NewPublicWhisperAPI(w)
	if api == nil {
		t.Fatalf("failed to create API.")
	}

	api.Start()
	defer api.Stop()

	sig, err := api.NewKeyPair()
	if err != nil {
		t.Fatalf("failed NewIdentity: %s.", err)
	}
	if len(sig) == 0 {
		t.Fatalf("wrong signature")
	}

	exist, err := api.HasKeyPair(sig)
	if err != nil {
		t.Fatalf("failed HasIdentity: %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasIdentity: false negative.")
	}

	sigPubKey, err := api.GetPublicKey(sig)
	if err != nil {
		t.Fatalf("failed GetPublicKey: %s.", err)
	}

	key, err := api.NewKeyPair()
	if err != nil {
		t.Fatalf("failed NewIdentity(): %s.", err)
	}
	if len(key) == 0 {
		t.Fatalf("wrong key")
	}

	dstPubKey, err := api.GetPublicKey(key)
	if err != nil {
		t.Fatalf("failed GetPublicKey: %s.", err)
	}

	var topics [2]TopicType
	topics[0] = TopicType{0x00, 0x64, 0x00, 0xff}
	topics[1] = TopicType{0xf2, 0x6e, 0x77, 0x79}
	var f WhisperFilterArgs
	f.Symmetric = false
	f.Key = key
	f.Sig = sigPubKey.String()
	f.Topics = make([][]byte, 2)
	f.Topics[0] = topics[0][:]
	f.Topics[1] = topics[1][:]
	f.MinPoW = DefaultMinimumPoW / 2
	f.AllowP2P = true

	id, err := api.Subscribe(f)
	if err != nil {
		t.Fatalf("failed to create new filter: %s.", err)
	}

	var p PostArgs
	p.Type = "asym"
	p.TTL = 2
	p.Sig = sig
	p.Key = dstPubKey.String()
	p.Padding = []byte("test string")
	p.Payload = []byte("extended test string")
	p.PowTarget = DefaultMinimumPoW
	p.PowTime = 2
	p.Topic = hexutil.Bytes{0xf2, 0x6e, 0x77, 0x79} // topics[1]

	err = api.Post(p)
	if err != nil {
		t.Errorf("failed to post message: %s.", err)
	}

	mail := waitForMessages(api, id, 1)
	if len(mail) != 1 {
		t.Fatalf("failed to GetFilterChanges: got %d messages.", len(mail))
	}

	text := string(common.FromHex(mail[0].Payload))
	if text != string("extended test string") {
		t.Fatalf("failed to decrypt first message: %s.", text)
	}

	p.Padding = []byte("new value")
	p.Payload = []byte("extended new value")
	err = api.Post(p)
	if err != nil {
		t.Fatalf("failed to post next message: %s.", err)
	}

	mail = waitForMessages(api, id, 1)
	if len(mail) != 1 {
		t.Fatalf("failed to GetFilterChanges: got %d messages.", len(mail))
	}

	text = string(common.FromHex(mail[0].Payload))
	if text != string("extended new value") {
		t.Fatalf("failed to decrypt second message: %s.", text)
	}
}

func TestIntegrationSym(t *testing.T) {
	w := New()
	api := NewPublicWhisperAPI(w)
	if api == nil {
		t.Fatalf("failed to create API.")
	}

	api.Start()
	defer api.Stop()

	symKeyID, err := api.GenerateSymmetricKey()
	if err != nil {
		t.Fatalf("failed GenerateSymKey: %s.", err)
	}

	sig, err := api.NewKeyPair()
	if err != nil {
		t.Fatalf("failed NewKeyPair: %s.", err)
	}
	if len(sig) == 0 {
		t.Fatalf("wrong signature")
	}

	sigPubKey, err := api.GetPublicKey(sig)
	if err != nil {
		t.Fatalf("failed GetPublicKey: %s.", err)
	}

	exist, err := api.HasKeyPair(sig)
	if err != nil {
		t.Fatalf("failed HasIdentity: %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasIdentity: false negative.")
	}

	var topics [2]TopicType
	topics[0] = TopicType{0x00, 0x7f, 0x80, 0xff}
	topics[1] = TopicType{0xf2, 0x6e, 0x77, 0x79}
	var f WhisperFilterArgs
	f.Symmetric = true
	f.Key = symKeyID
	f.Topics = make([][]byte, 2)
	f.Topics[0] = topics[0][:]
	f.Topics[1] = topics[1][:]
	f.MinPoW = DefaultMinimumPoW / 2
	f.Sig = sigPubKey.String()
	f.AllowP2P = false

	id, err := api.Subscribe(f)
	if err != nil {
		t.Fatalf("failed to create new filter: %s.", err)
	}

	var p PostArgs
	p.Type = "sym"
	p.TTL = 1
	p.Key = symKeyID
	p.Sig = sig
	p.Padding = []byte("test string")
	p.Payload = []byte("extended test string")
	p.PowTarget = DefaultMinimumPoW
	p.PowTime = 2
	p.Topic = hexutil.Bytes{0xf2, 0x6e, 0x77, 0x79}

	err = api.Post(p)
	if err != nil {
		t.Fatalf("failed to post first message: %s.", err)
	}

	mail := waitForMessages(api, id, 1)
	if len(mail) != 1 {
		t.Fatalf("failed GetFilterChanges: got %d messages.", len(mail))
	}

	text := string(common.FromHex(mail[0].Payload))
	if text != string("extended test string") {
		t.Fatalf("failed to decrypt first message: %s.", text)
	}

	p.Padding = []byte("new value")
	p.Payload = []byte("extended new value")
	err = api.Post(p)
	if err != nil {
		t.Fatalf("failed to post second message: %s.", err)
	}

	mail = waitForMessages(api, id, 1)
	if len(mail) != 1 {
		t.Fatalf("failed second GetFilterChanges: got %d messages.", len(mail))
	}

	text = string(common.FromHex(mail[0].Payload))
	if text != string("extended new value") {
		t.Fatalf("failed to decrypt second message: %s.", text)
	}
}

func TestIntegrationSymWithFilter(t *testing.T) {
	w := New()
	api := NewPublicWhisperAPI(w)
	if api == nil {
		t.Fatalf("failed to create API.")
	}

	api.Start()
	defer api.Stop()

	symKeyID, err := api.GenerateSymmetricKey()
	if err != nil {
		t.Fatalf("failed to GenerateSymKey: %s.", err)
	}

	sigKeyID, err := api.NewKeyPair()
	if err != nil {
		t.Fatalf("failed NewIdentity: %s.", err)
	}
	if len(sigKeyID) == 0 {
		t.Fatalf("wrong signature.")
	}

	exist, err := api.HasKeyPair(sigKeyID)
	if err != nil {
		t.Fatalf("failed HasIdentity: %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasIdentity: does not exist.")
	}

	sigPubKey, err := api.GetPublicKey(sigKeyID)
	if err != nil {
		t.Fatalf("failed GetPublicKey: %s.", err)
	}

	var topics [2]TopicType
	topics[0] = TopicType{0x00, 0x7f, 0x80, 0xff}
	topics[1] = TopicType{0xf2, 0x6e, 0x77, 0x79}
	var f WhisperFilterArgs
	f.Symmetric = true
	f.Key = symKeyID
	f.Topics = make([][]byte, 2)
	f.Topics[0] = topics[0][:]
	f.Topics[1] = topics[1][:]
	f.MinPoW = DefaultMinimumPoW / 2
	f.Sig = sigPubKey.String()
	f.AllowP2P = false

	id, err := api.Subscribe(f)
	if err != nil {
		t.Fatalf("failed to create new filter: %s.", err)
	}

	var p PostArgs
	p.Type = "sym"
	p.TTL = 1
	p.Key = symKeyID
	p.Sig = sigKeyID
	p.Padding = []byte("test string")
	p.Payload = []byte("extended test string")
	p.PowTarget = DefaultMinimumPoW
	p.PowTime = 2
	p.Topic = hexutil.Bytes{0xf2, 0x6e, 0x77, 0x79}

	err = api.Post(p)
	if err != nil {
		t.Fatalf("failed to post message: %s.", err)
	}

	mail := waitForMessages(api, id, 1)
	if len(mail) != 1 {
		t.Fatalf("failed to GetFilterChanges: got %d messages.", len(mail))
	}

	text := string(common.FromHex(mail[0].Payload))
	if text != string("extended test string") {
		t.Fatalf("failed to decrypt first message: %s.", text)
	}

	p.Padding = []byte("new value")
	p.Payload = []byte("extended new value")
	err = api.Post(p)
	if err != nil {
		t.Fatalf("failed to post next message: %s.", err)
	}

	mail = waitForMessages(api, id, 1)
	if len(mail) != 1 {
		t.Fatalf("failed to GetFilterChanges: got %d messages.", len(mail))
	}

	text = string(common.FromHex(mail[0].Payload))
	if text != string("extended new value") {
		t.Fatalf("failed to decrypt second message: %s.", text)
	}
}

func TestKey(t *testing.T) {
	w := New()
	api := NewPublicWhisperAPI(w)
	if api == nil {
		t.Fatalf("failed to create API.")
	}

	k, err := api.AddSymmetricKeyFromPassword("wwww")
	if err != nil {
		t.Fatalf("failed to create key: %s.", err)
	}

	s, err := api.GetSymmetricKey(k)
	if err != nil {
		t.Fatalf("failed to get sym key: %s.", err)
	}

	k2, err := api.AddSymmetricKeyDirect(s)
	if err != nil {
		t.Fatalf("failed to add sym key: %s.", err)
	}

	s2, err := api.GetSymmetricKey(k2)
	if err != nil {
		t.Fatalf("failed to get sym key: %s.", err)
	}

	if s.String() != "0x448652d595bd6ec00b2a9ea220ad6c26592d9bf4cf79023d3c1b30cb681e6e07" {
		t.Fatalf("wrong key from password: %s", s.String())
	}

	if !bytes.Equal(s, s2) {
		t.Fatalf("wrong key")
	}
}

func TestSubscribe(t *testing.T) {
	var err error
	var s string

	w := New()
	api := NewPublicWhisperAPI(w)
	if api == nil {
		t.Fatalf("failed to create API.")
	}

	symKeyID, err := api.GenerateSymmetricKey()
	if err != nil {
		t.Fatalf("failed to GenerateSymKey: %s.", err)
	}

	var f WhisperFilterArgs
	f.Symmetric = true
	f.Key = symKeyID
	f.Topics = make([][]byte, 5)
	f.Topics[0] = []byte{0x21}
	f.Topics[1] = []byte{0xd2, 0xe3}
	f.Topics[2] = []byte{0x64, 0x75, 0x76}
	f.Topics[3] = []byte{0xf8, 0xe9, 0xa0, 0xba}
	f.Topics[4] = []byte{0xcb, 0x3c, 0xdd, 0xee, 0xff}

	s, err = api.Subscribe(f)
	if err == nil {
		t.Fatalf("Subscribe: false positive.")
	}

	f.Topics[4] = []byte{}
	if err == nil {
		t.Fatalf("Subscribe: false positive again.")
	}

	f.Topics[4] = []byte{0x00}
	s, err = api.Subscribe(f)
	if err != nil {
		t.Fatalf("failed to subscribe: %s.", err)
	} else {
		api.Unsubscribe(s)
	}
}
