package types

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

func IsContractAddr(addr []byte) bool {
	return len(addr) == 0
}

type Transaction struct {
	AccountNonce uint64
	Price        *big.Int
	GasLimit     *big.Int
	Recipient    *common.Address // nil means contract creation
	Amount       *big.Int
	Payload      []byte
	V            byte
	R, S         []byte
}

func NewContractCreationTx(amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	return &Transaction{Recipient: nil, Amount: amount, GasLimit: gasLimit, Price: gasPrice, Payload: data}
}

func NewTransactionMessage(to common.Address, amount, gasAmount, gasPrice *big.Int, data []byte) *Transaction {
	return &Transaction{Recipient: &to, Amount: amount, GasLimit: gasAmount, Price: gasPrice, Payload: data}
}

func NewTransactionFromBytes(data []byte) *Transaction {
	tx := new(Transaction)
	rlp.Decode(bytes.NewReader(data), tx)
	return tx
}

func (tx *Transaction) Hash() (a common.Hash) {
	h := sha3.NewKeccak256()
	rlp.Encode(h, []interface{}{tx.AccountNonce, tx.Price, tx.GasLimit, tx.Recipient, tx.Amount, tx.Payload})
	h.Sum(a[:0])
	return a
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

func (self *Transaction) From() (common.Address, error) {
	pubkey := self.PublicKey()
	if len(pubkey) == 0 || pubkey[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Sha3(pubkey[1:]))
	return addr, nil
}

// To returns the recipient of the transaction.
// If transaction is a contract creation (with no recipient address)
// To returns nil.
func (tx *Transaction) To() *common.Address {
	return tx.Recipient
}

func (tx *Transaction) Curve() (v byte, r []byte, s []byte) {
	v = byte(tx.V)
	r = common.LeftPadBytes(tx.R, 32)
	s = common.LeftPadBytes(tx.S, 32)
	return
}

func (tx *Transaction) Signature(key []byte) []byte {
	hash := tx.Hash()
	sig, _ := secp256k1.Sign(hash[:], key)
	return sig
}

func (tx *Transaction) PublicKey() []byte {
	hash := tx.Hash()
	v, r, s := tx.Curve()
	sig := append(r, s...)
	sig = append(sig, v-27)

	//pubkey := crypto.Ecrecover(append(hash, sig...))
	pubkey, _ := secp256k1.RecoverPubkey(hash[:], sig)
	return pubkey
}

func (tx *Transaction) SetSignatureValues(sig []byte) error {
	tx.R = sig[:32]
	tx.S = sig[32:64]
	tx.V = sig[64] + 27
	return nil
}

func (tx Transaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{
		tx.AccountNonce,
		tx.Price, tx.GasLimit,
		tx.Recipient,
		tx.Amount,
		tx.Payload,
		tx.V,
		tx.R,
		tx.S,
	})
}

// TODO: remove
func (tx *Transaction) RlpData() interface{} {
	data := []interface{}{tx.AccountNonce, tx.Price, tx.GasLimit, tx.Recipient, tx.Amount, tx.Payload}
	return append(data, tx.V, new(big.Int).SetBytes(tx.R).Bytes(), new(big.Int).SetBytes(tx.S).Bytes())
}

// TODO: remove
func (tx *Transaction) RlpEncode() []byte {
	return common.Encode(tx)
}

func (tx *Transaction) String() string {
	var from, to string
	if f, err := tx.From(); err != nil {
		from = "[invalid sender]"
	} else {
		from = fmt.Sprintf("%x", f[:])
	}
	if t := tx.To(); t == nil {
		to = "[contract creation]"
	} else {
		to = fmt.Sprintf("%x", t[:])
	}
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
		len(tx.Recipient) == 0,
		from,
		to,
		tx.AccountNonce,
		tx.Price,
		tx.GasLimit,
		tx.Amount,
		tx.Payload,
		tx.V,
		tx.R,
		tx.S,
		common.Encode(tx),
	)
}

// Transaction slice type for basic sorting
type Transactions []*Transaction

// TODO: remove
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
func (s Transactions) GetRlp(i int) []byte { return common.Rlp(s[i]) }

type TxByNonce struct{ Transactions }

func (s TxByNonce) Less(i, j int) bool {
	return s.Transactions[i].AccountNonce < s.Transactions[j].AccountNonce
}
