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

package shhapi

import (
	"bytes"
	"testing"
	"time"

	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
)

func TestBasic(t *testing.T) {
	var id string = "test"
	api := NewPublicWhisperAPI()
	if api == nil {
		t.Fatalf("failed to create API.")
	}

	ver, err := api.Version()
	if err != nil {
		t.Fatalf("failed generateFilter: %s.", err)
	}

	if ver.Uint64() != whisperv5.ProtocolVersion {
		t.Fatalf("wrong version: %d.", ver.Uint64())
	}

	mail := api.GetFilterChanges(1)
	if len(mail) != 0 {
		t.Fatalf("failed GetFilterChanges: premature result")
	}

	exist, err := api.HasIdentity(id)
	if err != nil {
		t.Fatalf("failed initial HasIdentity: %s.", err)
	}
	if exist {
		t.Fatalf("failed initial HasIdentity: false positive.")
	}

	err = api.DeleteIdentity(id)
	if err != nil {
		t.Fatalf("failed DeleteIdentity: %s.", err)
	}

	pub, err := api.NewIdentity()
	if err != nil {
		t.Fatalf("failed NewIdentity: %s.", err)
	}
	if len(pub) == 0 {
		t.Fatalf("failed NewIdentity: empty")
	}

	exist, err = api.HasIdentity(pub)
	if err != nil {
		t.Fatalf("failed HasIdentity: %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasIdentity: false negative.")
	}

	err = api.DeleteIdentity(pub)
	if err != nil {
		t.Fatalf("failed to delete second identity: %s.", err)
	}

	exist, err = api.HasIdentity(pub)
	if err != nil {
		t.Fatalf("failed HasIdentity(): %s.", err)
	}
	if exist {
		t.Fatalf("failed HasIdentity(): false positive.")
	}

	id = "arbitrary text"
	id2 := "another arbitrary string"

	exist, err = api.HasSymKey(id)
	if err != nil {
		t.Fatalf("failed HasSymKey: %s.", err)
	}
	if exist {
		t.Fatalf("failed HasSymKey: false positive.")
	}

	err = api.GenerateSymKey(id)
	if err != nil {
		t.Fatalf("failed GenerateSymKey: %s.", err)
	}

	exist, err = api.HasSymKey(id)
	if err != nil {
		t.Fatalf("failed HasSymKey(): %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasSymKey(): false negative.")
	}

	err = api.AddSymKey(id, []byte("some stuff here"))
	if err == nil {
		t.Fatalf("failed AddSymKey: %s.", err)
	}

	err = api.AddSymKey(id2, []byte("some stuff here"))
	if err != nil {
		t.Fatalf("failed AddSymKey: %s.", err)
	}

	exist, err = api.HasSymKey(id2)
	if err != nil {
		t.Fatalf("failed HasSymKey(id2): %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasSymKey(id2): false negative.")
	}

	err = api.DeleteSymKey(id)
	if err != nil {
		t.Fatalf("failed DeleteSymKey(id): %s.", err)
	}

	exist, err = api.HasSymKey(id)
	if err != nil {
		t.Fatalf("failed HasSymKey(id): %s.", err)
	}
	if exist {
		t.Fatalf("failed HasSymKey(id): false positive.")
	}
}

func TestUnmarshalFilterArgs(t *testing.T) {
	s := []byte(`{
	"to":"0x70c87d191324e6712a591f304b4eedef6ad9bb9d",
	"from":"0x9b2055d370f73ec7d8a03e965129118dc8f5bf83",
	"keyname":"testname",
	"pow":2.34,
	"topics":["0x00000000", "0x007f80ff", "0xff807f00", "0xf26e7779"],
	"acceptP2P":true
	}`)

	var f WhisperFilterArgs
	err := f.UnmarshalJSON(s)
	if err != nil {
		t.Fatalf("failed UnmarshalJSON: %s.", err)
	}

	if f.To != "0x70c87d191324e6712a591f304b4eedef6ad9bb9d" {
		t.Fatalf("wrong To: %x.", f.To)
	}
	if f.From != "0x9b2055d370f73ec7d8a03e965129118dc8f5bf83" {
		t.Fatalf("wrong From: %x.", f.To)
	}
	if f.KeyName != "testname" {
		t.Fatalf("wrong KeyName: %s.", f.KeyName)
	}
	if f.PoW != 2.34 {
		t.Fatalf("wrong pow: %f.", f.PoW)
	}
	if !f.AcceptP2P {
		t.Fatalf("wrong AcceptP2P: %v.", f.AcceptP2P)
	}
	if len(f.Topics) != 4 {
		t.Fatalf("wrong topics number: %d.", len(f.Topics))
	}

	i := 0
	if f.Topics[i] != (whisperv5.TopicType{0x00, 0x00, 0x00, 0x00}) {
		t.Fatalf("wrong topic[%d]: %x.", i, f.Topics[i])
	}

	i++
	if f.Topics[i] != (whisperv5.TopicType{0x00, 0x7f, 0x80, 0xff}) {
		t.Fatalf("wrong topic[%d]: %x.", i, f.Topics[i])
	}

	i++
	if f.Topics[i] != (whisperv5.TopicType{0xff, 0x80, 0x7f, 0x00}) {
		t.Fatalf("wrong topic[%d]: %x.", i, f.Topics[i])
	}

	i++
	if f.Topics[i] != (whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}) {
		t.Fatalf("wrong topic[%d]: %x.", i, f.Topics[i])
	}
}

