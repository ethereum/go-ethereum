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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	ErrInvalidSig           = errors.New("invalid transaction v, r, s values")
	ErrUnexpectedProtection = errors.New("transaction type does not supported EIP-155 protected signatures")
	ErrInvalidTxType        = errors.New("transaction type not valid in this context")
	ErrTxTypeNotSupported   = errors.New("transaction type not supported")
)

const (
	LegacyTxId = iota
	AccessListTxId
)

type Transaction struct {
	typ   uint8     // EIP-2718 transaction type identifier
	inner inner     // Consensus contents of a transaction
	time  time.Time // Time first seen locally (spam avoidance)

	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type inner interface {
	// ChainId returns which chain id this transaction was signed for (if at all)
	ChainId() *big.Int
	// Protected returns whether the transaction is protected from replay protection.
	Protected() bool
	// AccessList returns the transactions optional EIP-2930 access list.
	AccessList() *AccessList
	Data() []byte
	Gas() uint64
	GasPrice() *big.Int
	Value() *big.Int
	Nonce() uint64
	CheckNonce() bool
	// To returns the recipient address of the transaction.
	// It returns nil if the transaction is a contract creation.
	To() *common.Address
	// RawSignatureValues returns the V, R, S signature values of the transaction.
	// The return values should not be modified by the caller.
	RawSignatureValues() (v, r, s *big.Int)
}

func (tx *Transaction) Type() uint8       { return tx.typ }
func (tx *Transaction) ChainId() *big.Int { return tx.inner.ChainId() }
func (tx *Transaction) Protected() bool   { return tx.inner.Protected() }

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28 && v != 1 && v != 0
	}
	// anything not 27 or 28 is considered protected
	return true
}

// EncodeRLP implements rlp.Encoder
// For legacy transactions, it outputs rlp(tx.inner). For typed transactions
// it outputs rlp(tx.type || tx.inner).
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	if tx.typ == LegacyTxId {
		return rlp.Encode(w, tx.inner)
	}
	buf := new(bytes.Buffer)
	if _, err := buf.Write([]byte{tx.typ}); err != nil {
		return err
	}
	if err := rlp.Encode(buf, tx.inner); err != nil {
		return err
	}
	return rlp.Encode(w, buf.Bytes())
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	var size uint64
	// If the tx isn't an RLP list, it's likely typed so pop off the first byte.
	kind, size, err := s.Kind()
	if err != nil {
		return err
	} else if kind == rlp.List {
		tx.typ = LegacyTxId
		var i *LegacyTransaction
		err = s.Decode(&i)
		tx.inner = i
	} else if kind == rlp.String {
		var b []byte
		b, err = s.Bytes()
		if err != nil {
			return err
		}

		tx.typ = b[0]
		size = uint64(len(b))

		if tx.typ == AccessListTxId {
			var i *AccessListTransaction
			err = rlp.DecodeBytes(b[1:], &i)
			tx.inner = i
		} else {
			return ErrTxTypeNotSupported
		}
	} else {
		return rlp.ErrExpectedList
	}

	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
		tx.time = time.Now()
	}

	return err
}

func sanityCheckSignature(v *big.Int, r *big.Int, s *big.Int, maybeProtected bool) error {
	if isProtectedV(v) && !maybeProtected {
		return ErrUnexpectedProtection
	}

	var plainV byte
	if isProtectedV(v) {
		chainID := deriveChainId(v).Uint64()
		plainV = byte(v.Uint64() - 35 - 2*chainID)
	} else if maybeProtected {
		// Only EIP-155 signatures can be optionally protected. Since
		// we determined this v value is not protected, it must be a
		// raw 27 or 28.
		plainV = byte(v.Uint64() - 27)
	} else {
		// If the signature is not optionally protected, we assume it
		// must already be equal to the recovery id.
		plainV = byte(v.Uint64())
	}
	if !crypto.ValidateSignatureValues(plainV, r, s, false) {
		return ErrInvalidSig
	}

	return nil
}

