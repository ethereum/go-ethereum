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
	"fmt"
	"io"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/hexutil"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

var (
	ErrInvalidSig           = errors.New("invalid transaction v, r, s values")
	ErrUnexpectedProtection = errors.New("transaction type does not supported EIP-155 protected signatures")
	ErrInvalidTxType        = errors.New("transaction type not valid in this context")
	ErrTxTypeNotSupported   = errors.New("transaction type not supported")
	ErrGasFeeCapTooLow      = errors.New("fee cap less than base fee")
	errShortTypedTx         = errors.New("typed transaction too short")

	skipNonceDestinationAddress = map[common.Address]bool{
		common.XDCXAddrBinary:                         true,
		common.TradingStateAddrBinary:                 true,
		common.XDCXLendingAddressBinary:               true,
		common.XDCXLendingFinalizedTradeAddressBinary: true,
	}
)

// Transaction types.
const (
	LegacyTxType = iota
	AccessListTxType
	DynamicFeeTxType
)

// Transaction is an Ethereum transaction.
type Transaction struct {
	inner TxData    // Consensus contents of a transaction
	time  time.Time // Time first seen locally (spam avoidance)

	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

// NewTx creates a new transaction.
func NewTx(inner TxData) *Transaction {
	tx := new(Transaction)
	tx.setDecoded(inner.copy(), 0)
	return tx
}

// TxData is the underlying data of a transaction.
//
// This is implemented by LegacyTx and AccessListTx.
type TxData interface {
	txType() byte // returns the type ID
	copy() TxData // creates a deep copy and initializes all fields

	chainID() *big.Int
	accessList() AccessList
	data() []byte
	gas() uint64
	gasPrice() *big.Int
	gasTipCap() *big.Int
	gasFeeCap() *big.Int
	value() *big.Int
	nonce() uint64
	to() *common.Address

	rawSignatureValues() (v, r, s *big.Int)
	setSignatureValues(chainID, v, r, s *big.Int)

	// effectiveGasPrice computes the gas price paid by the transaction, given
	// the inclusion block baseFee.
	//
	// Unlike other TxData methods, the returned *big.Int should be an independent
	// copy of the computed value, i.e. callers are allowed to mutate the result.
	// Method implementations can use 'dst' to store the result.
	effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int
}

// EncodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	if tx.Type() == LegacyTxType {
		return rlp.Encode(w, tx.inner)
	}
	// It's an EIP-2718 typed TX envelope.
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(buf)
	buf.Reset()
	if err := tx.encodeTyped(buf); err != nil {
		return err
	}
	return rlp.Encode(w, buf.Bytes())
}

// encodeTyped writes the canonical encoding of a typed transaction to w.
func (tx *Transaction) encodeTyped(w *bytes.Buffer) error {
	w.WriteByte(tx.Type())
	return rlp.Encode(w, tx.inner)
}

// MarshalBinary returns the canonical encoding of the transaction.
// For legacy transactions, it returns the RLP encoding. For EIP-2718 typed
// transactions, it returns the type and payload.
func (tx *Transaction) MarshalBinary() ([]byte, error) {
	if tx.Type() == LegacyTxType {
		return rlp.EncodeToBytes(tx.inner)
	}
	var buf bytes.Buffer
	err := tx.encodeTyped(&buf)
	return buf.Bytes(), err
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	kind, size, err := s.Kind()
	switch {
	case err != nil:
		return err
	case kind == rlp.List:
		// It's a legacy transaction.
		var inner LegacyTx
		err := s.Decode(&inner)
		if err == nil {
			tx.setDecoded(&inner, int(rlp.ListSize(size)))
		}
		return err
	default:
		// It's an EIP-2718 typed TX envelope.
		var b []byte
		if b, err = s.Bytes(); err != nil {
			return err
		}
		inner, err := tx.decodeTyped(b)
		if err == nil {
			tx.setDecoded(inner, len(b))
		}
		return err
	}
}

