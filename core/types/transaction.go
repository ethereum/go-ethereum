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
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

var ErrInvalidSig = errors.New("invalid transaction v, r, s values")

var (
	errMissingTxSignatureFields = errors.New("missing required JSON transaction signature fields")
	errMissingTxFields          = errors.New("missing required JSON transaction fields")
	errNoSigner                 = errors.New("missing signing methods")
)

// deriveSigner makes a *best* guess about which signer to use.
func deriveSigner(V *big.Int) Signer {
	if V.BitLen() > 0 && isProtectedV(V) {
		return EIP155Signer{chainId: deriveChainId(V)}
	} else {
		return HomesteadSigner{}
	}
}

type Transaction struct {
	data txdata
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type txdata struct {
	AccountNonce    uint64
	Price, GasLimit *big.Int
	Recipient       *common.Address `rlp:"nil"` // nil means contract creation
	Amount          *big.Int
	Payload         []byte
	V               *big.Int // signature
	R, S            *big.Int // signature
}

type jsonTransaction struct {
	Hash         *common.Hash    `json:"hash"`
	AccountNonce *hexutil.Uint64 `json:"nonce"`
	Price        *hexutil.Big    `json:"gasPrice"`
	GasLimit     *hexutil.Big    `json:"gas"`
	Recipient    *common.Address `json:"to"`
	Amount       *hexutil.Big    `json:"value"`
	Payload      *hexutil.Bytes  `json:"input"`
	V            *hexutil.Big    `json:"v"`
	R            *hexutil.Big    `json:"r"`
	S            *hexutil.Big    `json:"s"`
}

func NewTransaction(nonce uint64, to common.Address, amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(nonce, &to, amount, gasLimit, gasPrice, data)
}

func NewContractCreation(nonce uint64, amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(nonce, nil, amount, gasLimit, gasPrice, data)
}

func newTransaction(nonce uint64, to *common.Address, amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := txdata{
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     new(big.Int),
		Price:        new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasLimit != nil {
		d.GasLimit.Set(gasLimit)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}

	return &Transaction{data: d}
}

func pickSigner(rules params.Rules) Signer {
	var signer Signer
	switch {
	case rules.IsEIP155:
		signer = NewEIP155Signer(rules.ChainId)
	case rules.IsHomestead:
		signer = HomesteadSigner{}
	default:
		signer = FrontierSigner{}
	}
	return signer
}

// ChainId returns which chain id this transaction was signed for (if at all)
func (tx *Transaction) ChainId() *big.Int {
	return deriveChainId(tx.data.V)
}

// Protected returns whether the transaction is pretected from replay protection
func (tx *Transaction) Protected() bool {
	return isProtectedV(tx.data.V)
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28
	}
	// anything not 27 or 28 are considered unprotected
	return true
}

// DecodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}

	return err
}

// MarshalJSON encodes transactions into the web3 RPC response block format.
func (tx *Transaction) MarshalJSON() ([]byte, error) {
	hash := tx.Hash()

	return json.Marshal(&jsonTransaction{
		Hash:         &hash,
		AccountNonce: (*hexutil.Uint64)(&tx.data.AccountNonce),
		Price:        (*hexutil.Big)(tx.data.Price),
		GasLimit:     (*hexutil.Big)(tx.data.GasLimit),
		Recipient:    tx.data.Recipient,
		Amount:       (*hexutil.Big)(tx.data.Amount),
		Payload:      (*hexutil.Bytes)(&tx.data.Payload),
		V:            (*hexutil.Big)(tx.data.V),
		R:            (*hexutil.Big)(tx.data.R),
		S:            (*hexutil.Big)(tx.data.S),
	})
}

