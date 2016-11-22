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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
)

func TestBasic(x *testing.T) {
	var id string = "test"
	api := NewPublicWhisperAPI()
	if api == nil {
		x.Fatalf("failed to create API.")
	}

	ver, err := api.Version()
	if err != nil {
		x.Fatalf("failed generateFilter: %s.", err)
	}

	if ver.Uint64() != whisperv5.ProtocolVersion {
		x.Fatalf("wrong version: %d.", ver.Uint64())
	}

	var hexnum rpc.HexNumber
	mail := api.GetFilterChanges(hexnum)
	if len(mail) != 0 {
		x.Fatalf("failed GetFilterChanges")
	}

	exist, err := api.HasIdentity(id)
	if err != nil {
		x.Fatalf("failed test case 1: %s.", err)
	}
	if exist {
		x.Fatalf("failed test case 2, HasIdentity: false positive.")
	}

	err = api.DeleteIdentity(id)
	if err != nil {
		x.Fatalf("failed test case 3: %s.", err)
	}

	pub, err := api.NewIdentity()
	if err != nil {
		x.Fatalf("failed test case 4: %s.", err)
	}
	if len(pub) == 0 {
		x.Fatalf("test case 5, NewIdentity: empty")
	}

	exist, err = api.HasIdentity(pub)
	if err != nil {
		x.Fatalf("failed test case 6: %s.", err)
	}
	if !exist {
		x.Fatalf("failed test case 7, HasIdentity: false negative.")
	}

	err = api.DeleteIdentity(pub)
	if err != nil {
		x.Fatalf("failed 8 DeleteIdentity: %s.", err)
	}

	exist, err = api.HasIdentity(pub)
	if err != nil {
		x.Fatalf("failed test case 9: %s.", err)
	}
	if exist {
		x.Fatalf("failed test case 10, HasIdentity: false positive.")
	}

	id = "arbitrary text"
	id2 := "another arbitrary string"

	exist, err = api.HasSymKey(id)
	if err != nil {
		x.Fatalf("failed test case 11: %s.", err)
	}
	if exist {
		x.Fatalf("failed test case 12, HasSymKey: false positive.")
	}

	err = api.GenerateSymKey(id)
	if err != nil {
		x.Fatalf("failed 13 GenerateSymKey: %s.", err)
	}

	exist, err = api.HasSymKey(id)
	if err != nil {
		x.Fatalf("failed test case 14: %s.", err)
	}
	if !exist {
		x.Fatalf("failed test case 15, HasSymKey: false negative.")
	}

	err = api.AddSymKey(id, []byte("some stuff here"))
	if err == nil {
		x.Fatalf("failed test case 16: %s.", err)
	}

	err = api.AddSymKey(id2, []byte("some stuff here"))
	if err != nil {
		x.Fatalf("failed test case 17: %s.", err)
	}

	exist, err = api.HasSymKey(id2)
	if err != nil {
		x.Fatalf("failed test case 18: %s.", err)
	}
	if !exist {
		x.Fatalf("failed test case 19, HasSymKey: false negative.")
	}

	err = api.DeleteSymKey(id)
	if err != nil {
		x.Fatalf("failed test case 20: %s.", err)
	}

	exist, err = api.HasSymKey(id)
	if err != nil {
		x.Fatalf("failed test case 21: %s.", err)
	}
	if exist {
		x.Fatalf("failed test case 22, HasSymKey: false positive.")
	}
}

func TestUnmarshalFilterArgs(x *testing.T) {
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
		x.Fatalf("failed UnmarshalJSON: %s.", err)
	}

	if f.To != "0x70c87d191324e6712a591f304b4eedef6ad9bb9d" {
		x.Fatalf("wrong To: %x.", f.To)
	}
	if f.From != "0x9b2055d370f73ec7d8a03e965129118dc8f5bf83" {
		x.Fatalf("wrong From: %x.", f.To)
	}
	if f.KeyName != "testname" {
		x.Fatalf("wrong KeyName: %s.", f.KeyName)
	}
	if f.PoW != 2.34 {
		x.Fatalf("wrong pow: %f.", f.PoW)
	}
	if !f.AcceptP2P {
		x.Fatalf("wrong AcceptP2P: %v.", f.AcceptP2P)
	}
	if len(f.Topics) != 4 {
		x.Fatalf("wrong topics number: %d.", len(f.Topics))
	}

	i := 0
	if f.Topics[i] != (whisperv5.TopicType{0x00, 0x00, 0x00, 0x00}) {
		x.Fatalf("wrong topic[%d]: %x.", i, f.Topics[i])
	}

	i++
	if f.Topics[i] != (whisperv5.TopicType{0x00, 0x7f, 0x80, 0xff}) {
		x.Fatalf("wrong topic[%d]: %x.", i, f.Topics[i])
	}

	i++
	if f.Topics[i] != (whisperv5.TopicType{0xff, 0x80, 0x7f, 0x00}) {
		x.Fatalf("wrong topic[%d]: %x.", i, f.Topics[i])
	}

	i++
	if f.Topics[i] != (whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}) {
		x.Fatalf("wrong topic[%d]: %x.", i, f.Topics[i])
	}
}