func TestUnmarshalPostArgs(t *testing.T) {
	s := []byte(`{
	"ttl":12345,
	"from":"0x70c87d191324e6712a591f304b4eedef6ad9bb9d",
	"to":"0x9b2055d370f73ec7d8a03e965129118dc8f5bf83",
	"keyname":"shh_test",
	"topic":"0xf26e7779",
	"padding":"0x74686973206973206D79207465737420737472696E67",
	"payload":"0x7061796C6F61642073686F756C642062652070736575646F72616E646F6D",
	"worktime":777,
	"pow":3.1416,
	"filterID":64,
	"peerID":"0xf26e7779"
	}`)

	var a PostArgs
	err := json.Unmarshal(s, &a)
	if err != nil {
		t.Fatalf("failed UnmarshalJSON: %s.", err)
	}

	if a.TTL != 12345 {
		t.Fatalf("wrong ttl: %d.", a.TTL)
	}
	if a.From != "0x70c87d191324e6712a591f304b4eedef6ad9bb9d" {
		t.Fatalf("wrong From: %x.", a.To)
	}
	if a.To != "0x9b2055d370f73ec7d8a03e965129118dc8f5bf83" {
		t.Fatalf("wrong To: %x.", a.To)
	}
	if a.KeyName != "shh_test" {
		t.Fatalf("wrong KeyName: %s.", a.KeyName)
	}
	if a.Topic != (whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}) {
		t.Fatalf("wrong topic: %x.", a.Topic)
	}
	if string(a.Padding) != "this is my test string" {
		t.Fatalf("wrong Padding: %s.", string(a.Padding))
	}
	if string(a.Payload) != "payload should be pseudorandom" {
		t.Fatalf("wrong Payload: %s.", string(a.Payload))
	}
	if a.WorkTime != 777 {
		t.Fatalf("wrong WorkTime: %d.", a.WorkTime)
	}
	if a.PoW != 3.1416 {
		t.Fatalf("wrong pow: %f.", a.PoW)
	}
	if a.FilterID != 64 {
		t.Fatalf("wrong FilterID: %d.", a.FilterID)
	}
	if bytes.Compare(a.PeerID[:], a.Topic[:]) != 0 {
		t.Fatalf("wrong PeerID: %x.", a.PeerID)
	}
}

func waitForMessage(api *PublicWhisperAPI, id uint32, target int) bool {
	for i := 0; i < 64; i++ {
		all := api.GetMessages(id)
		if len(all) >= target {
			return true
		}
		time.Sleep(time.Millisecond * 16)
	}

	// timeout 1024 milliseconds
	return false
}

func TestIntegrationAsym(t *testing.T) {
	api := NewPublicWhisperAPI()
	if api == nil {
		t.Fatalf("failed to create API.")
	}

	sig, err := api.NewIdentity()
	if err != nil {
		t.Fatalf("failed NewIdentity: %s.", err)
	}
	if len(sig) == 0 {
		t.Fatalf("wrong signature")
	}

	exist, err := api.HasIdentity(sig)
	if err != nil {
		t.Fatalf("failed HasIdentity: %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasIdentity: false negative.")
	}

	key, err := api.NewIdentity()
	if err != nil {
		t.Fatalf("failed NewIdentity(): %s.", err)
	}
	if len(key) == 0 {
		t.Fatalf("wrong key")
	}

	var topics [2]whisperv5.TopicType
	topics[0] = whisperv5.TopicType{0x00, 0x64, 0x00, 0xff}
	topics[1] = whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}
	var f WhisperFilterArgs
	f.To = key
	f.From = sig
	f.Topics = topics[:]
	f.PoW = whisperv5.MinimumPoW / 2
	f.AcceptP2P = true

	id, err := api.NewFilter(f)
	if err != nil {
		t.Fatalf("failed to create new filter: %s.", err)
	}

	var p PostArgs
	p.TTL = 2
	p.From = f.From
	p.To = f.To
	p.Padding = []byte("test string")
	p.Payload = []byte("extended test string")
	p.PoW = whisperv5.MinimumPoW
	p.Topic = whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}
	p.WorkTime = 2

	err = api.Post(p)
	if err != nil {
		t.Errorf("failed to post message: %s.", err)
	}

	ok := waitForMessage(api, id, 1)
	if !ok {
		t.Fatalf("failed to receive first message: timeout.")
	}

	mail := api.GetFilterChanges(id)
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

	ok = waitForMessage(api, id, 2)
	if !ok {
		t.Fatalf("failed to receive second message: timeout.")
	}

	mail = api.GetFilterChanges(id)
	if len(mail) != 1 {
		t.Fatalf("failed to GetFilterChanges: got %d messages.", len(mail))
	}

	text = string(common.FromHex(mail[0].Payload))
	if text != string("extended new value") {
		t.Fatalf("failed to decrypt second message: %s.", text)
	}
}