// UnmarshalJSON decodes the web3 RPC transaction format.
func (tx *Transaction) UnmarshalJSON(input []byte) error {
	var dec jsonTransaction
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	// Ensure that all fields are set. V, R, S are checked separately because they're a
	// recent addition to the RPC spec (as of August 2016) and older implementations might
	// not provide them. Note that Recipient is not checked because it can be missing for
	// contract creations.
	if dec.V == nil || dec.R == nil || dec.S == nil {
		return errMissingTxSignatureFields
	}

	var V byte
	if isProtectedV((*big.Int)(dec.V)) {
		V = normaliseV(NewEIP155Signer(deriveChainId((*big.Int)(dec.V))), (*big.Int)(dec.V))
	} else {
		V = byte(((*big.Int)(dec.V)).Uint64())
	}
	if !crypto.ValidateSignatureValues(V, (*big.Int)(dec.R), (*big.Int)(dec.S), false) {
		return ErrInvalidSig
	}

	if dec.AccountNonce == nil || dec.Price == nil || dec.GasLimit == nil || dec.Amount == nil || dec.Payload == nil {
		return errMissingTxFields
	}
	// Assign the fields. This is not atomic but reusing transactions
	// for decoding isn't thread safe anyway.
	*tx = Transaction{}
	tx.data = txdata{
		AccountNonce: uint64(*dec.AccountNonce),
		Recipient:    dec.Recipient,
		Amount:       (*big.Int)(dec.Amount),
		GasLimit:     (*big.Int)(dec.GasLimit),
		Price:        (*big.Int)(dec.Price),
		Payload:      *dec.Payload,
		V:            (*big.Int)(dec.V),
		R:            (*big.Int)(dec.R),
		S:            (*big.Int)(dec.S),
	}
	return nil
}

func (tx *Transaction) Data() []byte       { return common.CopyBytes(tx.data.Payload) }
func (tx *Transaction) Gas() *big.Int      { return new(big.Int).Set(tx.data.GasLimit) }
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.data.Price) }
func (tx *Transaction) Value() *big.Int    { return new(big.Int).Set(tx.data.Amount) }
func (tx *Transaction) Nonce() uint64      { return tx.data.AccountNonce }
func (tx *Transaction) CheckNonce() bool   { return true }

func (tx *Transaction) To() *common.Address {
	if tx.data.Recipient == nil {
		return nil
	} else {
		to := *tx.data.Recipient
		return &to
	}
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

// SigHash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (tx *Transaction) SigHash(signer Signer) common.Hash {
	return signer.Hash(tx)
}

func (tx *Transaction) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.data)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

/*
// From returns the address derived from the signature (V, R, S) using secp256k1
// elliptic curve and an error if it failed deriving or upon an incorrect
// signature.
//
// From Uses the homestead consensus rules to determine whether the signature is
// valid.
//
// From caches the address, allowing it to be used regardless of
// Frontier / Homestead. however, the first time called it runs
// signature validations, so we need two versions. This makes it
// easier to ensure backwards compatibility of things like package rpc
// where eth_getblockbynumber uses tx.From() and needs to work for
// both txs before and after the first homestead block. Signatures
// valid in homestead are a subset of valid ones in Frontier)
func (tx *Transaction) From() (common.Address, error) {
	if tx.signer == nil {
		return common.Address{}, errNoSigner
	}

	if from := tx.from.Load(); from != nil {
		return from.(common.Address), nil
	}

	pubkey, err := tx.signer.PublicKey(tx)
	if err != nil {
		return common.Address{}, err
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pubkey[1:])[12:])
	tx.from.Store(addr)
	return addr, nil
}

// SignatureValues returns the ECDSA signature values contained in the transaction.
func (tx *Transaction) SignatureValues() (v byte, r *big.Int, s *big.Int, err error) {
	if tx.signer == nil {
		return 0, nil, nil,errNoSigner
	}

	return normaliseV(tx.signer, tx.data.V), new(big.Int).Set(tx.data.R),new(big.Int).Set(tx.data.S), nil
}

*/

// AsMessage returns the transaction as a core.Message.
//
// AsMessage requires a signer to derive the sender.
//
// XXX Rename message to something less arbitrary?
func (tx *Transaction) AsMessage(s Signer) (Message, error) {
	msg := Message{
		nonce:      tx.data.AccountNonce,
		price:      new(big.Int).Set(tx.data.Price),
		gasLimit:   new(big.Int).Set(tx.data.GasLimit),
		to:         tx.data.Recipient,
		amount:     tx.data.Amount,
		data:       tx.data.Payload,
		checkNonce: true,
	}

	var err error
	msg.from, err = Sender(s, tx)
	return msg, err
}

// SignECDSA signs the transaction using the given signer and private key
//
// XXX This only makes for a nice API: NewTx(...).SignECDSA(signer, prv). Should
// we keep this?
func (tx *Transaction) SignECDSA(signer Signer, prv *ecdsa.PrivateKey) (*Transaction, error) {
	return signer.SignECDSA(tx, prv)
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be formatted as described in the yellow paper (v+27).
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	return signer.WithSignature(tx, sig)
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.Price, tx.data.GasLimit)
	total.Add(total, tx.data.Amount)
	return total
}

