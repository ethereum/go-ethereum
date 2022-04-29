package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type BatchContextTx struct {
	ChainID       *big.Int
	DecryptionKey []byte
	BatchIndex    []byte
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *BatchContextTx) copy() TxData {
	cpy := &BatchContextTx{
		ChainID:       tx.ChainID,
		DecryptionKey: []byte{},
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.DecryptionKey != nil {
		cpy.DecryptionKey = make([]byte, len(tx.DecryptionKey))
		copy(cpy.DecryptionKey, tx.DecryptionKey)
	}
	if tx.BatchIndex != nil {
		cpy.BatchIndex = make([]byte, len(tx.BatchIndex))
		copy(cpy.BatchIndex, tx.BatchIndex)
	}
	return cpy
}

// accessors for innerTx.
func (tx *BatchContextTx) txType() byte             { return BatchContextTxType }
func (tx *BatchContextTx) chainID() *big.Int        { return tx.ChainID }
func (tx *BatchContextTx) protected() bool          { return true }
func (tx *BatchContextTx) accessList() AccessList   { return nil }
func (tx *BatchContextTx) data() []byte             { return nil }
func (tx *BatchContextTx) gas() uint64              { return 0 }
func (tx *BatchContextTx) gasFeeCap() *big.Int      { return big.NewInt(0) }
func (tx *BatchContextTx) gasTipCap() *big.Int      { return big.NewInt(0) }
func (tx *BatchContextTx) gasPrice() *big.Int       { return big.NewInt(0) }
func (tx *BatchContextTx) value() *big.Int          { return big.NewInt(0) }
func (tx *BatchContextTx) nonce() uint64            { return 0 }
func (tx *BatchContextTx) to() *common.Address      { return nil }
func (tx *BatchContextTx) encryptedPayload() []byte { return nil }
func (tx *BatchContextTx) decryptionKey() []byte    { return tx.DecryptionKey }
func (tx *BatchContextTx) batchIndex() []byte       { return tx.BatchIndex }

func (tx *BatchContextTx) rawSignatureValues() (v, r, s *big.Int) {
	return big.NewInt(0), big.NewInt(0), big.NewInt(0)
}

func (tx *BatchContextTx) setSignatureValues(chainID, v, r, s *big.Int) {
	// Decryption key transactions are not signed, so do nothing
}
