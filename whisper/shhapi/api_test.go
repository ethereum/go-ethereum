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
		x.Errorf("failed to create API.")
		return
	}

	ver, err := api.Version()
	if err != nil {
		x.Errorf("failed generateFilter: %s.", err)
		return
	}

	if ver.Uint64() != whisperv5.ProtocolVersion {
		x.Errorf("wrong version: %d.", ver.Uint64())
		return
	}

	var hexnum rpc.HexNumber
	mail := api.GetFilterChanges(hexnum)
	if len(mail) != 0 {
		x.Errorf("failed GetFilterChanges")
		return
	}

	exist, err := api.HasIdentity(id)
	if err != nil {
		x.Errorf("failed 1 HasIdentity: %s.", err)
		return
	}
	if exist {
		x.Errorf("failed 2 HasIdentity: false positive.")
		return
	}

	err = api.DeleteIdentity(id)
	if err != nil {
		x.Errorf("failed 3 DeleteIdentity: %s.", err)
		return
	}

	pub, err := api.NewIdentity()
	if err != nil {
		x.Errorf("failed 4 NewIdentity: %s.", err)
		return
	}
	if len(pub) == 0 {
		x.Errorf("NewIdentity 5: empty")
		return
	}

	exist, err = api.HasIdentity(pub)
	if err != nil {
		x.Errorf("failed 6 HasIdentity: %s.", err)
		return
	}
	if !exist {
		x.Errorf("failed 7 HasIdentity: false negative.")
		return
	}

	err = api.DeleteIdentity(pub)
	if err != nil {
		x.Errorf("failed 8 DeleteIdentity: %s.", err)
		return
	}

	exist, err = api.HasIdentity(pub)
	if err != nil {
		x.Errorf("failed 9 HasIdentity: %s.", err)
		return
	}
	if exist {
		x.Errorf("failed 10 HasIdentity: false positive.")
		return
	}

	id = "arbitrary text"
	id2 := "another arbitrary string"

	exist, err = api.HasSymKey(id)
	if err != nil {
		x.Errorf("failed 11 HasSymKey: %s.", err)
		return
	}
	if exist {
		x.Errorf("failed 12 HasSymKey: false positive.")
		return
	}

	err = api.GenerateSymKey(id)
	if err != nil {
		x.Errorf("failed 13 GenerateSymKey: %s.", err)
		return
	}

	exist, err = api.HasSymKey(id)
	if err != nil {
		x.Errorf("failed 14 HasSymKey: %s.", err)
		return
	}
	if !exist {
		x.Errorf("failed 15 HasSymKey: false negative.")
		return
	}

	err = api.AddSymKey(id, []byte("some stuff here"))
	if err == nil {
		x.Errorf("failed 16 AddSymKey: %s.", err)
		return
	}

	err = api.AddSymKey(id2, []byte("some stuff here"))
	if err != nil {
		x.Errorf("failed 17 AddSymKey: %s.", err)
		return
	}

	exist, err = api.HasSymKey(id2)
	if err != nil {
		x.Errorf("failed 18 HasSymKey: %s.", err)
		return
	}
	if !exist {
		x.Errorf("failed 19 HasSymKey: false negative.")
		return
	}

	err = api.DeleteSymKey(id)
	if err != nil {
		x.Errorf("failed 20 DeleteSymKey: %s.", err)
		return
	}

	exist, err = api.HasSymKey(id)
	if err != nil {
		x.Errorf("failed 21 HasSymKey: %s.", err)
		return
	}
	if exist {
		x.Errorf("failed 22 HasSymKey: false positive.")
		return
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
		x.Errorf("failed UnmarshalJSON: %s.", err)
		return
	}

	if f.To != "0x70c87d191324e6712a591f304b4eedef6ad9bb9d" {
		x.Errorf("wrong To: %x.", f.To)
		return
	}
	if f.From != "0x9b2055d370f73ec7d8a03e965129118dc8f5bf83" {
		x.Errorf("wrong From: %x.", f.To)
		return
	}
	if f.KeyName != "testname" {
		x.Errorf("wrong KeyName: %s.", f.KeyName)
		return
	}
	if f.PoW != 2.34 {
		x.Errorf("wrong pow: %f.", f.PoW)
		return
	}
	if !f.AcceptP2P {
		x.Errorf("wrong AcceptP2P: %v.", f.AcceptP2P)
		return
	}
	if len(f.Topics) != 4 {
		x.Errorf("wrong topics number: %d.", len(f.Topics))
		return
	}

	i := 0
	if f.Topics[i] != (whisperv5.TopicType{0x00, 0x00, 0x00, 0x00}) {
		x.Errorf("wrong topic[%d]: %x.", i, f.Topics[i])
		return
	}

	i++
	if f.Topics[i] != (whisperv5.TopicType{0x00, 0x7f, 0x80, 0xff}) {
		x.Errorf("wrong topic[%d]: %x.", i, f.Topics[i])
		return
	}

	i++
	if f.Topics[i] != (whisperv5.TopicType{0xff, 0x80, 0x7f, 0x00}) {
		x.Errorf("wrong topic[%d]: %x.", i, f.Topics[i])
		return
	}

	i++
	if f.Topics[i] != (whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}) {
		x.Errorf("wrong topic[%d]: %x.", i, f.Topics[i])
		return
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
		x.Errorf("failed UnmarshalJSON: %s.", err)
		return
	}

	if a.TTL != 12345 {
		x.Errorf("wrong ttl: %d.", a.TTL)
		return
	}
	if a.From != "0x70c87d191324e6712a591f304b4eedef6ad9bb9d" {
		x.Errorf("wrong From: %x.", a.To)
		return
	}
	if a.To != "0x9b2055d370f73ec7d8a03e965129118dc8f5bf83" {
		x.Errorf("wrong To: %x.", a.To)
		return
	}
	if a.KeyName != "shh_test" {
		x.Errorf("wrong KeyName: %s.", a.KeyName)
		return
	}
	if a.Topic != (whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}) {
		x.Errorf("wrong topic: %x.", a.Topic)
		return
	}
	if string(a.Padding) != "this is my test string" {
		x.Errorf("wrong Padding: %s.", string(a.Padding))
		return
	}
	if string(a.Payload) != "payload should be pseudorandom" {
		x.Errorf("wrong Payload: %s.", string(a.Payload))
		return
	}
	if a.WorkTime != 777 {
		x.Errorf("wrong WorkTime: %d.", a.WorkTime)
		return
	}
	if a.PoW != 3.1416 {
		x.Errorf("wrong pow: %f.", a.PoW)
		return
	}
	if a.FilterID != 64 {
		x.Errorf("wrong FilterID: %d.", a.FilterID)
		return
	}
	if bytes.Compare(a.PeerID[:], a.Topic[:]) != 0 {
		x.Errorf("wrong PeerID: %x.", a.PeerID)
		return
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
		x.Errorf("failed 1 to create API.")
		return
	}

	sig, err := api.NewIdentity()
	if err != nil {
		x.Errorf("failed 22 NewIdentity: %s.", err)
		return
	}
	if len(sig) == 0 {
		x.Errorf("NewIdentity 23: empty")
		return
	}

	exist, err := api.HasIdentity(sig)
	if err != nil {
		x.Errorf("failed 24 HasIdentity: %s.", err)
		return
	}
	if !exist {
		x.Errorf("failed 25 HasIdentity: false negative.")
		return
	}

	key, err := api.NewIdentity()
	if err != nil {
		x.Errorf("failed 26 NewIdentity: %s.", err)
		return
	}
	if len(key) == 0 {
		x.Errorf("failed 27 to generate new identity")
		return
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
		x.Errorf("failed 31 to create new filter: %s.", err)
		return
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
		x.Errorf("failed 32 to post message: %s.", err)
		return
	}

	ok := waitForMessage(api, id, 1)
	if !ok {
		x.Errorf("failed 33 receive first message: timeout.")
		return
	}

	mail := api.GetFilterChanges(*id)
	if len(mail) != 1 {
		x.Errorf("failed 34 to GetFilterChanges: got %d messages.", len(mail))
		return
	}

	text := string(common.FromHex(mail[0].Payload))
	if text != string("extended test string") {
		x.Errorf("failed 35 to decrypt first message: %s.", text)
		return
	}

	p.Padding = []byte("new value")
	p.Payload = []byte("extended new value")
	err = api.Post(p)
	if err != nil {
		x.Errorf("failed 42 to post message: %s.", err)
		return
	}

	ok = waitForMessage(api, id, 2)
	if !ok {
		x.Errorf("failed 43 receive second message: timeout.")
		return
	}

	mail = api.GetFilterChanges(*id)
	if len(mail) != 1 {
		x.Errorf("failed 44 to GetFilterChanges: got %d messages.", len(mail))
		return
	}

	text = string(common.FromHex(mail[0].Payload))
	if text != string("extended new value") {
		x.Errorf("failed 45 to decrypt second message: %s.", text)
		return
	}
}