// UnmarshalBinary decodes the canonical encoding of transactions.
// It supports legacy RLP transactions and EIP2718 typed transactions.
func (tx *Transaction) UnmarshalBinary(b []byte) error {
	if len(b) > 0 && b[0] > 0x7f {
		// It's a legacy transaction.
		var data LegacyTx
		err := rlp.DecodeBytes(b, &data)
		if err != nil {
			return err
		}
		tx.setDecoded(&data, len(b))
		return nil
	}
	// It's an EIP2718 typed transaction envelope.
	inner, err := tx.decodeTyped(b)
	if err != nil {
		return err
	}
	tx.setDecoded(inner, len(b))
	return nil
}

// decodeTyped decodes a typed transaction from the canonical format.
func (tx *Transaction) decodeTyped(b []byte) (TxData, error) {
	if len(b) <= 1 {
		return nil, errShortTypedTx
	}
	switch b[0] {
	case AccessListTxType:
		var inner AccessListTx
		err := rlp.DecodeBytes(b[1:], &inner)
		return &inner, err
	case DynamicFeeTxType:
		var inner DynamicFeeTx
		err := rlp.DecodeBytes(b[1:], &inner)
		return &inner, err
	default:
		return nil, ErrTxTypeNotSupported
	}
}

// setDecoded sets the inner transaction and size after decoding.
func (tx *Transaction) setDecoded(inner TxData, size int) {
	tx.inner = inner
	tx.time = time.Now()
	if size > 0 {
		tx.size.Store(common.StorageSize(size))
	}
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

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28 && v != 1 && v != 0
	}
	// anything not 27 or 28 is considered protected
	return true
}

// Protected says whether the transaction is replay-protected.
func (tx *Transaction) Protected() bool {
	switch tx := tx.inner.(type) {
	case *LegacyTx:
		return tx.V != nil && isProtectedV(tx.V)
	default:
		return true
	}
}

// Type returns the transaction type.
func (tx *Transaction) Type() uint8 {
	return tx.inner.txType()
}

// ChainId returns the EIP155 chain ID of the transaction. The return value will always be
// non-nil. For legacy transactions which are not replay-protected, the return value is
// zero.
func (tx *Transaction) ChainId() *big.Int {
	return tx.inner.chainID()
}

// Data returns the input data of the transaction.
func (tx *Transaction) Data() []byte { return tx.inner.data() }

// AccessList returns the access list of the transaction.
func (tx *Transaction) AccessList() AccessList { return tx.inner.accessList() }

// Gas returns the gas limit of the transaction.
func (tx *Transaction) Gas() uint64 { return tx.inner.gas() }

// GasPrice returns the gas price of the transaction.
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.inner.gasPrice()) }

// GasTipCap returns the gasTipCap per gas of the transaction.
func (tx *Transaction) GasTipCap() *big.Int { return new(big.Int).Set(tx.inner.gasTipCap()) }

// GasFeeCap returns the fee cap per gas of the transaction.
func (tx *Transaction) GasFeeCap() *big.Int { return new(big.Int).Set(tx.inner.gasFeeCap()) }

// Value returns the ether amount of the transaction.
func (tx *Transaction) Value() *big.Int { return new(big.Int).Set(tx.inner.value()) }

// Nonce returns the sender account nonce of the transaction.
func (tx *Transaction) Nonce() uint64 { return tx.inner.nonce() }

// To returns the recipient address of the transaction.
// For contract-creation transactions, To returns nil.
func (tx *Transaction) To() *common.Address {
	return copyAddressPtr(tx.inner.to())
}

func (tx *Transaction) From() *common.Address {
	var signer Signer
	if tx.Protected() {
		signer = LatestSignerForChainID(tx.ChainId())
	} else {
		signer = HomesteadSigner{}
	}
	from, err := Sender(signer, tx)
	if err != nil {
		return nil
	}
	return &from
}

