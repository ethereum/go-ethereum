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

	var id string = "test"

	exist, err := api.HasIdentity(id)
	if err != nil {
		x.Errorf("failed HasIdentity: %s.", err)
		return
	}
	if exist {
		x.Errorf("failed HasIdentity: false positive.")
		return
	}

	exist, err = api.HasSymKey(id)
	if err != nil {
		x.Errorf("failed HasSymKey: %s.", err)
		return
	}
	if exist {
		x.Errorf("failed HasSymKey: false positive.")
		return
	}

	err = api.DeleteIdentity(id)
	if err != nil {
		x.Errorf("failed DeleteIdentity: %s.", err)
		return
	}

	pub, err := api.NewIdentity()
	if err != nil {
		x.Errorf("failed NewIdentity: %s.", err)
		return
	}
	if len(pub) == 0 {
		x.Errorf("NewIdentity: empty")
		return
	}

	//spub := string(crypto.FromECDSAPub(pub))
	//fmt.Printf("%s \n", pub)

	exist, err = api.HasIdentity(pub)
	if err != nil {
		x.Errorf("failed HasIdentity: %s.", err)
		return
	}
	if !exist {
		x.Errorf("failed HasIdentity: false negative.")
		return
	}

	err = api.DeleteIdentity(pub)
	if err != nil {
		x.Errorf("failed DeleteIdentity 2: %s.", err)
		return
	}

	exist, err = api.HasIdentity(pub)
	if err != nil {
		x.Errorf("failed HasIdentity 3: %s.", err)
		return
	}
	if exist {
		x.Errorf("failed HasIdentity 3: false positive.")
		return
	}

	var hexnum rpc.HexNumber
	mail := api.GetFilterChanges(hexnum)
	if len(mail) != 0 {
		x.Errorf("failed GetFilterChanges")
		return
	}
}
