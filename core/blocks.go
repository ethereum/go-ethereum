// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

package core

import "github.com/ethereum/go-ethereum/common"

// Set of manually tracked bad hashes (usually hard forks)
var BadHashes = map[common.Hash]bool{
	common.HexToHash("f269c503aed286caaa0d114d6a5320e70abbc2febe37953207e76a2873f2ba79"): true,
	common.HexToHash("38f5bbbffd74804820ffa4bab0cd540e9de229725afb98c1a7e57936f4a714bc"): true,
	common.HexToHash("7064455b364775a16afbdecd75370e912c6e2879f202eda85b9beae547fff3ac"): true,
	common.HexToHash("5b7c80070a6eff35f3eb3181edb023465c776d40af2885571e1bc4689f3a44d8"): true,
}
