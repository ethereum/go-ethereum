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
	"container/heap"
	"errors"
	"io"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate gencodec -type txdata -field-override txdataMarshaling -out gen_tx_json.go

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
)

type Transaction struct {
	data txdata
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type txdata struct {
	AccountNonce uint64          `json:"nonce"    gencodec:"required"`
	Price        *big.Int        `json:"gasPrice" rlp:"nil"`
	GasLimit     uint64          `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"    gencodec:"required"`
	Payload      []byte          `json:"input"    gencodec:"required"`

	// EIP1559 gas values
	GasPremium *big.Int `json:"gasPremium" rlp:"nil"` // nil means legacy transaction
	FeeCap     *big.Int `json:"feeCap"     rlp:"nil"` // nil means legacy transaction

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash *common.Hash `json:"hash" rlp:"-"`
}

type txdataMarshaling struct {
	AccountNonce hexutil.Uint64
	Price        *hexutil.Big
	GasLimit     hexutil.Uint64
	Amount       *hexutil.Big
	Payload      hexutil.Bytes
	GasPremium   *hexutil.Big
	FeeCap       *hexutil.Big
	V            *hexutil.Big
	R            *hexutil.Big
	S            *hexutil.Big
}

func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, gasPremium, feeCap *big.Int) *Transaction {
	return newTransaction(nonce, &to, amount, gasLimit, gasPrice, data, gasPremium, feeCap)
}

func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, gasPremium, feeCap *big.Int) *Transaction {
	return newTransaction(nonce, nil, amount, gasLimit, gasPrice, data, gasPremium, feeCap)
}

func newTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, gasPremium, feeCap *big.Int) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := txdata{
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price = gasPrice
	}
	if gasPremium != nil {
		d.GasPremium = gasPremium
	}
	if feeCap != nil {
		d.FeeCap = feeCap
	}
	if gasPremium != nil {
		d.GasPremium = gasPremium
	}
	if feeCap != nil {
		d.FeeCap = feeCap
	}

	return &Transaction{data: d}
}

// ChainId returns which chain id this transaction was signed for (if at all)
func (tx *Transaction) ChainId() *big.Int {
	return deriveChainId(tx.data.V)
}

// Protected returns whether the transaction is protected from replay protection.
func (tx *Transaction) Protected() bool {
	return isProtectedV(tx.data.V)
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28
	}
	// anything not 27 or 28 is considered protected
	return true
}

// EncodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	if tx.data.FeeCap == nil || tx.data.GasPremium == nil {
		return rlp.Encode(w, []interface{}{
			tx.data.AccountNonce,
			tx.data.Price,
			tx.data.GasLimit,
			tx.data.Recipient,
			tx.data.Amount,
			tx.data.Payload,
			tx.data.V,
			tx.data.R,
			tx.data.S,
		})
	}
	return rlp.Encode(w, &tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(stream *rlp.Stream) error {
	size, err := stream.List()
	if err != nil {
		return err
	}
	accountNonce := new(uint64)
	if err = stream.Decode(accountNonce); err != nil {
		return err
	}
	price := new(big.Int)
	if err = stream.Decode(price); err != nil {
		return err
	}
	gasLimit := new(uint64)
	if err = stream.Decode(gasLimit); err != nil {
		return err
	}
	_, recipientSize, err := stream.Kind()
	if err != nil {
		return err
	}
	var recipient *common.Address
	// the below is to handle the "rlp: nil" tag (tag itself is not needed anymore because of this manual handling)
	// attempting to unpack a zero value into *common.Address throws an error
	// if there is a non-zero address, unpack it
	if recipientSize != 0 {
		recipient = new(common.Address)
		if err = stream.Decode(recipient); err != nil {
			return err
		}
	} else {
		// otherwise if the value is of size zero throw away the value, move to next value in the stream, and leave recipient nil
		if _, err = stream.Raw(); err != nil {
			return err
		}
	}
	amount := new(big.Int)
	if err = stream.Decode(amount); err != nil {
		return err
	}
	payload := new([]byte)
	if err = stream.Decode(payload); err != nil {
		return err
	}
	gasPremium := new(big.Int)
	if err = stream.Decode(gasPremium); err != nil {
		return err
	}
	feeCap := new(big.Int)
	if err = stream.Decode(feeCap); err != nil {
		return err
	}
	v := new(big.Int)
	if err = stream.Decode(v); err != nil {
		return err
	}
	// if this is the end of the list then we are decoding a legacy transaction
	// so the decoded gasPremium, feeCap, and v values are shifted into the v, r, and s values
	if err = stream.ListEnd(); err == nil {
		tx.data = txdata{
			AccountNonce: *accountNonce,
			Price:        price,
			GasLimit:     *gasLimit,
			Recipient:    recipient,
			Amount:       amount,
			Payload:      *payload,
			V:            gasPremium,
			R:            feeCap,
			S:            v,
		}
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
		return nil
	}
	// if we are not at the end of the list, continue decoding the 1559 transaction fields
	if err != rlp.ErrNotAtEOL {
		return err
	}
	r := new(big.Int)
	if err := stream.Decode(r); err != nil {
		return err
	}
	s := new(big.Int)
	if err := stream.Decode(s); err != nil {
		return err
	}
	// we should now be at the end of the list for a EIP1559 transaction
	if err = stream.ListEnd(); err != nil {
		return err
	}
	tx.data = txdata{
		AccountNonce: *accountNonce,
		Price:        nil,
		GasLimit:     *gasLimit,
		Recipient:    recipient,
		Amount:       amount,
		Payload:      *payload,
		GasPremium:   gasPremium,
		FeeCap:       feeCap,
		V:            v,
		R:            r,
		S:            s,
	}
	tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	return nil
}

// MarshalJSON encodes the web3 RPC transaction format.
func (tx *Transaction) MarshalJSON() ([]byte, error) {
	hash := tx.Hash()
	data := tx.data
	data.Hash = &hash
	return data.MarshalJSON()
}

// UnmarshalJSON decodes the web3 RPC transaction format.
func (tx *Transaction) UnmarshalJSON(input []byte) error {
	var dec txdata
	if err := dec.UnmarshalJSON(input); err != nil {
		return err
	}

	withSignature := dec.V.Sign() != 0 || dec.R.Sign() != 0 || dec.S.Sign() != 0
	if withSignature {
		var V byte
		if isProtectedV(dec.V) {
			chainID := deriveChainId(dec.V).Uint64()
			V = byte(dec.V.Uint64() - 35 - 2*chainID)
		} else {
			V = byte(dec.V.Uint64() - 27)
		}
		if !crypto.ValidateSignatureValues(V, dec.R, dec.S, false) {
			return ErrInvalidSig
		}
	}

	*tx = Transaction{data: dec}
	return nil
}

func (tx *Transaction) Data() []byte         { return common.CopyBytes(tx.data.Payload) }
func (tx *Transaction) Gas() uint64          { return tx.data.GasLimit }
func (tx *Transaction) GasPrice() *big.Int   { return tx.data.Price }
func (tx *Transaction) Value() *big.Int      { return new(big.Int).Set(tx.data.Amount) }
func (tx *Transaction) Nonce() uint64        { return tx.data.AccountNonce }
func (tx *Transaction) CheckNonce() bool     { return true }
func (tx *Transaction) GasPremium() *big.Int { return tx.data.GasPremium }
func (tx *Transaction) FeeCap() *big.Int     { return tx.data.FeeCap }

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *Transaction) To() *common.Address {
	if tx.data.Recipient == nil {
		return nil
	}
	to := *tx.data.Recipient
	return &to
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previsouly cached value.
func (tx *Transaction) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.data)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// AsMessage returns the transaction as a core.Message.
//
// AsMessage requires a signer to derive the sender.
//
// XXX Rename message to something less arbitrary?
func (tx *Transaction) AsMessage(s Signer) (Message, error) {
	msg := Message{
		nonce:      tx.data.AccountNonce,
		gasLimit:   tx.data.GasLimit,
		gasPrice:   tx.data.Price,
		to:         tx.data.Recipient,
		amount:     tx.data.Amount,
		data:       tx.data.Payload,
		checkNonce: true,
		gasPremium: tx.data.GasPremium,
		feeCap:     tx.data.FeeCap,
	}

	var err error
	msg.from, err = Sender(s, tx)
	return msg, err
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be in the [R || S || V] format where V is 0 or 1.
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &Transaction{data: tx.data}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost(baseFee *big.Int) *big.Int {
	if tx.data.Price != nil {
		total := new(big.Int).Mul(tx.data.Price, new(big.Int).SetUint64(tx.data.GasLimit))
		total.Add(total, tx.data.Amount)
		return total
	}
	if baseFee != nil && tx.data.GasPremium != nil && tx.data.FeeCap != nil {
		eip1559GasPrice := new(big.Int).Add(baseFee, tx.data.GasPremium)
		if eip1559GasPrice.Cmp(tx.data.FeeCap) > 0 {
			eip1559GasPrice.Set(tx.data.FeeCap)
		}
		total := new(big.Int).Mul(eip1559GasPrice, new(big.Int).SetUint64(tx.data.GasLimit))
		total.Add(total, tx.data.Amount)
		return total
	}
	return nil
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}

// Transactions is a Transaction slice type for basic sorting.
type Transactions []*Transaction

// Len returns the length of s.
func (s Transactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s.
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp.
func (s Transactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
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
func (s TxByNonce) Less(i, j int) bool { return s[i].data.AccountNonce < s[j].data.AccountNonce }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// TxByPrice implements both the sort and the heap interface, making it useful
// for all at once sorting as well as individually adding and removing elements.
type TxByPrice struct {
	txs     Transactions
	baseFee *big.Int
}

func (s TxByPrice) Len() int { return len(s.txs) }

// Note that this returns true if j is less than i, as the ordering needs to be from highest price to lowest
func (s TxByPrice) Less(i, j int) bool {
	iPrice := s.txs[i].data.Price
	jPrice := s.txs[j].data.Price
	if iPrice == nil {
		iPrice = new(big.Int).Add(s.baseFee, s.txs[i].data.GasPremium)
		if iPrice.Cmp(s.txs[i].data.FeeCap) > 0 {
			iPrice.Set(s.txs[i].data.FeeCap)
		}
	}
	if jPrice == nil {
		jPrice = new(big.Int).Add(s.baseFee, s.txs[j].data.GasPremium)
		if jPrice.Cmp(s.txs[j].data.FeeCap) > 0 {
			jPrice.Set(s.txs[j].data.FeeCap)
		}
	}
	return iPrice.Cmp(jPrice) > 0
}

func (s *TxByPrice) Swap(i, j int) { s.txs[i], s.txs[j] = s.txs[j], s.txs[i] }

func (s *TxByPrice) Push(x interface{}) {
	s.txs = append(s.txs, x.(*Transaction))
}

func (s *TxByPrice) Pop() interface{} {
	old := s.txs
	n := len(old)
	x := old[n-1]
	s.txs = old[0 : n-1]
	return x
}

// TransactionsByPriceAndNonce represents a set of transactions that can return
// transactions in a profit-maximizing sorted order, while supporting removing
// entire batches of transactions for non-executable accounts.
type TransactionsByPriceAndNonce struct {
	txs    map[common.Address]Transactions // Per account nonce-sorted list of transactions
	heads  *TxByPrice                      // Next transaction for each unique account (price heap)
	signer Signer                          // Signer for the set of transactions
}

// NewTransactionsByPriceAndNonce creates a transaction set that can retrieve
// price sorted transactions in a nonce-honouring way.
//
// Note, the input map is reowned so the caller should not interact any more with
// if after providing it to the constructor.
func NewTransactionsByPriceAndNonce(signer Signer, txs map[common.Address]Transactions, baseFee *big.Int) *TransactionsByPriceAndNonce {
	// Initialize a price based heap with the head transactions
	heads := new(TxByPrice)
	heads.txs = make(Transactions, 0, len(txs))
	heads.baseFee = baseFee
	for from, accTxs := range txs {
		heads.txs = append(heads.txs, accTxs[0])
		// Ensure the sender address is from the signer
		acc, _ := Sender(signer, accTxs[0])
		txs[acc] = accTxs[1:]
		if from != acc {
			delete(txs, from)
		}
	}
	heap.Init(heads)

	// Assemble and return the transaction set
	return &TransactionsByPriceAndNonce{
		txs:    txs,
		heads:  heads,
		signer: signer,
	}
}

// Peek returns the next transaction by price.
func (t *TransactionsByPriceAndNonce) Peek() *Transaction {
	if len(t.heads.txs) == 0 {
		return nil
	}
	return t.heads.txs[0]
}

// Shift replaces the current best head with the next one from the same account.
func (t *TransactionsByPriceAndNonce) Shift() {
	acc, _ := Sender(t.signer, t.heads.txs[0])
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		t.heads.txs[0], t.txs[acc] = txs[0], txs[1:]
		heap.Fix(t.heads, 0)
	} else {
		heap.Pop(t.heads)
	}
}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (t *TransactionsByPriceAndNonce) Pop() {
	heap.Pop(t.heads)
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
	checkNonce bool
	gasPremium *big.Int
	feeCap     *big.Int
}

// NewMessage creates and returns a new message
func NewMessage(from common.Address, to *common.Address, nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, checkNonce bool, gasPremium, feeCap *big.Int) Message {
	return Message{
		from:       from,
		to:         to,
		nonce:      nonce,
		amount:     amount,
		gasLimit:   gasLimit,
		gasPrice:   gasPrice,
		data:       data,
		checkNonce: checkNonce,
		gasPremium: gasPremium,
		feeCap:     feeCap,
	}
}

func (m Message) From() common.Address { return m.from }
func (m Message) To() *common.Address  { return m.to }
func (m Message) GasPrice() *big.Int   { return m.gasPrice }
func (m Message) Value() *big.Int      { return m.amount }
func (m Message) Gas() uint64          { return m.gasLimit }
func (m Message) Nonce() uint64        { return m.nonce }
func (m Message) Data() []byte         { return m.data }
func (m Message) CheckNonce() bool     { return m.checkNonce }
func (m Message) GasPremium() *big.Int { return m.gasPremium }
func (m Message) FeeCap() *big.Int     { return m.feeCap }
