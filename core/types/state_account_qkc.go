// Copyright 2024 The go-ethereum Authors
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
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// qkcAccountRLP is the wire struct for QuarkChain's 6-element account format:
// [Nonce, TokenBal(bytes), Root, CodeHash, FullShardKey(4B fixed), Optional].
// TokenBal is stored as raw serialized bytes (matching pyquarkchain's `binary` type),
// so nil encodes as 0x80 (empty string), not 0xC0 (empty list).
type qkcAccountRLP struct {
	Nonce        uint64
	TokenBal     []byte // SerializeToBytes output; nil = no balances
	Root         common.Hash
	CodeHash     []byte
	FullShardKey Uint32
	Optional     []byte
}

// mergeQKCTokenBalances combines the QKC native balance (tokenID=35760) and MNT
// balances into a single TokenBalances for wire encoding. Returns nil if both empty,
// which causes EncodeRLP to write 0x80 (RLP nil/empty) matching goquarkchain behavior.
func mergeQKCTokenBalances(balance *uint256.Int, mnt *TokenBalances) *TokenBalances {
	if (balance == nil || balance.IsZero()) && (mnt == nil || mnt.IsBlank()) {
		return nil
	}
	merged := NewEmptyTokenBalances()
	if mnt != nil {
		for id, bal := range mnt.GetBalanceMap() {
			merged.SetValue(bal, id)
		}
	}
	if balance != nil && !balance.IsZero() {
		merged.SetValue(balance, DefaultTokenID)
	}
	return merged
}

// EncodeRLP implements rlp.Encoder for StateAccount using QuarkChain's
// 6-element format. Root is always written as 32 bytes (no nil optimization).
func (acct *StateAccount) EncodeRLP(w io.Writer) error {
	var tokenBal []byte
	switch {
	case acct.MntBalances == nil && (acct.Balance == nil || acct.Balance.IsZero()):
		// No QKC balance, no MNT tokens — encode TokenBal as nil → 0x80.
		// Covers: new accounts and accounts that never entered the TokenBalances map.
		tokenBal = nil

	case acct.MntBalances != nil && acct.MntBalances.IsBlank() && acct.Balance != nil && acct.Balance.IsZero():
		// QKC balance zero, MNT map explicitly empty → re-serialize as
		// list-format with zero pairs → 0x8200c0. This preserves the
		// "account touched TokenBalances map then emptied it" history.
		tokenBal, _ = acct.MntBalances.SerializeToBytes()

	default:
		// Normal case: has QKC balance and/or non-empty MNT tokens.
		tb := mergeQKCTokenBalances(acct.Balance, acct.MntBalances)
		var err error
		tokenBal, err = tb.SerializeToBytes()
		if err != nil {
			return err
		}
	}
	qkc := &qkcAccountRLP{
		Nonce:        acct.Nonce,
		Root:         acct.Root,
		CodeHash:     acct.CodeHash,
		TokenBal:     tokenBal,
		FullShardKey: Uint32(acct.FullShardKey),
		Optional:     nil,
	}
	return rlp.Encode(w, qkc)
}

// DecodeRLP implements rlp.Decoder for StateAccount using QuarkChain's
// 6-element format.
func (acct *StateAccount) DecodeRLP(s *rlp.Stream) error {
	raw, err := s.Raw()
	if err != nil {
		return err
	}
	var qkc qkcAccountRLP
	if err := rlp.DecodeBytes(raw, &qkc); err != nil {
		return err
	}
	acct.Nonce = qkc.Nonce
	acct.CodeHash = qkc.CodeHash
	acct.Root = qkc.Root
	acct.FullShardKey = uint32(qkc.FullShardKey)
	acct.Balance = new(uint256.Int)
	if len(qkc.TokenBal) > 0 {
		tb, err := NewTokenBalancesFromBytes(qkc.TokenBal)
		if err != nil {
			return err
		}
		balMap := tb.GetBalanceMap()
		if qkcBal, ok := balMap[DefaultTokenID]; ok {
			acct.Balance.Set(qkcBal)
			delete(balMap, DefaultTokenID)
		}
		// Always set MntBalances when TokenBal has content — even if empty
		// after stripping QKC, this lets EncodeRLP produce 0x8200c0 instead
		// of 0x80, preserving byte-identical round-trip.
		acct.MntBalances = &TokenBalances{balances: balMap}
	}
	return nil
}
