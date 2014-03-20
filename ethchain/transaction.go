package ethchain

import (
	"bytes"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/obscuren/secp256k1-go"
	"math/big"
)

var ContractAddr = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

type Transaction struct {
	Nonce     uint64
	Recipient []byte
	Value     *big.Int
	Gas       *big.Int
	Gasprice  *big.Int
	Data      []string
	v         byte
	r, s      []byte
}

func NewTransaction(to []byte, value *big.Int, data []string) *Transaction {
	tx := Transaction{Recipient: to, Value: value, Nonce: 0, Data: data}

	return &tx
}

func NewContractCreationTx(value, gasprice *big.Int, data []string) *Transaction {
	return &Transaction{Value: value, Gasprice: gasprice, Data: data}
}

func NewContractMessageTx(to []byte, value, gasprice, gas *big.Int, data []string) *Transaction {
	return &Transaction{Recipient: to, Value: value, Gasprice: gasprice, Gas: gas, Data: data}
}

func NewTx(to []byte, value *big.Int, data []string) *Transaction {
	return &Transaction{Recipient: to, Value: value, Gasprice: big.NewInt(0), Gas: big.NewInt(0), Nonce: 0, Data: data}
}

// XXX Deprecated
func NewTransactionFromData(data []byte) *Transaction {
	return NewTransactionFromBytes(data)
}

func NewTransactionFromBytes(data []byte) *Transaction {
	tx := &Transaction{}
	tx.RlpDecode(data)

	return tx
}

func NewTransactionFromValue(val *ethutil.Value) *Transaction {
	tx := &Transaction{}
	tx.RlpValueDecode(val)

	return tx
}

func (tx *Transaction) Hash() []byte {
	data := make([]interface{}, len(tx.Data))
	for i, val := range tx.Data {
		data[i] = val
	}

	preEnc := []interface{}{
		tx.Nonce,
		tx.Recipient,
		tx.Value,
		data,
	}

	return ethutil.Sha3Bin(ethutil.Encode(preEnc))
}

func (tx *Transaction) IsContract() bool {
	return bytes.Compare(tx.Recipient, ContractAddr) == 0
}

func (tx *Transaction) Signature(key []byte) []byte {
	hash := tx.Hash()

	sig, _ := secp256k1.Sign(hash, key)

	return sig
}

func (tx *Transaction) PublicKey() []byte {
	hash := tx.Hash()

	// If we don't make a copy we will overwrite the existing underlying array
	dst := make([]byte, len(tx.r))
	copy(dst, tx.r)

	sig := append(dst, tx.s...)
	sig = append(sig, tx.v-27)

	pubkey, _ := secp256k1.RecoverPubkey(hash, sig)

	return pubkey
}

func (tx *Transaction) Sender() []byte {
	pubkey := tx.PublicKey()

	// Validate the returned key.
	// Return nil if public key isn't in full format
	if pubkey[0] != 4 {
		return nil
	}

	return ethutil.Sha3Bin(pubkey[1:])[12:]
}

func (tx *Transaction) Sign(privk []byte) error {

	sig := tx.Signature(privk)

	tx.r = sig[:32]
	tx.s = sig[32:64]
	tx.v = sig[64] + 27

	return nil
}

func (tx *Transaction) RlpData() interface{} {
	// Prepare the transaction for serialization
	return []interface{}{
		tx.Nonce,
		tx.Recipient,
		tx.Value,
		ethutil.NewSliceValue(tx.Data).Slice(),
		tx.v,
		tx.r,
		tx.s,
	}
}

func (tx *Transaction) RlpValue() *ethutil.Value {
	return ethutil.NewValue(tx.RlpData())
}

func (tx *Transaction) RlpEncode() []byte {
	return tx.RlpValue().Encode()
}

func (tx *Transaction) RlpDecode(data []byte) {
	tx.RlpValueDecode(ethutil.NewValueFromBytes(data))
}

func (tx *Transaction) RlpValueDecode(decoder *ethutil.Value) {
	tx.Nonce = decoder.Get(0).Uint()
	tx.Recipient = decoder.Get(1).Bytes()
	tx.Value = decoder.Get(2).BigInt()

	d := decoder.Get(3)
	tx.Data = make([]string, d.Len())
	for i := 0; i < d.Len(); i++ {
		tx.Data[i] = d.Get(i).Str()
	}

	// TODO something going wrong here
	tx.v = byte(decoder.Get(4).Uint())
	tx.r = decoder.Get(5).Bytes()
	tx.s = decoder.Get(6).Bytes()
}