func TestUnmarshalPostArgs(x *testing.T) {
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
	"filterID":"0x40",
	"peerID":"0xf26e7779"
	}`)

	var a PostArgs
	err := a.UnmarshalJSON(s)
	if err != nil {
		x.Fatalf("failed UnmarshalJSON: %s.", err)
	}

	if a.TTL != 12345 {
		x.Fatalf("wrong ttl: %d.", a.TTL)
	}
	if a.From != "0x70c87d191324e6712a591f304b4eedef6ad9bb9d" {
		x.Fatalf("wrong From: %x.", a.To)
	}
	if a.To != "0x9b2055d370f73ec7d8a03e965129118dc8f5bf83" {
		x.Fatalf("wrong To: %x.", a.To)
	}
	if a.KeyName != "shh_test" {
		x.Fatalf("wrong KeyName: %s.", a.KeyName)
	}
	if a.Topic != (whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}) {
		x.Fatalf("wrong topic: %x.", a.Topic)
	}
	if string(a.Padding) != "this is my test string" {
		x.Fatalf("wrong Padding: %s.", string(a.Padding))
	}
	if string(a.Payload) != "payload should be pseudorandom" {
		x.Fatalf("wrong Payload: %s.", string(a.Payload))
	}
	if a.WorkTime != 777 {
		x.Fatalf("wrong WorkTime: %d.", a.WorkTime)
	}
	if a.PoW != 3.1416 {
		x.Fatalf("wrong pow: %f.", a.PoW)
	}
	if a.FilterID != 64 {
		x.Fatalf("wrong FilterID: %d.", a.FilterID)
	}
	if bytes.Compare(a.PeerID[:], a.Topic[:]) != 0 {
		x.Fatalf("wrong PeerID: %x.", a.PeerID)
	}
}

func waitForMessage(api *PublicWhisperAPI, id *rpc.HexNumber, target int) bool {
	for i := 0; i < 64; i++ {
		all := api.GetMessages(*id)
		if len(all) >= target {
			return true
		}
		time.Sleep(time.Millisecond * 16)
	}

	// timeout 1024 milliseconds
	return false
}

func TestIntegrationAsym(x *testing.T) {
	api := NewPublicWhisperAPI()
	if api == nil {
		x.Fatalf("failed to create API.")
	}

	sig, err := api.NewIdentity()
	if err != nil {
		x.Fatalf("failed test case 22: %s.", err)
	}
	if len(sig) == 0 {
		x.Fatalf("failed test case 23")
	}

	exist, err := api.HasIdentity(sig)
	if err != nil {
		x.Fatalf("failed test case 24: %s.", err)
	}
	if !exist {
		x.Fatalf("failed test case 25, HasIdentity: false negative.")
	}

	key, err := api.NewIdentity()
	if err != nil {
		x.Fatalf("failed test case 26: %s.", err)
	}
	if len(key) == 0 {
		x.Fatalf("failed test case 27")
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
		x.Fatalf("failed to create new filter: %s.", err)
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
		x.Errorf("failed to post message: %s.", err)
	}

	ok := waitForMessage(api, id, 1)
	if !ok {
		x.Fatalf("failed to receive first message: timeout.")
	}

	mail := api.GetFilterChanges(*id)
	if len(mail) != 1 {
		x.Fatalf("failed to GetFilterChanges: got %d messages.", len(mail))
	}

	text := string(common.FromHex(mail[0].Payload))
	if text != string("extended test string") {
		x.Fatalf("failed to decrypt first message: %s.", text)
	}

	p.Padding = []byte("new value")
	p.Payload = []byte("extended new value")
	err = api.Post(p)
	if err != nil {
		x.Fatalf("failed to post next message: %s.", err)
	}

	ok = waitForMessage(api, id, 2)
	if !ok {
		x.Fatalf("failed to receive second message: timeout.")
	}

	mail = api.GetFilterChanges(*id)
	if len(mail) != 1 {
		x.Fatalf("failed to GetFilterChanges: got %d messages.", len(mail))
	}

	text = string(common.FromHex(mail[0].Payload))
	if text != string("extended new value") {
		x.Fatalf("failed to decrypt second message: %s.", text)
	}
}

func TestIntegrationSym(x *testing.T) {
	api := NewPublicWhisperAPI()
	if api == nil {
		x.Fatalf("failed to create API.")
	}

	keyname := "schluessel"
	err := api.GenerateSymKey(keyname)
	if err != nil {
		x.Fatalf("failed test case 1: %s.", err)
	}

	sig, err := api.NewIdentity()
	if err != nil {
		x.Fatalf("failed test case 2: %s.", err)
	}
	if len(sig) == 0 {
		x.Fatalf("failed test case 3")
	}

	exist, err := api.HasIdentity(sig)
	if err != nil {
		x.Fatalf("failed test case 4: %s.", err)
	}
	if !exist {
		x.Fatalf("failed test case 5, HasIdentity: false negative.")
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
		x.Fatalf("failed 31 to create new filter: %s.", err)
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
		x.Fatalf("failed test case 32 (post message): %s.", err)
	}

	ok := waitForMessage(api, id, 1)
	if !ok {
		x.Fatalf("failed test case 33 (receive first message: timeout).")
	}

	mail := api.GetFilterChanges(*id)
	if len(mail) != 1 {
		x.Fatalf("failed test case 34 (GetFilterChanges: got %d messages).", len(mail))
	}

	text := string(common.FromHex(mail[0].Payload))
	if text != string("extended test string") {
		x.Fatalf("failed test case 35 (decrypt first message): %s.", text)
	}

	p.Padding = []byte("new value")
	p.Payload = []byte("extended new value")
	err = api.Post(p)
	if err != nil {
		x.Fatalf("failed test case 42 (post message): %s.", err)
	}

	ok = waitForMessage(api, id, 2)
	if !ok {
		x.Fatalf("failed test case 43 (receive second message: timeout).")
	}

	mail = api.GetFilterChanges(*id)
	if len(mail) != 1 {
		x.Fatalf("failed test case 44 (GetFilterChanges: got %d messages).", len(mail))
	}

	text = string(common.FromHex(mail[0].Payload))
	if text != string("extended new value") {
		x.Fatalf("failed test case 45 (decrypt second message: %s).", text)
	}
}

func TestIntegrationSymWithFilter(x *testing.T) {
	api := NewPublicWhisperAPI()
	if api == nil {
		x.Fatalf("failed to create API.")
	}

	keyname := "schluessel"
	err := api.GenerateSymKey(keyname)
	if err != nil {
		x.Fatalf("failed to GenerateSymKey: %s.", err)
	}

	sig, err := api.NewIdentity()
	if err != nil {
		x.Fatalf("failed test case 2: %s.", err)
	}
	if len(sig) == 0 {
		x.Fatalf("failed test case 3.")
	}

	exist, err := api.HasIdentity(sig)
	if err != nil {
		x.Fatalf("failed test case 4: %s.", err)
	}
	if !exist {
		x.Fatalf("failed test case 5.")
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
		x.Fatalf("failed to create new filter: %s.", err)
	}

	var p PostArgs
	p.TTL = 1
	p.FilterID = id.Int()
	p.From = sig
	p.Padding = []byte("test string")
	p.Payload = []byte("extended test string")
	p.PoW = whisperv5.MinimumPoW
	p.Topic = whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}
	p.WorkTime = 2

	err = api.Post(p)
	if err != nil {
		x.Fatalf("failed to post message: %s.", err)
	}

	ok := waitForMessage(api, id, 1)
	if !ok {
		x.Fatalf("failed to receive first message: timeout.")
	}

	mail := api.GetFilterChanges(*id)
	if len(mail) != 1 {
		x.Fatalf("failed to GetFilterChanges: got %d messages.", len(mail))
	}

	text := string(common.FromHex(mail[0].Payload))
	if text != string("extended test string") {
		x.Fatalf("failed to decrypt first message: %s.", text)
	}

	p.Padding = []byte("new value")
	p.Payload = []byte("extended new value")
	err = api.Post(p)
	if err != nil {
		x.Fatalf("failed to post next message: %s.", err)
	}

	ok = waitForMessage(api, id, 2)
	if !ok {
		x.Fatalf("failed to receive second message: timeout.")
	}

	mail = api.GetFilterChanges(*id)
	if len(mail) != 1 {
		x.Fatalf("failed to GetFilterChanges: got %d messages.", len(mail))
	}

	text = string(common.FromHex(mail[0].Payload))
	if text != string("extended new value") {
		x.Fatalf("failed to decrypt second message: %s.", text)
	}
}