// Cost returns gas * gasPrice + value.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas()))
	total.Add(total, tx.Value())
	return total
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.inner.rawSignatureValues()
}

// GasFeeCapCmp compares the fee cap of two transactions.
func (tx *Transaction) GasFeeCapCmp(other *Transaction) int {
	return tx.inner.gasFeeCap().Cmp(other.inner.gasFeeCap())
}

// GasFeeCapIntCmp compares the fee cap of the transaction against the given fee cap.
func (tx *Transaction) GasFeeCapIntCmp(other *big.Int) int {
	return tx.inner.gasFeeCap().Cmp(other)
}

// GasTipCapCmp compares the gasTipCap of two transactions.
func (tx *Transaction) GasTipCapCmp(other *Transaction) int {
	return tx.inner.gasTipCap().Cmp(other.inner.gasTipCap())
}

// GasTipCapIntCmp compares the gasTipCap of the transaction against the given gasTipCap.
func (tx *Transaction) GasTipCapIntCmp(other *big.Int) int {
	return tx.inner.gasTipCap().Cmp(other)
}

// EffectiveGasTip returns the effective miner gasTipCap for the given base fee.
// Note: if the effective gasTipCap is negative, this method returns both error
// the actual negative value, _and_ ErrGasFeeCapTooLow
func (tx *Transaction) EffectiveGasTip(baseFee *big.Int) (*big.Int, error) {
	if baseFee == nil {
		return tx.GasTipCap(), nil
	}
	var err error
	gasFeeCap := tx.GasFeeCap()
	if gasFeeCap.Cmp(baseFee) < 0 {
		err = ErrGasFeeCapTooLow
	}
	gasFeeCap = gasFeeCap.Sub(gasFeeCap, baseFee)

	gasTipCap := tx.GasTipCap()
	if gasTipCap.Cmp(gasFeeCap) < 0 {
		return gasTipCap, err
	}
	return gasFeeCap, err
}

// EffectiveGasTipValue is identical to EffectiveGasTip, but does not return an
// error in case the effective gasTipCap is negative
func (tx *Transaction) EffectiveGasTipValue(baseFee *big.Int) *big.Int {
	effectiveTip, _ := tx.EffectiveGasTip(baseFee)
	return effectiveTip
}

// EffectiveGasTipCmp compares the effective gasTipCap of two transactions assuming the given base fee.
func (tx *Transaction) EffectiveGasTipCmp(other *Transaction, baseFee *big.Int) int {
	if baseFee == nil {
		return tx.GasTipCapCmp(other)
	}
	return tx.EffectiveGasTipValue(baseFee).Cmp(other.EffectiveGasTipValue(baseFee))
}

// EffectiveGasTipIntCmp compares the effective gasTipCap of a transaction to the given gasTipCap.
func (tx *Transaction) EffectiveGasTipIntCmp(other *big.Int, baseFee *big.Int) int {
	if baseFee == nil {
		return tx.GasTipCapIntCmp(other)
	}
	return tx.EffectiveGasTipValue(baseFee).Cmp(other)
}

// Hash returns the transaction hash.
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}

	var h common.Hash
	if tx.Type() == LegacyTxType {
		h = rlpHash(tx.inner)
	} else {
		h = prefixedRlpHash(tx.Type(), tx.inner)
	}
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

func (tx *Transaction) EffectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	return tx.inner.effectiveGasPrice(dst, baseFee)
}