func TestIntegrationSym(x *testing.T) {
	api := NewPublicWhisperAPI()
	if api == nil {
		x.Errorf("failed 1 to create API.")
		return
	}

	keyname := "schluessel"
	err := api.GenerateSymKey(keyname)
	if err != nil {
		x.Errorf("failed 2 to GenerateSymKey: %s.", err)
		return
	}

	sig, err := api.NewIdentity()
	if err != nil {
		x.Errorf("failed 22 NewIdentity: %s.", err)
		return
	}
	if len(sig) == 0 {
		x.Errorf("NewIdentity 23: empty")
		return
	}

	exist, err := api.HasIdentity(sig)
	if err != nil {
		x.Errorf("failed 24 HasIdentity: %s.", err)
		return
	}
	if !exist {
		x.Errorf("failed 25 HasIdentity: false negative.")
		return
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
		x.Errorf("failed 31 to create new filter: %s.", err)
		return
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
		x.Errorf("failed 32 to post message: %s.", err)
		return
	}

	ok := waitForMessage(api, id, 1)
	if !ok {
		x.Errorf("failed 33 to receive first message: timeout.")
		return
	}

	mail := api.GetFilterChanges(*id)
	if len(mail) != 1 {
		x.Errorf("failed 34 to GetFilterChanges: got %d messages.", len(mail))
		return
	}

	text := string(common.FromHex(mail[0].Payload))
	if text != string("extended test string") {
		x.Errorf("failed 35 to decrypt first message: %s.", text)
		return
	}

	p.Padding = []byte("new value")
	p.Payload = []byte("extended new value")
	err = api.Post(p)
	if err != nil {
		x.Errorf("failed 42 to post message: %s.", err)
		return
	}

	ok = waitForMessage(api, id, 2)
	if !ok {
		x.Errorf("failed 43 receive second message: timeout.")
		return
	}

	mail = api.GetFilterChanges(*id)
	if len(mail) != 1 {
		x.Errorf("failed 44 to GetFilterChanges: got %d messages.", len(mail))
		return
	}

	text = string(common.FromHex(mail[0].Payload))
	if text != string("extended new value") {
		x.Errorf("failed 45 to decrypt second message: %s.", text)
		return
	}
}

