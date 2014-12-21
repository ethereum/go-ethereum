package types

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/obscuren/secp256k1-go"
)

func IsContractAddr(addr []byte) bool {
	return len(addr) == 0
}

type Transaction struct {
	nonce     uint64
	recipient []byte
	value     *big.Int
	gas       *big.Int
	gasPrice  *big.Int
	data      []byte
	v         byte
	r, s      []byte
}

func NewContractCreationTx(value, gas, gasPrice *big.Int, script []byte) *Transaction {
	return &Transaction{recipient: nil, value: value, gas: gas, gasPrice: gasPrice, data: script}
}

func NewTransactionMessage(to []byte, value, gas, gasPrice *big.Int, data []byte) *Transaction {
	return &Transaction{recipient: to, value: value, gasPrice: gasPrice, gas: gas, data: data}
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
	data := []interface{}{tx.nonce, tx.gasPrice, tx.gas, tx.recipient, tx.value, tx.data}

	return crypto.Sha3(ethutil.NewValue(data).Encode())
}

func (self *Transaction) Data() []byte {
	return self.data
}

func (self *Transaction) Gas() *big.Int {
	return self.gas
}

func (self *Transaction) GasPrice() *big.Int {
	return self.gasPrice
}

func (self *Transaction) Value() *big.Int {
	return self.value
}

func (self *Transaction) Nonce() uint64 {
	return self.nonce
}

func (self *Transaction) SetNonce(nonce uint64) {
	self.nonce = nonce
}

func (self *Transaction) From() []byte {
	return self.Sender()
}

func (self *Transaction) To() []byte {
	return self.recipient
}

func (tx *Transaction) Curve() (v byte, r []byte, s []byte) {
	v = tx.v
	r = ethutil.LeftPadBytes(tx.r, 32)
	s = ethutil.LeftPadBytes(tx.s, 32)

	return
}

func (tx *Transaction) Signature(key []byte) []byte {
	hash := tx.Hash()

	sig, _ := secp256k1.Sign(hash, key)

	return sig
}

func (tx *Transaction) PublicKey() []byte {
	hash := tx.Hash()

	v, r, s := tx.Curve()

	sig := append(r, s...)
	sig = append(sig, v-27)

	//pubkey := crypto.Ecrecover(append(hash, sig...))
	pubkey, _ := secp256k1.RecoverPubkey(hash, sig)

	return pubkey
}

func (tx *Transaction) Sender() []byte {
	pubkey := tx.PublicKey()

	// Validate the returned key.
	// Return nil if public key isn't in full format
	if len(pubkey) != 0 && pubkey[0] != 4 {
		return nil
	}

	return crypto.Sha3(pubkey[1:])[12:]
}

func (tx *Transaction) Sign(privk []byte) error {

	sig := tx.Signature(privk)

	tx.r = sig[:32]
	tx.s = sig[32:64]
	tx.v = sig[64] + 27

	return nil
}

func (tx *Transaction) RlpData() interface{} {
	data := []interface{}{tx.nonce, tx.gasPrice, tx.gas, tx.recipient, tx.value, tx.data}

	return append(data, tx.v, new(big.Int).SetBytes(tx.r).Bytes(), new(big.Int).SetBytes(tx.s).Bytes())
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
	tx.nonce = decoder.Get(0).Uint()
	tx.gasPrice = decoder.Get(1).BigInt()
	tx.gas = decoder.Get(2).BigInt()
	tx.recipient = decoder.Get(3).Bytes()
	tx.value = decoder.Get(4).BigInt()
	tx.data = decoder.Get(5).Bytes()
	tx.v = byte(decoder.Get(6).Uint())

	tx.r = decoder.Get(7).Bytes()
	tx.s = decoder.Get(8).Bytes()
}

func (tx *Transaction) String() string {
	return fmt.Sprintf(`
	TX(%x)
	Contract: %v
	From:     %x
	To:       %x
	Nonce:    %v
	GasPrice: %v
	Gas:      %v
	Value:    %v
	Data:     0x%x
	V:        0x%x
	R:        0x%x
	S:        0x%x
	Hex:      %x
	`,
		tx.Hash(),
		len(tx.recipient) == 0,
		tx.Sender(),
		tx.recipient,
		tx.nonce,
		tx.gasPrice,
		tx.gas,
		tx.value,
		tx.data,
		tx.v,
		tx.r,
		tx.s,
		ethutil.Encode(tx),
	)
}

// Transaction slice type for basic sorting
type Transactions []*Transaction

func (self Transactions) RlpData() interface{} {
	// Marshal the transactions of this block
	enc := make([]interface{}, len(self))
	for i, tx := range self {
		// Cast it to a string (safe)
		enc[i] = tx.RlpData()
	}

	return enc
}
func (s Transactions) Len() int            { return len(s) }
func (s Transactions) Swap(i, j int)       { s[i], s[j] = s[j], s[i] }
func (s Transactions) GetRlp(i int) []byte { return ethutil.Rlp(s[i]) }

type TxByNonce struct{ Transactions }

func (s TxByNonce) Less(i, j int) bool {
	return s.Transactions[i].nonce < s.Transactions[j].nonce
}
