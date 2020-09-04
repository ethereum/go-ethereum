package types

import (
	"math/big"
	"time"

	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type AccessList []AccessTuple

func (al *AccessList) Addresses() int { return len(*al) }
func (al *AccessList) StorageKeys() int {
	count := 0
	for _, tuple := range *al {
		count += len(tuple.StorageKeys)
	}

	return count
}

type AccessTuple struct {
	Address     *common.Address
	StorageKeys []*common.Hash
}

type AccessListTransaction struct {
	Chain        *big.Int
	AccountNonce uint64          `json:"nonce"    gencodec:"required"`
	Price        *big.Int        `json:"gasPrice" gencodec:"required"`
	GasLimit     uint64          `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"    gencodec:"required"`
	Payload      []byte          `json:"input"    gencodec:"required"`
	Accesses     *AccessList     `json:"accessList" rlp:"nil"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
}

func NewAccessListTransaction(chainId *big.Int, nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, accesses *AccessList) *Transaction {
	return newAccessListTransaction(chainId, nonce, &to, amount, gasLimit, gasPrice, data, accesses)
}

func NewAccessListContractCreation(chainId *big.Int, nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, accesses *AccessList) *Transaction {
	return newAccessListTransaction(chainId, nonce, nil, amount, gasLimit, gasPrice, data, accesses)
}

func newAccessListTransaction(chainId *big.Int, nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, accesses *AccessList) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	i := AccessListTransaction{
		Chain:        new(big.Int),
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Accesses:     accesses,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
	}
	if chainId != nil {
		i.Chain.Set(chainId)
	}
	if amount != nil {
		i.Amount.Set(amount)
	}
	if gasPrice != nil {
		i.Price.Set(gasPrice)
	}
	return &Transaction{
		typ:   AccessListTxId,
		inner: &i,
		time:  time.Now(),
	}
}

func (tx *AccessListTransaction) ChainId() *big.Int {
	return tx.Chain
}

func (tx *AccessListTransaction) Protected() bool {
	return true
}

func (tx *AccessListTransaction) Data() []byte       { return common.CopyBytes(tx.Payload) }
func (tx *AccessListTransaction) Gas() uint64        { return tx.GasLimit }
func (tx *AccessListTransaction) GasPrice() *big.Int { return new(big.Int).Set(tx.Price) }
func (tx *AccessListTransaction) Value() *big.Int    { return new(big.Int).Set(tx.Amount) }
func (tx *AccessListTransaction) Nonce() uint64      { return tx.AccountNonce }
func (tx *AccessListTransaction) CheckNonce() bool   { return true }

func (tx *AccessListTransaction) Hash() common.Hash {
	return rlpHash(tx)
}

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *AccessListTransaction) To() *common.Address {
	if tx.Recipient == nil {
		return nil
	}
	to := *tx.Recipient
	return &to
}

func (tx *AccessListTransaction) AccessList() *AccessList {
	return tx.Accesses
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *AccessListTransaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

// MarshalJSONWithHash marshals as JSON with a hash.
func (t *AccessListTransaction) MarshalJSONWithHash(hash *common.Hash) ([]byte, error) {
	type txdata struct {
		ChainId      *hexutil.Big    `json:"chainId"    gencodec:"required"`
		AccountNonce hexutil.Uint64  `json:"nonce"    gencodec:"required"`
		Price        *hexutil.Big    `json:"gasPrice" gencodec:"required"`
		GasLimit     hexutil.Uint64  `json:"gas"      gencodec:"required"`
		Recipient    *common.Address `json:"to"       rlp:"nil"`
		Amount       *hexutil.Big    `json:"value"    gencodec:"required"`
		Payload      hexutil.Bytes   `json:"input"    gencodec:"required"`
		V            *hexutil.Big    `json:"v" gencodec:"required"`
		R            *hexutil.Big    `json:"r" gencodec:"required"`
		S            *hexutil.Big    `json:"s" gencodec:"required"`
		Hash         *common.Hash    `json:"hash" rlp:"-"`
	}

	var enc txdata

	enc.ChainId = (*hexutil.Big)(t.Chain)
	enc.AccountNonce = hexutil.Uint64(t.AccountNonce)
	enc.Price = (*hexutil.Big)(t.Price)
	enc.GasLimit = hexutil.Uint64(t.GasLimit)
	enc.Recipient = t.Recipient
	enc.Amount = (*hexutil.Big)(t.Amount)
	enc.Payload = t.Payload
	enc.V = (*hexutil.Big)(t.V)
	enc.R = (*hexutil.Big)(t.R)
	enc.S = (*hexutil.Big)(t.S)
	enc.Hash = hash

	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (t *AccessListTransaction) UnmarshalJSON(input []byte) error {
	type txdata struct {
		ChainId      *hexutil.Big    `json:"chainId"    gencodec:"required"`
		AccountNonce *hexutil.Uint64 `json:"nonce"    gencodec:"required"`
		Price        *hexutil.Big    `json:"gasPrice" gencodec:"required"`
		GasLimit     *hexutil.Uint64 `json:"gas"      gencodec:"required"`
		Recipient    *common.Address `json:"to"       rlp:"nil"`
		Amount       *hexutil.Big    `json:"value"    gencodec:"required"`
		Payload      *hexutil.Bytes  `json:"input"    gencodec:"required"`
		V            *hexutil.Big    `json:"v" gencodec:"required"`
		R            *hexutil.Big    `json:"r" gencodec:"required"`
		S            *hexutil.Big    `json:"s" gencodec:"required"`
	}
	var dec txdata
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.ChainId == nil {
		return errors.New("missing required field 'chainId' for txdata")
	}
	t.Chain = (*big.Int)(dec.ChainId)
	if dec.AccountNonce == nil {
		return errors.New("missing required field 'nonce' for txdata")
	}
	t.AccountNonce = uint64(*dec.AccountNonce)
	if dec.Price == nil {
		return errors.New("missing required field 'gasPrice' for txdata")
	}
	t.Price = (*big.Int)(dec.Price)
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gas' for txdata")
	}
	t.GasLimit = uint64(*dec.GasLimit)
	if dec.Recipient != nil {
		t.Recipient = dec.Recipient
	}
	if dec.Amount == nil {
		return errors.New("missing required field 'value' for txdata")
	}
	t.Amount = (*big.Int)(dec.Amount)
	if dec.Payload == nil {
		return errors.New("missing required field 'input' for txdata")
	}
	t.Payload = *dec.Payload
	if dec.V == nil {
		return errors.New("missing required field 'v' for txdata")
	}
	t.V = (*big.Int)(dec.V)
	if dec.R == nil {
		return errors.New("missing required field 'r' for txdata")
	}
	t.R = (*big.Int)(dec.R)
	if dec.S == nil {
		return errors.New("missing required field 's' for txdata")
	}
	t.S = (*big.Int)(dec.S)

	return nil
}
