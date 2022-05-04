package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type ShutterTx struct {
	ChainID   *big.Int
	Nonce     uint64
	GasTipCap *big.Int
	GasFeeCap *big.Int
	Gas       uint64

	EncryptedPayload []byte
	BatchIndex       uint64

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *ShutterTx) copy() TxData {
	cpy := &ShutterTx{
		Nonce: tx.Nonce,
		Gas:   tx.Gas,

		// These are copied below.
		ChainID:          new(big.Int),
		GasTipCap:        new(big.Int),
		GasFeeCap:        new(big.Int),
		EncryptedPayload: []byte{},
		BatchIndex:       tx.BatchIndex,
		V:                new(big.Int),
		R:                new(big.Int),
		S:                new(big.Int),
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap.Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap.Set(tx.GasFeeCap)
	}
	if tx.EncryptedPayload != nil {
		cpy.EncryptedPayload = make([]byte, len(tx.EncryptedPayload))
		copy(cpy.EncryptedPayload, tx.EncryptedPayload)
	}
	if tx.V != nil {
		cpy.V.Set(tx.V)
	}
	if tx.R != nil {
		cpy.R.Set(tx.R)
	}
	if tx.S != nil {
		cpy.S.Set(tx.S)
	}
	return cpy
}

// accessors for innerTx.
func (tx *ShutterTx) txType() byte             { return ShutterTxType }
func (tx *ShutterTx) chainID() *big.Int        { return tx.ChainID }
func (tx *ShutterTx) protected() bool          { return true }
func (tx *ShutterTx) accessList() AccessList   { return nil }
func (tx *ShutterTx) data() []byte             { return nil }
func (tx *ShutterTx) gas() uint64              { return tx.Gas }
func (tx *ShutterTx) gasFeeCap() *big.Int      { return tx.GasFeeCap }
func (tx *ShutterTx) gasTipCap() *big.Int      { return tx.GasTipCap }
func (tx *ShutterTx) gasPrice() *big.Int       { return tx.GasFeeCap }
func (tx *ShutterTx) value() *big.Int          { return nil }
func (tx *ShutterTx) nonce() uint64            { return tx.Nonce }
func (tx *ShutterTx) to() *common.Address      { return nil }
func (tx *ShutterTx) encryptedPayload() []byte { return tx.EncryptedPayload }
func (tx *ShutterTx) decryptionKey() []byte    { return nil }
func (tx *ShutterTx) batchIndex() uint64       { return tx.BatchIndex }
func (tx *ShutterTx) l1BlockNumber() *big.Int  { return nil }
func (tx *ShutterTx) timestamp() *big.Int      { return nil }
func (tx *ShutterTx) transactions() [][]byte   { return nil }

func (tx *ShutterTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *ShutterTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}
