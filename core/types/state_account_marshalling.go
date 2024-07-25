// Copyright 2021 The go-ethereum Authors
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

package types

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/iden3/go-iden3-crypto/utils"

	zkt "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
)

var (
	ErrInvalidLength = errors.New("StateAccount: invalid input length")
)

// MarshalFields marshalls a StateAccount into a sequence of bytes. The bytes scheme is:
// [0:32] (bytes in big-endian)
//
//	[0:16] Reserved with all 0
//	[16:24] CodeSize, uint64 in big-endian
//	[24:32] Nonce, uint64 in big-endian
//
// [32:64] Balance
// [64:96] StorageRoot
// [96:128] KeccakCodeHash
// [128:160] PoseidonCodehash
// (total 160 bytes)
func (s *StateAccount) MarshalFields() ([]zkt.Byte32, uint32) {
	fields := make([]zkt.Byte32, 5)

	if s.Balance == nil {
		panic("StateAccount balance nil")
	}

	if !utils.CheckBigIntInField(s.Balance) {
		panic("StateAccount balance overflow")
	}

	if !utils.CheckBigIntInField(s.Root.Big()) {
		panic("StateAccount root overflow")
	}

	if !utils.CheckBigIntInField(new(big.Int).SetBytes(s.PoseidonCodeHash)) {
		panic("StateAccount poseidonCodeHash overflow")
	}

	binary.BigEndian.PutUint64(fields[0][16:], s.CodeSize)
	binary.BigEndian.PutUint64(fields[0][24:], s.Nonce)
	s.Balance.FillBytes(fields[1][:])
	copy(fields[2][:], s.Root.Bytes())
	copy(fields[3][:], s.KeccakCodeHash)
	copy(fields[4][:], s.PoseidonCodeHash)

	// The returned flag shows which items cannot be encoded as field elements.
	// KeccakCodeHash can be larger than the field size so we set the 3rd (LSB) bit to 1.
	//
	// +---+---+---+---+---+
	// | 0 | 1 | 2 | 3 | 4 |
	// +---+---+---+---+---+
	//   0   0   0   1   0

	flag := uint32(8)

	return fields, flag
}

func UnmarshalStateAccount(bytes []byte) (*StateAccount, error) {
	if len(bytes) != 160 {
		return nil, ErrInvalidLength
	}

	acc := new(StateAccount)

	acc.CodeSize = binary.BigEndian.Uint64(bytes[16:24])
	acc.Nonce = binary.BigEndian.Uint64(bytes[24:32])
	acc.Balance = new(big.Int).SetBytes(bytes[32:64])

	acc.Root = common.Hash{}
	acc.Root.SetBytes(bytes[64:96])

	acc.KeccakCodeHash = make([]byte, 32)
	copy(acc.KeccakCodeHash, bytes[96:128])

	acc.PoseidonCodeHash = make([]byte, 32)
	copy(acc.PoseidonCodeHash, bytes[128:160])

	return acc, nil
}
