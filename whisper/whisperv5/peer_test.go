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

import "testing"

func TestPeerBasic(x *testing.T) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		x.Errorf("failed 1 with seed %d.", seed)
		return
	}

	params.PoW = 0.001
	msg := NewSentMessage(params)
	env, err := msg.Wrap(params)
	if err != nil {
		x.Errorf("failed 2 with seed %d.", seed)
		return
	}

	p := newPeer(nil, nil, nil)
	p.mark(env)
	if !p.marked(env) {
		x.Errorf("failed 3 with seed %d.", seed)
		return
	}
}
