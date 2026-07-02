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
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// TokenTrieThreshold is the maximum number of non-zero token balances supported
// in list format. Accounts exceeding this limit are not supported.
//
// NOTE: goquarkchain supports a trie format (0x01 prefix + 32-byte SecureTrie root)
// for accounts with > 16 MNT tokens. That format is NOT implemented here because:
//  1. The trie format requires a SecureTrie backed by a database, which would
//     introduce an import cycle (core/types → trie → core/rawdb → core/types).
//  2. No real mainnet accounts with > 16 MNT tokens have been observed in practice.
//
// If trie-format support becomes necessary, a standalone MPT implementation must
// be added (without importing the trie package) and verified byte-for-byte against
// goquarkchain's SecureTrie output.
const TokenTrieThreshold = 16

// listFormatPrefix is the prefix byte for the list serialization format (≤ 16 tokens).
const listFormatPrefix = byte(0x00)

// trieFormatPrefix is the goquarkchain trie format prefix (> 16 tokens).
// Deserialization of this format returns an error; serialization is unsupported.
const trieFormatPrefix = byte(0x01)

// DefaultTokenID is the QKC token ID (= TokenIDEncode("QKC") = 35760).
const DefaultTokenID = uint64(35760)

// TokenBalancePair is a (TokenID, Balance) pair used in list-format encoding.
type TokenBalancePair struct {
	TokenID uint64
	Balance *uint256.Int
}

// TokenBalances holds multi-token balances for an account.
// Serialization uses list format (0x00 prefix) for ≤ TokenTrieThreshold non-zero
// balances. Trie format (> 16 tokens) is not supported; SerializeToBytes returns
// an error if more than TokenTrieThreshold non-zero balances are present.
type TokenBalances struct {
	balances map[uint64]*uint256.Int
}

// NewEmptyTokenBalances creates an empty TokenBalances.
func NewEmptyTokenBalances() *TokenBalances {
	return &TokenBalances{
		balances: make(map[uint64]*uint256.Int),
	}
}

// NewTokenBalancesWithMap creates a TokenBalances from a map.
// The provided values are deep-copied.
func NewTokenBalancesWithMap(data map[uint64]*uint256.Int) *TokenBalances {
	tb := NewEmptyTokenBalances()
	for id, bal := range data {
		if bal != nil && !bal.IsZero() {
			tb.balances[id] = new(uint256.Int).Set(bal)
		}
	}
	return tb
}

// NewTokenBalancesFromBytes deserializes a TokenBalances from its serialized form.
// Prefix 0x00 → list format (RLP-decoded, ≤ 16 tokens).
// Prefix 0x01 → trie format (> 16 tokens); NOT SUPPORTED — returns an error.
func NewTokenBalancesFromBytes(data []byte) (*TokenBalances, error) {
	if len(data) == 0 {
		return NewEmptyTokenBalances(), nil
	}
	switch data[0] {
	case listFormatPrefix:
		var pairs []TokenBalancePair
		if err := rlp.DecodeBytes(data[1:], &pairs); err != nil {
			return nil, fmt.Errorf("token_balances: list decode error: %w", err)
		}
		tb := NewEmptyTokenBalances()
		for _, p := range pairs {
			if p.Balance != nil && !p.Balance.IsZero() {
				tb.balances[p.TokenID] = new(uint256.Int).Set(p.Balance)
			}
		}
		return tb, nil

	case trieFormatPrefix:
		// Trie format (> 16 MNT tokens) is not implemented.
		// See TokenTrieThreshold comment for details.
		return nil, fmt.Errorf("token_balances: trie format (0x01, >%d tokens) is not supported", TokenTrieThreshold)

	default:
		return nil, fmt.Errorf("token_balances: unknown format prefix 0x%02x", data[0])
	}
}

// SetValue sets the balance for the given tokenID.
// A nil or zero amount removes the entry.
func (t *TokenBalances) SetValue(amount *uint256.Int, tokenID uint64) {
	if amount == nil || amount.IsZero() {
		delete(t.balances, tokenID)
		return
	}
	t.balances[tokenID] = new(uint256.Int).Set(amount)
}

// GetTokenBalance returns the balance for the given tokenID.
// Returns a new zero uint256 if not present.
func (t *TokenBalances) GetTokenBalance(tokenID uint64) *uint256.Int {
	if bal, ok := t.balances[tokenID]; ok {
		return new(uint256.Int).Set(bal)
	}
	return new(uint256.Int)
}

// GetBalanceMap returns a copy of the internal balance map.
func (t *TokenBalances) GetBalanceMap() map[uint64]*uint256.Int {
	out := make(map[uint64]*uint256.Int, len(t.balances))
	for id, bal := range t.balances {
		out[id] = new(uint256.Int).Set(bal)
	}
	return out
}

// IsBlank reports whether there are no non-zero balances.
func (t *TokenBalances) IsBlank() bool {
	return len(t.balances) == 0
}

// Len returns the number of non-zero token balances.
func (t *TokenBalances) Len() int {
	return len(t.balances)
}

// Copy returns a deep copy of the TokenBalances.
func (t *TokenBalances) Copy() *TokenBalances {
	cp := NewEmptyTokenBalances()
	for id, bal := range t.balances {
		cp.balances[id] = new(uint256.Int).Set(bal)
	}
	return cp
}

// SerializeToBytes encodes TokenBalances to bytes.
//
// List format (≤ TokenTrieThreshold non-zero balances):
//   - Output: 0x00 + RLP([]TokenBalancePair sorted by TokenID ascending)
//
// Trie format (> TokenTrieThreshold) is NOT supported and returns an error.
// See TokenTrieThreshold comment for details.
func (t *TokenBalances) SerializeToBytes() ([]byte, error) {
	if t.Len() > TokenTrieThreshold {
		// Trie format (0x01) is not implemented.
		// See TokenTrieThreshold comment for details.
		return nil, fmt.Errorf("token_balances: %d tokens exceeds the supported maximum of %d; trie format (0x01) is not implemented", t.Len(), TokenTrieThreshold)
	}
	return t.serializeListFormat()
}

// serializeListFormat encodes in list format: 0x00 + RLP(sorted pairs).
func (t *TokenBalances) serializeListFormat() ([]byte, error) {
	pairs := make([]TokenBalancePair, 0, len(t.balances))
	for id, bal := range t.balances {
		pairs = append(pairs, TokenBalancePair{
			TokenID: id,
			Balance: bal,
		})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].TokenID < pairs[j].TokenID
	})

	encoded, err := rlp.EncodeToBytes(pairs)
	if err != nil {
		return nil, fmt.Errorf("token_balances: list encode error: %w", err)
	}

	out := make([]byte, 1+len(encoded))
	out[0] = listFormatPrefix
	copy(out[1:], encoded)
	return out, nil
}
