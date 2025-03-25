// Copyright 2014 The go-ethereum Authors
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
	"bytes"
	"container/heap"
	"errors"
	"io"
	"math/big"
	"sync/atomic"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

var (
	// ErrInvalidLengdingSig invalidate signer
	ErrInvalidLengdingSig = errors.New("invalid transaction v, r, s values")
)

const (
	LendingStatusNew           = "NEW"
	LendingStatusOpen          = "OPEN"
	LendingStatusPartialFilled = "PARTIAL_FILLED"
	LendingStatusFilled        = "FILLED"
	LendingStatusCancelled     = "CANCELLED"
	LendingTypeMo              = "MO"
	LendingTypeLo              = "LO"
	LendingSideBorrow          = "BORROW"
	LendingSideInvest          = "INVEST"
	LendingRePay               = "REPAY"
	LendingTopup               = "TOPUP"
)

// LendingTransaction lending transaction
type LendingTransaction struct {
	data lendingtxdata
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type lendingtxdata struct {
	AccountNonce    uint64         `json:"nonce"    gencodec:"required"`
	Quantity        *big.Int       `json:"quantity,omitempty"`
	Interest        uint64         `json:"interest"`
	RelayerAddress  common.Address `json:"relayerAddress,omitempty"`
	UserAddress     common.Address `json:"userAddress,omitempty"`
	CollateralToken common.Address `json:"collateralToken,omitempty"`
	AutoTopUp       bool           `json:"autoTopUp,omitempty"`
	LendingToken    common.Address `json:"lendingToken,omitempty"`
	Term            uint64         `json:"term"`
	Status          string         `json:"status,omitempty"`
	Side            string         `json:"side,omitempty"`
	Type            string         `json:"type,omitempty"`
	LendingId       uint64         `json:"lendingId,omitempty"`
	LendingTradeId  uint64         `json:"tradeId,omitempty"`
	ExtraData       string         `json:"extraData,omitempty"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash common.Hash `json:"hash"`
}

// IsCreatedLending check if tx is cancelled transaction
func (tx *LendingTransaction) IsCreatedLending() bool {
	return (tx.IsLoTypeLending() || tx.IsMoTypeLending()) && tx.Status() == LendingStatusNew
}

// IsCancelledLending check if tx is cancelled transaction
func (tx *LendingTransaction) IsCancelledLending() bool {
	return tx.Status() == LendingStatusCancelled
}

// IsRepayLending check if tx is repay lending transaction
func (tx *LendingTransaction) IsRepayLending() bool {
	return tx.Type() == LendingRePay
}

// IsTopupLending check if tx is repay lending transaction
func (tx *LendingTransaction) IsTopupLending() bool {
	return tx.Type() == LendingTopup
}

// IsMoTypeLending check if tx type is MO lending
func (tx *LendingTransaction) IsMoTypeLending() bool {
	return tx.Type() == LendingTypeMo
}

// IsLoTypeLending check if tx type is LO lending
func (tx *LendingTransaction) IsLoTypeLending() bool {
	return tx.Type() == LendingTypeLo
}

// EncodeRLP implements rlp.Encoder
func (tx *LendingTransaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *LendingTransaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}

	return err
}

// Nonce return nonce of account
func (tx *LendingTransaction) Nonce() uint64 { return tx.data.AccountNonce }

// Quantity return quantity of transaction
func (tx *LendingTransaction) Quantity() *big.Int { return tx.data.Quantity }

// RelayerAddress return relayer address transaction
func (tx *LendingTransaction) RelayerAddress() common.Address { return tx.data.RelayerAddress }

// UserAddress return user address transaction
func (tx *LendingTransaction) UserAddress() common.Address { return tx.data.UserAddress }

// Interest return interest percent of transaction
func (tx *LendingTransaction) Interest() uint64 { return tx.data.Interest }

// Duration return period of transaction
func (tx *LendingTransaction) Duration() uint64 { return tx.data.Term }

// Term return period of transaction
func (tx *LendingTransaction) Term() uint64 { return tx.data.Term }

// CollateralToken return collateral token address
func (tx *LendingTransaction) CollateralToken() common.Address { return tx.data.CollateralToken }

// CollateralToken return autoTopUp flag
func (tx *LendingTransaction) AutoTopUp() bool { return tx.data.AutoTopUp }

// LendingToken return lending token address of transaction
func (tx *LendingTransaction) LendingToken() common.Address { return tx.data.LendingToken }

// Status return status of lending transaction
func (tx *LendingTransaction) Status() string { return tx.data.Status }

// Side return side of lending transaction
func (tx *LendingTransaction) Side() string { return tx.data.Side }

// Type return type of lending transaction
func (tx *LendingTransaction) Type() string { return tx.data.Type }

// Type return extraData of lending transaction
func (tx *LendingTransaction) ExtraData() string { return tx.data.ExtraData }

// Signature return signature of lending transaction
func (tx *LendingTransaction) Signature() (V, R, S *big.Int) { return tx.data.V, tx.data.R, tx.data.S }

// LendingHash return hash of lending transaction
func (tx *LendingTransaction) LendingHash() common.Hash { return tx.data.Hash }

// LendingId return lending id
func (tx *LendingTransaction) LendingId() uint64 { return tx.data.LendingId }

// LendingId return lendingTradeId
func (tx *LendingTransaction) LendingTradeId() uint64 { return tx.data.LendingTradeId }

// SetLendingHash set hash of lending transaction hash
func (tx *LendingTransaction) SetLendingHash(h common.Hash) { tx.data.Hash = h }

// From get transaction from
func (tx *LendingTransaction) From() *common.Address {
	if tx.data.V != nil {
		signer := LendingTxSigner{}
		if f, err := LendingSender(signer, tx); err != nil {
			return nil
		} else {
			return &f
		}

	}
	return nil

}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be formatted as described in the yellow paper (v+27).
func (tx *LendingTransaction) WithSignature(signer LendingSigner, sig []byte) (*LendingTransaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &LendingTransaction{data: tx.data}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}

// ImportSignature make lending tx with specific signature
func (tx *LendingTransaction) ImportSignature(V, R, S *big.Int) *LendingTransaction {

	if V != nil {
		tx.data.V = V
	}
	if R != nil {
		tx.data.R = R
	}
	if S != nil {
		tx.data.S = S
	}
	return tx
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (tx *LendingTransaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

// CacheHash cache hash
func (tx *LendingTransaction) CacheHash() {
	v := rlpHash(tx)
	tx.hash.Store(v)
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previsouly cached value.
func (tx *LendingTransaction) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.data)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// NewLendingTransaction init lending from value
func NewLendingTransaction(nonce uint64, quantity *big.Int, interest, duration uint64, relayerAddress, userAddress, lendingToken, collateralToken common.Address, autoTopUp bool, status, side, typeLending string, hash common.Hash, id, tradeId uint64, extraData string) *LendingTransaction {
	return newLendingTransaction(nonce, quantity, interest, duration, relayerAddress, userAddress, lendingToken, collateralToken, autoTopUp, status, side, typeLending, hash, id, tradeId, extraData)
}

func newLendingTransaction(nonce uint64, quantity *big.Int, interest, duration uint64, relayerAddress, userAddress, lendingToken, collateralToken common.Address, autoTopUp bool, status, side, typeLending string, hash common.Hash, id, tradeId uint64, extraData string) *LendingTransaction {
	d := lendingtxdata{
		AccountNonce:    nonce,
		Quantity:        new(big.Int),
		Interest:        interest,
		Term:            duration,
		RelayerAddress:  relayerAddress,
		UserAddress:     userAddress,
		LendingToken:    lendingToken,
		CollateralToken: collateralToken,
		AutoTopUp:       autoTopUp,
		Status:          status,
		Side:            side,
		Type:            typeLending,
		Hash:            hash,
		LendingId:       id,
		LendingTradeId:  tradeId,
		ExtraData:       extraData,
		V:               new(big.Int),
		R:               new(big.Int),
		S:               new(big.Int),
	}
	if quantity != nil {
		d.Quantity.Set(quantity)
	}

	return &LendingTransaction{data: d}
}

// LendingTransactions is a Transaction slice type for basic sorting.
type LendingTransactions []*LendingTransaction

// Len returns the length of s.
func (s LendingTransactions) Len() int { return len(s) }

// EncodeIndex encodes the i'th element of s to w.
func (s LendingTransactions) EncodeIndex(i int, w *bytes.Buffer) {
	rlp.Encode(w, s[i])
}

// LendingTxDifference returns a new set t which is the difference between a to b.
func LendingTxDifference(a, b LendingTransactions) (keep LendingTransactions) {
	keep = make(LendingTransactions, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

// LendingTxByNonce sorted lending by nonce defined
type LendingTxByNonce LendingTransactions

func (s LendingTxByNonce) Len() int           { return len(s) }
func (s LendingTxByNonce) Less(i, j int) bool { return s[i].data.AccountNonce < s[j].data.AccountNonce }

func (s LendingTxByNonce) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s *LendingTxByNonce) Push(x interface{}) {
	*s = append(*s, x.(*LendingTransaction))
}

func (s *LendingTxByNonce) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

// LendingTransactionByNonce sort transaction by nonce
type LendingTransactionByNonce struct {
	txs    map[common.Address]LendingTransactions
	heads  LendingTxByNonce
	signer LendingSigner
}

// NewLendingTransactionByNonce sort transaction by nonce
func NewLendingTransactionByNonce(signer LendingSigner, txs map[common.Address]LendingTransactions) *LendingTransactionByNonce {
	// Initialize a price based heap with the head transactions
	heads := make(LendingTxByNonce, 0, len(txs))
	for from, accTxs := range txs {
		heads = append(heads, accTxs[0])
		// Ensure the sender address is from the signer
		acc, _ := LendingSender(signer, accTxs[0])
		txs[acc] = accTxs[1:]
		if from != acc {
			delete(txs, from)
		}
	}
	heap.Init(&heads)

	// Assemble and return the transaction set
	return &LendingTransactionByNonce{
		txs:    txs,
		heads:  heads,
		signer: signer,
	}
}

// Peek returns the next transaction by price.
func (t *LendingTransactionByNonce) Peek() *LendingTransaction {
	if len(t.heads) == 0 {
		return nil
	}
	return t.heads[0]
}

// Shift replaces the current best head with the next one from the same account.
func (t *LendingTransactionByNonce) Shift() {
	acc, _ := LendingSender(t.signer, t.heads[0])
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		t.heads[0], t.txs[acc] = txs[0], txs[1:]
		heap.Fix(&t.heads, 0)
	} else {
		heap.Pop(&t.heads)
	}
}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (t *LendingTransactionByNonce) Pop() {
	heap.Pop(&t.heads)
}