func TestIntegrationSym(t *testing.T) {
	api := NewPublicWhisperAPI()
	if api == nil {
		t.Fatalf("failed to create API.")
	}

	keyname := "schluessel"
	err := api.GenerateSymKey(keyname)
	if err != nil {
		t.Fatalf("failed GenerateSymKey: %s.", err)
	}

	sig, err := api.NewIdentity()
	if err != nil {
		t.Fatalf("failed NewIdentity: %s.", err)
	}
	if len(sig) == 0 {
		t.Fatalf("wrong signature")
	}

	exist, err := api.HasIdentity(sig)
	if err != nil {
		t.Fatalf("failed HasIdentity: %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasIdentity: false negative.")
	}

	var topics [2]whisperv5.TopicType
	topics[0] = whisperv5.TopicType{0x00, 0x7f, 0x80, 0xff}
	topics[1] = whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}
	var f WhisperFilterArgs
	f.KeyName = keyname
	f.Topics = topics[:]
	f.PoW = 0.324
	f.From = sig
	f.AcceptP2P = false

	id, err := api.NewFilter(f)
	if err != nil {
		t.Fatalf("failed to create new filter: %s.", err)
	}

	var p PostArgs
	p.TTL = 1
	p.KeyName = keyname
	p.From = f.From
	p.Padding = []byte("test string")
	p.Payload = []byte("extended test string")
	p.PoW = whisperv5.MinimumPoW
	p.Topic = whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}
	p.WorkTime = 2

	err = api.Post(p)
	if err != nil {
		t.Fatalf("failed to post first message: %s.", err)
	}

	ok := waitForMessage(api, id, 1)
	if !ok {
		t.Fatalf("failed to receive first message: timeout.")
	}

	mail := api.GetFilterChanges(id)
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

	ok = waitForMessage(api, id, 2)
	if !ok {
		t.Fatalf("failed to receive second message: timeout.")
	}

	mail = api.GetFilterChanges(id)
	if len(mail) != 1 {
		t.Fatalf("failed second GetFilterChanges: got %d messages.", len(mail))
	}

	text = string(common.FromHex(mail[0].Payload))
	if text != string("extended new value") {
		t.Fatalf("failed to decrypt second message: %s.", text)
	}
}

func TestIntegrationSymWithFilter(t *testing.T) {
	api := NewPublicWhisperAPI()
	if api == nil {
		t.Fatalf("failed to create API.")
	}

	keyname := "schluessel"
	err := api.GenerateSymKey(keyname)
	if err != nil {
		t.Fatalf("failed to GenerateSymKey: %s.", err)
	}

	sig, err := api.NewIdentity()
	if err != nil {
		t.Fatalf("failed NewIdentity: %s.", err)
	}
	if len(sig) == 0 {
		t.Fatalf("wrong signature.")
	}

	exist, err := api.HasIdentity(sig)
	if err != nil {
		t.Fatalf("failed HasIdentity: %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasIdentity: does not exist.")
	}

	var topics [2]whisperv5.TopicType
	topics[0] = whisperv5.TopicType{0x00, 0x7f, 0x80, 0xff}
	topics[1] = whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}
	var f WhisperFilterArgs
	f.KeyName = keyname
	f.Topics = topics[:]
	f.PoW = 0.324
	f.From = sig
	f.AcceptP2P = false

	id, err := api.NewFilter(f)
	if err != nil {
		t.Fatalf("failed to create new filter: %s.", err)
	}

	var p PostArgs
	p.TTL = 1
	p.FilterID = id
	p.From = sig
	p.Padding = []byte("test string")
	p.Payload = []byte("extended test string")
	p.PoW = whisperv5.MinimumPoW
	p.Topic = whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}
	p.WorkTime = 2

	err = api.Post(p)
	if err != nil {
		t.Fatalf("failed to post message: %s.", err)
	}

	ok := waitForMessage(api, id, 1)
	if !ok {
		t.Fatalf("failed to receive first message: timeout.")
	}

	mail := api.GetFilterChanges(id)
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

	ok = waitForMessage(api, id, 2)
	if !ok {
		t.Fatalf("failed to receive second message: timeout.")
	}

	mail = api.GetFilterChanges(id)
	if len(mail) != 1 {
		t.Fatalf("failed to GetFilterChanges: got %d messages.", len(mail))
	}

	text = string(common.FromHex(mail[0].Payload))
	if text != string("extended new value") {
		t.Fatalf("failed to decrypt second message: %s.", text)
	}
}
