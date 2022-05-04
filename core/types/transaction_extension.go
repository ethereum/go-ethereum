package types

import "math/big"

type TxDataExtension interface {
	encryptedPayload() []byte
	decryptionKey() []byte
	batchIndex() uint64
	l1BlockNumber() *big.Int
	timestamp() *big.Int
	shutterTXs() [][]byte
	plainTextTXs() [][]byte
}

func (tx *DynamicFeeTx) encryptedPayload() []byte { return nil }
func (tx *DynamicFeeTx) decryptionKey() []byte    { return nil }
func (tx *DynamicFeeTx) batchIndex() uint64       { return 0 }
func (tx *DynamicFeeTx) l1BlockNumber() *big.Int  { return nil }
func (tx *DynamicFeeTx) timestamp() *big.Int      { return nil }
func (tx *DynamicFeeTx) shutterTXs() [][]byte     { return nil }
func (tx *DynamicFeeTx) plainTextTXs() [][]byte   { return nil }

func (tx *AccessListTx) encryptedPayload() []byte { return nil }
func (tx *AccessListTx) decryptionKey() []byte    { return nil }
func (tx *AccessListTx) batchIndex() uint64       { return 0 }
func (tx *AccessListTx) l1BlockNumber() *big.Int  { return nil }
func (tx *AccessListTx) timestamp() *big.Int      { return nil }
func (tx *AccessListTx) shutterTXs() [][]byte     { return nil }
func (tx *AccessListTx) plainTextTXs() [][]byte   { return nil }

func (tx *LegacyTx) encryptedPayload() []byte { return nil }
func (tx *LegacyTx) decryptionKey() []byte    { return nil }
func (tx *LegacyTx) batchIndex() uint64       { return 0 }
func (tx *LegacyTx) l1BlockNumber() *big.Int  { return nil }
func (tx *LegacyTx) timestamp() *big.Int      { return nil }
func (tx *LegacyTx) shutterTXs() [][]byte     { return nil }
func (tx *LegacyTx) plainTextTXs() [][]byte   { return nil }