func (tx *Transaction) Data() []byte            { return tx.inner.Data() }
func (tx *Transaction) AccessList() *AccessList { return tx.inner.AccessList() }
func (tx *Transaction) Gas() uint64             { return tx.inner.Gas() }
func (tx *Transaction) GasPrice() *big.Int      { return new(big.Int).Set(tx.inner.GasPrice()) }
func (tx *Transaction) GasPriceCmp(other *Transaction) int {
	return tx.inner.GasPrice().Cmp(other.GasPrice())
}
func (tx *Transaction) GasPriceIntCmp(other *big.Int) int {
	return tx.inner.GasPrice().Cmp(other)
}
func (tx *Transaction) Value() *big.Int     { return new(big.Int).Set(tx.inner.Value()) }
func (tx *Transaction) Nonce() uint64       { return tx.inner.Nonce() }
func (tx *Transaction) CheckNonce() bool    { return true }
func (tx *Transaction) To() *common.Address { return tx.inner.To() }
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	n := rawtx(*tx)
	h := rlpHash(&n)
	tx.hash.Store(h)

	return h
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previously cached value.
func (tx *Transaction) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.inner)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be in the [R || S || V] format where V is 0 or 1.
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}

	var ret *Transaction
	if tx.typ == LegacyTxId {
		inner := tx.inner.(*LegacyTransaction)
		cpy := &LegacyTransaction{
			AccountNonce: inner.AccountNonce,
			Price:        inner.Price,
			GasLimit:     inner.GasLimit,
			Recipient:    inner.Recipient,
			Amount:       inner.Amount,
			Payload:      inner.Payload,

			V: inner.V,
			R: inner.R,
			S: inner.S,
		}
		cpy.R, cpy.S, cpy.V = r, s, v

		ret = &Transaction{
			typ:   LegacyTxId,
			inner: cpy,
			time:  tx.time,
		}
	} else if tx.typ == AccessListTxId {
		inner := tx.inner.(*AccessListTransaction)
		cpy := &AccessListTransaction{
			Chain:        inner.Chain,
			AccountNonce: inner.AccountNonce,
			Price:        inner.Price,
			GasLimit:     inner.GasLimit,
			Recipient:    inner.Recipient,
			Amount:       inner.Amount,
			Payload:      inner.Payload,
			Accesses:     inner.Accesses,

			V: inner.V,
			R: inner.R,
			S: inner.S,
		}
		cpy.R, cpy.S, cpy.V = r, s, v

		ret = &Transaction{
			typ:   AccessListTxId,
			inner: cpy,
			time:  tx.time,
		}

	} else {
		return nil, ErrInvalidTxType
	}

	return ret, nil
}
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas()))
	total.Add(total, tx.Value())
	return total
}
func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) { return tx.inner.RawSignatureValues() }

// Raw transactions are used for internal processes which need the raw
// consensus representation of typed transactions, not the RLP string
// wrapped version (e.g. type || payload vs. rlp(type || payload)).
type rawtx Transaction

func (tx *rawtx) EncodeRLP(w io.Writer) error {
	if tx.typ != LegacyTxId {
		if _, err := w.Write([]byte{tx.typ}); err != nil {
			return err
		}
	}

	return rlp.Encode(w, tx.inner)
}

// Transactions is a Transaction slice type for basic sorting.
type Transactions []*Transaction

// Len returns the length of s.
func (s Transactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s.
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp.
func (s Transactions) GetRlp(i int) []byte {
	raw := rawtx(*s[i])
	enc, _ := rlp.EncodeToBytes(&raw)
	return enc
}

// TxDifference returns a new set which is the difference between a and b.
func TxDifference(a, b Transactions) Transactions {
	keep := make(Transactions, 0, len(a))

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

// TxByNonce implements the sort interface to allow sorting a list of transactions
// by their nonces. This is usually only useful for sorting transactions from a
// single account, otherwise a nonce comparison doesn't make much sense.
type TxByNonce Transactions

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].Nonce() < s[j].Nonce() }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// TxByPriceAndTime implements both the sort and the heap interface, making it useful
// for all at once sorting as well as individually adding and removing elements.
type TxByPriceAndTime Transactions