func (tx *Transaction) RawSignatureValues() (*big.Int, *big.Int, *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}

func (tx *Transaction) String() string {
	// make a best guess about the signer and use that to derive
	// the sender.
	signer := deriveSigner(tx.data.V)

	var from, to string
	if f, err := Sender(signer, tx); err != nil { // derive but don't cache
		from = "[invalid sender: invalid sig]"
	} else {
		from = fmt.Sprintf("%x", f[:])
	}
	if tx.data.Recipient == nil {
		to = "[contract creation]"
	} else {
		to = fmt.Sprintf("%x", tx.data.Recipient[:])
	}
	enc, _ := rlp.EncodeToBytes(&tx.data)
	return fmt.Sprintf(`
	TX(%x)
	Contract: %v
	From:     %s
	To:       %s
	Nonce:    %v
	GasPrice: %v
	GasLimit  %v
	Value:    %v
	Data:     0x%x
	V:        0x%x
	R:        0x%x
	S:        0x%x
	Hex:      %x
`,
		tx.Hash(),
		len(tx.data.Recipient) == 0,
		from,
		to,
		tx.data.AccountNonce,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Amount,
		tx.data.Payload,
		tx.data.V,
		tx.data.R,
		tx.data.S,
		enc,
	)
}

// Transaction slice type for basic sorting.
type Transactions []*Transaction

// Len returns the length of s
func (s Transactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp
func (s Transactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

// Returns a new set t which is the difference between a to b
func TxDifference(a, b Transactions) (keep Transactions) {
	keep = make(Transactions, 0, len(a))

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
type TxByPrice Transactions

func (s TxByPrice) Len() int           { return len(s) }
func (s TxByPrice) Less(i, j int) bool { return s[i].data.Price.Cmp(s[j].data.Price) > 0 }
func (s TxByPrice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s *TxByPrice) Push(x interface{}) {
	*s = append(*s, x.(*Transaction))
}

func (s *TxByPrice) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

// TransactionsByPriceAndNonce represents a set of transactions that can return
// transactions in a profit-maximising sorted order, while supporting removing
// entire batches of transactions for non-executable accounts.
type TransactionsByPriceAndNonce struct {
	txs   map[common.Address]Transactions // Per account nonce-sorted list of transactions
	heads TxByPrice                       // Next transaction for each unique account (price heap)
}

// NewTransactionsByPriceAndNonce creates a transaction set that can retrieve
// price sorted transactions in a nonce-honouring way.
//
// Note, the input map is reowned so the caller should not interact any more with
// if after providng it to the constructor.
func NewTransactionsByPriceAndNonce(txs map[common.Address]Transactions) *TransactionsByPriceAndNonce {
	// Initialize a price based heap with the head transactions
	heads := make(TxByPrice, 0, len(txs))
	for acc, accTxs := range txs {
		heads = append(heads, accTxs[0])
		txs[acc] = accTxs[1:]
	}
	heap.Init(&heads)

	// Assemble and return the transaction set
	return &TransactionsByPriceAndNonce{
		txs:   txs,
		heads: heads,
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
	signer := deriveSigner(t.heads[0].data.V)
	// derive signer but don't cache.
	acc, _ := Sender(signer, t.heads[0]) // we only sort valid txs so this cannot fail
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
	to                      *common.Address
	from                    common.Address
	nonce                   uint64
	amount, price, gasLimit *big.Int
	data                    []byte
	checkNonce              bool
}

func NewMessage(from common.Address, to *common.Address, nonce uint64, amount, gasLimit, price *big.Int, data []byte, checkNonce bool) Message {
	return Message{
		from:       from,
		to:         to,
		nonce:      nonce,
		amount:     amount,
		price:      price,
		gasLimit:   gasLimit,
		data:       data,
		checkNonce: checkNonce,
	}
}

func (m Message) From() common.Address { return m.from }
func (m Message) To() *common.Address  { return m.to }
func (m Message) GasPrice() *big.Int   { return m.price }
func (m Message) Value() *big.Int      { return m.amount }
func (m Message) Gas() *big.Int        { return m.gasLimit }
func (m Message) Nonce() uint64        { return m.nonce }
func (m Message) Data() []byte         { return m.data }
func (m Message) CheckNonce() bool     { return m.checkNonce }