// AsMessage returns the transaction as a core.Message.
func (tx *Transaction) AsMessage(s Signer, balanceFee, blockNumber, baseFee *big.Int) (Message, error) {
	msg := Message{
		nonce:           tx.Nonce(),
		gasLimit:        tx.Gas(),
		gasPrice:        new(big.Int).Set(tx.GasPrice()),
		gasFeeCap:       new(big.Int).Set(tx.GasFeeCap()),
		gasTipCap:       new(big.Int).Set(tx.GasTipCap()),
		to:              tx.To(),
		amount:          tx.Value(),
		data:            tx.Data(),
		accessList:      tx.AccessList(),
		isFake:          false,
		balanceTokenFee: balanceFee,
	}

	if balanceFee != nil {
		if blockNumber != nil {
			if blockNumber.Cmp(common.BlockNumberGas50x) >= 0 {
				msg.gasPrice = common.GasPrice50x
			} else if blockNumber.Cmp(common.TIPTRC21Fee) > 0 {
				msg.gasPrice = common.TRC21GasPrice
			} else {
				msg.gasPrice = common.TRC21GasPriceBefore
			}
		}
	} else if baseFee != nil {
		// If baseFee provided, set gasPrice to effectiveGasPrice.
		msg.gasPrice = msg.gasPrice.Add(msg.gasTipCap, baseFee)
		if msg.gasPrice.Cmp(msg.gasFeeCap) > 0 {
			msg.gasPrice.Set(msg.gasFeeCap)
		}
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
	cpy := tx.inner.copy()
	cpy.setSignatureValues(signer.ChainID(), v, r, s)
	return &Transaction{inner: cpy, time: tx.time}, nil
}

// TxCost returns gas * gasPrice + value.
func (tx *Transaction) TxCost(number *big.Int) *big.Int {
	total := new(big.Int).Mul(common.GetGasPrice(number), new(big.Int).SetUint64(tx.Gas()))
	total.Add(total, tx.Value())
	return total
}

func IsSpecialTx(to *common.Address) bool {
	return to != nil && (*to == common.BlockSignersBinary || *to == common.RandomizeSMCBinary)
}

func (tx *Transaction) IsSpecialTransaction() bool {
	return tx != nil && IsSpecialTx(tx.To())
}

func (tx *Transaction) IsTradingTransaction() bool {
	to := tx.To()
	return to != nil && *to == common.XDCXAddrBinary
}

func (tx *Transaction) IsLendingTransaction() bool {
	to := tx.To()
	return to != nil && *to == common.XDCXLendingAddressBinary
}

func (tx *Transaction) IsLendingFinalizedTradeTransaction() bool {
	to := tx.To()
	return to != nil && *to == common.XDCXLendingFinalizedTradeAddressBinary
}

func (tx *Transaction) IsSkipNonceTransaction() bool {
	to := tx.To()
	return to != nil && skipNonceDestinationAddress[*to]
}

func (tx *Transaction) IsSigningTransaction() bool {
	to := tx.To()
	if to == nil || *to != common.BlockSignersBinary {
		return false
	}
	data := tx.Data()
	if len(data) != (32*2 + 4) {
		return false
	}
	method := hexutil.Encode(data[0:4])
	return method == common.SignMethod
}

func (tx *Transaction) IsVotingTransaction() (bool, *common.Address) {
	to := tx.To()
	if to == nil || *to != common.MasternodeVotingSMCBinary {
		return false, nil
	}
	var end int
	data := tx.Data()
	method := hexutil.Encode(data[0:4])
	if method == common.VoteMethod || method == common.ProposeMethod || method == common.ResignMethod {
		end = len(data)
	} else if method == common.UnvoteMethod {
		end = len(data) - 32
	} else {
		return false, nil
	}

	addr := data[end-20 : end]
	m := common.BytesToAddress(addr)
	return true, &m

}

func (tx *Transaction) IsXDCXApplyTransaction() bool {
	to := tx.To()
	if to == nil {
		return false
	}

	addr := common.XDCXListingSMC
	if common.IsTestnet {
		addr = common.XDCXListingSMCTestNet
	}
	if *to != addr {
		return false
	}
	data := tx.Data()
	// 4 bytes for function name
	// 32 bytes for 1 parameter
	if len(data) != (32 + 4) {
		return false
	}
	method := hexutil.Encode(data[0:4])
	return method == common.XDCXApplyMethod
}

func (tx *Transaction) IsXDCZApplyTransaction() bool {
	to := tx.To()
	if to == nil {
		return false
	}

	addr := common.TRC21IssuerSMC
	if common.IsTestnet {
		addr = common.TRC21IssuerSMCTestNet
	}
	if *to != addr {
		return false
	}
	data := tx.Data()
	// 4 bytes for function name
	// 32 bytes for 1 parameter
	if len(data) != (32 + 4) {
		return false
	}
	method := hexutil.Encode(data[0:4])
	return method == common.XDCZApplyMethod
}

func (tx *Transaction) String() string {
	var from, to string

	sender := tx.From()
	if sender != nil {
		from = fmt.Sprintf("%x", sender[:])
	} else {
		from = "[invalid sender]"
	}

	receiver := tx.To()
	if receiver == nil {
		to = "[contract creation]"
	} else {
		to = fmt.Sprintf("%x", receiver[:])
	}

	enc, _ := rlp.EncodeToBytes(tx.Data())
	v, r, s := tx.RawSignatureValues()

	return fmt.Sprintf(`
	TX(%x)
	Contract: %v
	From:     %s
	To:       %s
	Nonce:    %v
	GasPrice: %#x
	GasLimit  %#x
	Value:    %#x
	Data:     %#x
	V:        %#x
	R:        %#x
	S:        %#x
	Hex:      %x
`,
		tx.Hash(),
		receiver == nil,
		from,
		to,
		tx.Nonce(),
		tx.GasPrice(),
		tx.Gas(),
		tx.Value(),
		tx.Data(),
		v,
		r,
		s,
		enc,
	)
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

// TxDifference returns a new set t which is the difference between a to b.
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

// HashDifference returns a new set of hashes that are present in a but not in b.
func HashDifference(a, b []common.Hash) []common.Hash {
	keep := make([]common.Hash, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, hash := range b {
		remove[hash] = struct{}{}
	}

	for _, hash := range a {
		if _, ok := remove[hash]; !ok {
			keep = append(keep, hash)
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
type TxByPriceAndTime struct {
	txs        Transactions
	payersSwap map[common.Address]*big.Int
}

func (s TxByPriceAndTime) Len() int { return len(s.txs) }
func (s TxByPriceAndTime) Less(i, j int) bool {
	i_price := s.txs[i].GasPrice()
	if s.txs[i].To() != nil {
		if _, ok := s.payersSwap[*s.txs[i].To()]; ok {
			i_price = common.TRC21GasPrice
		}
	}

	j_price := s.txs[j].GasPrice()
	if s.txs[j].To() != nil {
		if _, ok := s.payersSwap[*s.txs[j].To()]; ok {
			j_price = common.TRC21GasPrice
		}
	}

	// If the prices are equal, use the time the transaction was first seen for
	// deterministic sorting
	cmp := i_price.Cmp(j_price)
	if cmp == 0 {
		return s.txs[i].time.Before(s.txs[j].time)
	}
	return cmp > 0
}
func (s TxByPriceAndTime) Swap(i, j int) { s.txs[i], s.txs[j] = s.txs[j], s.txs[i] }

func (s *TxByPriceAndTime) Push(x interface{}) {
	s.txs = append(s.txs, x.(*Transaction))
}

func (s *TxByPriceAndTime) Pop() interface{} {
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
	heads  TxByPriceAndTime                // Next transaction for each unique account (price heap)
	signer Signer                          // Signer for the set of transactions
}

// NewTransactionsByPriceAndNonce creates a transaction set that can retrieve
// price sorted transactions in a nonce-honouring way.
//
// Note, the input map is reowned so the caller should not interact any more with
// if after providing it to the constructor.
//
// It also classifies special txs and normal txs
func NewTransactionsByPriceAndNonce(signer Signer, txs map[common.Address]Transactions, signers map[common.Address]struct{}, payersSwap map[common.Address]*big.Int) (*TransactionsByPriceAndNonce, Transactions) {
	// Initialize a price and received time based heap with the head transactions
	heads := TxByPriceAndTime{}
	heads.payersSwap = payersSwap
	specialTxs := Transactions{}
	for _, accTxs := range txs {
		from, _ := Sender(signer, accTxs[0])
		var normalTxs Transactions
		lastSpecialTx := -1
		if len(signers) > 0 {
			if _, ok := signers[from]; ok {
				for i, tx := range accTxs {
					if tx.IsSpecialTransaction() {
						lastSpecialTx = i
					}
				}
			}
		}
		if lastSpecialTx >= 0 {
			for i := 0; i <= lastSpecialTx; i++ {
				specialTxs = append(specialTxs, accTxs[i])
			}
			normalTxs = accTxs[lastSpecialTx+1:]
		} else {
			normalTxs = accTxs
		}
		if len(normalTxs) > 0 {
			heads.txs = append(heads.txs, normalTxs[0])
			// Ensure the sender address is from the signer
			txs[from] = normalTxs[1:]
		}
	}
	heap.Init(&heads)

	// Assemble and return the transaction set
	return &TransactionsByPriceAndNonce{
		txs:    txs,
		heads:  heads,
		signer: signer,
	}, specialTxs
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
	to              *common.Address
	from            common.Address
	nonce           uint64
	amount          *big.Int
	gasLimit        uint64
	gasPrice        *big.Int
	gasFeeCap       *big.Int
	gasTipCap       *big.Int
	data            []byte
	accessList      AccessList
	isFake          bool
	balanceTokenFee *big.Int
}

func NewMessage(from common.Address, to *common.Address, nonce uint64, amount *big.Int, gasLimit uint64, gasPrice, gasFeeCap, gasTipCap *big.Int, data []byte, accessList AccessList, isFake bool, balanceTokenFee *big.Int, number *big.Int) Message {
	if balanceTokenFee != nil {
		gasPrice = common.GetGasPrice(number)
	}
	return Message{
		from:            from,
		to:              to,
		nonce:           nonce,
		amount:          amount,
		gasLimit:        gasLimit,
		gasPrice:        gasPrice,
		gasFeeCap:       gasFeeCap,
		gasTipCap:       gasTipCap,
		data:            data,
		accessList:      accessList,
		isFake:          isFake,
		balanceTokenFee: balanceTokenFee,
	}
}

func (m Message) From() common.Address      { return m.from }
func (m Message) BalanceTokenFee() *big.Int { return m.balanceTokenFee }
func (m Message) To() *common.Address       { return m.to }
func (m Message) GasPrice() *big.Int        { return m.gasPrice }
func (m Message) GasFeeCap() *big.Int       { return m.gasFeeCap }
func (m Message) GasTipCap() *big.Int       { return m.gasTipCap }
func (m Message) Value() *big.Int           { return m.amount }
func (m Message) Gas() uint64               { return m.gasLimit }
func (m Message) Nonce() uint64             { return m.nonce }
func (m Message) Data() []byte              { return m.data }
func (m Message) IsFake() bool              { return m.isFake }
func (m Message) AccessList() AccessList    { return m.accessList }

func (m *Message) SetNonce(nonce uint64) { m.nonce = nonce }

func (m *Message) SetBalanceTokenFeeForCall() {
	m.balanceTokenFee = new(big.Int).SetUint64(m.gasLimit)
	m.balanceTokenFee.Mul(m.balanceTokenFee, m.gasPrice)
}

func (m *Message) SetBalanceTokenFee(balanceTokenFee *big.Int) {
	m.balanceTokenFee = balanceTokenFee
}

// copyAddressPtr copies an address.
func copyAddressPtr(a *common.Address) *common.Address {
	if a == nil {
		return nil
	}
	cpy := *a
	return &cpy
}