func (s TxByPriceAndTime) Len() int { return len(s) }
func (s TxByPriceAndTime) Less(i, j int) bool {
	// If the prices are equal, use the time the transaction was first seen for
	// deterministic sorting
	cmp := s[i].GasPrice().Cmp(s[j].GasPrice())
	if cmp == 0 {
		return s[i].time.Before(s[j].time)
	}
	return cmp > 0
}
func (s TxByPriceAndTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s *TxByPriceAndTime) Push(x interface{}) {
	*s = append(*s, x.(*Transaction))
}

func (s *TxByPriceAndTime) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

// TransactionsByPriceAndNonce represents a set of transactions that can return
// transactions in a profit-maximizing sorted order, while supporting removing
// entire batches of transactions for non-executable accounts.
type TransactionsByPriceAndNonce struct {
	txs    map[common.Address]Transactions // Per account nonce-sorted list of transactions
	heads  TxByPriceAndTime                // Next transaction for each unique account (price heap)
	signer Signer                          // Signer for the set of transactions
}

// NewTransactionsByPriceAndNonce creates a transaction set that can retrieve
// price sorted transactions in a nonce-honouring way.
//
// Note, the input map is reowned so the caller should not interact any more with
// if after providing it to the constructor.
func NewTransactionsByPriceAndNonce(signer Signer, txs map[common.Address]Transactions) *TransactionsByPriceAndNonce {
	// Initialize a price and received time based heap with the head transactions
	heads := make(TxByPriceAndTime, 0, len(txs))
	for from, accTxs := range txs {
		heads = append(heads, accTxs[0])
		// Ensure the sender address is from the signer
		acc, _ := Sender(signer, accTxs[0])
		txs[acc] = accTxs[1:]
		if from != acc {
			delete(txs, from)
		}
	}
	heap.Init(&heads)

	// Assemble and return the transaction set
	return &TransactionsByPriceAndNonce{
		txs:    txs,
		heads:  heads,
		signer: signer,
	}
}

// Peek returns the next transaction by price.
func (t *TransactionsByPriceAndNonce) Peek() *Transaction {
	if len(t.heads) == 0 {
		return nil
	}
	return t.heads[0]
}

// Shift replaces the current best head with the next one from the same account.
func (t *TransactionsByPriceAndNonce) Shift() {
	acc, _ := Sender(t.signer, t.heads[0])
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
func (t *TransactionsByPriceAndNonce) Pop() {
	heap.Pop(&t.heads)
}

// Message is a fully derived transaction and implements core.Message
//
// NOTE: In a future PR this will be removed.
type Message struct {
	to         *common.Address
	from       common.Address
	nonce      uint64
	amount     *big.Int
	gasLimit   uint64
	gasPrice   *big.Int
	data       []byte
	accessList *AccessList
	checkNonce bool
}

func NewMessage(from common.Address, to *common.Address, nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, accessList *AccessList, checkNonce bool) Message {
	return Message{
		from:       from,
		to:         to,
		nonce:      nonce,
		amount:     amount,
		gasLimit:   gasLimit,
		gasPrice:   gasPrice,
		data:       data,
		accessList: accessList,
		checkNonce: checkNonce,
	}
}

// AsMessage returns the transaction as a core.Message.
func (tx *Transaction) AsMessage(s Signer) (Message, error) {
	msg := Message{
		nonce:      tx.Nonce(),
		gasLimit:   tx.Gas(),
		gasPrice:   new(big.Int).Set(tx.GasPrice()),
		to:         tx.To(),
		amount:     tx.Value(),
		data:       tx.Data(),
		accessList: tx.AccessList(),
		checkNonce: true,
	}

	var err error
	msg.from, err = Sender(s, tx)
	return msg, err
}

func (m Message) From() common.Address    { return m.from }
func (m Message) To() *common.Address     { return m.to }
func (m Message) GasPrice() *big.Int      { return m.gasPrice }
func (m Message) Value() *big.Int         { return m.amount }
func (m Message) Gas() uint64             { return m.gasLimit }
func (m Message) Nonce() uint64           { return m.nonce }
func (m Message) Data() []byte            { return m.data }
func (m Message) AccessList() *AccessList { return m.accessList }
func (m Message) CheckNonce() bool        { return m.checkNonce }