func TestIntegrationSymWithFilter(x *testing.T) {
	api := NewPublicWhisperAPI()
	if api == nil {
		x.Errorf("failed 1 to create API.")
		return
	}

	keyname := "schluessel"
	err := api.GenerateSymKey(keyname)
	if err != nil {
		x.Errorf("failed 2 to GenerateSymKey: %s.", err)
		return
	}

	sig, err := api.NewIdentity()
	if err != nil {
		x.Errorf("failed 22 NewIdentity: %s.", err)
		return
	}
	if len(sig) == 0 {
		x.Errorf("NewIdentity 23: empty")
		return
	}

	exist, err := api.HasIdentity(sig)
	if err != nil {
		x.Errorf("failed 24 HasIdentity: %s.", err)
		return
	}
	if !exist {
		x.Errorf("failed 25 HasIdentity: false negative.")
		return
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
		x.Errorf("failed 31 to create new filter: %s.", err)
		return
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
		x.Errorf("failed 32 to post message: %s.", err)
		return
	}

	ok := waitForMessage(api, id, 1)
	if !ok {
		x.Errorf("failed 33 to receive first message: timeout.")
		return
	}

	mail := api.GetFilterChanges(*id)
	if len(mail) != 1 {
		x.Errorf("failed 34 to GetFilterChanges: got %d messages.", len(mail))
		return
	}

	text := string(common.FromHex(mail[0].Payload))
	if text != string("extended test string") {
		x.Errorf("failed 35 to decrypt first message: %s.", text)
		return
	}

	p.Padding = []byte("new value")
	p.Payload = []byte("extended new value")
	err = api.Post(p)
	if err != nil {
		x.Errorf("failed 42 to post message: %s.", err)
		return
	}

	ok = waitForMessage(api, id, 2)
	if !ok {
		x.Errorf("failed 43 receive second message: timeout.")
		return
	}

	mail = api.GetFilterChanges(*id)
	if len(mail) != 1 {
		x.Errorf("failed 44 to GetFilterChanges: got %d messages.", len(mail))
		return
	}

	text = string(common.FromHex(mail[0].Payload))
	if text != string("extended new value") {
		x.Errorf("failed 45 to decrypt second message: %s.", text)
		return
	}
}
