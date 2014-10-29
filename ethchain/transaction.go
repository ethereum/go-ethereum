package ethchain

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethcrypto"
	"github.com/ethereum/go-ethereum/ethstate"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/vm"
	"github.com/obscuren/secp256k1-go"
)

var ContractAddr = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func IsContractAddr(addr []byte) bool {
	return len(addr) == 0
	//return bytes.Compare(addr, ContractAddr) == 0
}

type Transaction struct {
	Nonce     uint64
	Recipient []byte
	Value     *big.Int
	Gas       *big.Int
	GasPrice  *big.Int
	Data      []byte
	v         byte
	r, s      []byte

	logs []vm.Log

	// Indicates whether this tx is a contract creation transaction
	contractCreation bool
}

func NewContractCreationTx(value, gas, gasPrice *big.Int, script []byte) *Transaction {
	return &Transaction{Recipient: nil, Value: value, Gas: gas, GasPrice: gasPrice, Data: script, contractCreation: true}
}

func NewTransactionMessage(to []byte, value, gas, gasPrice *big.Int, data []byte) *Transaction {
	return &Transaction{Recipient: to, Value: value, GasPrice: gasPrice, Gas: gas, Data: data, contractCreation: IsContractAddr(to)}
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

func (self *Transaction) addLog(log vm.Log) {
	self.logs = append(self.logs, log)
}

func (self *Transaction) GasValue() *big.Int {
	return new(big.Int).Mul(self.Gas, self.GasPrice)
}

func (self *Transaction) TotalValue() *big.Int {
	v := self.GasValue()
	return v.Add(v, self.Value)
}

func (tx *Transaction) Hash() []byte {
	data := []interface{}{tx.Nonce, tx.GasPrice, tx.Gas, tx.Recipient, tx.Value, tx.Data}

	return ethcrypto.Sha3(ethutil.NewValue(data).Encode())
}

func (tx *Transaction) CreatesContract() bool {
	return tx.contractCreation
}

/* Deprecated */
func (tx *Transaction) IsContract() bool {
	return tx.CreatesContract()
}

func (tx *Transaction) CreationAddress(state *ethstate.State) []byte {
	// Generate a new address
	addr := ethcrypto.Sha3(ethutil.NewValue([]interface{}{tx.Sender(), tx.Nonce}).Encode())[12:]
	//for i := uint64(0); state.GetStateObject(addr) != nil; i++ {
	//	addr = ethcrypto.Sha3(ethutil.NewValue([]interface{}{tx.Sender(), tx.Nonce + i}).Encode())[12:]
	//}

	return addr
}

func (tx *Transaction) Signature(key []byte) []byte {
	hash := tx.Hash()

	sig, _ := secp256k1.Sign(hash, key)

	return sig
}

func (tx *Transaction) PublicKey() []byte {
	hash := tx.Hash()

	// TODO
	r := ethutil.LeftPadBytes(tx.r, 32)
	s := ethutil.LeftPadBytes(tx.s, 32)

	sig := append(r, s...)
	sig = append(sig, tx.v-27)

	pubkey := ethcrypto.Ecrecover(append(hash, sig...))
	//pubkey, _ := secp256k1.RecoverPubkey(hash, sig)

	return pubkey
}

func (tx *Transaction) Sender() []byte {
	pubkey := tx.PublicKey()

	// Validate the returned key.
	// Return nil if public key isn't in full format
	if pubkey[0] != 4 {
		return nil
	}

	return ethcrypto.Sha3(pubkey[1:])[12:]
}

func (tx *Transaction) Sign(privk []byte) error {

	sig := tx.Signature(privk)

	tx.r = sig[:32]
	tx.s = sig[32:64]
	tx.v = sig[64] + 27

	return nil
}

func (tx *Transaction) RlpData() interface{} {
	data := []interface{}{tx.Nonce, tx.GasPrice, tx.Gas, tx.Recipient, tx.Value, tx.Data}

	// TODO Remove prefixing zero's

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
	tx.Nonce = decoder.Get(0).Uint()
	tx.GasPrice = decoder.Get(1).BigInt()
	tx.Gas = decoder.Get(2).BigInt()
	tx.Recipient = decoder.Get(3).Bytes()
	tx.Value = decoder.Get(4).BigInt()
	tx.Data = decoder.Get(5).Bytes()
	tx.v = byte(decoder.Get(6).Uint())

	tx.r = decoder.Get(7).Bytes()
	tx.s = decoder.Get(8).Bytes()

	if IsContractAddr(tx.Recipient) {
		tx.contractCreation = true
	}
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
	`,
		tx.Hash(),
		len(tx.Recipient) == 0,
		tx.Sender(),
		tx.Recipient,
		tx.Nonce,
		tx.GasPrice,
		tx.Gas,
		tx.Value,
		tx.Data,
		tx.v,
		tx.r,
		tx.s)
}

type Receipt struct {
	PostState         []byte
	CumulativeGasUsed *big.Int
	Bloom             []byte
	Logs              vm.Logs
}

func NewRecieptFromValue(val *ethutil.Value) *Receipt {
	r := &Receipt{}
	r.RlpValueDecode(val)

	return r
}

func (self *Receipt) RlpValueDecode(decoder *ethutil.Value) {
	self.PostState = decoder.Get(0).Bytes()
	self.CumulativeGasUsed = decoder.Get(1).BigInt()
	self.Bloom = decoder.Get(2).Bytes()

	it := decoder.Get(3).NewIterator()
	for it.Next() {
		self.Logs = append(self.Logs, vm.NewLogFromValue(it.Value()))
	}
}

func (self *Receipt) RlpData() interface{} {
	return []interface{}{self.PostState, self.CumulativeGasUsed, self.Bloom, self.Logs.RlpData()}
}

func (self *Receipt) RlpEncode() []byte {
	return ethutil.Encode(self.RlpData())
}

func (self *Receipt) Cmp(other *Receipt) bool {
	if bytes.Compare(self.PostState, other.PostState) != 0 {
		return false
	}

	return true
}

type Receipts []*Receipt

func (self Receipts) Len() int            { return len(self) }
func (self Receipts) GetRlp(i int) []byte { return ethutil.Rlp(self[i]) }

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
	return s.Transactions[i].Nonce < s.Transactions[j].Nonce
}
