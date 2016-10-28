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
	"testing"

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
