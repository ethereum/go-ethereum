package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type BatchTx struct {
	ChainID       *big.Int
	DecryptionKey []byte
	BatchIndex    uint64
	L1BlockNumber *big.Int
	Timestamp     *big.Int
	Transactions  [][]byte

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *BatchTx) copy() TxData {
	cpy := &BatchTx{
		ChainID:       new(big.Int),
		DecryptionKey: []byte{},
		BatchIndex:    tx.BatchIndex,
		L1BlockNumber: new(big.Int),
		Timestamp:     new(big.Int),
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.DecryptionKey != nil {
		cpy.DecryptionKey = make([]byte, len(tx.DecryptionKey))
		copy(cpy.DecryptionKey, tx.DecryptionKey)
	}
	if tx.L1BlockNumber != nil {
		cpy.L1BlockNumber.Set(tx.L1BlockNumber)
	}
	if tx.Timestamp != nil {
		cpy.Timestamp.Set(tx.Timestamp)
	}
	if tx.Transactions != nil {
		cpy.Transactions = make([][]byte, len(tx.Transactions))
		for i, b := range tx.Transactions {
			c := make([]byte, len(b))
			copy(c, b)
			cpy.Transactions[i] = c
		}
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
func (tx *BatchTx) txType() byte             { return BatchTxType }
func (tx *BatchTx) chainID() *big.Int        { return tx.ChainID }
func (tx *BatchTx) protected() bool          { return true }
func (tx *BatchTx) accessList() AccessList   { return nil }
func (tx *BatchTx) data() []byte             { return nil }
func (tx *BatchTx) gas() uint64              { return 0 }
func (tx *BatchTx) gasFeeCap() *big.Int      { return big.NewInt(0) }
func (tx *BatchTx) gasTipCap() *big.Int      { return big.NewInt(0) }
func (tx *BatchTx) gasPrice() *big.Int       { return big.NewInt(0) }
func (tx *BatchTx) value() *big.Int          { return big.NewInt(0) }
func (tx *BatchTx) nonce() uint64            { return 0 }
func (tx *BatchTx) to() *common.Address      { return nil }
func (tx *BatchTx) encryptedPayload() []byte { return nil }
func (tx *BatchTx) decryptionKey() []byte    { return tx.DecryptionKey }
func (tx *BatchTx) batchIndex() uint64       { return tx.BatchIndex }
func (tx *BatchTx) l1BlockNumber() *big.Int  { return tx.L1BlockNumber }
func (tx *BatchTx) timestamp() *big.Int      { return tx.Timestamp }
func (tx *BatchTx) transactions() [][]byte   { return tx.Transactions }

func (tx *BatchTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *BatchTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}
