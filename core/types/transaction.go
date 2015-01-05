package types

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/obscuren/secp256k1-go"
)

func IsContractAddr(addr []byte) bool {
	return len(addr) == 0
}

type Transaction struct {
	AccountNonce uint64
	Price        *big.Int
	GasLimit     *big.Int
	Recipient    []byte
	Amount       *big.Int
	Payload      []byte
	V            uint64
	R, S         []byte
}

func NewContractCreationTx(Amount, gasAmount, price *big.Int, data []byte) *Transaction {
	return NewTransactionMessage(nil, Amount, gasAmount, price, data)
}

func NewTransactionMessage(to []byte, Amount, gasAmount, price *big.Int, data []byte) *Transaction {
	return &Transaction{Recipient: to, Amount: Amount, Price: price, GasLimit: gasAmount, Payload: data}
}

func NewTransactionFromBytes(data []byte) *Transaction {
	tx := &Transaction{}
	tx.RlpDecode(data)

	return tx
}

func NewTransactionFromAmount(val *ethutil.Value) *Transaction {
	tx := &Transaction{}
	tx.RlpValueDecode(val)

	return tx
}

func (tx *Transaction) Hash() []byte {
	data := []interface{}{tx.AccountNonce, tx.Price, tx.GasLimit, tx.Recipient, tx.Amount, tx.Payload}

	return crypto.Sha3(ethutil.Encode(data))
}

func (self *Transaction) Data() []byte {
	return self.Payload
}

func (self *Transaction) Gas() *big.Int {
	return self.GasLimit
}

func (self *Transaction) GasPrice() *big.Int {
	return self.Price
}

func (self *Transaction) Value() *big.Int {
	return self.Amount
}

func (self *Transaction) Nonce() uint64 {
	return self.AccountNonce
}

func (self *Transaction) SetNonce(AccountNonce uint64) {
	self.AccountNonce = AccountNonce
}

func (self *Transaction) From() []byte {
	return self.sender()
}

func (self *Transaction) To() []byte {
	return self.Recipient
}

func (tx *Transaction) Curve() (v byte, r []byte, s []byte) {
	v = byte(tx.V)
	r = ethutil.LeftPadBytes(tx.R, 32)
	s = ethutil.LeftPadBytes(tx.S, 32)

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

func (tx *Transaction) sender() []byte {
	pubkey := tx.PublicKey()

	// Validate the returned key.
	// Return nil if public key isn't in full format
	if len(pubkey) == 0 || pubkey[0] != 4 {
		return nil
	}

	return crypto.Sha3(pubkey[1:])[12:]
}

func (tx *Transaction) Sign(privk []byte) error {

	sig := tx.Signature(privk)

	tx.R = sig[:32]
	tx.S = sig[32:64]
	tx.V = uint64(sig[64] + 27)

	return nil
}

func (tx *Transaction) SignECDSA(key *ecdsa.PrivateKey) error {
	return tx.Sign(crypto.FromECDSA(key))
}

func (tx *Transaction) RlpData() interface{} {
	data := []interface{}{tx.AccountNonce, tx.Price, tx.GasLimit, tx.Recipient, tx.Amount, tx.Payload}

	return append(data, tx.V, new(big.Int).SetBytes(tx.R).Bytes(), new(big.Int).SetBytes(tx.S).Bytes())
}

func (tx *Transaction) RlpEncode() []byte {
	return ethutil.Encode(tx)
}

func (tx *Transaction) RlpDecode(data []byte) {
	rlp.Decode(bytes.NewReader(data), tx)
}

func (tx *Transaction) RlpValueDecode(decoder *ethutil.Value) {
	tx.AccountNonce = decoder.Get(0).Uint()
	tx.Price = decoder.Get(1).BigInt()
	tx.GasLimit = decoder.Get(2).BigInt()
	tx.Recipient = decoder.Get(3).Bytes()
	tx.Amount = decoder.Get(4).BigInt()
	tx.Payload = decoder.Get(5).Bytes()
	tx.V = decoder.Get(6).Uint()
	tx.R = decoder.Get(7).Bytes()
	tx.S = decoder.Get(8).Bytes()
}

func (tx *Transaction) String() string {
	return fmt.Sprintf(`
	TX(%x)
	Contract: %v
	From:     %x
	To:       %x
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
		len(tx.Recipient) == 0,
		tx.From(),
		tx.To(),
		tx.AccountNonce,
		tx.Price,
		tx.GasLimit,
		tx.Amount,
		tx.Payload,
		tx.V,
		tx.R,
		tx.S,
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
	return s.Transactions[i].AccountNonce < s.Transactions[j].AccountNonce
}
