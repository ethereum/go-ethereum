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
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
)

var ErrInvalidSig = errors.New("invalid v, r, s values")

func IsContractAddr(addr []byte) bool {
	return len(addr) == 0
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
	V               byte     // signature
	R, S            *big.Int // signature
}

func NewContractCreation(nonce uint64, amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	return &Transaction{data: txdata{
		AccountNonce: nonce,
		Recipient:    nil,
		Amount:       new(big.Int).Set(amount),
		GasLimit:     new(big.Int).Set(gasLimit),
		Price:        new(big.Int).Set(gasPrice),
		Payload:      data,
		R:            new(big.Int),
		S:            new(big.Int),
	}}
}

func NewTransaction(nonce uint64, to common.Address, amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := txdata{
		AccountNonce: nonce,
		Recipient:    &to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     new(big.Int),
		Price:        new(big.Int),
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

func NewTransactionFromBytes(data []byte) *Transaction {
	// TODO: remove this function if possible. callers would
	// much better off decoding into transaction directly.
	// it's not that hard.
	tx := new(Transaction)
	rlp.DecodeBytes(data, tx)
	return tx
}

func (tx *Transaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}
	return err
}

func (tx *Transaction) Data() []byte       { return common.CopyBytes(tx.data.Payload) }
func (tx *Transaction) Gas() *big.Int      { return new(big.Int).Set(tx.data.GasLimit) }
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.data.Price) }
func (tx *Transaction) Value() *big.Int    { return new(big.Int).Set(tx.data.Amount) }
func (tx *Transaction) Nonce() uint64      { return tx.data.AccountNonce }

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
func (tx *Transaction) SigHash() common.Hash {
	return rlpHash([]interface{}{
		tx.data.AccountNonce,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Recipient,
		tx.data.Amount,
		tx.data.Payload,
	})
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

func (tx *Transaction) From() (common.Address, error) {
	if from := tx.from.Load(); from != nil {
		return from.(common.Address), nil
	}
	pubkey, err := tx.publicKey()
	if err != nil {
		return common.Address{}, err
	}
	var addr common.Address
	copy(addr[:], crypto.Sha3(pubkey[1:])[12:])
	tx.from.Store(addr)
	return addr, nil
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.Price, tx.data.GasLimit)
	total.Add(total, tx.data.Amount)
	return total
}

func (tx *Transaction) SignatureValues() (v byte, r *big.Int, s *big.Int) {
	return tx.data.V, new(big.Int).Set(tx.data.R), new(big.Int).Set(tx.data.S)
}

func (tx *Transaction) publicKey() ([]byte, error) {
	if !crypto.ValidateSignatureValues(tx.data.V, tx.data.R, tx.data.S) {
		return nil, ErrInvalidSig
	}

	// encode the signature in uncompressed format
	r, s := tx.data.R.Bytes(), tx.data.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = tx.data.V - 27

	// recover the public key from the signature
	hash := tx.SigHash()
	pub, err := crypto.Ecrecover(hash[:], sig)
	if err != nil {
		glog.V(logger.Error).Infof("Could not get pubkey from signature: ", err)
		return nil, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return nil, errors.New("invalid public key")
	}
	return pub, nil
}

func (tx *Transaction) WithSignature(sig []byte) (*Transaction, error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	cpy := &Transaction{data: tx.data}
	cpy.data.R = new(big.Int).SetBytes(sig[:32])
	cpy.data.S = new(big.Int).SetBytes(sig[32:64])
	cpy.data.V = sig[64] + 27
	return cpy, nil
}

func (tx *Transaction) SignECDSA(prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := tx.SigHash()
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(sig)
}

func (tx *Transaction) String() string {
	var from, to string
	if f, err := tx.From(); err != nil {
		from = "[invalid sender]"
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

func (s Transactions) Len() int      { return len(s) }
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s Transactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

type TxByNonce struct{ Transactions }

func (s TxByNonce) Less(i, j int) bool {
	return s.Transactions[i].data.AccountNonce < s.Transactions[j].data.AccountNonce
}
